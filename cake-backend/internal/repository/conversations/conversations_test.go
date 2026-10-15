package conversations

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

func createTestConversation(t *testing.T, repo *Repository, scope testScope, title string) Conversation {
	t.Helper()
	conversation, err := repo.CreateConversation(scope.ownerUuid, scope.spaceUuid, Conversation{
		Uuid:    uuid.NewV4().String(),
		OwnerId: scope.ownerID,
		Title:   title,
		SpaceId: scope.spaceID,
	})
	require.NoError(t, err)
	return conversation
}

func insertTestMessage(t *testing.T, conversationID int, createdAt time.Time) {
	t.Helper()
	var roleID int
	require.NoError(t, testDB.Get(&roleID, "SELECT id FROM message_roles WHERE name = ?", "user"))
	_, err := testDB.Exec(`INSERT INTO messages (uuid, content, created_at, conversation_id, role_id)
		VALUES (?, ?, ?, ?, ?)`, uuid.NewV4().String(), "test message", createdAt, conversationID, roleID)
	require.NoError(t, err)
}

func TestGetSpaceConversationsIncludesEmptyConversationsOnceAndOrdersByLatestMessage(t *testing.T) {
	repo := New(testDB)
	scope := makeTestScope(t)
	otherScope := makeTestScope(t)

	olderActivity := createTestConversation(t, repo, scope, "older activity")
	newerActivity := createTestConversation(t, repo, scope, "newer activity")
	empty := createTestConversation(t, repo, scope, "no messages")
	otherOwnerConversation := createTestConversation(t, repo, otherScope, "other owner")

	base := time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC)
	insertTestMessage(t, olderActivity.ID, base.Add(1*time.Minute))
	insertTestMessage(t, olderActivity.ID, base.Add(3*time.Minute))
	insertTestMessage(t, newerActivity.ID, base.Add(2*time.Minute))
	insertTestMessage(t, otherOwnerConversation.ID, base.Add(4*time.Minute))

	got, err := repo.GetSpaceConversations(scope.ownerUuid, scope.spaceUuid)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, []string{olderActivity.Uuid, newerActivity.Uuid, empty.Uuid}, []string{
		got[0].Uuid,
		got[1].Uuid,
		got[2].Uuid,
	})

	wrongSpace, err := repo.GetSpaceConversations(scope.ownerUuid, otherScope.spaceUuid)
	require.NoError(t, err)
	assert.Empty(t, wrongSpace)
}

func TestUpdateConversationUpdatesTitleAndRespectsScope(t *testing.T) {
	repo := New(testDB)
	scope := makeTestScope(t)
	otherScope := makeTestScope(t)
	created := createTestConversation(t, repo, scope, "original title")

	created.Title = "updated title"
	updated, err := repo.UpdateConversation(scope.ownerUuid, scope.spaceUuid, created)
	require.NoError(t, err)
	assert.Equal(t, "updated title", updated.Title)
	assert.Equal(t, created.Uuid, updated.Uuid)

	_, err = repo.UpdateConversation(otherScope.ownerUuid, scope.spaceUuid, created)
	assert.ErrorIs(t, err, &repo_errors.ConversationNotFound{})

	unchanged, err := repo.GetConversation(scope.ownerUuid, scope.spaceUuid, uuid.MustParse(created.Uuid))
	require.NoError(t, err)
	assert.Equal(t, "updated title", unchanged.Title)
}

func TestDeleteConversationDeletesWithinScopeAndCascadesMessages(t *testing.T) {
	repo := New(testDB)
	scope := makeTestScope(t)
	otherScope := makeTestScope(t)
	created := createTestConversation(t, repo, scope, "delete me")
	insertTestMessage(t, created.ID, time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC))

	err := repo.DeleteConversation(otherScope.ownerUuid, scope.spaceUuid, created)
	assert.ErrorIs(t, err, &repo_errors.ConversationNotFound{})

	err = repo.DeleteConversation(scope.ownerUuid, scope.spaceUuid, created)
	require.NoError(t, err)
	_, err = repo.GetConversation(scope.ownerUuid, scope.spaceUuid, uuid.MustParse(created.Uuid))
	assert.ErrorIs(t, err, &repo_errors.ConversationNotFound{})

	var messageCount int
	require.NoError(t, testDB.Get(&messageCount, "SELECT COUNT(*) FROM messages WHERE conversation_id = ?", created.ID))
	assert.Zero(t, messageCount)
}

func TestDeleteConversationReturnsNotFoundWhenMissingOrOutOfScope(t *testing.T) {
	repo := New(testDB)
	scope := makeTestScope(t)
	otherScope := makeTestScope(t)
	created := createTestConversation(t, repo, scope, "scoped")

	err := repo.DeleteConversation(scope.ownerUuid, otherScope.spaceUuid, created)
	assert.ErrorIs(t, err, &repo_errors.ConversationNotFound{})

	err = repo.DeleteConversation(scope.ownerUuid, scope.spaceUuid, Conversation{Uuid: uuid.NewV4().String()})
	assert.ErrorIs(t, err, &repo_errors.ConversationNotFound{})

	_, err = repo.GetConversation(scope.ownerUuid, scope.spaceUuid, uuid.MustParse(created.Uuid))
	assert.NoError(t, err)
}
