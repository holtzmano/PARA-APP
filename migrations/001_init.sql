CREATE TABLE items (
    id          INTEGER PRIMARY KEY,
    type        TEXT NOT NULL CHECK (type IN ('project','area','resource')),
    title       TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','someday','done','archived')),
    content_md  TEXT NOT NULL DEFAULT '',
    parent_id   INTEGER REFERENCES items(id) ON DELETE SET NULL,
    due_date    TEXT,
    metadata    TEXT NOT NULL DEFAULT '{}',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    archived_at TEXT
);

CREATE TABLE tags (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE item_tags (
    item_id INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (item_id, tag_id)
);

CREATE TABLE notes (
    id         INTEGER PRIMARY KEY,
    item_id    INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    body_md    TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE attachments (
    id         INTEGER PRIMARY KEY,
    item_id    INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('proton_drive','onepassword','bitwarden','url','local_note')),
    ref        TEXT NOT NULL,
    label      TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX items_type_status_live ON items (type, status) WHERE archived_at IS NULL;
CREATE INDEX items_parent_id        ON items (parent_id);
CREATE INDEX items_due_date_active  ON items (due_date) WHERE status = 'active';

CREATE VIRTUAL TABLE items_fts USING fts5(
    title,
    content_md,
    content='items',
    content_rowid='id'
);

-- External-content FTS needs explicit sync triggers; without these the index never populates.
CREATE TRIGGER items_ai AFTER INSERT ON items BEGIN
    INSERT INTO items_fts (rowid, title, content_md)
    VALUES (new.id, new.title, new.content_md);
END;

CREATE TRIGGER items_ad AFTER DELETE ON items BEGIN
    INSERT INTO items_fts (items_fts, rowid, title, content_md)
    VALUES ('delete', old.id, old.title, old.content_md);
END;

CREATE TRIGGER items_au AFTER UPDATE ON items BEGIN
    INSERT INTO items_fts (items_fts, rowid, title, content_md)
    VALUES ('delete', old.id, old.title, old.content_md);
    INSERT INTO items_fts (rowid, title, content_md)
    VALUES (new.id, new.title, new.content_md);
END;
