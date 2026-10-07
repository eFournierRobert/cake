package providers

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
	"uuid"

	"efournierrobert/cake-backend/internal/repository"
	"efournierrobert/cake-backend/internal/repository/repo_errors"

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
	os.Setenv("DATABASE_URL", dsn)

	// NewDbConnection resolves "migrations" relative to the working
	// directory, which is this package's directory. Cd to the
	// repository root so the migrations directory is found, no matter
	// where the test binary is launched from.
	if _, statErr := os.Stat("migrations"); os.IsNotExist(statErr) {
		// This package sits three levels below the module root.
		if err := os.Chdir("../../.."); err != nil {
			log.Fatalf("test setup: failed to chdir into the module root: %v", err)
		}
	}

	db, err := repository.NewDbConnection()
	if err != nil {
		if db != nil {
			db.Close()
		}
		log.Fatalf("test setup: %v", err)
	}
	defer db.Close()

	testDB = db

	code := m.Run()
	os.Exit(code)
}

// makeProvider builds a provider with a fresh random uuid, ready to
// be created or used as a foreign key reference.
func makeProvider(t *testing.T, name string) Provider {
	t.Helper()
	return Provider{
		Uuid:    uuid.NewV4().String(),
		Name:    name,
		BaseUrl: "https://api." + name + ".example.com",
		ApiKey:  []byte("key-" + name),
	}
}

// insertModel inserts a model row referencing the provider directly
// through the test database, since no model repository exists yet.
// It returns the model row's id.
func insertModel(t *testing.T, provider Provider) int {
	t.Helper()

	var modelID int
	_, err := testDB.Exec(
		"INSERT INTO models (uuid, name, context_length, provider_model_id, provider_id, created_at) VALUES (?, ?, ?, ?, ?, NOW())",
		uuid.NewV4().String(), "gpt-test", 4096, "provider-model-id", provider.Id,
	)
	require.NoError(t, err, "failed to insert a model referencing the provider")
	err = testDB.QueryRow(
		"SELECT id FROM models WHERE provider_id = ? AND provider_model_id = ?", provider.Id, "provider-model-id",
	).Scan(&modelID)
	require.NoError(t, err, "failed to look up the inserted model")

	return modelID
}

// countRows runs the given query and returns the single-column count
// it produces.
func countRows(t *testing.T, query string, args ...any) int {
	t.Helper()
	var count int
	err := testDB.QueryRow(query, args...).Scan(&count)
	require.NoError(t, err, "failed to run %q", query)
	return count
}

// A fresh migrated database contains no providers: GetAllProviders
// must return an empty list, not an error. The name is ordered before
// the other tests so it runs against the empty table — tests in this
// binary share one migrated database.
func TestGetAllProvidersEmpty(t *testing.T) {
	repo := New(testDB)

	all, err := repo.GetAllProviders()
	require.NoError(t, err, "expected GetAllProviders to succeed on an empty providers table")
	assert.Empty(t, all, "expected no providers before anything is created")
}

func TestCreateProvider(t *testing.T) {
	repo := New(testDB)
	provider := makeProvider(t, "create-provider")

	created, err := repo.CreateProvider(provider)
	require.NoError(t, err, "expected CreateProvider to succeed")

	assert.Positive(t, created.Id, "expected a positive auto-generated id")
	assert.Equal(t, provider.Uuid, created.Uuid)
	assert.Equal(t, provider.Name, created.Name)
	assert.Equal(t, provider.BaseUrl, created.BaseUrl)
	assert.Equal(t, provider.ApiKey, created.ApiKey)
	assert.False(t, created.CreatedAt.IsZero(), "expected created_at to be populated")
	assert.False(t, created.UpdatedAt.IsZero(), "expected updated_at to be populated")
}

func TestGetProviderNotFound(t *testing.T) {
	repo := New(testDB)
	missing := uuid.NewV4()

	_, err := repo.GetProvider(missing)
	assert.ErrorIs(t, err, &repo_errors.ProviderNotFound{}, "expected *repo_errors.ProviderNotFound for a missing uuid")
}

func TestUpdateProvider(t *testing.T) {
	repo := New(testDB)
	provider := makeProvider(t, "update-provider")
	created, err := repo.CreateProvider(provider)
	require.NoError(t, err, "expected CreateProvider to succeed")

	// DATETIME(6) stores microseconds, so updated_at is guaranteed to
	// move forward as soon as any measurable time passes; the sleep
	// only guards against the two measurements landing on the same
	// microsecond.
	time.Sleep(10 * time.Millisecond)

	updated := created
	updated.Name = "update-provider-renamed"
	updated.BaseUrl = "https://api.renamed.example.com"

	result, err := repo.UpdateProvider(updated)
	require.NoError(t, err, "expected UpdateProvider to succeed")

	assert.Positive(t, result.Id, "expected a positive auto-generated id")
	assert.Equal(t, created.Id, result.Id, "expected the id to be unchanged")
	assert.Equal(t, "update-provider-renamed", result.Name, "expected the updated name to be persisted")
	assert.Equal(t, "https://api.renamed.example.com", result.BaseUrl, "expected the updated base_url to be persisted")
	assert.Equal(t, created.ApiKey, result.ApiKey, "expected untouched fields to be preserved")
	assert.False(t, result.UpdatedAt.IsZero(), "expected updated_at to be populated")
	assert.True(t, result.UpdatedAt.After(result.CreatedAt),
		"expected updated_at (%v) to be after created_at (%v)", result.UpdatedAt, result.CreatedAt)
	assert.NotEqual(t, result.CreatedAt.UnixMicro(), result.UpdatedAt.UnixMicro(),
		"expected updated_at (%v) to differ from created_at (%v)", result.UpdatedAt, result.CreatedAt)

	_, err = repo.UpdateProvider(makeProvider(t, "ghost-provider"))
	assert.ErrorIs(t, err, &repo_errors.ProviderNotFound{}, "expected *repo_errors.ProviderNotFound when updating a missing provider")
}

func TestDeleteProvider(t *testing.T) {
	repo := New(testDB)
	provider := makeProvider(t, "delete-provider")
	created, err := repo.CreateProvider(provider)
	require.NoError(t, err, "expected CreateProvider to succeed")

	require.NoError(t, repo.DeleteProvider(uuid.MustParse(created.Uuid)), "expected DeleteProvider to succeed")

	_, err = repo.GetProvider(uuid.MustParse(created.Uuid))
	assert.ErrorIs(t, err, &repo_errors.ProviderNotFound{}, "expected *repo_errors.ProviderNotFound after deletion")

	err = repo.DeleteProvider(uuid.MustParse(created.Uuid))
	assert.ErrorIs(t, err, &repo_errors.ProviderNotFound{}, "expected *repo_errors.ProviderNotFound when deleting an already deleted provider")
}

func TestDeleteProviderCascadesModels(t *testing.T) {
	repo := New(testDB)
	provider := makeProvider(t, "cascade-provider")
	created, err := repo.CreateProvider(provider)
	require.NoError(t, err, "expected CreateProvider to succeed")
	modelID := insertModel(t, created)
	t.Cleanup(func() {
		_, _ = testDB.Exec("DELETE FROM models WHERE id = ?", modelID)
		_, _ = testDB.Exec("DELETE FROM providers WHERE id = ?", created.Id)
	})

	assert.Equal(t, 1, countRows(t, "SELECT COUNT(*) FROM models WHERE provider_id = ?", created.Id),
		"expected 1 model referencing the provider before deletion")

	require.NoError(t, repo.DeleteProvider(uuid.MustParse(created.Uuid)), "expected DeleteProvider to succeed")

	assert.Equal(t, 0, countRows(t, "SELECT COUNT(*) FROM models WHERE provider_id = ?", created.Id),
		"expected the referencing models to be cascade deleted")
}

func TestGetAllProvidersOrdering(t *testing.T) {
	repo := New(testDB)
	first := makeProvider(t, "oldest-provider")
	createdFirst, err := repo.CreateProvider(first)
	require.NoError(t, err, "expected CreateProvider to succeed")

	second := makeProvider(t, "newest-provider")
	createdSecond, err := repo.CreateProvider(second)
	require.NoError(t, err, "expected CreateProvider to succeed")

	all, err := repo.GetAllProviders()
	require.NoError(t, err, "expected GetAllProviders to succeed")

	firstIndex, secondIndex := -1, -1
	for i, p := range all {
		switch p.Uuid {
		case first.Uuid:
			firstIndex = i
		case second.Uuid:
			secondIndex = i
		}
	}
	require.NotEqual(t, -1, firstIndex, "expected the oldest provider to be present in GetAllProviders")
	require.NotEqual(t, -1, secondIndex, "expected the newest provider to be present in GetAllProviders")
	assert.True(t, createdSecond.CreatedAt.After(createdFirst.CreatedAt),
		"expected the second provider to have been created after the first (%v vs %v)",
		createdSecond.CreatedAt, createdFirst.CreatedAt)
	assert.Less(t, secondIndex, firstIndex,
		"expected newer providers to come first; newest-provider at index %d, oldest-provider at index %d", secondIndex, firstIndex)
}

func TestGetAllProvidersContent(t *testing.T) {
	repo := New(testDB)

	first := makeProvider(t, "content-provider")
	second := makeProvider(t, "content-provider-two")
	createdFirst, err := repo.CreateProvider(first)
	require.NoError(t, err, "expected CreateProvider to succeed")
	createdSecond, err := repo.CreateProvider(second)
	require.NoError(t, err, "expected CreateProvider to succeed")
	t.Cleanup(func() {
		_, _ = testDB.Exec("DELETE FROM providers WHERE id = ?", createdFirst.Id)
		_, _ = testDB.Exec("DELETE FROM providers WHERE id = ?", createdSecond.Id)
	})

	all, err := repo.GetAllProviders()
	require.NoError(t, err, "expected GetAllProviders to succeed")

	byName := map[string]Provider{}
	for _, p := range all {
		byName[p.Name] = p
	}

	firstEntry, ok := byName["content-provider"]
	require.True(t, ok, "expected content-provider to be present in GetAllProviders")
	assert.Equal(t, createdFirst.Uuid, firstEntry.Uuid)
	assert.Equal(t, createdFirst.Id, firstEntry.Id, "expected the auto-generated id to be returned")
	assert.Equal(t, "content-provider", firstEntry.Name)
	assert.Equal(t, first.BaseUrl, firstEntry.BaseUrl)
	assert.Equal(t, first.ApiKey, firstEntry.ApiKey)
	assert.False(t, firstEntry.CreatedAt.IsZero(), "expected created_at to be populated")

	secondEntry, ok := byName["content-provider-two"]
	require.True(t, ok, "expected content-provider-two to be present in GetAllProviders")
	assert.Equal(t, createdSecond.Uuid, secondEntry.Uuid)
	assert.Equal(t, createdSecond.Id, secondEntry.Id, "expected the auto-generated id to be returned")
	assert.Equal(t, "content-provider-two", secondEntry.Name)
	assert.Equal(t, second.BaseUrl, secondEntry.BaseUrl)
	assert.Equal(t, second.ApiKey, secondEntry.ApiKey)
	assert.False(t, secondEntry.CreatedAt.IsZero(), "expected created_at to be populated")
}
