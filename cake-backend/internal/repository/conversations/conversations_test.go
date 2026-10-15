package conversations

import (
	"context"
	"log"
	"os"
	"testing"
	"uuid"

	"efournierrobert/cake-backend/internal/repository"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
)

var testDB *sqlx.DB

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
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		if err := os.Chdir("../../.."); err != nil {
			_ = container.Terminate(ctx)
			log.Fatalf("test setup: failed to chdir into module root: %v", err)
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

type testScope struct {
	ownerUuid uuid.UUID
	ownerID   int
	spaceUuid uuid.UUID
	spaceID   int
}

func makeTestScope(t *testing.T) testScope {
	t.Helper()
	scope := testScope{ownerUuid: uuid.NewV4(), spaceUuid: uuid.NewV4()}
	var roleID int
	require.NoError(t, testDB.Get(&roleID, "SELECT id FROM user_roles WHERE name = ?", "user"))
	result, err := testDB.Exec(`INSERT INTO users (uuid, username, password_hash, first_name, last_name, created_at, updated_at, role_id)
		VALUES (?, ?, ?, ?, ?, NOW(6), NOW(6), ?)`, scope.ownerUuid.String(), "conversations-test-"+uuid.NewV4().String(), []byte("unused"), "Test", "Owner", roleID)
	require.NoError(t, err)
	ownerID, err := result.LastInsertId()
	require.NoError(t, err)
	scope.ownerID = int(ownerID)
	result, err = testDB.Exec(`INSERT INTO spaces (uuid, name, system_prompt, created_at, updated_at, owner_id)
		VALUES (?, ?, ?, NOW(6), NOW(6), ?)`, scope.spaceUuid.String(), "test space", "prompt", scope.ownerID)
	require.NoError(t, err)
	spaceID, err := result.LastInsertId()
	require.NoError(t, err)
	scope.spaceID = int(spaceID)
	t.Cleanup(func() { _, _ = testDB.Exec("DELETE FROM users WHERE id = ?", scope.ownerID) })
	return scope
}

func TestCreateAndGetConversation(t *testing.T) {
	repo := New(testDB)
	scope := makeTestScope(t)
	want := Conversation{Uuid: uuid.NewV4().String(), OwnerId: scope.ownerID, Title: "A test conversation", SpaceId: scope.spaceID}

	created, err := repo.CreateConversation(scope.ownerUuid, scope.spaceUuid, want)
	require.NoError(t, err)
	assert.Positive(t, created.ID)
	assert.Equal(t, want.Uuid, created.Uuid)
	assert.Equal(t, want.OwnerId, created.OwnerId)
	assert.Equal(t, want.Title, created.Title)
	assert.Equal(t, want.SpaceId, created.SpaceId)
	assert.False(t, created.CreatedAt.IsZero())

	got, err := repo.GetConversation(scope.ownerUuid, scope.spaceUuid, uuid.MustParse(want.Uuid))
	require.NoError(t, err)
	assert.Equal(t, created, got)
}

func TestGetConversationIsScopedToOwnerAndSpace(t *testing.T) {
	repo := New(testDB)
	scope := makeTestScope(t)
	otherScope := makeTestScope(t)
	created, err := repo.CreateConversation(scope.ownerUuid, scope.spaceUuid, Conversation{
		Uuid: uuid.NewV4().String(), OwnerId: scope.ownerID, Title: "Scoped", SpaceId: scope.spaceID,
	})
	require.NoError(t, err)
	conversationUuid := uuid.MustParse(created.Uuid)

	_, err = repo.GetConversation(otherScope.ownerUuid, scope.spaceUuid, conversationUuid)
	assert.ErrorIs(t, err, &repo_errors.ConversationNotFound{})
	_, err = repo.GetConversation(scope.ownerUuid, otherScope.spaceUuid, conversationUuid)
	assert.ErrorIs(t, err, &repo_errors.ConversationNotFound{})
	_, err = repo.GetConversation(scope.ownerUuid, scope.spaceUuid, uuid.NewV4())
	assert.ErrorIs(t, err, &repo_errors.ConversationNotFound{})
	assert.False(t, created.CreatedAt.IsZero())
}
