package users

import (
	"efournierrobert/cake-backend/internal/handlers"
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/middleware/auth"
	userService "efournierrobert/cake-backend/internal/services/users"
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
)

type Handler struct {
	service *userService.Service
}

func New(db *sqlx.DB, router *http.ServeMux) *Handler {
	h := Handler{
		userService.New(db),
	}

	router.HandleFunc("GET /user", auth.RequireAuth(h.getCurrentUser))
	router.HandleFunc("PATCH /user", auth.RequireAuth(h.patchCurrentUser))
	router.HandleFunc("POST /user/password", auth.RequireAuth(h.patchCurrentUser))

	router.HandleFunc("GET /users", auth.RequireAdminAuth(h.getUsers))
	router.HandleFunc("POST /users", auth.RequireAdminAuth(h.postUser))

	router.HandleFunc("GET /users/{uuid}", auth.RequireAdminAuth(h.getUsers))
	router.HandleFunc("PATCH /users/{uuid}", auth.RequireAdminAuth(h.patchUser))
	router.HandleFunc("DELETE /users/{uuid}", auth.RequireAdminAuth(h.deleteUser))
	router.HandleFunc("POST /users/{uuid}/password", auth.RequireAdminAuth(h.postUserPassword))

	return &h
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

	w.Write(requestBody)
	w.WriteHeader(http.StatusOK)
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

	w.Write(requestBody)
	w.WriteHeader(http.StatusOK)
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

	w.Write(requestBody)
	w.WriteHeader(http.StatusOK)
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

	w.Write(requestBody)
	w.WriteHeader(http.StatusOK)
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

	w.Write(requestBody)
	w.WriteHeader(http.StatusOK)
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

	w.Write(requestBody)
	w.WriteHeader(http.StatusOK)
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
