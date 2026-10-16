package conversations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	conversationService "efournierrobert/cake-backend/internal/services/conversations"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret      = "test-only conversations jwt secret"
	testOwnerUuid      = "3b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a"
	testSpaceUuid      = "5b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a"
	testConversationID = "6b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

type mockConversationService struct {
	calls []string

	getConversationsFunc    func(ownerUuid, spaceUuid string) ([]models.ConversationDto, error)
	getConversationFunc     func(ownerUuid, spaceUuid, conversationUuid string) (models.ConversationDto, error)
	updateConversationFunc  func(ownerUuid, spaceUuid, conversationUuid, title string) (models.ConversationDto, error)
	deleteConversationFunc  func(ownerUuid, spaceUuid, conversationUuid string) error
	getFullConversationFunc func(ownerUuid, spaceUuid, conversationUuid string) ([]models.ConversationMessagesDto, error)

	ownerUuid        string
	spaceUuid        string
	conversationUuid string
	title            string
}

var _ conversationService.ConversationService = (*mockConversationService)(nil)

func (m *mockConversationService) GetConversations(ownerUuid, spaceUuid string) ([]models.ConversationDto, error) {
	m.calls = append(m.calls, "GetConversations")
	m.ownerUuid, m.spaceUuid = ownerUuid, spaceUuid
	if m.getConversationsFunc == nil {
		return nil, nil
	}
	return m.getConversationsFunc(ownerUuid, spaceUuid)
}

func (m *mockConversationService) GetConversation(ownerUuid, spaceUuid, conversationUuid string) (models.ConversationDto, error) {
	m.calls = append(m.calls, "GetConversation")
	m.ownerUuid, m.spaceUuid, m.conversationUuid = ownerUuid, spaceUuid, conversationUuid
	if m.getConversationFunc == nil {
		return models.ConversationDto{}, nil
	}
	return m.getConversationFunc(ownerUuid, spaceUuid, conversationUuid)
}

func (m *mockConversationService) UpdateConversation(ownerUuid, spaceUuid, conversationUuid, title string) (models.ConversationDto, error) {
	m.calls = append(m.calls, "UpdateConversation")
	m.ownerUuid, m.spaceUuid, m.conversationUuid, m.title = ownerUuid, spaceUuid, conversationUuid, title
	if m.updateConversationFunc == nil {
		return models.ConversationDto{}, nil
	}
	return m.updateConversationFunc(ownerUuid, spaceUuid, conversationUuid, title)
}

func (m *mockConversationService) DeleteConversation(ownerUuid, spaceUuid, conversationUuid string) error {
	m.calls = append(m.calls, "DeleteConversation")
	m.ownerUuid, m.spaceUuid, m.conversationUuid = ownerUuid, spaceUuid, conversationUuid
	if m.deleteConversationFunc == nil {
		return nil
	}
	return m.deleteConversationFunc(ownerUuid, spaceUuid, conversationUuid)
}

func (m *mockConversationService) GetFullConversation(ownerUuid, spaceUuid, conversationUuid string) ([]models.ConversationMessagesDto, error) {
	m.calls = append(m.calls, "GetFullConversation")
	m.ownerUuid, m.spaceUuid, m.conversationUuid = ownerUuid, spaceUuid, conversationUuid
	if m.getFullConversationFunc == nil {
		return nil, nil
	}
	return m.getFullConversationFunc(ownerUuid, spaceUuid, conversationUuid)
}

func makeToken(t *testing.T) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  testOwnerUuid,
		"role": "user",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
}

func doRequest(t *testing.T, handler http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "jwt-token", Value: token})
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func newHandler(service conversationService.ConversationService) http.Handler {
	router := http.NewServeMux()
	New(service, router)
	return router
}

func sampleConversationDto() models.ConversationDto {
	return models.ConversationDto{
		Uuid:      testConversationID,
		Title:     "Planning",
		CreatedAt: time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
	}
}

func TestConversationEndpointsRequireAuthentication(t *testing.T) {
	service := &mockConversationService{}
	handler := newHandler(service)
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/spaces/" + testSpaceUuid + "/conversations"},
		{http.MethodGet, "/spaces/" + testSpaceUuid + "/conversations/" + testConversationID},
		{http.MethodPatch, "/spaces/" + testSpaceUuid + "/conversations/" + testConversationID},
		{http.MethodDelete, "/spaces/" + testSpaceUuid + "/conversations/" + testConversationID},
		{http.MethodGet, "/spaces/" + testSpaceUuid + "/conversations/" + testConversationID + "/messages"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			recorder := doRequest(t, handler, route.method, route.path, "", `{}`)
			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}
	assert.Empty(t, service.calls, "unauthenticated requests must not reach the service")
}

func TestConversationEndpoints(t *testing.T) {
	token := makeToken(t)
	dto := sampleConversationDto()

	t.Run("list conversations", func(t *testing.T) {
		service := &mockConversationService{
			getConversationsFunc: func(ownerUuid, spaceUuid string) ([]models.ConversationDto, error) {
				return []models.ConversationDto{dto}, nil
			},
		}
		recorder := doRequest(t, newHandler(service), http.MethodGet,
			"/spaces/"+testSpaceUuid+"/conversations", token, "")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.Equal(t, testOwnerUuid, service.ownerUuid)
		assert.Equal(t, testSpaceUuid, service.spaceUuid)
		var got []models.ConversationDto
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
		assert.Equal(t, []models.ConversationDto{dto}, got)
	})

	t.Run("get conversation", func(t *testing.T) {
		service := &mockConversationService{
			getConversationFunc: func(ownerUuid, spaceUuid, conversationUuid string) (models.ConversationDto, error) {
				return dto, nil
			},
		}
		recorder := doRequest(t, newHandler(service), http.MethodGet,
			"/spaces/"+testSpaceUuid+"/conversations/"+testConversationID, token, "")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, testOwnerUuid, service.ownerUuid)
		assert.Equal(t, testSpaceUuid, service.spaceUuid)
		assert.Equal(t, testConversationID, service.conversationUuid)
		var got models.ConversationDto
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
		assert.Equal(t, dto, got)
	})

	t.Run("update conversation", func(t *testing.T) {
		updated := dto
		updated.Title = "Updated planning"
		service := &mockConversationService{
			updateConversationFunc: func(ownerUuid, spaceUuid, conversationUuid, title string) (models.ConversationDto, error) {
				return updated, nil
			},
		}
		recorder := doRequest(t, newHandler(service), http.MethodPatch,
			"/spaces/"+testSpaceUuid+"/conversations/"+testConversationID, token,
			`{"title":"Updated planning"}`)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, testOwnerUuid, service.ownerUuid)
		assert.Equal(t, testSpaceUuid, service.spaceUuid)
		assert.Equal(t, testConversationID, service.conversationUuid)
		assert.Equal(t, "Updated planning", service.title)
		var got models.ConversationDto
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
		assert.Equal(t, updated, got)
	})

	t.Run("delete conversation", func(t *testing.T) {
		service := &mockConversationService{}
		recorder := doRequest(t, newHandler(service), http.MethodDelete,
			"/spaces/"+testSpaceUuid+"/conversations/"+testConversationID, token, "")

		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Equal(t, testOwnerUuid, service.ownerUuid)
		assert.Equal(t, testSpaceUuid, service.spaceUuid)
		assert.Equal(t, testConversationID, service.conversationUuid)
		assert.Empty(t, recorder.Body.String())
	})

	t.Run("list conversation messages", func(t *testing.T) {
		message := models.ConversationMessagesDto{
			Uuid:      "7b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a",
			Content:   "Hello",
			CreatedAt: dto.CreatedAt,
			RoleName:  "user",
		}
		service := &mockConversationService{
			getFullConversationFunc: func(ownerUuid, spaceUuid, conversationUuid string) ([]models.ConversationMessagesDto, error) {
				return []models.ConversationMessagesDto{message}, nil
			},
		}
		recorder := doRequest(t, newHandler(service), http.MethodGet,
			"/spaces/"+testSpaceUuid+"/conversations/"+testConversationID+"/messages", token, "")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, testOwnerUuid, service.ownerUuid)
		assert.Equal(t, testSpaceUuid, service.spaceUuid)
		assert.Equal(t, testConversationID, service.conversationUuid)
		var got []models.ConversationMessagesDto
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
		assert.Equal(t, []models.ConversationMessagesDto{message}, got)
	})
}

func TestConversationHandlerRejectsInvalidJSONAndForwardsServiceErrors(t *testing.T) {
	token := makeToken(t)
	path := "/spaces/" + testSpaceUuid + "/conversations/" + testConversationID

	t.Run("invalid update body", func(t *testing.T) {
		service := &mockConversationService{}
		recorder := doRequest(t, newHandler(service), http.MethodPatch, path, token, `{"title":`)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Empty(t, service.calls, "invalid JSON must not reach the service")
	})

	t.Run("service error", func(t *testing.T) {
		service := &mockConversationService{
			getConversationFunc: func(string, string, string) (models.ConversationDto, error) {
				return models.ConversationDto{}, handler_errors.ErrConvoNotFound
			},
		}
		recorder := doRequest(t, newHandler(service), http.MethodGet, path, token, "")
		assert.Equal(t, http.StatusNotFound, recorder.Code)
		var got handler_errors.AppError
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
		assert.Equal(t, "convo_not_found", got.Code)
	})
}
