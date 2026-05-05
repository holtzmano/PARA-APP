package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"para/internal/store"
	"para/internal/view"
)

type Search struct {
	Store *store.Store
}

func NewSearch(s *store.Store) *Search { return &Search{Store: s} }

func (h *Search) Routes(r chi.Router) {
	r.Get("/search", h.show)
}

func (h *Search) show(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := h.Store.Search(r.Context(), q)
	if err != nil {
		serverError(w, err)
		return
	}
	render(w, r, view.SearchPage(q, results))
}
