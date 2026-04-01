CREATE TABLE shares (
    id            TEXT NOT NULL PRIMARY KEY,
    owner_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type   TEXT NOT NULL CHECK(target_type IN ('item', 'outfit', 'location')),
    target_id     TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    UNIQUE(owner_id, recipient_id, target_type, target_id)
);

CREATE INDEX idx_shares_owner_id ON shares(owner_id);
CREATE INDEX idx_shares_recipient_id ON shares(recipient_id);
CREATE INDEX idx_shares_target ON shares(target_type, target_id);
