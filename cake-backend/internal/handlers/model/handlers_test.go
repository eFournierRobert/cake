package model

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	apiModels "efournierrobert/cake-backend/internal/handlers/models"
	modelService "efournierrobert/cake-backend/internal/services/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret = "test-only model jwt secret"
	testModelUUID = "3b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

type mockModelService struct {
	getAllModelsFunc          func() ([]apiModels.ModelDto, error)
	getAllActivatedModelsFunc func() ([]apiModels.ModelDto, error)
	updateModelFunc           func(string, apiModels.ModelUpdate) (apiModels.ModelDto, error)
	deleteModelFunc           func(string) error

	updateUuidArg string
	updateArg     apiModels.ModelUpdate
	deleteUuidArg string
}

var _ modelService.ModelService = (*mockModelService)(nil)

func (m *mockModelService) GetAllModels() ([]apiModels.ModelDto, error) {
	if m.getAllModelsFunc == nil {
		return nil, nil
	}
	return m.getAllModelsFunc()
}

func (m *mockModelService) GetAllActivatedModels() ([]apiModels.ModelDto, error) {
	if m.getAllActivatedModelsFunc == nil {
		return nil, nil
	}
	return m.getAllActivatedModelsFunc()
}

func (m *mockModelService) UpdateModel(uuid string, update apiModels.ModelUpdate) (apiModels.ModelDto, error) {
	m.updateUuidArg = uuid
	m.updateArg = update
	if m.updateModelFunc == nil {
		return apiModels.ModelDto{}, nil
	}
	return m.updateModelFunc(uuid, update)
}

func (m *mockModelService) DeleteModel(uuid string) error {
	m.deleteUuidArg = uuid
	if m.deleteModelFunc == nil {
		return nil
	}
	return m.deleteModelFunc(uuid)
}

func token(t *testing.T, role string) string {
	t.Helper()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  testModelUUID,
		"role": role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, err := jwtToken.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
}

func request(t *testing.T, mux http.Handler, method, path, jwtToken, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if jwtToken != "" {
		req.AddCookie(&http.Cookie{Name: "jwt-token", Value: jwtToken})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func sampleModel() apiModels.ModelDto {
	description := "test model"
	return apiModels.ModelDto{
		Uuid:            testModelUUID,
		Name:            "Test model",
		Description:     &description,
		ProviderModelId: "provider-model-id",
		ProviderUuid:    "4b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a",
		Activated:       true,
		CreatedAt:       time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
	}
}

func assertAppError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	assert.Equal(t, status, rec.Code, "body: %s", rec.Body.String())
	var appErr handler_errors.AppError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &appErr))
	assert.Equal(t, code, appErr.Code)
}

func TestModelRoutesAuthorization(t *testing.T) {
	for _, route := range []struct {
		method    string
		path      string
		adminOnly bool
	}{
		{method: http.MethodGet, path: "/models"},
		{method: http.MethodGet, path: "/models/all", adminOnly: true},
		{method: http.MethodPut, path: "/models/" + testModelUUID, adminOnly: true},
		{method: http.MethodDelete, path: "/models/" + testModelUUID, adminOnly: true},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			mock := &mockModelService{
				getAllModelsFunc: func() ([]apiModels.ModelDto, error) {
					t.Error("unexpected service call")
					return nil, nil
				},
				getAllActivatedModelsFunc: func() ([]apiModels.ModelDto, error) {
					if route.adminOnly {
						t.Error("unexpected service call")
						return nil, nil
					}
					return []apiModels.ModelDto{}, nil
				},
				updateModelFunc: func(string, apiModels.ModelUpdate) (apiModels.ModelDto, error) {
					t.Error("unexpected service call")
					return apiModels.ModelDto{}, nil
				},
				deleteModelFunc: func(string) error {
					t.Error("unexpected service call")
					return nil
				},
			}
			mux := http.NewServeMux()
			New(mock, mux)

			assert.Equal(t, http.StatusUnauthorized, request(t, mux, route.method, route.path, "", "{}").Code)
			if route.adminOnly {
				assert.Equal(t, http.StatusForbidden,
					request(t, mux, route.method, route.path, token(t, "user"), "{}").Code)
			} else {
				assert.Equal(t, http.StatusOK,
					request(t, mux, route.method, route.path, token(t, "user"), "{}").Code)
			}
		})
	}
}

func TestGetActivatedModels(t *testing.T) {
	list := []apiModels.ModelDto{sampleModel()}
	mock := &mockModelService{
		getAllActivatedModelsFunc: func() ([]apiModels.ModelDto, error) { return list, nil },
	}
	mux := http.NewServeMux()
	New(mock, mux)

	rec := request(t, mux, http.MethodGet, "/models", token(t, "user"), "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var got []apiModels.ModelDto
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, list, got)
}

func TestGetAllModels(t *testing.T) {
	list := []apiModels.ModelDto{sampleModel()}
	mock := &mockModelService{
		getAllModelsFunc: func() ([]apiModels.ModelDto, error) { return list, nil },
	}
	mux := http.NewServeMux()
	New(mock, mux)

	rec := request(t, mux, http.MethodGet, "/models/all", token(t, "admin"), "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var got []apiModels.ModelDto
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, list, got)
}

func TestPutModel(t *testing.T) {
	updated := sampleModel()
	description := "updated description"
	update := apiModels.ModelUpdate{Name: "updated", Description: &description, Activated: false}
	mock := &mockModelService{
		updateModelFunc: func(uuid string, got apiModels.ModelUpdate) (apiModels.ModelDto, error) {
			assert.Equal(t, testModelUUID, uuid)
			assert.Equal(t, update, got)
			return updated, nil
		},
	}
	mux := http.NewServeMux()
	New(mock, mux)
	body, err := json.Marshal(update)
	require.NoError(t, err)

	rec := request(t, mux, http.MethodPut, "/models/"+testModelUUID, token(t, "admin"), string(body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, testModelUUID, mock.updateUuidArg)
	assert.Equal(t, update, mock.updateArg)
	var got apiModels.ModelDto
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, updated, got)

	badJSONMock := &mockModelService{}
	badJSONMux := http.NewServeMux()
	New(badJSONMock, badJSONMux)
	assertAppError(t, request(t, badJSONMux, http.MethodPut, "/models/"+testModelUUID,
		token(t, "admin"), "{not json"), http.StatusBadRequest, "invalid_request")
	assert.Empty(t, badJSONMock.updateUuidArg)
}

func TestDeleteModel(t *testing.T) {
	mock := &mockModelService{}
	mux := http.NewServeMux()
	New(mock, mux)

	rec := request(t, mux, http.MethodDelete, "/models/"+testModelUUID, token(t, "admin"), "")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Equal(t, testModelUUID, mock.deleteUuidArg)
}

func TestModelServiceErrorsAreWritten(t *testing.T) {
	t.Run("list models", func(t *testing.T) {
		mock := &mockModelService{
			getAllActivatedModelsFunc: func() ([]apiModels.ModelDto, error) {
				return nil, handler_errors.ErrUnexpectedError
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		assertAppError(t, request(t, mux, http.MethodGet, "/models", token(t, "user"), ""),
			http.StatusInternalServerError, "unexpected_error")
	})

	t.Run("update model", func(t *testing.T) {
		mock := &mockModelService{
			updateModelFunc: func(string, apiModels.ModelUpdate) (apiModels.ModelDto, error) {
				return apiModels.ModelDto{}, handler_errors.ErrModelNotFound
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		assertAppError(t, request(t, mux, http.MethodPut, "/models/"+testModelUUID,
			token(t, "admin"), `{}`), http.StatusNotFound, "model_not_found")
	})

	t.Run("delete model", func(t *testing.T) {
		mock := &mockModelService{
			deleteModelFunc: func(string) error { return errors.New("database failure") },
		}
		mux := http.NewServeMux()
		New(mock, mux)
		assertAppError(t, request(t, mux, http.MethodDelete, "/models/"+testModelUUID,
			token(t, "admin"), ""), http.StatusInternalServerError, "unexpected_error")
	})
}
