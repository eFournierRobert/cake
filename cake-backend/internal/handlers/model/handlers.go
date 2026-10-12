// Package model provides HTTP handlers for model catalog operations.
package model

import (
	"encoding/json"
	"net/http"

	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	apiModels "efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/middleware/auth"
	modelService "efournierrobert/cake-backend/internal/services/models"
)

// Handler serves model catalog endpoints.
type Handler struct {
	service modelService.ModelService
}

// New creates a model handler and registers its routes. Listing activated
// models requires authentication; listing all models and modifying models are
// administrator-only operations.
func New(svc modelService.ModelService, router *http.ServeMux) *Handler {
	h := &Handler{service: svc}
	router.HandleFunc("GET /models", auth.RequireAuth(h.getActivatedModels))
	router.HandleFunc("GET /models/all", auth.RequireAdminAuth(h.getAllModels))
	router.HandleFunc("PUT /models/{uuid}", auth.RequireAdminAuth(h.putModel))
	router.HandleFunc("DELETE /models/{uuid}", auth.RequireAdminAuth(h.deleteModel))
	return h
}

// getActivatedModels handles GET /models and lists the models available for
// use by an authenticated user.
func (h *Handler) getActivatedModels(w http.ResponseWriter, _ *http.Request) {
	models, err := h.service.GetAllActivatedModels()
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, models)
}

// getAllModels handles GET /models/all and lists every model, including
// inactive models. The route is administrator-only.
func (h *Handler) getAllModels(w http.ResponseWriter, _ *http.Request) {
	models, err := h.service.GetAllModels()
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, models)
}

// putModel handles PUT /models/{uuid} and updates a model. The route is
// administrator-only.
func (h *Handler) putModel(w http.ResponseWriter, r *http.Request) {
	var update apiModels.ModelUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		handler_errors.WriteError(w, handler_errors.ErrInvalidRequest)
		return
	}

	model, err := h.service.UpdateModel(r.PathValue("uuid"), update)
	if err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	writeJSON(w, model)
}

// deleteModel handles DELETE /models/{uuid} and removes a model. The route is
// administrator-only.
func (h *Handler) deleteModel(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteModel(r.PathValue("uuid")); err != nil {
		handler_errors.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func writeJSON(w http.ResponseWriter, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		handler_errors.WriteError(w, handler_errors.ErrUnexpectedError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
