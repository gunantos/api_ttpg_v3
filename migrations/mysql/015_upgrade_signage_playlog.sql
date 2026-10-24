-- Digital Signage v2: idempotent proof-of-play.
ALTER TABLE ds_play_logs
  ADD COLUMN event_id VARCHAR(100) DEFAULT NULL,
  ADD COLUMN ended_at DATETIME(3) DEFAULT NULL,
  ADD COLUMN error_message VARCHAR(255) DEFAULT NULL,
  ADD COLUMN item_id INT DEFAULT NULL,
  ADD COLUMN layout_id INT DEFAULT NULL,
  ADD COLUMN revision_id BIGINT DEFAULT NULL;

UPDATE ds_play_logs SET event_id = CONCAT('legacy-', id) WHERE event_id IS NULL OR event_id = '';

ALTER TABLE ds_play_logs
  MODIFY COLUMN event_id VARCHAR(100) NOT NULL,
  ADD UNIQUE KEY uq_ds_play_logs_event (event_id),
  ADD KEY idx_ds_play_logs_revision (revision_id),
  ADD KEY idx_ds_play_logs_created (created_at);
