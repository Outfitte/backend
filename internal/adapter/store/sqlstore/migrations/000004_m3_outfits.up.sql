CREATE TABLE outfits (
    id          TEXT NOT NULL PRIMARY KEY,
    owner_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT,
    notes       TEXT,
    created_at  TEXT NOT NULL
);

CREATE TABLE outfit_items (
    outfit_id   TEXT NOT NULL REFERENCES outfits(id) ON DELETE CASCADE,
    item_id     TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (outfit_id, item_id)
);

CREATE TABLE outfit_photos (
    id          TEXT NOT NULL PRIMARY KEY,
    outfit_id   TEXT NOT NULL REFERENCES outfits(id) ON DELETE CASCADE,
    media_key   TEXT NOT NULL,
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL
);

CREATE TABLE outfit_logs (
    id          TEXT NOT NULL PRIMARY KEY,
    outfit_id   TEXT NOT NULL REFERENCES outfits(id) ON DELETE CASCADE,
    owner_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    worn_on     TEXT NOT NULL,
    notes       TEXT,
    created_at  TEXT NOT NULL
);

CREATE TABLE outfit_log_wear_logs (
    outfit_log_id TEXT NOT NULL REFERENCES outfit_logs(id) ON DELETE CASCADE,
    wear_log_id   TEXT NOT NULL REFERENCES wear_logs(id) ON DELETE CASCADE,
    PRIMARY KEY (outfit_log_id, wear_log_id)
);

CREATE INDEX idx_outfits_owner_id ON outfits(owner_id);
CREATE INDEX idx_outfit_items_item_id ON outfit_items(item_id);
CREATE INDEX idx_outfit_photos_outfit_id ON outfit_photos(outfit_id);
CREATE INDEX idx_outfit_logs_outfit_id ON outfit_logs(outfit_id);
CREATE INDEX idx_outfit_logs_owner_worn ON outfit_logs(owner_id, worn_on);
CREATE INDEX idx_outfit_log_wear_logs_wear_log_id ON outfit_log_wear_logs(wear_log_id);
