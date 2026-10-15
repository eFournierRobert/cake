package spaces

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	spaceService "efournierrobert/cake-backend/internal/services/spaces"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret = "test-only spaces jwt secret"
	testOwnerUuid = "3b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a"
	testSpaceUuid = "5b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

type mockSpaceService struct {
	calls []string

	getAllSpacesFunc func(ownerUuid string) ([]models.SpaceDto, error)
	getAllOwnerArg   string

	getSpaceFunc     func(spaceUuid, ownerUuid string) (models.SpaceDto, error)
	getSpaceUuidArg  string
	getSpaceOwnerArg string

	createSpaceFunc func(ownerUuid string, request models.SpaceCreate) (models.SpaceDto, error)
	createOwnerArg  string
	createRequest   models.SpaceCreate

	updateSpaceFunc func(ownerUuid, spaceUuid string, request models.SpaceUpdate) (models.SpaceDto, error)
	updateOwnerArg  string
	updateSpaceArg  string
	updateRequest   models.SpaceUpdate

	deleteSpaceFunc func(ownerUuid, spaceUuid string) error
	deleteOwnerArg  string
	deleteSpaceArg  string

	createChatFunc     func(ctx context.Context, ownerUuid, spaceUuid string, request models.ChatCreate) (<-chan models.ChatDto, error)
	createChatOwnerArg string
	createChatSpaceArg string
	createChatRequest  models.ChatCreate
}

var _ spaceService.SpaceService = (*mockSpaceService)(nil)

func (m *mockSpaceService) GetAllSpaces(ownerUuid string) ([]models.SpaceDto, error) {
	m.calls = append(m.calls, "GetAllSpaces")
	m.getAllOwnerArg = ownerUuid
	if m.getAllSpacesFunc == nil {
		return nil, nil
	}
	return m.getAllSpacesFunc(ownerUuid)
}

func (m *mockSpaceService) GetSpace(spaceUuid, ownerUuid string) (models.SpaceDto, error) {
	m.calls = append(m.calls, "GetSpace")
	m.getSpaceUuidArg = spaceUuid
	m.getSpaceOwnerArg = ownerUuid
	if m.getSpaceFunc == nil {
		return models.SpaceDto{}, nil
	}
	return m.getSpaceFunc(spaceUuid, ownerUuid)
}

func (m *mockSpaceService) CreateSpace(ownerUuid string, request models.SpaceCreate) (models.SpaceDto, error) {
	m.calls = append(m.calls, "CreateSpace")
	m.createOwnerArg = ownerUuid
	m.createRequest = request
	if m.createSpaceFunc == nil {
		return models.SpaceDto{}, nil
	}
	return m.createSpaceFunc(ownerUuid, request)
}

func (m *mockSpaceService) UpdateSpace(ownerUuid, spaceUuid string, request models.SpaceUpdate) (models.SpaceDto, error) {
	m.calls = append(m.calls, "UpdateSpace")
	m.updateOwnerArg = ownerUuid
	m.updateSpaceArg = spaceUuid
	m.updateRequest = request
	if m.updateSpaceFunc == nil {
		return models.SpaceDto{}, nil
	}
	return m.updateSpaceFunc(ownerUuid, spaceUuid, request)
}

func (m *mockSpaceService) DeleteSpace(ownerUuid, spaceUuid string) error {
	m.calls = append(m.calls, "DeleteSpace")
	m.deleteOwnerArg = ownerUuid
	m.deleteSpaceArg = spaceUuid
	if m.deleteSpaceFunc == nil {
		return nil
	}
	return m.deleteSpaceFunc(ownerUuid, spaceUuid)
}

func (m *mockSpaceService) CreateChat(ctx context.Context, ownerUuid, spaceUuid string, request models.ChatCreate) (<-chan models.ChatDto, error) {
	m.calls = append(m.calls, "CreateChat")
	m.createChatOwnerArg = ownerUuid
	m.createChatSpaceArg = spaceUuid
	m.createChatRequest = request
	if m.createChatFunc == nil {
		return nil, nil
	}
	return m.createChatFunc(ctx, ownerUuid, spaceUuid, request)
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

func newHandler(service spaceService.SpaceService) http.Handler {
	router := http.NewServeMux()
	New(service, router)
	return router
}

func sampleSpaceDto() models.SpaceDto {
	return models.SpaceDto{
		Uuid:         testSpaceUuid,
		Name:         "Research",
		SystemPrompt: "Be concise",
		CreatedAt:    time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 10, 2, 11, 30, 0, 0, time.UTC),
	}
}

func TestSpaceEndpointsRequireAuthentication(t *testing.T) {
	service := &mockSpaceService{}
	handler := newHandler(service)
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/spaces"},
		{http.MethodPost, "/spaces"},
		{http.MethodGet, "/spaces/" + testSpaceUuid},
		{http.MethodPatch, "/spaces/" + testSpaceUuid},
		{http.MethodDelete, "/spaces/" + testSpaceUuid},
		{http.MethodPost, "/spaces/" + testSpaceUuid + "/chat"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			recorder := doRequest(t, handler, route.method, route.path, "", `{}`)
			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}
	assert.Empty(t, service.calls, "unauthenticated requests must not reach the service")
}

func TestSpaceEndpoints(t *testing.T) {
	token := makeToken(t)
	dto := sampleSpaceDto()

	t.Run("list spaces", func(t *testing.T) {
		service := &mockSpaceService{
			getAllSpacesFunc: func(ownerUuid string) ([]models.SpaceDto, error) {
				return []models.SpaceDto{dto}, nil
			},
		}
		recorder := doRequest(t, newHandler(service), http.MethodGet, "/spaces", token, "")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.Equal(t, testOwnerUuid, service.getAllOwnerArg)
		var got []models.SpaceDto
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
		assert.Equal(t, []models.SpaceDto{dto}, got)
	})

	t.Run("get a space", func(t *testing.T) {
		service := &mockSpaceService{
			getSpaceFunc: func(spaceUuid, ownerUuid string) (models.SpaceDto, error) {
				return dto, nil
			},
		}
		recorder := doRequest(t, newHandler(service), http.MethodGet, "/spaces/"+testSpaceUuid, token, "")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, testSpaceUuid, service.getSpaceUuidArg)
		assert.Equal(t, testOwnerUuid, service.getSpaceOwnerArg)
		var got models.SpaceDto
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
		assert.Equal(t, dto, got)
	})

	t.Run("create a space", func(t *testing.T) {
		service := &mockSpaceService{
			createSpaceFunc: func(ownerUuid string, request models.SpaceCreate) (models.SpaceDto, error) {
				return dto, nil
			},
		}
		recorder := doRequest(t, newHandler(service), http.MethodPost, "/spaces", token,
			`{"name":"Research","system_prompt":"Be concise"}`)

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, testOwnerUuid, service.createOwnerArg)
		assert.Equal(t, models.SpaceCreate{Name: "Research", SystemPrompt: "Be concise"}, service.createRequest)
		var got models.SpaceDto
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
		assert.Equal(t, dto, got)
	})

	t.Run("update a space", func(t *testing.T) {
		service := &mockSpaceService{
			updateSpaceFunc: func(ownerUuid, spaceUuid string, request models.SpaceUpdate) (models.SpaceDto, error) {
				return dto, nil
			},
		}
		recorder := doRequest(t, newHandler(service), http.MethodPatch, "/spaces/"+testSpaceUuid, token,
			`{"name":"Updated"}`)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, testOwnerUuid, service.updateOwnerArg)
		assert.Equal(t, testSpaceUuid, service.updateSpaceArg)
		require.NotNil(t, service.updateRequest.Name)
		assert.Equal(t, "Updated", *service.updateRequest.Name)
		assert.Nil(t, service.updateRequest.SystemPrompt)
		var got models.SpaceDto
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
		assert.Equal(t, dto, got)
	})

	t.Run("delete a space", func(t *testing.T) {
		service := &mockSpaceService{}
		recorder := doRequest(t, newHandler(service), http.MethodDelete, "/spaces/"+testSpaceUuid, token, "")

		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Empty(t, recorder.Body.String())
		assert.Equal(t, testOwnerUuid, service.deleteOwnerArg)
		assert.Equal(t, testSpaceUuid, service.deleteSpaceArg)
	})
}

func TestSpaceHandlersRejectMalformedJSON(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/spaces"},
		{http.MethodPatch, "/spaces/" + testSpaceUuid},
		{http.MethodPost, "/spaces/" + testSpaceUuid + "/chat"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			service := &mockSpaceService{}
			recorder := doRequest(t, newHandler(service), route.method, route.path, makeToken(t), "{")
			assertError(t, recorder, http.StatusBadRequest, "invalid_request")
			assert.Empty(t, service.calls, "malformed requests must not reach the service")
		})
	}
}

func TestChatHandlerStreamsServiceEvents(t *testing.T) {
	conversationUuid := "5b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a"
	messageUuid := "7b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a"
	delta := "Hello"
	service := &mockSpaceService{
		createChatFunc: func(_ context.Context, _, _ string, _ models.ChatCreate) (<-chan models.ChatDto, error) {
			ch := make(chan models.ChatDto, 3)
			ch <- models.ChatDto{Event: "response.start"}
			ch <- models.ChatDto{Event: "response.output_text.delta", Data: models.ChatDtoData{Content: &delta, ConversationUuid: &conversationUuid}}
			ch <- models.ChatDto{Event: "response.completed", Data: models.ChatDtoData{MessageUuid: &messageUuid, ConversationUuid: &conversationUuid}}
			close(ch)
			return ch, nil
		},
	}

	body := `{"conversation_uuid":"` + conversationUuid + `","model_uuid":"` + messageUuid + `","message":"Hi","reasoning":"medium"}`
	recorder := doRequest(t, newHandler(service), http.MethodPost, "/spaces/"+testSpaceUuid+"/chat", makeToken(t), body)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", recorder.Header().Get("Cache-Control"))
	assert.Equal(t, testOwnerUuid, service.createChatOwnerArg)
	assert.Equal(t, testSpaceUuid, service.createChatSpaceArg)
	assert.Equal(t, models.ChatCreate{ConversationUuid: &conversationUuid, ModelUuid: messageUuid, Message: "Hi", Reasoning: "medium"}, service.createChatRequest)
	assert.Contains(t, recorder.Body.String(), "event: response.start\ndata:")
	assert.Contains(t, recorder.Body.String(), "event: response.output_text.delta\ndata:")
	assert.Contains(t, recorder.Body.String(), "\"content\":\"Hello\"")
	assert.Contains(t, recorder.Body.String(), "event: response.completed\ndata:")
	assert.Contains(t, recorder.Body.String(), "\"message_uuid\":\""+messageUuid+"\"")
}

func TestChatHandlerReturnsServiceErrorBeforeStreaming(t *testing.T) {
	service := &mockSpaceService{
		createChatFunc: func(context.Context, string, string, models.ChatCreate) (<-chan models.ChatDto, error) {
			return nil, handler_errors.ErrModelNotFound
		},
	}
	body := `{"model_uuid":"7b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a","message":"Hi"}`
	recorder := doRequest(t, newHandler(service), http.MethodPost, "/spaces/"+testSpaceUuid+"/chat", makeToken(t), body)

	assertError(t, recorder, http.StatusNotFound, "model_not_found")
	assert.NotEqual(t, "text/event-stream", recorder.Header().Get("Content-Type"))
}

func TestSpaceHandlerPropagatesServiceErrors(t *testing.T) {
	service := &mockSpaceService{
		getSpaceFunc: func(spaceUuid, ownerUuid string) (models.SpaceDto, error) {
			return models.SpaceDto{}, handler_errors.ErrSpaceNotFound
		},
	}
	recorder := doRequest(t, newHandler(service), http.MethodGet, "/spaces/"+testSpaceUuid, makeToken(t), "")

	assertError(t, recorder, http.StatusNotFound, "space_not_found")
}

func TestSpaceHandlerMapsUnknownServiceErrors(t *testing.T) {
	service := &mockSpaceService{
		getAllSpacesFunc: func(ownerUuid string) ([]models.SpaceDto, error) {
			return nil, errors.New("database failure")
		},
	}
	recorder := doRequest(t, newHandler(service), http.MethodGet, "/spaces", makeToken(t), "")

	assertError(t, recorder, http.StatusInternalServerError, "unexpected_error")
}

func assertError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	assert.Equal(t, status, recorder.Code, "response body: %s", recorder.Body.String())
	var got handler_errors.AppError
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
	assert.Equal(t, code, got.Code)
}
