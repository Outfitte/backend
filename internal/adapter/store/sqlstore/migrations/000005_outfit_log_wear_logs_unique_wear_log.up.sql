DROP INDEX IF EXISTS idx_outfit_log_wear_logs_wear_log_id;

CREATE UNIQUE INDEX idx_outfit_log_wear_logs_wear_log_id
    ON outfit_log_wear_logs(wear_log_id);
