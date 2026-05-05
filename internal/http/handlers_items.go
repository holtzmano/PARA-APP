// Package web hosts the HTTP handlers. It lives in internal/http to match
// the project layout; the package name avoids collision with stdlib net/http.
package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	"para/internal/store"
	"para/internal/view"
)

type Items struct {
	Store *store.Store
}

func NewItems(s *store.Store) *Items { return &Items{Store: s} }

func (h *Items) Routes(r chi.Router) {
	r.Get("/items", h.list)
	r.Get("/items/new", h.newForm)
	r.Post("/items", h.create)
	r.Get("/items/{id}", h.detail)
	r.Post("/items/{id}", h.update)
	r.Post("/items/{id}/archive", h.archive)
	r.Post("/items/{id}/restore", h.restore)
}

func (h *Items) list(w http.ResponseWriter, r *http.Request) {
	f := store.ListFilter{
		Type:   r.URL.Query().Get("type"),
		Status: r.URL.Query().Get("status"),
	}
	items, err := h.Store.ListItems(r.Context(), f)
	if err != nil {
		serverError(w, err)
		return
	}
	render(w, r, view.ItemsList(items, f))
}

func (h *Items) newForm(w http.ResponseWriter, r *http.Request) {
	render(w, r, view.ItemForm(nil, "/items", "New item"))
}

func (h *Items) create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	it := &store.Item{
		Type:      r.FormValue("type"),
		Title:     r.FormValue("title"),
		Status:    r.FormValue("status"),
		ContentMD: r.FormValue("content_md"),
		DueDate:   nullStr(r.FormValue("due_date")),
		ParentID:  nullInt64(r.FormValue("parent_id")),
	}
	id, err := h.Store.CreateItem(r.Context(), it)
	if err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/items/%d", id), http.StatusSeeOther)
}

func (h *Items) detail(w http.ResponseWriter, r *http.Request) {
	id, err := itemID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	it, err := h.Store.GetItem(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	render(w, r, view.ItemForm(it, fmt.Sprintf("/items/%d", id), it.Title))
}

func (h *Items) update(w http.ResponseWriter, r *http.Request) {
	id, err := itemID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cur, err := h.Store.GetItem(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	cur.Title = r.FormValue("title")
	cur.Status = r.FormValue("status")
	cur.ContentMD = r.FormValue("content_md")
	cur.DueDate = nullStr(r.FormValue("due_date"))
	cur.ParentID = nullInt64(r.FormValue("parent_id"))
	if err := h.Store.UpdateItem(r.Context(), cur); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/items/%d", id), http.StatusSeeOther)
}

func (h *Items) archive(w http.ResponseWriter, r *http.Request) {
	h.toggleArchived(w, r, h.Store.ArchiveItem)
}

func (h *Items) restore(w http.ResponseWriter, r *http.Request) {
	h.toggleArchived(w, r, h.Store.RestoreItem)
}

func (h *Items) toggleArchived(w http.ResponseWriter, r *http.Request, op func(context.Context, int64) error) {
	id, err := itemID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := op(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		serverError(w, err)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		it, err := h.Store.GetItem(r.Context(), id)
		if err != nil {
			serverError(w, err)
			return
		}
		render(w, r, view.ItemRow(*it))
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/items/%d", id), http.StatusSeeOther)
}

func itemID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt64(s string) sql.NullInt64 {
	if s == "" {
		return sql.NullInt64{}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: n, Valid: true}
}

func render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(r.Context(), w); err != nil {
		log.Printf("render: %v", err)
	}
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}
