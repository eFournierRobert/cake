package users

import (
	"efournierrobert/cake-backend/internal/handlers"
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/middleware/auth"
	userService "efournierrobert/cake-backend/internal/services/users"
	"encoding/json"
	"log"
	"net/http"
)

type Handler struct {
	service userService.UserService
}

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

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req handlers.LoginRequest
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

func (h *Handler) patchCurrentUser(w http.ResponseWriter, r *http.Request) {
	currentUserUuid := r.Context().Value(auth.UserUuidKey).(string)

	var req handlers.UserUpdate
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

func (h *Handler) postCurrentUserPassword(w http.ResponseWriter, r *http.Request) {
	currentUserUuid := r.Context().Value(auth.UserUuidKey).(string)

	var req handlers.PasswordChangeRequest
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

func (h *Handler) postUser(w http.ResponseWriter, r *http.Request) {
	var req handlers.UserCreate
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

func (h *Handler) patchUser(w http.ResponseWriter, r *http.Request) {
	userUuid := r.PathValue("uuid")

	var req handlers.AdminUserUpdate
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

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	userUuid := r.PathValue("uuid")

	err := h.service.DeleteUser(userUuid)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) postUserPassword(w http.ResponseWriter, r *http.Request) {
	userUuid := r.PathValue("uuid")

	var req handlers.PasswordChangeRequest
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
