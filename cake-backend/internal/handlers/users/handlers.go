package users

import (
	"efournierrobert/cake-backend/internal/middleware/auth"
	userService "efournierrobert/cake-backend/internal/services/users"
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

	return &h
}

func (h *Handler) getCurrentUser(w http.ResponseWriter, r *http.Request) {

}
