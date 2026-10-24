-- Digital Signage v2: device auth, settings, telemetry, publish pointer.
ALTER TABLE ds_devices
  ADD COLUMN token_hash VARCHAR(128) DEFAULT NULL,
  ADD COLUMN token_prefix VARCHAR(16) DEFAULT NULL,
  ADD COLUMN token_rotated_at DATETIME(3) DEFAULT NULL,
  ADD COLUMN app_version VARCHAR(50) DEFAULT NULL,
  ADD COLUMN ip_address VARCHAR(64) DEFAULT NULL,
  ADD COLUMN last_sync_at DATETIME(3) DEFAULT NULL,
  ADD COLUMN last_metrics_json MEDIUMTEXT DEFAULT NULL,
  ADD COLUMN current_media_id INT DEFAULT NULL,
  ADD COLUMN current_playlist_id INT DEFAULT NULL,
  ADD COLUMN current_layout_id INT DEFAULT NULL,
  ADD COLUMN screenshot_url VARCHAR(500) DEFAULT NULL,
  ADD COLUMN timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Jakarta',
  ADD COLUMN cache_limit_mb INT NOT NULL DEFAULT 2048,
  ADD COLUMN settings_json TEXT DEFAULT NULL,
  ADD COLUMN published_revision_id BIGINT DEFAULT NULL,
  ADD COLUMN approved_at DATETIME(3) DEFAULT NULL,
  ADD COLUMN last_offline_event_at DATETIME(3) DEFAULT NULL,
  ADD KEY idx_ds_devices_last_seen (last_seen_at),
  ADD KEY idx_ds_devices_status (status),
  ADD KEY idx_ds_devices_token_hash (token_hash);

SET @sql := (
  SELECT IF(COUNT(*) > 0,
    'ALTER TABLE ds_device_playlists DROP INDEX uq_ds_dev_playlist',
    'SELECT 1')
  FROM information_schema.statistics
  WHERE table_schema = DATABASE() AND table_name = 'ds_device_playlists' AND index_name = 'uq_ds_dev_playlist'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := (
  SELECT IF(COUNT(*) > 0,
    'ALTER TABLE ds_devices RENAME INDEX idx_ds_devices_device_id TO uq_ds_devices_device_id',
    'SELECT 1')
  FROM information_schema.statistics
  WHERE table_schema = DATABASE() AND table_name = 'ds_devices' AND index_name = 'idx_ds_devices_device_id'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := (
  SELECT IF(COUNT(*) > 0,
    'ALTER TABLE ds_configs RENAME INDEX idx_ds_configs_device_id TO uq_ds_configs_device',
    'SELECT 1')
  FROM information_schema.statistics
  WHERE table_schema = DATABASE() AND table_name = 'ds_configs' AND index_name = 'idx_ds_configs_device_id'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
