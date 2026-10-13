package spaces

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

// TestMain starts a migrated MariaDB instance for the repository tests.
func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := mariadb.Run(ctx, "mariadb:latest",
		mariadb.WithDatabase("cake"),
		mariadb.WithUsername("cake-user"),
		mariadb.WithPassword("cake-user"),
	)
	if err != nil {
		log.Fatalf("test setup: failed to start MariaDB testcontainer: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "parseTime=true", "loc=UTC")
	if err != nil {
		_ = container.Terminate(ctx)
		log.Fatalf("test setup: failed to build DATABASE_URL from the container endpoint: %v", err)
	}
	if err := os.Setenv("DATABASE_URL", dsn); err != nil {
		_ = container.Terminate(ctx)
		log.Fatalf("test setup: failed to set DATABASE_URL: %v", err)
	}

	// Migrations are resolved relative to the module root.
	if _, statErr := os.Stat("migrations"); os.IsNotExist(statErr) {
		if err := os.Chdir("../../.."); err != nil {
			_ = container.Terminate(ctx)
			log.Fatalf("test setup: failed to chdir into the module root: %v", err)
		}
	}
	db, err := repository.NewDbConnection()
	if err != nil {
		if db != nil {
			_ = db.Close()
		}
		_ = container.Terminate(ctx)
		log.Fatalf("test setup: %v", err)
	}
	testDB = db

	code := m.Run()
	if err := db.Close(); err != nil {
		log.Printf("test teardown: failed to close database: %v", err)
	}
	if err := container.Terminate(ctx); err != nil {
		log.Printf("test teardown: failed to terminate MariaDB testcontainer: %v", err)
	}
	os.Exit(code)
}

type testOwner struct {
	id   int
	uuid uuid.UUID
}

func makeOwner(t *testing.T) testOwner {
	t.Helper()

	owner := testOwner{uuid: uuid.NewV4()}
	username := "spaces-test-" + uuid.NewV4().String()
	var roleID int
	require.NoError(t, testDB.Get(&roleID, "SELECT id FROM user_roles WHERE name = ?", "user"),
		"failed to find the seeded user role")

	result, err := testDB.Exec(`
		INSERT INTO users (
			uuid, username, password_hash, first_name, last_name,
			created_at, updated_at, role_id
		) VALUES (?, ?, ?, ?, ?, NOW(6), NOW(6), ?)
	`, owner.uuid.String(), username, []byte("not-used"), "Test", "Owner", roleID)
	require.NoError(t, err, "failed to insert test owner")
	id, err := result.LastInsertId()
	require.NoError(t, err, "failed to retrieve test owner id")
	owner.id = int(id)
	t.Cleanup(func() {
		_, _ = testDB.Exec("DELETE FROM users WHERE id = ?", owner.id)
	})

	return owner
}

func makeSpace(t *testing.T, repo *Repository, owner testOwner, name string) Space {
	t.Helper()
	created, err := repo.CreateSpace(Space{
		Uuid:         uuid.NewV4().String(),
		Name:         name,
		SystemPrompt: "Prompt for " + name,
		OwnerId:      owner.id,
	})
	require.NoError(t, err, "failed to create test space")
	return created
}

func TestGetAllUserSpacesEmpty(t *testing.T) {
	repo := New(testDB)

	all, err := repo.GetAllUserSpaces(uuid.NewV4())
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestCreateSpaceAndGetSpace(t *testing.T) {
	repo := New(testDB)
	owner := makeOwner(t)
	want := Space{
		Uuid:         uuid.NewV4().String(),
		Name:         "personal space",
		SystemPrompt: "Be helpful",
		OwnerId:      owner.id,
	}

	created, err := repo.CreateSpace(want)
	require.NoError(t, err)
	assert.Positive(t, created.ID)
	assert.Equal(t, want.Uuid, created.Uuid)
	assert.Equal(t, want.Name, created.Name)
	assert.Equal(t, want.SystemPrompt, created.SystemPrompt)
	assert.Equal(t, want.OwnerId, created.OwnerId)
	assert.False(t, created.CreatedAt.IsZero())
	assert.False(t, created.UpdatedAt.IsZero())

	got, err := repo.getSpace(uuid.MustParse(want.Uuid))
	require.NoError(t, err)
	assert.Equal(t, created, got)
}

func TestGetAllUserSpacesReturnsOnlyOwnersSpaces(t *testing.T) {
	repo := New(testDB)
	owner := makeOwner(t)
	otherOwner := makeOwner(t)
	want := makeSpace(t, repo, owner, "owned-space")
	makeSpace(t, repo, otherOwner, "other-owner-space")

	all, err := repo.GetAllUserSpaces(owner.uuid)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, want, all[0])
}

func TestUpdateSpace(t *testing.T) {
	repo := New(testDB)
	owner := makeOwner(t)
	created := makeSpace(t, repo, owner, "before-update")
	// DATETIME(6) stores microseconds; ensure the update gets a later value.
	time.Sleep(10 * time.Millisecond)

	input := created
	input.Name = "after-update"
	input.SystemPrompt = "Updated prompt"
	updated, err := repo.UpdateSpace(input)
	require.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, created.Uuid, updated.Uuid)
	assert.Equal(t, created.OwnerId, updated.OwnerId)
	assert.Equal(t, "after-update", updated.Name)
	assert.Equal(t, "Updated prompt", updated.SystemPrompt)
	assert.True(t, updated.UpdatedAt.After(created.UpdatedAt),
		"expected updated_at to advance from %v, got %v", created.UpdatedAt, updated.UpdatedAt)
}

func TestGetSpaceNotFound(t *testing.T) {
	_, err := New(testDB).getSpace(uuid.NewV4())
	assert.ErrorIs(t, err, &repo_errors.SpaceNotFound{})
}

func TestUpdateSpaceNotFound(t *testing.T) {
	missing := Space{Uuid: uuid.NewV4().String(), Name: "missing"}
	_, err := New(testDB).UpdateSpace(missing)
	assert.ErrorIs(t, err, &repo_errors.SpaceNotFound{})
}

func TestDeleteSpace(t *testing.T) {
	repo := New(testDB)
	owner := makeOwner(t)
	created := makeSpace(t, repo, owner, "to-delete")
	spaceUUID := uuid.MustParse(created.Uuid)

	require.NoError(t, repo.DeleteSpace(spaceUUID))
	_, err := repo.getSpace(spaceUUID)
	assert.ErrorIs(t, err, &repo_errors.SpaceNotFound{})
	assert.ErrorIs(t, repo.DeleteSpace(spaceUUID), &repo_errors.SpaceNotFound{})
}
