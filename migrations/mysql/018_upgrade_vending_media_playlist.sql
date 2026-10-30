-- Upgrade vending: kategori media (media/screensaver), URL sumber, tabel playlist & playback.
-- Idempoten: aman dijalankan ulang (dipakai reset.sh lokal dan manual di produksi).

SET @has_purpose := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'vending_media' AND COLUMN_NAME = 'purpose');
SET @sql := IF(@has_purpose = 0, 'ALTER TABLE vending_media ADD COLUMN purpose VARCHAR(20) NOT NULL DEFAULT ''media'' AFTER type', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @has_source := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'vending_media' AND COLUMN_NAME = 'source_url');
SET @sql := IF(@has_source = 0, 'ALTER TABLE vending_media ADD COLUMN source_url VARCHAR(500) DEFAULT NULL AFTER checksum', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @has_idx := (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'vending_media' AND INDEX_NAME = 'idx_vending_media_purpose');
SET @sql := IF(@has_idx = 0, 'ALTER TABLE vending_media ADD KEY idx_vending_media_purpose (purpose, enabled, `order`)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS vending_playlists (
  id INT NOT NULL AUTO_INCREMENT,
  name VARCHAR(100) NOT NULL,
  priority INT NOT NULL DEFAULT 0,
  loop_mode ENUM('loop','once') NOT NULL DEFAULT 'loop',
  repeat_count INT NOT NULL DEFAULT 1,
  weight_percent INT NOT NULL DEFAULT 100,
  days_mask TINYINT UNSIGNED NOT NULL DEFAULT 127,
  start_time TIME DEFAULT NULL,
  end_time TIME DEFAULT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_vending_playlists_priority (priority),
  KEY idx_vending_playlists_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vending_playlist_items (
  id INT NOT NULL AUTO_INCREMENT,
  playlist_id INT NOT NULL,
  media_id INT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  repeat_count INT NOT NULL DEFAULT 1,
  weight_percent INT NOT NULL DEFAULT 100,
  duration_seconds INT DEFAULT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_vending_playlist_items_playlist (playlist_id, sort_order),
  KEY idx_vending_playlist_items_media (media_id),
  CONSTRAINT fk_vending_playlist_items_media FOREIGN KEY (media_id) REFERENCES vending_media (id) ON DELETE CASCADE,
  CONSTRAINT fk_vending_playlist_items_playlist FOREIGN KEY (playlist_id) REFERENCES vending_playlists (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vending_machine_playlists (
  id INT NOT NULL AUTO_INCREMENT,
  machine_id VARCHAR(50) NOT NULL,
  playlist_id INT NOT NULL,
  priority_override INT DEFAULT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_vending_machine_playlist (machine_id, playlist_id),
  KEY idx_vending_machine_playlists_machine (machine_id, enabled),
  KEY fk_vending_machine_playlists_playlist (playlist_id),
  CONSTRAINT fk_vending_machine_playlists_playlist FOREIGN KEY (playlist_id) REFERENCES vending_playlists (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vending_play_logs (
  id BIGINT NOT NULL AUTO_INCREMENT,
  machine_id VARCHAR(50) NOT NULL,
  playlist_id INT DEFAULT NULL,
  media_id INT DEFAULT NULL,
  started_at DATETIME NOT NULL,
  duration_seconds INT NOT NULL DEFAULT 0,
  completed TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_vending_play_logs_machine_started (machine_id, started_at),
  KEY idx_vending_play_logs_media_started (media_id, started_at),
  KEY idx_vending_play_logs_playlist_started (playlist_id, started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
