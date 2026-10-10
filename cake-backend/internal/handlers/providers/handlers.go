package providers

import (
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/middleware/auth"
	providerService "efournierrobert/cake-backend/internal/services/providers"
	"encoding/json"
	"net/http"
)

// Handler provides HTTP handlers for provider-related operations.
// It registers the routes on the supplied router.
type Handler struct {
	service providerService.ProviderService
}

// New creates a Handler and registers all provider endpoints. Every endpoint
// is restricted to administrators.
func New(svc providerService.ProviderService, router *http.ServeMux) *Handler {
	h := Handler{service: svc}

	router.HandleFunc("GET /providers", auth.RequireAdminAuth(h.getProviders))
	router.HandleFunc("POST /providers", auth.RequireAdminAuth(h.postProvider))
	router.HandleFunc("GET /providers/{uuid}", auth.RequireAdminAuth(h.getProvider))
	router.HandleFunc("PUT /providers/{uuid}", auth.RequireAdminAuth(h.putProvider))
	router.HandleFunc("DELETE /providers/{uuid}", auth.RequireAdminAuth(h.deleteProvider))
	router.HandleFunc("POST /providers/{uuid}/test", auth.RequireAdminAuth(h.postProviderTest))

	return &h
}

// getProviders handles GET /providers and lists every provider (admin only).
func (h *Handler) getProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.service.GetAllProviders()
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	responseBody, err := json.Marshal(providers)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseBody)
}

// postProvider handles POST /providers and creates a provider (admin only).
func (h *Handler) postProvider(w http.ResponseWriter, r *http.Request) {
	var req models.ProviderCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	provider, err := h.service.CreateProvider(req)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	responseBody, err := json.Marshal(provider)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseBody)
}

// getProvider handles GET /providers/{uuid} and returns one provider.
func (h *Handler) getProvider(w http.ResponseWriter, r *http.Request) {
	provider, err := h.service.GetProvider(r.PathValue("uuid"))
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	responseBody, err := json.Marshal(provider)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseBody)
}

// putProvider handles PUT /providers/{uuid} and updates a provider.
func (h *Handler) putProvider(w http.ResponseWriter, r *http.Request) {
	var req models.ProviderUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	provider, err := h.service.ModifyProvider(r.PathValue("uuid"), req)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	responseBody, err := json.Marshal(provider)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseBody)
}

// deleteProvider handles DELETE /providers/{uuid} and removes a provider.
func (h *Handler) deleteProvider(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteProvider(r.PathValue("uuid")); err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// postProviderTest handles POST /providers/{uuid}/test and returns the
// result of the provider connectivity check.
func (h *Handler) postProviderTest(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.TestProvider(r.PathValue("uuid"))
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}

	responseBody, err := json.Marshal(result)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseBody)
}
