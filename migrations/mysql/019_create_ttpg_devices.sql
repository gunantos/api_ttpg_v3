-- Unifikasi device: satu tabel `devices` untuk vending + digital signage.
-- Idempoten: aman dijalankan ulang. Salinan data lama tidak menghapus tabel sumber
-- (vending_machines / ds_devices tetap sebagai arsip; handler memakai `devices`).

CREATE TABLE IF NOT EXISTS devices (
  id INT NOT NULL AUTO_INCREMENT,
  type VARCHAR(20) NOT NULL DEFAULT 'vending',
  device_id VARCHAR(50) NOT NULL,
  name VARCHAR(100) DEFAULT NULL,
  location VARCHAR(150) DEFAULT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  app_version VARCHAR(50) DEFAULT NULL,
  config_version VARCHAR(50) DEFAULT NULL,
  last_sync_at DATETIME(3) DEFAULT NULL,
  last_seen_at DATETIME(3) DEFAULT NULL,
  ip_address VARCHAR(64) DEFAULT NULL,
  resolution VARCHAR(30) DEFAULT NULL,
  orientation VARCHAR(20) DEFAULT 'landscape',
  volume INT DEFAULT 80,
  brightness INT DEFAULT 80,
  timezone VARCHAR(64) DEFAULT 'Asia/Jakarta',
  cache_limit_mb INT DEFAULT 2048,
  extra TEXT,
  settings_json TEXT,
  last_metrics_json MEDIUMTEXT,
  token_hash VARCHAR(128) DEFAULT NULL,
  token_prefix VARCHAR(16) DEFAULT NULL,
  token_rotated_at DATETIME(3) DEFAULT NULL,
  current_media_id INT DEFAULT NULL,
  current_playlist_id INT DEFAULT NULL,
  current_layout_id INT DEFAULT NULL,
  main_layout_id INT DEFAULT NULL,
  screenshot_url VARCHAR(500) DEFAULT NULL,
  published_revision_id BIGINT UNSIGNED DEFAULT NULL,
  approved_at DATETIME(3) DEFAULT NULL,
  last_offline_event_at DATETIME(3) DEFAULT NULL,
  cpu_percent INT NOT NULL DEFAULT 0,
  memory_percent INT NOT NULL DEFAULT 0,
  disk_percent INT NOT NULL DEFAULT 0,
  last_metric_at DATETIME(3) DEFAULT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uq_devices_type_device (type, device_id),
  KEY idx_devices_type_status (type, status),
  KEY idx_devices_last_seen (last_seen_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET @has_v := (
  SELECT COUNT(*) FROM information_schema.TABLES
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'vending_machines'
);
SET @sql := IF(@has_v > 0,
  'INSERT IGNORE INTO devices
     (type, device_id, name, status, app_version, config_version, last_sync_at, last_seen_at, extra, created_at, updated_at)
   SELECT ''vending'', machine_id, name, COALESCE(status,''pending''), app_version, config_version,
          last_sync_at, last_seen_at, extra, COALESCE(created_at, NOW(3)), COALESCE(updated_at, NOW(3))
   FROM vending_machines',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @has_ds := (
  SELECT COUNT(*) FROM information_schema.TABLES
  WHERE TABLE_SCHEMA = 'ttpg_digital_signage' AND TABLE_NAME = 'ds_devices'
);
SET @sql := IF(@has_ds > 0,
  'INSERT IGNORE INTO devices
     (type, device_id, name, location, status, app_version, config_version, last_sync_at, last_seen_at,
      ip_address, resolution, orientation, volume, brightness, timezone, cache_limit_mb,
      settings_json, last_metrics_json, token_hash, token_prefix, token_rotated_at,
      current_media_id, current_playlist_id, current_layout_id, main_layout_id, screenshot_url,
      published_revision_id, approved_at, last_offline_event_at, created_at, updated_at)
   SELECT ''signage'', device_id, name, location, COALESCE(status,''pending''), app_version, config_version,
      last_sync_at, last_seen_at, ip_address, resolution, COALESCE(orientation,''landscape''),
      COALESCE(volume,80), COALESCE(brightness,80), COALESCE(timezone,''Asia/Jakarta''), COALESCE(cache_limit_mb,2048),
      settings_json, last_metrics_json, token_hash, token_prefix, token_rotated_at,
      current_media_id, current_playlist_id, current_layout_id, main_layout_id, screenshot_url,
      published_revision_id, approved_at, last_offline_event_at,
      COALESCE(created_at, NOW(3)), COALESCE(updated_at, NOW(3))
   FROM ttpg_digital_signage.ds_devices',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
