package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"para/internal/store"
	"para/internal/view"
)

type Review struct {
	Store *store.Store
}

func NewReview(s *store.Store) *Review { return &Review{Store: s} }

func (h *Review) Routes(r chi.Router) {
	r.Get("/review", h.show)
}

func (h *Review) show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stale, err := h.Store.StaleActiveProjects(ctx)
	if err != nil {
		serverError(w, err)
		return
	}
	dueSoon, err := h.Store.DueSoonProjects(ctx)
	if err != nil {
		serverError(w, err)
		return
	}
	someday, err := h.Store.SomedayItems(ctx)
	if err != nil {
		serverError(w, err)
		return
	}
	recent, err := h.Store.RecentlyArchived(ctx)
	if err != nil {
		serverError(w, err)
		return
	}
	render(w, r, view.Review(stale, dueSoon, someday, recent))
}
