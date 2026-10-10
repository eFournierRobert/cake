package models

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"
	"uuid"

	"efournierrobert/cake-backend/internal/repository"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
)

var testDB *sqlx.DB

// TestMain starts a fresh MariaDB container, points DATABASE_URL at
// it and opens the connection through repository.NewDbConnection so
// all pending migrations run, exactly like the application does.
// testDB holds that migrated connection for the whole test binary.
func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := mariadb.Run(ctx, "mariadb:latest",
		mariadb.WithDatabase("cake"),
		mariadb.WithUsername("cake-user"),
		mariadb.WithPassword("cake-user"),
	)
	if err != nil {
		log.Fatalf("test setup: failed to start the MariaDB testcontainer: %v", err)
	}
	defer container.Terminate(ctx)

	// The MariaDB server runs in UTC; set loc so the driver decodes
	// DATETIME columns into the same instant time.Now() measures in
	// the host's timezone.
	dsn, err := container.ConnectionString(ctx, "parseTime=true", "loc=UTC")
	if err != nil {
		log.Fatalf("test setup: failed to build the DATABASE_URL from the container endpoint: %v", err)
	}
	if err := os.Setenv("DATABASE_URL", dsn); err != nil {
		log.Fatalf("test setup: failed to set DATABASE_URL: %v", err)
	}

	// NewDbConnection resolves "migrations" relative to the working
	// directory. Move to the module root so the migrations are found
	// regardless of where the test binary is launched from.
	if _, statErr := os.Stat("migrations"); os.IsNotExist(statErr) {
		if err := os.Chdir("../../.."); err != nil {
			log.Fatalf("test setup: failed to chdir into the module root: %v", err)
		}
	}

	db, err := repository.NewDbConnection()
	if err != nil {
		if db != nil {
			_ = db.Close()
		}
		log.Fatalf("test setup: %v", err)
	}
	defer db.Close()

	testDB = db
	code := m.Run()
	os.Exit(code)
}

// makeModel builds a model with a fresh provider row to satisfy the
// models.provider_id foreign key. The provider and its model rows are
// removed together during cleanup.
func makeModel(t *testing.T, name, providerModelID string) Model {
	t.Helper()

	providerUUID := uuid.NewV4().String()
	result, err := testDB.Exec(
		`INSERT INTO providers (uuid, name, base_url, created_at, updated_at)
		 VALUES (?, ?, ?, NOW(), NOW())`,
		providerUUID, "provider-"+name, "https://api."+name+".example.com",
	)
	require.NoError(t, err, "failed to insert a provider for the model")

	providerID, err := result.LastInsertId()
	require.NoError(t, err, "failed to get the inserted provider id")
	t.Cleanup(func() {
		_, _ = testDB.Exec("DELETE FROM providers WHERE id = ?", providerID)
	})

	return Model{
		Uuid:            uuid.NewV4().String(),
		Name:            name,
		Description:     sqlNullString("Description for " + name),
		ContextLength:   8192,
		ProviderModelId: providerModelID,
		ProviderId:      int(providerID),
	}
}

func sqlNullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: true}
}

func countModels(t *testing.T, query string, args ...any) int {
	t.Helper()
	var count int
	err := testDB.QueryRow(query, args...).Scan(&count)
	require.NoError(t, err, "failed to run %q", query)
	return count
}

// A fresh migrated database contains no models: GetAllModels must
// return an empty list, not an error. This test runs first because the
// package tests share one database.
func TestGetAllModelsEmpty(t *testing.T) {
	repo := New(testDB)

	all, err := repo.GetAllModels()
	require.NoError(t, err, "expected GetAllModels to succeed on an empty models table")
	assert.Empty(t, all, "expected no models before anything is created")
}

func TestGetAllActivatedModelsEmpty(t *testing.T) {
	repo := New(testDB)

	all, err := repo.GetAllActivatedModels()
	require.NoError(t, err, "expected GetAllActivatedModels to succeed when no models are activated")
	assert.Empty(t, all, "expected no activated models before anything is created")
}

func TestCreateModel(t *testing.T) {
	repo := New(testDB)
	model := makeModel(t, "create-model", "provider-model-create")

	require.NoError(t, repo.CreateModel(model), "expected CreateModel to succeed")

	var created Model
	err := testDB.Get(&created, "SELECT * FROM models WHERE uuid = ?", model.Uuid)
	require.NoError(t, err, "expected the created model to be persisted")

	assert.Positive(t, created.Id, "expected a positive auto-generated id")
	assert.Equal(t, model.Uuid, created.Uuid)
	assert.Equal(t, model.Name, created.Name)
	assert.Equal(t, model.Description, created.Description)
	assert.Equal(t, model.ContextLength, created.ContextLength)
	assert.Equal(t, model.ProviderModelId, created.ProviderModelId)
	assert.Equal(t, model.ProviderId, created.ProviderId)
	assert.False(t, created.CreatedAt.IsZero(), "expected created_at to be populated")
	assert.False(t, created.Activated, "expected new models to use the database's inactive default")
}

func TestCreateModelWithNullDescription(t *testing.T) {
	repo := New(testDB)
	model := makeModel(t, "null-description-model", "provider-model-null-description")
	model.Description = sql.NullString{}

	require.NoError(t, repo.CreateModel(model), "expected CreateModel to accept a null description")

	var created Model
	err := testDB.Get(&created, "SELECT * FROM models WHERE uuid = ?", model.Uuid)
	require.NoError(t, err, "expected the created model to be persisted")
	assert.False(t, created.Description.Valid, "expected description to remain NULL")
}

func TestGetAllModelsContentAndOrdering(t *testing.T) {
	repo := New(testDB)
	older := makeModel(t, "older-model", "provider-model-older")
	require.NoError(t, repo.CreateModel(older), "expected CreateModel to create the older model")

	// The repository sets created_at from time.Now(); allow it to advance
	// beyond the DATETIME(6) microsecond precision before creating the next row.
	time.Sleep(10 * time.Millisecond)

	newer := makeModel(t, "newer-model", "provider-model-newer")
	require.NoError(t, repo.CreateModel(newer), "expected CreateModel to create the newer model")

	all, err := repo.GetAllModels()
	require.NoError(t, err, "expected GetAllModels to succeed")
	require.Len(t, all, 2, "expected GetAllModels to return the two fixture models")

	byUUID := make(map[string]Model, len(all))
	for _, model := range all {
		byUUID[model.Uuid] = model
	}
	olderEntry, ok := byUUID[older.Uuid]
	require.True(t, ok, "expected the older model to be present in GetAllModels")
	newerEntry, ok := byUUID[newer.Uuid]
	require.True(t, ok, "expected the newer model to be present in GetAllModels")

	assert.Equal(t, older.Name, olderEntry.Name)
	assert.Equal(t, older.Description, olderEntry.Description)
	assert.Equal(t, older.ContextLength, olderEntry.ContextLength)
	assert.Equal(t, older.ProviderModelId, olderEntry.ProviderModelId)
	assert.Equal(t, older.ProviderId, olderEntry.ProviderId)
	assert.Positive(t, olderEntry.Id, "expected the auto-generated id to be returned")
	assert.False(t, olderEntry.CreatedAt.IsZero(), "expected created_at to be returned")

	assert.True(t, newerEntry.CreatedAt.After(olderEntry.CreatedAt),
		"expected newer model's created_at (%v) to be after older model's created_at (%v)",
		newerEntry.CreatedAt, olderEntry.CreatedAt)
	assert.Equal(t, newer.Uuid, all[0].Uuid, "expected models to be ordered newest first")
	assert.Equal(t, older.Uuid, all[1].Uuid, "expected the older model to follow the newer model")
}

func TestGetAllActivatedModelsFiltersInactiveModels(t *testing.T) {
	repo := New(testDB)
	active := makeModel(t, "active-model", "provider-model-active")
	inactive := makeModel(t, "inactive-model", "provider-model-inactive")
	require.NoError(t, repo.CreateModel(active), "expected CreateModel to create the active model")
	require.NoError(t, repo.CreateModel(inactive), "expected CreateModel to create the inactive model")

	_, err := testDB.Exec("UPDATE models SET activated = TRUE WHERE uuid = ?", active.Uuid)
	require.NoError(t, err, "failed to activate the fixture model")

	all, err := repo.GetAllActivatedModels()
	require.NoError(t, err, "expected GetAllActivatedModels to succeed")
	require.Len(t, all, 1, "expected only activated models to be returned")
	assert.Equal(t, active.Uuid, all[0].Uuid)
	assert.True(t, all[0].Activated)
}

func TestUpdateModel(t *testing.T) {
	repo := New(testDB)
	model := makeModel(t, "update-model", "provider-model-before")
	require.NoError(t, repo.CreateModel(model), "expected CreateModel to create the model before update")

	model.Name = "updated-model"
	model.Description = sqlNullString("Updated description")
	model.ProviderModelId = "provider-model-after"
	model.Activated = true
	require.NoError(t, repo.UpdateModel(model), "expected UpdateModel to succeed")

	var updated Model
	err := testDB.Get(&updated, "SELECT * FROM models WHERE uuid = ?", model.Uuid)
	require.NoError(t, err, "expected the updated model to remain persisted")
	assert.Equal(t, model.Name, updated.Name)
	assert.Equal(t, model.Description, updated.Description)
	assert.Equal(t, model.ProviderModelId, updated.ProviderModelId)
	assert.Equal(t, model.Activated, updated.Activated)
	assert.Equal(t, uint32(8192), updated.ContextLength, "expected untouched context_length to be preserved")
	assert.Equal(t, model.ProviderId, updated.ProviderId, "expected untouched provider_id to be preserved")
}

func TestDeleteModel(t *testing.T) {
	repo := New(testDB)
	model := makeModel(t, "delete-model", "provider-model-delete")
	require.NoError(t, repo.CreateModel(model), "expected CreateModel to create the model before deletion")

	require.NoError(t, repo.DeleteModel(uuid.MustParse(model.Uuid)), "expected model deletion to succeed")
	assert.Equal(t, 0, countModels(t, "SELECT COUNT(*) FROM models WHERE uuid = ?", model.Uuid),
		"expected the model row to be removed")
}
