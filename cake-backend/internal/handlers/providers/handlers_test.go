package providers

import (
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	providerService "efournierrobert/cake-backend/internal/services/providers"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret = "test-only provider jwt secret"
	testUuid      = "3b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

func testToken(t *testing.T, role string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  testUuid,
		"role": role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
}

func providerDto() models.ProviderDto {
	return models.ProviderDto{
		Uuid:      testUuid,
		Name:      "Example provider",
		BaseUrl:   "https://provider.example.com/v1",
		HasApiKey: true,
		CreatedAt: time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 9, 2, 11, 30, 0, 0, time.UTC),
	}
}

func doRequest(t *testing.T, mux http.Handler, method, path, token string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "jwt-token", Value: token})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	assert.Equal(t, status, rec.Code, "response body: %s", rec.Body.String())
	var got handler_errors.AppError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, code, got.Code)
}

func TestProviderEndpointsRequireAdminAuth(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/providers"},
		{http.MethodPost, "/providers"},
		{http.MethodGet, "/providers/" + testUuid},
		{http.MethodPut, "/providers/" + testUuid},
		{http.MethodDelete, "/providers/" + testUuid},
		{http.MethodPost, "/providers/" + testUuid + "/test"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			mock := &providerService.MockService{
				GetAllProvidersFunc: func() ([]models.ProviderDto, error) {
					t.Error("unexpected service call")
					return nil, nil
				},
				GetProviderFunc: func(string) (models.ProviderDto, error) {
					t.Error("unexpected service call")
					return models.ProviderDto{}, nil
				},
				CreateProviderFunc: func(models.ProviderCreate) (models.ProviderDto, error) {
					t.Error("unexpected service call")
					return models.ProviderDto{}, nil
				},
				ModifyProviderFunc: func(string, models.ProviderUpdate) (models.ProviderDto, error) {
					t.Error("unexpected service call")
					return models.ProviderDto{}, nil
				},
				DeleteProviderFunc: func(string) error { t.Error("unexpected service call"); return nil },
				TestProviderFunc: func(string) (models.ProviderTestResponse, error) {
					t.Error("unexpected service call")
					return models.ProviderTestResponse{}, nil
				},
			}
			mux := http.NewServeMux()
			New(mock, mux)

			assert.Equal(t, http.StatusUnauthorized, doRequest(t, mux, route.method, route.path, "", "{}").Code)
			assert.Equal(t, http.StatusForbidden, doRequest(t, mux, route.method, route.path, testToken(t, "user"), "{}").Code)
		})
	}
}

func TestGetProviders(t *testing.T) {
	list := []models.ProviderDto{providerDto()}
	mock := &providerService.MockService{
		GetAllProvidersFunc: func() ([]models.ProviderDto, error) { return list, nil },
	}
	mux := http.NewServeMux()
	New(mock, mux)

	rec := doRequest(t, mux, http.MethodGet, "/providers", testToken(t, "admin"), "")
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var got []models.ProviderDto
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, list, got)
}

func TestPostProvider(t *testing.T) {
	dto := providerDto()
	create := models.ProviderCreate{Name: "Example provider", BaseUrl: "https://provider.example.com/v1", ApiKey: "secret"}
	mock := &providerService.MockService{
		CreateProviderFunc: func(got models.ProviderCreate) (models.ProviderDto, error) {
			assert.Equal(t, create, got)
			return dto, nil
		},
	}
	mux := http.NewServeMux()
	New(mock, mux)
	body, err := json.Marshal(create)
	require.NoError(t, err)

	rec := doRequest(t, mux, http.MethodPost, "/providers", testToken(t, "admin"), string(body))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var got models.ProviderDto
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, dto, got)

	t.Run("malformed JSON is rejected", func(t *testing.T) {
		mock := &providerService.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)
		assertError(t, doRequest(t, mux, http.MethodPost, "/providers", testToken(t, "admin"), "{not json"), http.StatusBadRequest, "invalid_request")
		assert.Empty(t, mock.CreateProviderArg)
	})
}

func TestGetProvider(t *testing.T) {
	dto := providerDto()
	mock := &providerService.MockService{
		GetProviderFunc: func(uuid string) (models.ProviderDto, error) {
			assert.Equal(t, testUuid, uuid)
			return dto, nil
		},
	}
	mux := http.NewServeMux()
	New(mock, mux)

	rec := doRequest(t, mux, http.MethodGet, "/providers/"+testUuid, testToken(t, "admin"), "")
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var got models.ProviderDto
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, dto, got)
}

func TestPutProvider(t *testing.T) {
	dto := providerDto()
	apiKey := "replacement-secret"
	update := models.ProviderUpdate{Name: "Updated provider", BaseUrl: "https://new.example.com", ApiKey: &apiKey}
	mock := &providerService.MockService{
		ModifyProviderFunc: func(uuid string, got models.ProviderUpdate) (models.ProviderDto, error) {
			assert.Equal(t, testUuid, uuid)
			assert.Equal(t, update, got)
			return dto, nil
		},
	}
	mux := http.NewServeMux()
	New(mock, mux)
	body, err := json.Marshal(update)
	require.NoError(t, err)

	rec := doRequest(t, mux, http.MethodPut, "/providers/"+testUuid, testToken(t, "admin"), string(body))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var got models.ProviderDto
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, dto, got)

	t.Run("malformed JSON is rejected", func(t *testing.T) {
		mock := &providerService.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)
		assertError(t, doRequest(t, mux, http.MethodPut, "/providers/"+testUuid, testToken(t, "admin"), "{not json"), http.StatusBadRequest, "invalid_request")
		assert.Empty(t, mock.ModifyProviderUuidArg)
	})
}

func TestDeleteProvider(t *testing.T) {
	mock := &providerService.MockService{
		DeleteProviderFunc: func(uuid string) error {
			assert.Equal(t, testUuid, uuid)
			return nil
		},
	}
	mux := http.NewServeMux()
	New(mock, mux)

	rec := doRequest(t, mux, http.MethodDelete, "/providers/"+testUuid, testToken(t, "admin"), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestPostProviderTest(t *testing.T) {
	result := models.ProviderTestResponse{Success: false, Error: "provider returned HTTP 503"}
	mock := &providerService.MockService{
		TestProviderFunc: func(uuid string) (models.ProviderTestResponse, error) {
			assert.Equal(t, testUuid, uuid)
			return result, nil
		},
	}
	mux := http.NewServeMux()
	New(mock, mux)

	rec := doRequest(t, mux, http.MethodPost, "/providers/"+testUuid+"/test", testToken(t, "admin"), "")
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var got models.ProviderTestResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, result, got)
}

func TestProviderServiceErrorsAreWritten(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		code string
	}{
		{name: "known", err: handler_errors.ErrProviderNotFound, code: "provider_not_found"},
		{name: "unexpected", err: errors.New("unexpected failure"), code: "unexpected_error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mock := &providerService.MockService{
				GetProviderFunc: func(string) (models.ProviderDto, error) { return models.ProviderDto{}, test.err },
			}
			mux := http.NewServeMux()
			New(mock, mux)
			rec := doRequest(t, mux, http.MethodGet, "/providers/"+testUuid, testToken(t, "admin"), "")
			assertError(t, rec, func() int {
				if test.code == "provider_not_found" {
					return http.StatusNotFound
				}
				return http.StatusInternalServerError
			}(), test.code)
		})
	}
}
