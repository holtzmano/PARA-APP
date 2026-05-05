package store

import (
	"context"
	"fmt"
	"html"
	"strings"
)

// SearchResult groups one item with whatever matched it: a snippet from FTS5
// over the item's title/content_md (empty if only notes matched), and any
// notes whose body_md contained the query (empty if only the item matched).
type SearchResult struct {
	Item        Item
	ItemSnippet string // safe HTML with <mark> highlights
	Notes       []Note
}

// Search runs the user query through FTS5 against items and a LIKE scan
// against notes, then groups by item. Order: FTS-rank winners first,
// then notes-only matches by most recent matching note.
func (s *Store) Search(ctx context.Context, query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	results := map[int64]*SearchResult{}
	var order []int64 // preserves insertion order

	ftsQ := buildFTSQuery(query)
	if ftsQ != "" {
		rows, err := s.DB.QueryContext(ctx, `
			SELECT i.id, i.type, i.title, i.status, i.content_md, i.parent_id,
			       i.due_date, i.metadata, i.created_at, i.updated_at, i.archived_at,
			       snippet(items_fts, -1, '[[m]]', '[[/m]]', '…', 16) AS snip
			FROM items_fts
			JOIN items i ON i.id = items_fts.rowid
			WHERE items_fts MATCH ?
			ORDER BY rank
			LIMIT 50`, ftsQ)
		if err != nil {
			return nil, fmt.Errorf("fts search: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var it Item
			var snip string
			if err := rows.Scan(
				&it.ID, &it.Type, &it.Title, &it.Status, &it.ContentMD, &it.ParentID,
				&it.DueDate, &it.Metadata, &it.CreatedAt, &it.UpdatedAt, &it.ArchivedAt,
				&snip,
			); err != nil {
				return nil, err
			}
			results[it.ID] = &SearchResult{Item: it, ItemSnippet: renderSnippet(snip)}
			order = append(order, it.ID)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	notes, err := s.searchNotes(ctx, query)
	if err != nil {
		return nil, err
	}
	for _, n := range notes {
		r, ok := results[n.ItemID]
		if !ok {
			it, err := s.GetItem(ctx, n.ItemID)
			if err != nil {
				return nil, err
			}
			r = &SearchResult{Item: *it}
			results[it.ID] = r
			order = append(order, it.ID)
		}
		r.Notes = append(r.Notes, n)
	}

	out := make([]SearchResult, 0, len(order))
	for _, id := range order {
		out = append(out, *results[id])
	}
	return out, nil
}

func (s *Store) searchNotes(ctx context.Context, query string) ([]Note, error) {
	pattern := "%" + escapeLike(query) + "%"
	rows, err := s.DB.QueryContext(ctx, `
		SELECT n.id, n.item_id, n.body_md, n.created_at
		FROM notes n
		WHERE n.body_md LIKE ? ESCAPE '\'
		ORDER BY n.created_at DESC
		LIMIT 100`, pattern)
	if err != nil {
		return nil, fmt.Errorf("notes search: %w", err)
	}
	defer rows.Close()
	var out []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.ItemID, &n.BodyMD, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// buildFTSQuery wraps each whitespace-separated token as a quoted FTS5 phrase
// so user input can't accidentally invoke FTS5 query operators.
func buildFTSQuery(q string) string {
	fields := strings.Fields(q)
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = `"` + strings.ReplaceAll(f, `"`, `""`) + `"`
	}
	return strings.Join(parts, " ")
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// renderSnippet HTML-escapes the FTS snippet, then swaps the sentinel tokens
// for real <mark> tags. This avoids reflecting raw user content as HTML.
func renderSnippet(s string) string {
	s = html.EscapeString(s)
	s = strings.ReplaceAll(s, "[[m]]", "<mark>")
	s = strings.ReplaceAll(s, "[[/m]]", "</mark>")
	return s
}
