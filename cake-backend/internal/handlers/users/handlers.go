package users

import (
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/middleware/auth"
	userService "efournierrobert/cake-backend/internal/services/users"
	"encoding/json"
	"log"
	"net/http"
)

// Handler provides HTTP handlers for user-related operations.
// It is initialized with a UserService and registers routes on the provided router.
type Handler struct {
	service userService.UserService
}

// New creates a new Handler and registers all user-related routes on the router.
// It registers handlers for both authenticated user endpoints and admin endpoints.
func New(svc userService.UserService, router *http.ServeMux) *Handler {
	h := Handler{
		service: svc,
	}

	router.HandleFunc("GET /user", auth.RequireAuth(h.getCurrentUser))
	router.HandleFunc("PATCH /user", auth.RequireAuth(h.patchCurrentUser))
	router.HandleFunc("POST /user/password", auth.RequireAuth(h.postCurrentUserPassword))

	router.HandleFunc("GET /users", auth.RequireAdminAuth(h.getUsers))
	router.HandleFunc("POST /users", auth.RequireAdminAuth(h.postUser))

	router.HandleFunc("GET /users/{uuid}", auth.RequireAdminAuth(h.getUser))
	router.HandleFunc("PATCH /users/{uuid}", auth.RequireAdminAuth(h.patchUser))
	router.HandleFunc("DELETE /users/{uuid}", auth.RequireAdminAuth(h.deleteUser))
	router.HandleFunc("POST /users/{uuid}/password", auth.RequireAdminAuth(h.postUserPassword))

	router.HandleFunc("POST /login", h.login)
	router.HandleFunc("POST /logout", h.logout)

	return &h
}

// login handles POST /login and authenticates a user with username and password.
// On success, it sets a JWT cookie and returns 200 OK.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	tokenString, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		log.Println("Error on login: " + err.Error())
		handler_errors.WriteError(w, handler_errors.ErrInvalidPassword)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt-token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60 * 60,
	})
	w.WriteHeader(http.StatusOK)
}

// logout handles POST /logout and clears the JWT cookie.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt-token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusOK)
}

// getCurrentUser handles GET /user and returns the authenticated user's profile.
func (h *Handler) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	currentUserUuid := r.Context().Value(auth.UserUuidKey).(string)

	u, err := h.service.GetUser(currentUserUuid)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	requestBody, err := json.Marshal(u)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(requestBody)
}

// patchCurrentUser handles PATCH /user and allows the authenticated user to
// update their own profile (first name, last name, username).
func (h *Handler) patchCurrentUser(w http.ResponseWriter, r *http.Request) {
	currentUserUuid := r.Context().Value(auth.UserUuidKey).(string)

	var req models.UserUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	u, err := h.service.ModifyUser(currentUserUuid, req)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	requestBody, err := json.Marshal(u)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(requestBody)
}

// postCurrentUserPassword handles POST /user/password and allows the authenticated
// user to change their own password.
func (h *Handler) postCurrentUserPassword(w http.ResponseWriter, r *http.Request) {
	currentUserUuid := r.Context().Value(auth.UserUuidKey).(string)

	var req models.PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	err := h.service.ChangePassword(currentUserUuid, req.Password)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// getUsers handles GET /users and lists all users (admin only).
func (h *Handler) getUsers(w http.ResponseWriter, r *http.Request) {
	us, err := h.service.GetAllUsers()
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	requestBody, err := json.Marshal(us)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(requestBody)
}

// postUser handles POST /users and creates a new user (admin only).
func (h *Handler) postUser(w http.ResponseWriter, r *http.Request) {
	var req models.UserCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	u, err := h.service.CreateUser(req)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	requestBody, err := json.Marshal(u)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(requestBody)
}

// getUser handles GET /users/{uuid} and returns a specific user (admin only).
func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	userUuid := r.PathValue("uuid")

	u, err := h.service.GetUser(userUuid)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	requestBody, err := json.Marshal(u)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(requestBody)
}

// patchUser handles PATCH /users/{uuid} and updates a user (admin only).
// Can update the user's role in addition to profile fields.
func (h *Handler) patchUser(w http.ResponseWriter, r *http.Request) {
	userUuid := r.PathValue("uuid")

	var req models.AdminUserUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	u, err := h.service.AdminModifyUser(userUuid, req)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	requestBody, err := json.Marshal(u)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(requestBody)
}

// deleteUser handles DELETE /users/{uuid} and removes a user (admin only).
func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	userUuid := r.PathValue("uuid")

	err := h.service.DeleteUser(userUuid)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// postUserPassword handles POST /users/{uuid}/password and changes a user's password (admin only).
func (h *Handler) postUserPassword(w http.ResponseWriter, r *http.Request) {
	userUuid := r.PathValue("uuid")

	var req models.PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	err := h.service.ChangePassword(userUuid, req.Password)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
