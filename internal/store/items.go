package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Item struct {
	ID         int64
	Type       string
	Title      string
	Status     string
	ContentMD  string
	ParentID   sql.NullInt64
	DueDate    sql.NullString
	Metadata   string
	CreatedAt  string
	UpdatedAt  string
	ArchivedAt sql.NullString
}

func (it Item) IsArchived() bool { return it.ArchivedAt.Valid }

// ListFilter narrows ListItems. Empty/zero fields mean "no filter on this column".
// ListItems always excludes archived rows; use GetItem to fetch one regardless.
type ListFilter struct {
	Type     string
	Status   string
	ParentID *int64
}

var ErrNotFound = errors.New("item not found")

const itemColumns = `id, type, title, status, content_md, parent_id, due_date,
	metadata, created_at, updated_at, archived_at`

func scanItem(s interface{ Scan(...any) error }) (*Item, error) {
	var it Item
	if err := s.Scan(
		&it.ID, &it.Type, &it.Title, &it.Status, &it.ContentMD,
		&it.ParentID, &it.DueDate, &it.Metadata,
		&it.CreatedAt, &it.UpdatedAt, &it.ArchivedAt,
	); err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Store) CreateItem(ctx context.Context, it *Item) (int64, error) {
	if it.Status == "" {
		it.Status = "active"
	}
	if it.Metadata == "" {
		it.Metadata = "{}"
	}
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO items (type, title, status, content_md, parent_id, due_date, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		it.Type, it.Title, it.Status, it.ContentMD, it.ParentID, it.DueDate, it.Metadata,
	)
	if err != nil {
		return 0, fmt.Errorf("insert item: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) GetItem(ctx context.Context, id int64) (*Item, error) {
	row := s.DB.QueryRowContext(ctx,
		`SELECT `+itemColumns+` FROM items WHERE id = ?`, id)
	it, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return it, err
}

func (s *Store) ListItems(ctx context.Context, f ListFilter) ([]Item, error) {
	q := `SELECT ` + itemColumns + ` FROM items WHERE archived_at IS NULL`
	var args []any
	if f.Type != "" {
		q += " AND type = ?"
		args = append(args, f.Type)
	}
	if f.Status != "" {
		q += " AND status = ?"
		args = append(args, f.Status)
	}
	if f.ParentID != nil {
		q += " AND parent_id = ?"
		args = append(args, *f.ParentID)
	}
	q += " ORDER BY COALESCE(due_date, '9999-12-31'), updated_at DESC"

	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
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

func (s *Store) UpdateItem(ctx context.Context, it *Item) error {
	res, err := s.DB.ExecContext(ctx, `
		UPDATE items SET
			title      = ?,
			status     = ?,
			content_md = ?,
			parent_id  = ?,
			due_date   = ?,
			metadata   = ?,
			updated_at = datetime('now')
		WHERE id = ?`,
		it.Title, it.Status, it.ContentMD, it.ParentID, it.DueDate, it.Metadata, it.ID,
	)
	return checkAffected(res, err, "update item")
}

func (s *Store) ArchiveItem(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE items SET archived_at = datetime('now'), updated_at = datetime('now')
		 WHERE id = ? AND archived_at IS NULL`, id)
	return checkAffected(res, err, "archive item")
}

func (s *Store) RestoreItem(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE items SET archived_at = NULL, updated_at = datetime('now')
		 WHERE id = ? AND archived_at IS NOT NULL`, id)
	return checkAffected(res, err, "restore item")
}

func checkAffected(res sql.Result, err error, op string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: rows affected: %w", op, err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ItemTypes / ItemStatuses are exposed for form rendering.
var (
	ItemTypes    = []string{"project", "area", "resource"}
	ItemStatuses = []string{"active", "someday", "done", "archived"}
)

func (f ListFilter) String() string {
	var parts []string
	if f.Type != "" {
		parts = append(parts, "type="+f.Type)
	}
	if f.Status != "" {
		parts = append(parts, "status="+f.Status)
	}
	return strings.Join(parts, " ")
}
