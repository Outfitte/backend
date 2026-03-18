-- 000001_initial_schema.up.sql

CREATE TABLE users (
    id          TEXT NOT NULL PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'member',
    created_at  TEXT NOT NULL
);

CREATE TABLE sessions (
    id          TEXT NOT NULL PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL,
    expires_at  TEXT NOT NULL,
    created_at  TEXT NOT NULL
);

CREATE TABLE app_settings (
    id                   TEXT NOT NULL PRIMARY KEY DEFAULT 'app_settings',
    registration_enabled INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE locations (
    id          TEXT NOT NULL PRIMARY KEY,
    owner_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id   TEXT REFERENCES locations(id) ON DELETE RESTRICT,
    label       TEXT NOT NULL,
    created_at  TEXT NOT NULL
);

CREATE TABLE items (
    id              TEXT NOT NULL PRIMARY KEY,
    owner_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    category_id     TEXT,
    brand           TEXT,
    color           TEXT,
    location_id     TEXT REFERENCES locations(id) ON DELETE SET NULL,
    wear_count      INTEGER NOT NULL DEFAULT 0,
    last_worn_at    TEXT,
    archived_at     TEXT,
    disposal_reason TEXT,
    purchase_price  TEXT,
    purchase_date   TEXT,
    created_at      TEXT NOT NULL,
    metadata        TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE item_photos (
    id          TEXT NOT NULL PRIMARY KEY,
    item_id     TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    media_key   TEXT NOT NULL,
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL
);

CREATE TABLE wear_logs (
    id          TEXT NOT NULL PRIMARY KEY,
    item_id     TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    owner_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    worn_on     TEXT NOT NULL,
    notes       TEXT,
    created_at  TEXT NOT NULL
);

-- Indexes
CREATE INDEX idx_users_email         ON users(email);
CREATE INDEX idx_sessions_user_id    ON sessions(user_id);
CREATE INDEX idx_items_owner_id      ON items(owner_id);
CREATE INDEX idx_items_owner_archived ON items(owner_id, archived_at);
CREATE INDEX idx_item_photos_item_id ON item_photos(item_id);
CREATE INDEX idx_wear_logs_item_id   ON wear_logs(item_id);
CREATE INDEX idx_locations_owner_id  ON locations(owner_id);
CREATE INDEX idx_locations_parent_id ON locations(parent_id);
