// Package spaces provides HTTP handlers for user-owned conversation spaces.
package spaces

import (
	"encoding/json"
	"net/http"

	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/middleware/auth"
	spaceService "efournierrobert/cake-backend/internal/services/spaces"
)

// Handler serves authenticated space operations.
type Handler struct {
	service spaceService.SpaceService
}

// New creates a space handler and registers its routes. All space operations
// are restricted to authenticated users; the service enforces ownership.
func New(svc spaceService.SpaceService, router *http.ServeMux) *Handler {
	h := &Handler{service: svc}
	router.HandleFunc("GET /spaces", auth.RequireAuth(h.getSpaces))
	router.HandleFunc("POST /spaces", auth.RequireAuth(h.postSpace))
	router.HandleFunc("GET /spaces/{uuid}", auth.RequireAuth(h.getSpace))
	router.HandleFunc("PATCH /spaces/{uuid}", auth.RequireAuth(h.patchSpace))
	router.HandleFunc("DELETE /spaces/{uuid}", auth.RequireAuth(h.deleteSpace))
	return h
}

func (h *Handler) getSpaces(w http.ResponseWriter, r *http.Request) {
	ownerUuid := r.Context().Value(auth.UserUuidKey).(string)
	spaces, err := h.service.GetAllSpaces(ownerUuid)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, spaces)
}

func (h *Handler) postSpace(w http.ResponseWriter, r *http.Request) {
	var req models.SpaceCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	ownerUuid := r.Context().Value(auth.UserUuidKey).(string)
	space, err := h.service.CreateSpace(ownerUuid, req)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, space)
}

func (h *Handler) getSpace(w http.ResponseWriter, r *http.Request) {
	ownerUuid := r.Context().Value(auth.UserUuidKey).(string)
	space, err := h.service.GetSpace(r.PathValue("uuid"), ownerUuid)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, space)
}

func (h *Handler) patchSpace(w http.ResponseWriter, r *http.Request) {
	var req models.SpaceUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	ownerUuid := r.Context().Value(auth.UserUuidKey).(string)
	space, err := h.service.UpdateSpace(ownerUuid, r.PathValue("uuid"), req)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, space)
}

func (h *Handler) deleteSpace(w http.ResponseWriter, r *http.Request) {
	ownerUuid := r.Context().Value(auth.UserUuidKey).(string)
	if err := h.service.DeleteSpace(ownerUuid, r.PathValue("uuid")); err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
