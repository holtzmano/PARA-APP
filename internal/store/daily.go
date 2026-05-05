package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Note struct {
	ID        int64
	ItemID    int64
	BodyMD    string
	CreatedAt string
}

// GetOrCreateDailyItem looks up the resource for the given date (YYYY-MM-DD)
// and creates it on first use. Title format: "Daily / 2026-05-05".
//
// Single-user assumption: a race here would just produce two daily items.
// Acceptable, and avoids the schema churn of a UNIQUE(title) constraint.
func (s *Store) GetOrCreateDailyItem(ctx context.Context, date string) (*Item, error) {
	title := dailyTitle(date)
	row := s.DB.QueryRowContext(ctx,
		`SELECT `+itemColumns+` FROM items WHERE type = 'resource' AND title = ?`, title)
	it, err := scanItem(row)
	if err == nil {
		return it, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup daily %s: %w", date, err)
	}
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO items (type, title) VALUES ('resource', ?)`, title)
	if err != nil {
		return nil, fmt.Errorf("create daily %s: %w", date, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetItem(ctx, id)
}

func dailyTitle(date string) string { return "Daily / " + date }

func (s *Store) AppendNote(ctx context.Context, itemID int64, body string) (*Note, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, errors.New("note body is empty")
	}
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO notes (item_id, body_md) VALUES (?, ?)`, itemID, body)
	if err != nil {
		return nil, fmt.Errorf("insert note: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	var n Note
	err = s.DB.QueryRowContext(ctx,
		`SELECT id, item_id, body_md, created_at FROM notes WHERE id = ?`, id,
	).Scan(&n.ID, &n.ItemID, &n.BodyMD, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (s *Store) ListNotes(ctx context.Context, itemID int64) ([]Note, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, item_id, body_md, created_at FROM notes
		 WHERE item_id = ? ORDER BY created_at DESC, id DESC`, itemID)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
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
