package spaces

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"uuid"

	"efournierrobert/cake-backend/internal/encryptor"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/oai"
	"efournierrobert/cake-backend/internal/repository"
	messagesRepo "efournierrobert/cake-backend/internal/repository/messages"
	"efournierrobert/cake-backend/internal/repository/providers"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
)

var chatTestDB *sqlx.DB

func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := mariadb.Run(ctx, "mariadb:latest",
		mariadb.WithDatabase("cake"),
		mariadb.WithUsername("cake-user"),
		mariadb.WithPassword("cake-user"),
	)
	if err != nil {
		log.Fatalf("chat test setup: failed to start MariaDB testcontainer: %v", err)
	}
	dsn, err := container.ConnectionString(ctx, "parseTime=true", "loc=UTC")
	if err != nil {
		_ = container.Terminate(ctx)
		log.Fatalf("chat test setup: failed to build DATABASE_URL: %v", err)
	}
	if err := os.Setenv("DATABASE_URL", dsn); err != nil {
		_ = container.Terminate(ctx)
		log.Fatalf("chat test setup: failed to set DATABASE_URL: %v", err)
	}
	if err := os.Setenv("ENCRYPTION_SECRET", "0123456789abcdef0123456789abcdef"); err != nil {
		_ = container.Terminate(ctx)
		log.Fatalf("chat test setup: failed to set ENCRYPTION_SECRET: %v", err)
	}
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		if err := os.Chdir("../../.."); err != nil {
			_ = container.Terminate(ctx)
			log.Fatalf("chat test setup: failed to chdir into module root: %v", err)
		}
	}
	db, err := repository.NewDbConnection()
	if err != nil {
		if db != nil {
			_ = db.Close()
		}
		_ = container.Terminate(ctx)
		log.Fatalf("chat test setup: %v", err)
	}
	chatTestDB = db

	code := m.Run()
	if err := db.Close(); err != nil {
		log.Printf("chat test teardown: failed to close database: %v", err)
	}
	if err := container.Terminate(ctx); err != nil {
		log.Printf("chat test teardown: failed to terminate MariaDB testcontainer: %v", err)
	}
	os.Exit(code)
}

type recordedResponseRequest struct {
	body          oai.ResponseRequestBody
	authorization string
}

func TestCreateChatPersistsAndStreamsAssistantResponse(t *testing.T) {
	requests := make(chan recordedResponseRequest, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body oai.ResponseRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		requests <- recordedResponseRequest{body: body, authorization: r.Header.Get("Authorization")}
		w.Header().Set("Content-Type", "application/json")
		if !body.Stream {
			_, _ = fmt.Fprint(w, `{"output":[{"type":"message","content":[{"type":"output_text","text":"A title"}]}]}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Answer\"}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"response.completed\"}\n\n")
	}))
	t.Cleanup(server.Close)

	ownerUuid := uuid.NewV4()
	spaceUuid := uuid.NewV4()
	modelUuid := uuid.NewV4()
	var userRoleID int
	require.NoError(t, chatTestDB.Get(&userRoleID, "SELECT id FROM user_roles WHERE name = ?", "user"))
	result, err := chatTestDB.Exec(`INSERT INTO users (uuid, username, password_hash, first_name, last_name, created_at, updated_at, role_id)
		VALUES (?, ?, ?, ?, ?, NOW(6), NOW(6), ?)`, ownerUuid.String(), "chat-test-"+uuid.NewV4().String(), []byte("unused"), "Chat", "Tester", userRoleID)
	require.NoError(t, err)
	ownerID64, err := result.LastInsertId()
	require.NoError(t, err)
	ownerID := int(ownerID64)
	result, err = chatTestDB.Exec(`INSERT INTO spaces (uuid, name, system_prompt, created_at, updated_at, owner_id)
		VALUES (?, ?, ?, NOW(6), NOW(6), ?)`, spaceUuid.String(), "Chat test", "Be helpful", ownerID)
	require.NoError(t, err)
	spaceID64, err := result.LastInsertId()
	require.NoError(t, err)
	spaceID := int(spaceID64)
	provider, err := providers.New(chatTestDB).CreateProvider(providers.Provider{
		Uuid: uuid.NewV4().String(), Name: "Chat test provider", BaseUrl: server.URL,
	})
	require.NoError(t, err)
	_, err = chatTestDB.Exec(`INSERT INTO models (uuid, name, description, provider_model_id, provider_id, created_at, activated)
		VALUES (?, ?, NULL, ?, ?, NOW(6), TRUE)`, modelUuid.String(), "Chat model", "provider-chat-model", provider.Id)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = chatTestDB.Exec("DELETE FROM users WHERE id = ?", ownerID)
		_, _ = chatTestDB.Exec("DELETE FROM providers WHERE id = ?", provider.Id)
	})

	enc, err := encryptor.New()
	require.NoError(t, err)
	svc := New(chatTestDB, enc)
	chatEvents, err := svc.CreateChat(context.Background(), ownerUuid.String(), spaceUuid.String(), models.ChatCreate{
		ModelUuid: modelUuid.String(), Message: "Hello", Reasoning: "medium",
	})
	require.NoError(t, err)
	events := make([]models.ChatDto, 0, 4)
	for event := range chatEvents {
		events = append(events, event)
	}
	require.Len(t, events, 3)
	assert.Equal(t, "response.start", events[0].Event)
	assert.Equal(t, "response.output_text.delta", events[1].Event)
	require.NotNil(t, events[1].Data.Content)
	assert.Equal(t, "Answer", *events[1].Data.Content)
	assert.Equal(t, "response.completed", events[2].Event)
	require.NotNil(t, events[2].Data.MessageUuid)
	assert.NotEmpty(t, *events[2].Data.MessageUuid)
	require.NotNil(t, events[2].Data.ConversationUuid)
	conversationUuid := *events[2].Data.ConversationUuid

	titleRequest := <-requests
	assert.False(t, titleRequest.body.Stream)
	assert.Empty(t, titleRequest.authorization)
	assert.Equal(t, "none", titleRequest.body.Reasoning.Effort)
	assert.Equal(t, "Hello", titleRequest.body.Input[0].Content)
	streamRequest := <-requests
	assert.True(t, streamRequest.body.Stream)
	assert.Empty(t, streamRequest.authorization)
	assert.Equal(t, "Be helpful", streamRequest.body.Instructions)
	assert.Equal(t, "medium", streamRequest.body.Reasoning.Effort)
	assert.Equal(t, []oai.ResponseInput{{Role: "user", Content: "Hello"}}, streamRequest.body.Input)

	var conversationID int
	require.NoError(t, chatTestDB.Get(&conversationID, "SELECT id FROM conversations WHERE uuid = ?", conversationUuid))
	wholeConversation, err := messagesRepo.New(chatTestDB).GetWholeConversation(conversationID)
	require.NoError(t, err)
	require.Len(t, wholeConversation, 2)
	assert.Equal(t, "user", wholeConversation[0].RoleName)
	assert.Equal(t, "Hello", wholeConversation[0].Content)
	assert.Equal(t, "assistant", wholeConversation[1].RoleName)
	assert.Equal(t, "Answer", wholeConversation[1].Content)
	var title string
	require.NoError(t, chatTestDB.Get(&title, "SELECT title FROM conversations WHERE id = ?", conversationID))
	assert.Equal(t, "A title", title)

	chatEvents, err = svc.CreateChat(context.Background(), ownerUuid.String(), spaceUuid.String(), models.ChatCreate{
		ConversationUuid: &conversationUuid,
		ModelUuid:        modelUuid.String(),
		Message:          "Again",
	})
	require.NoError(t, err)
	for event := range chatEvents {
		if event.Event == "response.completed" {
			break
		}
	}
	continuedRequest := <-requests
	assert.True(t, continuedRequest.body.Stream)
	assert.Equal(t, "Be helpful", continuedRequest.body.Instructions)
	assert.Equal(t, []oai.ResponseInput{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Answer"},
		{Role: "user", Content: "Again"},
	}, continuedRequest.body.Input)
	var conversationCount int
	require.NoError(t, chatTestDB.Get(&conversationCount, "SELECT COUNT(*) FROM conversations WHERE owner_id = ? AND space_id = ?", ownerID, spaceID))
	assert.Equal(t, 1, conversationCount, "continuing a conversation must not create another conversation")
	wholeConversation, err = messagesRepo.New(chatTestDB).GetWholeConversation(conversationID)
	require.NoError(t, err)
	require.Len(t, wholeConversation, 4)
	assert.Equal(t, "Again", wholeConversation[2].Content)
	assert.Equal(t, "assistant", wholeConversation[3].RoleName)
}
