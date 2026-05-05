package store

import (
	"context"
	"fmt"
)

func (s *Store) StaleActiveProjects(ctx context.Context) ([]Item, error) {
	return s.queryItems(ctx, `
		SELECT `+itemColumns+` FROM items
		WHERE type = 'project' AND status = 'active' AND archived_at IS NULL
		  AND updated_at < datetime('now', '-14 days')
		ORDER BY updated_at ASC
		LIMIT 100`)
}

func (s *Store) DueSoonProjects(ctx context.Context) ([]Item, error) {
	return s.queryItems(ctx, `
		SELECT `+itemColumns+` FROM items
		WHERE type = 'project' AND status = 'active' AND archived_at IS NULL
		  AND due_date IS NOT NULL
		  AND due_date BETWEEN date('now') AND date('now', '+14 days')
		ORDER BY due_date ASC
		LIMIT 100`)
}

func (s *Store) SomedayItems(ctx context.Context) ([]Item, error) {
	return s.queryItems(ctx, `
		SELECT `+itemColumns+` FROM items
		WHERE status = 'someday' AND archived_at IS NULL
		ORDER BY updated_at DESC
		LIMIT 100`)
}

func (s *Store) RecentlyArchived(ctx context.Context) ([]Item, error) {
	return s.queryItems(ctx, `
		SELECT `+itemColumns+` FROM items
		WHERE archived_at IS NOT NULL
		  AND archived_at >= datetime('now', '-7 days')
		ORDER BY archived_at DESC
		LIMIT 100`)
}

func (s *Store) queryItems(ctx context.Context, q string, args ...any) ([]Item, error) {
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}
