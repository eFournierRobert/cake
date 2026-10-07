package users

import (
	"efournierrobert/cake-backend/internal/handlers"
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	mock "efournierrobert/cake-backend/internal/services/users"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testJWTSecret is the secret the auth middleware expects in
// JWT_SECRET for every test in this binary.
const testJWTSecret = "test-only jwt secret"

// testUuid is a fixed, parseable uuid used in request paths.
const testUuid = "3b2418f9-8c2d-4b7a-9e51-2c6a0f4d1e8a"

func TestMain(m *testing.M) {
	// The auth middleware reads JWT_SECRET from the environment on
	// every request; set it once for the whole test binary.
	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

// testToken signs a real HS256 jwt with the given subject and role so
// the auth middleware accepts it.
func testToken(t *testing.T, sub, role string, exp time.Time) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"iat":  time.Now().Unix(),
		"exp":  exp.Unix(),
	})
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err, "failed to sign the test token")
	return signed
}

// testDto builds a UserDto with fixed, non-zero timestamps.
func testDto(uuid, username, role, firstName, lastName string) handlers.UserDto {
	return handlers.UserDto{
		Uuid:        uuid,
		Username:    username,
		Role:        role,
		FirstName:   firstName,
		LastName:    lastName,
		CreatedAt:   time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
		LastUpdated: time.Date(2026, 9, 2, 11, 30, 0, 0, time.UTC),
	}
}

// doRequest runs one request through the router, attaching the given
// token as the jwt-token cookie when it is non-empty and a JSON body
// when one is provided.
func doRequest(t *testing.T, mux http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err, "failed to marshal the test request body")
		bodyReader = strings.NewReader(string(data))
	} else {
		bodyReader = strings.NewReader("")
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "jwt-token", Value: token})
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// assertError checks the response carries the expected status and an
// AppError body with the expected error code.
func assertError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	assert.Equal(t, wantStatus, rec.Code, "expected status %d, got %d; body: %s", wantStatus, rec.Code, rec.Body.String())
	var appErr handler_errors.AppError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &appErr), "expected the response body to be an AppError, got: %s", rec.Body.String())
	assert.Equal(t, wantCode, appErr.Code, "expected the AppError code to be %q", wantCode)
}

// jwtCookie returns the jwt-token cookie from the response, if present.
func jwtCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "jwt-token" {
			return c
		}
	}
	return nil
}

func TestLogin(t *testing.T) {
	t.Run("valid credentials set the jwt-token cookie", func(t *testing.T) {
		mock := &mock.MockService{
			LoginFunc: func(username, password string) (string, error) {
				return "mock-issued-token", nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		rec := doRequest(t, mux, http.MethodPost, "/login", "", handlers.LoginRequest{
			Username: "ada",
			Password: "sup3r-s3cure-pass!",
		})

		require.Equal(t, http.StatusOK, rec.Code, "expected login to succeed; body: %s", rec.Body.String())
		assert.Equal(t, "", rec.Body.String(), "expected no body on login success, the token lives in the cookie")

		cookie := jwtCookie(rec)
		require.NotNil(t, cookie, "expected the jwt-token cookie to be set")
		assert.Equal(t, "mock-issued-token", cookie.Value)
		assert.Equal(t, "/", cookie.Path)
		assert.True(t, cookie.HttpOnly)
		assert.Equal(t, 60*60, cookie.MaxAge, "expected the cookie to live for one hour")

		assert.Equal(t, "ada", mock.LoginUsernameArg, "expected the handler to forward the username to the service")
		assert.Equal(t, "sup3r-s3cure-pass!", mock.LoginPasswordArg, "expected the handler to forward the password to the service")
	})

	t.Run("malformed JSON body", func(t *testing.T) {
		mock := &mock.MockService{
			LoginFunc: func(username, password string) (string, error) {
				t.Error("expected the login service method not to be called for an undecodable body")
				return "token", nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("{not json"))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assertError(t, rec, http.StatusBadRequest, "invalid_request")
	})

	t.Run("service error is mapped to invalid credentials", func(t *testing.T) {
		mock := &mock.MockService{
			LoginFunc: func(username, password string) (string, error) {
				return "", handler_errors.ErrInvalidPassword
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		rec := doRequest(t, mux, http.MethodPost, "/login", "", handlers.LoginRequest{
			Username: "ada",
			Password: "wrong-password",
		})

		assertError(t, rec, http.StatusUnauthorized, "invalid_credentials")
	})
}

func TestLogout(t *testing.T) {
	mux := http.NewServeMux()
	New(&mock.MockService{}, mux)

	rec := doRequest(t, mux, http.MethodPost, "/logout", "", nil)
	require.Equal(t, http.StatusOK, rec.Code, "expected logout to succeed")

	cookie := jwtCookie(rec)
	require.NotNil(t, cookie, "expected the jwt-token cookie to be reset")
	assert.Equal(t, "", cookie.Value, "expected the cookie value to be emptied")
	assert.Equal(t, -1, cookie.MaxAge, "expected the cookie to be expired")
	assert.True(t, cookie.HttpOnly)
}

func TestGetCurrentUser(t *testing.T) {
	t.Run("returns the current user's profile", func(t *testing.T) {
		dto := testDto(testUuid, "ada", "user", "Ada", "Lovelace")
		mock := &mock.MockService{
			GetUserFunc: func(strUuid string) (handlers.UserDto, error) {
				return dto, nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		rec := doRequest(t, mux, http.MethodGet, "/user", testToken(t, testUuid, "user", time.Now().Add(time.Hour)), nil)

		require.Equal(t, http.StatusOK, rec.Code, "expected the profile to load; body: %s", rec.Body.String())
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		var got handlers.UserDto
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, dto, got, "expected the decoded profile to match the service dto")

		assert.Equal(t, testUuid, mock.GetUserArg, "expected the handler to use the token subject as the user uuid")
	})

	t.Run("service error maps to the matching status", func(t *testing.T) {
		mock := &mock.MockService{
			GetUserFunc: func(strUuid string) (handlers.UserDto, error) {
				return handlers.UserDto{}, handler_errors.ErrUserDoesNotExist
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		rec := doRequest(t, mux, http.MethodGet, "/user", testToken(t, testUuid, "user", time.Now().Add(time.Hour)), nil)
		assertError(t, rec, http.StatusNotFound, "user_does_not_exist")
	})

	t.Run("no cookie", func(t *testing.T) {
		mock := &mock.MockService{
			GetUserFunc: func(strUuid string) (handlers.UserDto, error) {
				t.Error("expected the service not to be called without a valid token")
				return handlers.UserDto{}, nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodGet, "/user", "", nil)
		// The middleware rejects with a plain-text 401, not an
		// AppError body.
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "", mock.GetUserArg, "expected no getUser call without a cookie")
	})

	t.Run("garbage token", func(t *testing.T) {
		mock := &mock.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodGet, "/user", "not.a.token", nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "", mock.GetUserArg, "expected no service call for a garbage token")
	})

	t.Run("expired token", func(t *testing.T) {
		mock := &mock.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)
		expired := testToken(t, testUuid, "user", time.Now().Add(-time.Hour))
		rec := doRequest(t, mux, http.MethodGet, "/user", expired, nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "expected an expired token to be rejected")
		assert.Equal(t, "", mock.GetUserArg, "expected no service call for an expired token")
	})
}

func TestPatchCurrentUser(t *testing.T) {
	t.Run("updates the provided fields", func(t *testing.T) {
		dto := testDto(testUuid, "augusta", "user", "Augusta", "King")
		mock := &mock.MockService{
			ModifyUserFunc: func(currentUserUuid string, update handlers.UserUpdate) (handlers.UserDto, error) {
				return dto, nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		update := handlers.UserUpdate{FirstName: "Augusta", LastName: "King"}
		rec := doRequest(t, mux, http.MethodPatch, "/user", testToken(t, testUuid, "user", time.Now().Add(time.Hour)), update)

		require.Equal(t, http.StatusOK, rec.Code, "expected the update to succeed; body: %s", rec.Body.String())
		var got handlers.UserDto
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, dto, got)

		assert.Equal(t, testUuid, mock.ModifyUserUuidArg, "expected the token subject to be forwarded")
		assert.Equal(t, update, mock.ModifyUserUpdateArg, "expected the decoded update payload to be forwarded")
	})

	t.Run("malformed JSON body", func(t *testing.T) {
		mock := &mock.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)

		req := httptest.NewRequest(http.MethodPatch, "/user", strings.NewReader("{not json"))
		req.AddCookie(&http.Cookie{Name: "jwt-token", Value: testToken(t, testUuid, "user", time.Now().Add(time.Hour))})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assertError(t, rec, http.StatusBadRequest, "invalid_request")
	})

	t.Run("service conflict", func(t *testing.T) {
		mock := &mock.MockService{
			ModifyUserFunc: func(string, handlers.UserUpdate) (handlers.UserDto, error) {
				return handlers.UserDto{}, handler_errors.ErrResourceConflict
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPatch, "/user", testToken(t, testUuid, "user", time.Now().Add(time.Hour)), handlers.UserUpdate{Username: "taken"})
		assertError(t, rec, http.StatusConflict, "resource_conflict")
	})

	t.Run("no cookie", func(t *testing.T) {
		mock := &mock.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPatch, "/user", "", handlers.UserUpdate{FirstName: "X"})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Empty(t, mock.ModifyUserUuidArg, "expected no modify call without a cookie")
	})
}

func TestPostCurrentUserPassword(t *testing.T) {
	t.Run("updates the password", func(t *testing.T) {
		mock := &mock.MockService{
			ChangePasswordFunc: func(currentUserUuid string, newPassword string) error {
				return nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		rec := doRequest(t, mux, http.MethodPost, "/user/password", testToken(t, testUuid, "user", time.Now().Add(time.Hour)),
			handlers.PasswordChangeRequest{Password: "new-new-new-pass"})

		require.Equal(t, http.StatusOK, rec.Code, "expected the password change to succeed; body: %s", rec.Body.String())
		assert.Empty(t, rec.Body.String(), "expected no body on success")
		assert.Equal(t, testUuid, mock.ChangePasswordUuidArg, "expected the token subject to be forwarded")
		assert.Equal(t, "new-new-new-pass", mock.ChangePasswordNewArg, "expected the new password to be forwarded")
	})

	t.Run("malformed JSON body", func(t *testing.T) {
		mock := &mock.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)

		req := httptest.NewRequest(http.MethodPost, "/user/password", strings.NewReader("{not json"))
		req.AddCookie(&http.Cookie{Name: "jwt-token", Value: testToken(t, testUuid, "user", time.Now().Add(time.Hour))})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assertError(t, rec, http.StatusBadRequest, "invalid_request")
	})

	t.Run("service reports an invalid password", func(t *testing.T) {
		mock := &mock.MockService{
			ChangePasswordFunc: func(string, string) error {
				return handler_errors.ErrInvalidPassword
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPost, "/user/password", testToken(t, testUuid, "user", time.Now().Add(time.Hour)),
			handlers.PasswordChangeRequest{Password: "short"})
		assertError(t, rec, http.StatusUnauthorized, "invalid_credentials")
	})

	t.Run("service reports a missing user", func(t *testing.T) {
		mock := &mock.MockService{
			ChangePasswordFunc: func(string, string) error {
				return handler_errors.ErrUserDoesNotExist
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPost, "/user/password", testToken(t, testUuid, "user", time.Now().Add(time.Hour)),
			handlers.PasswordChangeRequest{Password: "valid-new-pass-1"})
		assertError(t, rec, http.StatusNotFound, "user_does_not_exist")
	})

	t.Run("no cookie", func(t *testing.T) {
		mock := &mock.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPost, "/user/password", "", handlers.PasswordChangeRequest{Password: "valid-new-pass-1"})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Empty(t, mock.ChangePasswordUuidArg, "expected no password change call without a cookie")
	})
}

// The six admin routes must be wrapped by RequireAdminAuth: every
// request without a valid token is rejected with 401, and a valid
// token for a non-admin user with 403 — both before any service
// method runs.
func TestAdminEndpointsRequireAuth(t *testing.T) {
	userToken := testToken(t, testUuid, "user", time.Now().Add(time.Hour))

	rows := []struct {
		method        string
		path          string
		serviceMethod string
	}{
		{http.MethodGet, "/users", "GetAllUsers"},
		{http.MethodPost, "/users", "CreateUser"},
		{http.MethodGet, "/users/" + testUuid, "GetUser"},
		{http.MethodPatch, "/users/" + testUuid, "AdminModifyUser"},
		{http.MethodDelete, "/users/" + testUuid, "DeleteUser"},
		{http.MethodPost, "/users/" + testUuid + "/password", "ChangePassword"},
	}

	for _, row := range rows {
		t.Run(row.method+" "+row.path, func(t *testing.T) {
			mock := &mock.MockService{
				GetUserFunc: func(string) (handlers.UserDto, error) {
					t.Error("unexpected GetUser call in the auth gate test")
					return handlers.UserDto{}, nil
				},
				GetAllUsersFunc: func() ([]handlers.UserDto, error) {
					t.Error("unexpected GetAllUsers call in the auth gate test")
					return nil, nil
				},
				CreateUserFunc: func(handlers.UserCreate) (handlers.UserDto, error) {
					t.Error("unexpected CreateUser call in the auth gate test")
					return handlers.UserDto{}, nil
				},
				AdminModifyUserFunc: func(string, handlers.AdminUserUpdate) (handlers.UserDto, error) {
					t.Error("unexpected AdminModifyUser call in the auth gate test")
					return handlers.UserDto{}, nil
				},
				DeleteUserFunc: func(string) error { t.Error("unexpected DeleteUser call in the auth gate test"); return nil },
				ChangePasswordFunc: func(string, string) error {
					t.Error("unexpected ChangePassword call in the auth gate test")
					return nil
				},
			}
			mux := http.NewServeMux()
			New(mock, mux)

			t.Run("no cookie", func(t *testing.T) {
				rec := doRequest(t, mux, row.method, row.path, "", nil)
				assert.Equal(t, http.StatusUnauthorized, rec.Code, "expected an unauthenticated request to be rejected")
			})

			t.Run("non-admin token", func(t *testing.T) {
				rec := doRequest(t, mux, row.method, row.path, userToken, nil)
				assert.Equal(t, http.StatusForbidden, rec.Code, "expected a non-admin token to be rejected")
			})
		})
	}
}

func TestGetUsers(t *testing.T) {
	adminToken := testToken(t, testUuid, "admin", time.Now().Add(time.Hour))

	t.Run("returns every user as JSON", func(t *testing.T) {
		list := []handlers.UserDto{
			testDto("11111111-1111-4111-8111-111111111111", "ada", "user", "Ada", "Lovelace"),
			testDto("22222222-2222-4222-8222-222222222222", "alan", "admin", "Alan", "Turing"),
		}
		mock := &mock.MockService{
			GetAllUsersFunc: func() ([]handlers.UserDto, error) {
				return list, nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		rec := doRequest(t, mux, http.MethodGet, "/users", adminToken, nil)

		require.Equal(t, http.StatusOK, rec.Code, "expected the list to load; body: %s", rec.Body.String())
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		var got []handlers.UserDto
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, list, got, "expected the decoded list to match the service output, order included")
	})

	t.Run("unexpected service error maps to 500", func(t *testing.T) {
		mock := &mock.MockService{
			GetAllUsersFunc: func() ([]handlers.UserDto, error) {
				// A plain error: exercises WriteError's fallback to
				// ErrUnexpectedError for anything that is not an
				// AppError.
				return nil, errors.New("boom")
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		rec := doRequest(t, mux, http.MethodGet, "/users", adminToken, nil)
		assertError(t, rec, http.StatusInternalServerError, "unexpected_error")
	})
}

func TestPostUser(t *testing.T) {
	adminToken := testToken(t, testUuid, "admin", time.Now().Add(time.Hour))

	t.Run("creates a user", func(t *testing.T) {
		dto := testDto("33333333-3333-4333-8333-333333333333", "grace", "user", "Grace", "Hopper")
		mock := &mock.MockService{
			CreateUserFunc: func(create handlers.UserCreate) (handlers.UserDto, error) {
				return dto, nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		create := handlers.UserCreate{Username: "grace", Password: "sup3r-s3cure-pass!", Role: "user", FirstName: "Grace", LastName: "Hopper"}
		rec := doRequest(t, mux, http.MethodPost, "/users", adminToken, create)

		require.Equal(t, http.StatusOK, rec.Code, "expected the user to be created; body: %s", rec.Body.String())
		var got handlers.UserDto
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, dto, got)
		assert.Equal(t, create, mock.CreateUserArg, "expected the decoded create payload to be forwarded")
	})

	t.Run("creates an admin", func(t *testing.T) {
		dto := testDto("44444444-4444-4444-8444-444444444444", "carl", "admin", "Carl", "Sagan")
		mock := &mock.MockService{
			CreateUserFunc: func(create handlers.UserCreate) (handlers.UserDto, error) {
				return dto, nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		create := handlers.UserCreate{Username: "carl", Password: "sup3r-s3cure-pass!", Role: "admin"}
		rec := doRequest(t, mux, http.MethodPost, "/users", adminToken, create)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "admin", mock.CreateUserArg.Role, "expected the role to reach the service")
	})

	t.Run("malformed JSON body", func(t *testing.T) {
		mock := &mock.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)

		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{not json"))
		req.AddCookie(&http.Cookie{Name: "jwt-token", Value: adminToken})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assertError(t, rec, http.StatusBadRequest, "invalid_request")
	})

	t.Run("service reports an invalid password", func(t *testing.T) {
		mock := &mock.MockService{
			CreateUserFunc: func(handlers.UserCreate) (handlers.UserDto, error) {
				return handlers.UserDto{}, handler_errors.ErrInvalidPassword
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPost, "/users", adminToken, handlers.UserCreate{Username: "x", Password: "short", Role: "user"})
		assertError(t, rec, http.StatusUnauthorized, "invalid_credentials")
	})

	t.Run("service reports an unknown role", func(t *testing.T) {
		mock := &mock.MockService{
			CreateUserFunc: func(handlers.UserCreate) (handlers.UserDto, error) {
				return handlers.UserDto{}, handler_errors.ErrRoleNotFound
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPost, "/users", adminToken, handlers.UserCreate{Username: "x", Password: "sup3r-s3cure-pass!", Role: "superadmin"})
		assertError(t, rec, http.StatusNotFound, "role_not_found")
	})

	t.Run("service reports a conflict", func(t *testing.T) {
		mock := &mock.MockService{
			CreateUserFunc: func(handlers.UserCreate) (handlers.UserDto, error) {
				return handlers.UserDto{}, handler_errors.ErrResourceConflict
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPost, "/users", adminToken, handlers.UserCreate{Username: "taken", Password: "sup3r-s3cure-pass!", Role: "user"})
		assertError(t, rec, http.StatusConflict, "resource_conflict")
	})
}

func TestGetUserByUUID(t *testing.T) {
	adminToken := testToken(t, testUuid, "admin", time.Now().Add(time.Hour))

	t.Run("returns the user", func(t *testing.T) {
		dto := testDto(testUuid, "ada", "user", "Ada", "Lovelace")
		mock := &mock.MockService{
			GetUserFunc: func(strUuid string) (handlers.UserDto, error) {
				return dto, nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		rec := doRequest(t, mux, http.MethodGet, "/users/"+testUuid, adminToken, nil)

		require.Equal(t, http.StatusOK, rec.Code, "expected the user to load; body: %s", rec.Body.String())
		var got handlers.UserDto
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, dto, got)
		assert.Equal(t, testUuid, mock.GetUserArg, "expected the path uuid to be forwarded to the service")
	})

	t.Run("service reports a missing user", func(t *testing.T) {
		mock := &mock.MockService{
			GetUserFunc: func(string) (handlers.UserDto, error) {
				return handlers.UserDto{}, handler_errors.ErrUserDoesNotExist
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodGet, "/users/"+uuid.NewV4().String(), adminToken, nil)
		assertError(t, rec, http.StatusNotFound, "user_does_not_exist")
	})

	t.Run("malformed uuid path", func(t *testing.T) {
		// The real service rejects unparseable uuids with
		// ErrInvalidRequest; the mock mirrors that behavior.
		mock := &mock.MockService{
			GetUserFunc: func(string) (handlers.UserDto, error) {
				return handlers.UserDto{}, handler_errors.ErrInvalidRequest
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodGet, "/users/nonsense", adminToken, nil)
		assertError(t, rec, http.StatusBadRequest, "invalid_request")
	})
}

func TestPatchUser(t *testing.T) {
	adminToken := testToken(t, testUuid, "admin", time.Now().Add(time.Hour))

	t.Run("changes role and fields", func(t *testing.T) {
		dto := testDto(testUuid, "ada", "admin", "Augusta", "King")
		mock := &mock.MockService{
			AdminModifyUserFunc: func(userUuid string, update handlers.AdminUserUpdate) (handlers.UserDto, error) {
				return dto, nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		update := handlers.AdminUserUpdate{Role: "admin", FirstName: "Augusta", LastName: "King"}
		rec := doRequest(t, mux, http.MethodPatch, "/users/"+testUuid, adminToken, update)

		require.Equal(t, http.StatusOK, rec.Code, "expected the update to succeed; body: %s", rec.Body.String())
		var got handlers.UserDto
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, dto, got)
		assert.Equal(t, testUuid, mock.AdminModifyUserUuidArg, "expected the path uuid to be forwarded")
		assert.Equal(t, update, mock.AdminModifyUserUpdateArg, "expected the decoded update payload to be forwarded")
	})

	t.Run("malformed JSON body", func(t *testing.T) {
		mock := &mock.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)

		req := httptest.NewRequest(http.MethodPatch, "/users/"+testUuid, strings.NewReader("{not json"))
		rec := httptest.NewRecorder()
		req.AddCookie(&http.Cookie{Name: "jwt-token", Value: adminToken})
		mux.ServeHTTP(rec, req)

		assertError(t, rec, http.StatusBadRequest, "invalid_request")
	})

	t.Run("service reports an unknown role", func(t *testing.T) {
		mock := &mock.MockService{
			AdminModifyUserFunc: func(string, handlers.AdminUserUpdate) (handlers.UserDto, error) {
				return handlers.UserDto{}, handler_errors.ErrRoleNotFound
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPatch, "/users/"+testUuid, adminToken, handlers.AdminUserUpdate{Role: "superadmin"})
		assertError(t, rec, http.StatusNotFound, "role_not_found")
	})

	t.Run("service reports a conflict", func(t *testing.T) {
		mock := &mock.MockService{
			AdminModifyUserFunc: func(string, handlers.AdminUserUpdate) (handlers.UserDto, error) {
				return handlers.UserDto{}, handler_errors.ErrResourceConflict
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPatch, "/users/"+testUuid, adminToken, handlers.AdminUserUpdate{Username: "taken"})
		assertError(t, rec, http.StatusConflict, "resource_conflict")
	})
}

func TestDeleteUser(t *testing.T) {
	adminToken := testToken(t, testUuid, "admin", time.Now().Add(time.Hour))

	t.Run("deletes the user", func(t *testing.T) {
		mock := &mock.MockService{
			DeleteUserFunc: func(userUuid string) error {
				return nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		rec := doRequest(t, mux, http.MethodDelete, "/users/"+testUuid, adminToken, nil)

		require.Equal(t, http.StatusOK, rec.Code, "expected the delete to succeed; body: %s", rec.Body.String())
		assert.Empty(t, rec.Body.String(), "expected no body on success")
		assert.Equal(t, testUuid, mock.DeleteUserArg, "expected the path uuid to be forwarded")
	})

	t.Run("service reports a missing user", func(t *testing.T) {
		mock := &mock.MockService{
			DeleteUserFunc: func(string) error {
				return handler_errors.ErrUserDoesNotExist
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodDelete, "/users/"+uuid.NewV4().String(), adminToken, nil)
		assertError(t, rec, http.StatusNotFound, "user_does_not_exist")
	})

	t.Run("malformed uuid path", func(t *testing.T) {
		mock := &mock.MockService{
			DeleteUserFunc: func(string) error {
				return handler_errors.ErrInvalidRequest
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodDelete, "/users/nonsense", adminToken, nil)
		assertError(t, rec, http.StatusBadRequest, "invalid_request")
	})
}

func TestPostUserPassword(t *testing.T) {
	adminToken := testToken(t, testUuid, "admin", time.Now().Add(time.Hour))

	t.Run("updates the user's password", func(t *testing.T) {
		mock := &mock.MockService{
			ChangePasswordFunc: func(userUuid string, newPassword string) error {
				return nil
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)

		rec := doRequest(t, mux, http.MethodPost, "/users/"+testUuid+"/password", adminToken,
			handlers.PasswordChangeRequest{Password: "new-new-new-pass"})

		require.Equal(t, http.StatusOK, rec.Code, "expected the password change to succeed; body: %s", rec.Body.String())
		assert.Empty(t, rec.Body.String(), "expected no body on success")
		assert.Equal(t, testUuid, mock.ChangePasswordUuidArg, "expected the path uuid to be forwarded")
		assert.Equal(t, "new-new-new-pass", mock.ChangePasswordNewArg, "expected the new password to be forwarded")
	})

	t.Run("malformed JSON body", func(t *testing.T) {
		mock := &mock.MockService{}
		mux := http.NewServeMux()
		New(mock, mux)

		req := httptest.NewRequest(http.MethodPost, "/users/"+testUuid+"/password", strings.NewReader("{not json"))
		rec := httptest.NewRecorder()
		req.AddCookie(&http.Cookie{Name: "jwt-token", Value: adminToken})
		mux.ServeHTTP(rec, req)

		assertError(t, rec, http.StatusBadRequest, "invalid_request")
	})

	t.Run("service reports an invalid password", func(t *testing.T) {
		mock := &mock.MockService{
			ChangePasswordFunc: func(string, string) error {
				return handler_errors.ErrInvalidPassword
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPost, "/users/"+testUuid+"/password", adminToken, handlers.PasswordChangeRequest{Password: "short"})
		assertError(t, rec, http.StatusUnauthorized, "invalid_credentials")
	})

	t.Run("service reports a missing user", func(t *testing.T) {
		mock := &mock.MockService{
			ChangePasswordFunc: func(string, string) error {
				return handler_errors.ErrUserDoesNotExist
			},
		}
		mux := http.NewServeMux()
		New(mock, mux)
		rec := doRequest(t, mux, http.MethodPost, "/users/"+uuid.NewV4().String()+"/password", adminToken, handlers.PasswordChangeRequest{Password: "valid-new-pass-1"})
		assertError(t, rec, http.StatusNotFound, "user_does_not_exist")
	})
}
