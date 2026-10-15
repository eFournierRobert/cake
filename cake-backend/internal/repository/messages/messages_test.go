package messages

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

type testConversation struct {
	id int
}

func makeTestConversation(t *testing.T) testConversation {
	t.Helper()
	var userRoleID int
	require.NoError(t, testDB.Get(&userRoleID, "SELECT id FROM user_roles WHERE name = ?", "user"))
	ownerUuid := uuid.NewV4()
	result, err := testDB.Exec(`INSERT INTO users (uuid, username, password_hash, first_name, last_name, created_at, updated_at, role_id)
		VALUES (?, ?, ?, ?, ?, NOW(6), NOW(6), ?)`, ownerUuid.String(), "messages-test-"+uuid.NewV4().String(), []byte("unused"), "Test", "Owner", userRoleID)
	require.NoError(t, err)
	ownerID64, err := result.LastInsertId()
	require.NoError(t, err)
	ownerID := int(ownerID64)

	spaceUuid := uuid.NewV4()
	result, err = testDB.Exec(`INSERT INTO spaces (uuid, name, system_prompt, created_at, updated_at, owner_id)
		VALUES (?, ?, ?, NOW(6), NOW(6), ?)`, spaceUuid.String(), "test space", "prompt", ownerID)
	require.NoError(t, err)
	spaceID64, err := result.LastInsertId()
	require.NoError(t, err)
	spaceID := int(spaceID64)

	result, err = testDB.Exec(`INSERT INTO conversations (uuid, owner_id, title, created_at, space_id)
		VALUES (?, ?, ?, NOW(6), ?)`, uuid.NewV4().String(), ownerID, "test conversation", spaceID)
	require.NoError(t, err)
	conversationID64, err := result.LastInsertId()
	require.NoError(t, err)
	conversationID := int(conversationID64)
	t.Cleanup(func() { _, _ = testDB.Exec("DELETE FROM users WHERE id = ?", ownerID) })
	return testConversation{id: conversationID}
}

func TestCreateMessageAndGetWholeConversation(t *testing.T) {
	repo := New(testDB)
	conversation := makeTestConversation(t)
	userRole, err := repo.GetRole("user")
	require.NoError(t, err)
	assistantRole, err := repo.GetRole("assistant")
	require.NoError(t, err)

	userMessage := Message{Uuid: uuid.NewV4().String(), Content: "Hello", ConversationId: conversation.id, RoleId: userRole.ID}
	require.NoError(t, repo.CreateMessage(userMessage))
	// Deliberately separate timestamps so chronological ordering is deterministic.
	time.Sleep(10 * time.Millisecond)
	assistantMessage := Message{Uuid: uuid.NewV4().String(), Content: "Hi there", ConversationId: conversation.id, RoleId: assistantRole.ID}
	require.NoError(t, repo.CreateMessage(assistantMessage))

	got, err := repo.GetWholeConversation(conversation.id)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, userMessage.Uuid, got[0].Uuid)
	assert.Equal(t, "Hello", got[0].Content)
	assert.Equal(t, "user", got[0].RoleName)
	assert.Equal(t, assistantMessage.Uuid, got[1].Uuid)
	assert.Equal(t, "Hi there", got[1].Content)
	assert.Equal(t, "assistant", got[1].RoleName)
	assert.True(t, got[0].CreatedAt.Before(got[1].CreatedAt))
}

func TestMessageRoleNotFoundMapsToRepositoryError(t *testing.T) {
	_, err := New(testDB).GetRole("missing-role")
	var dbErr *repo_errors.InternalDbError
	assert.ErrorAs(t, err, &dbErr)
}
