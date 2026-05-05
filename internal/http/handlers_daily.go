package web

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"para/internal/store"
	"para/internal/view"
)

const dateLayout = "2006-01-02"

type Daily struct {
	Store *store.Store
}

func NewDaily(s *store.Store) *Daily { return &Daily{Store: s} }

func (h *Daily) Routes(r chi.Router) {
	r.Get("/daily", h.today)
	r.Get("/daily/{date}", h.show)
	r.Post("/daily/{date}/notes", h.addNote)
}

func (h *Daily) today(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format(dateLayout)
	http.Redirect(w, r, "/daily/"+today, http.StatusSeeOther)
}

func (h *Daily) show(w http.ResponseWriter, r *http.Request) {
	date, ok := parseDate(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	it, err := h.Store.GetOrCreateDailyItem(r.Context(), date)
	if err != nil {
		serverError(w, err)
		return
	}
	notes, err := h.Store.ListNotes(r.Context(), it.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	t, _ := time.Parse(dateLayout, date)
	prev := t.AddDate(0, 0, -1).Format(dateLayout)
	next := t.AddDate(0, 0, 1).Format(dateLayout)
	render(w, r, view.Daily(*it, notes, date, prev, next))
}

func (h *Daily) addNote(w http.ResponseWriter, r *http.Request) {
	date, ok := parseDate(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	body := r.FormValue("body_md")
	it, err := h.Store.GetOrCreateDailyItem(r.Context(), date)
	if err != nil {
		serverError(w, err)
		return
	}
	n, err := h.Store.AppendNote(r.Context(), it.ID, body)
	if err != nil {
		// Empty body is the only expected error here; just no-op for htmx.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		render(w, r, view.NoteItem(*n))
		return
	}
	http.Redirect(w, r, "/daily/"+date, http.StatusSeeOther)
}

func parseDate(r *http.Request) (string, bool) {
	d := chi.URLParam(r, "date")
	if _, err := time.Parse(dateLayout, d); err != nil {
		return "", false
	}
	return d, true
}
