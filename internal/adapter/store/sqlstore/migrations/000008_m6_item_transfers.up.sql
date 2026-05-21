CREATE TABLE item_transfers (
    id               TEXT NOT NULL PRIMARY KEY,
    item_id          TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    sender_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status           TEXT NOT NULL CHECK(status IN ('pending', 'accepted', 'rejected', 'cancelled')),
    transfer_history INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT NOT NULL,
    decided_at       TEXT
);

CREATE UNIQUE INDEX idx_item_transfers_pending_item
    ON item_transfers (item_id)
    WHERE status = 'pending';

CREATE INDEX idx_item_transfers_sender_id ON item_transfers (sender_id);
CREATE INDEX idx_item_transfers_recipient_id ON item_transfers (recipient_id);
