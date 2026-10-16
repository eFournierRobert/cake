// Package conversations provides authenticated HTTP handlers for conversations
// and their messages.
package conversations

import (
	"encoding/json"
	"net/http"

	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/middleware/auth"
	conversationService "efournierrobert/cake-backend/internal/services/conversations"
)

// Handler serves authenticated conversation operations.
type Handler struct {
	service conversationService.ConversationService
}

// New creates a conversation handler and registers its routes. All endpoints
// require authentication; the service enforces conversation ownership.
func New(svc conversationService.ConversationService, router *http.ServeMux) *Handler {
	h := &Handler{service: svc}
	router.HandleFunc("GET /spaces/{uuid}/conversations", auth.RequireAuth(h.getConversations))
	router.HandleFunc("GET /spaces/{uuid}/conversations/{conversation_uuid}", auth.RequireAuth(h.getConversation))
	router.HandleFunc("PATCH /spaces/{uuid}/conversations/{conversation_uuid}", auth.RequireAuth(h.patchConversation))
	router.HandleFunc("DELETE /spaces/{uuid}/conversations/{conversation_uuid}", auth.RequireAuth(h.deleteConversation))
	router.HandleFunc("GET /spaces/{uuid}/conversations/{conversation_uuid}/messages", auth.RequireAuth(h.getConversationMessages))
	return h
}

func (h *Handler) getConversations(w http.ResponseWriter, r *http.Request) {
	ownerUuid := r.Context().Value(auth.UserUuidKey).(string)
	conversations, err := h.service.GetConversations(ownerUuid, r.PathValue("uuid"))
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, conversations)
}

func (h *Handler) getConversation(w http.ResponseWriter, r *http.Request) {
	ownerUuid := r.Context().Value(auth.UserUuidKey).(string)
	conversation, err := h.service.GetConversation(ownerUuid, r.PathValue("uuid"), r.PathValue("conversation_uuid"))
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, conversation)
}

func (h *Handler) patchConversation(w http.ResponseWriter, r *http.Request) {
	var update models.ConversationUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	ownerUuid := r.Context().Value(auth.UserUuidKey).(string)
	conversation, err := h.service.UpdateConversation(ownerUuid, r.PathValue("uuid"), r.PathValue("conversation_uuid"), update.Title)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, conversation)
}

func (h *Handler) deleteConversation(w http.ResponseWriter, r *http.Request) {
	ownerUuid := r.Context().Value(auth.UserUuidKey).(string)
	if err := h.service.DeleteConversation(ownerUuid, r.PathValue("uuid"), r.PathValue("conversation_uuid")); err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getConversationMessages(w http.ResponseWriter, r *http.Request) {
	ownerUuid := r.Context().Value(auth.UserUuidKey).(string)
	messages, err := h.service.GetFullConversation(ownerUuid, r.PathValue("uuid"), r.PathValue("conversation_uuid"))
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, messages)
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
