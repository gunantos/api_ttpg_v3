-- Digital Signage (DB terpisah `ttpg_digital_signage`) — jalankan manual di produksi.
CREATE TABLE IF NOT EXISTS ds_devices (
  id INT NOT NULL AUTO_INCREMENT,
  device_id VARCHAR(50) NOT NULL,
  name VARCHAR(100) DEFAULT NULL,
  location VARCHAR(150) DEFAULT NULL,
  resolution VARCHAR(30) DEFAULT NULL,
  orientation VARCHAR(20) NOT NULL DEFAULT 'landscape',
  volume INT NOT NULL DEFAULT 80,
  brightness INT NOT NULL DEFAULT 80,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  last_seen_at DATETIME DEFAULT NULL,
  config_version VARCHAR(50) DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_ds_devices_device_id (device_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_media (
  id INT NOT NULL AUTO_INCREMENT,
  type VARCHAR(20) NOT NULL,
  url VARCHAR(500) NOT NULL,
  checksum VARCHAR(128) DEFAULT NULL,
  duration_seconds INT NOT NULL DEFAULT 10,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_playlists (
  id INT NOT NULL AUTO_INCREMENT,
  name VARCHAR(100) NOT NULL,
  priority INT NOT NULL DEFAULT 0,
  loop_mode VARCHAR(10) NOT NULL DEFAULT 'loop',
  repeat_count INT NOT NULL DEFAULT 1,
  weight_percent INT NOT NULL DEFAULT 100,
  days_mask INT NOT NULL DEFAULT 127,
  start_time TIME DEFAULT NULL,
  end_time TIME DEFAULT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_playlist_items (
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
  KEY idx_ds_playlist_items_playlist (playlist_id, sort_order),
  CONSTRAINT fk_ds_playlist_items_playlist FOREIGN KEY (playlist_id) REFERENCES ds_playlists(id) ON DELETE CASCADE,
  CONSTRAINT fk_ds_playlist_items_media FOREIGN KEY (media_id) REFERENCES ds_media(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_device_playlists (
  id INT NOT NULL AUTO_INCREMENT,
  device_id VARCHAR(50) NOT NULL,
  playlist_id INT NOT NULL,
  priority_override INT DEFAULT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_ds_device_playlist (device_id, playlist_id),
  CONSTRAINT fk_ds_device_playlists_playlist FOREIGN KEY (playlist_id) REFERENCES ds_playlists(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_schedules (
  id INT NOT NULL AUTO_INCREMENT,
  device_id VARCHAR(50) DEFAULT NULL,
  playlist_id INT NOT NULL,
  days_mask INT NOT NULL DEFAULT 127,
  start_time TIME DEFAULT NULL,
  end_time TIME DEFAULT NULL,
  start_date DATE DEFAULT NULL,
  end_date DATE DEFAULT NULL,
  priority INT NOT NULL DEFAULT 0,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ds_schedules_device (device_id, enabled),
  CONSTRAINT fk_ds_schedules_playlist FOREIGN KEY (playlist_id) REFERENCES ds_playlists(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_play_logs (
  id BIGINT NOT NULL AUTO_INCREMENT,
  device_id VARCHAR(50) NOT NULL,
  playlist_id INT DEFAULT NULL,
  media_id INT DEFAULT NULL,
  started_at DATETIME NOT NULL,
  duration_seconds INT NOT NULL DEFAULT 0,
  completed TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_ds_play_logs_device_started (device_id, started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_configs (
  id INT NOT NULL AUTO_INCREMENT,
  device_id VARCHAR(50) NOT NULL,
  version VARCHAR(50) NOT NULL,
  payload TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_ds_configs_device (device_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO ds_configs (device_id, version, payload, updated_at)
SELECT '__global__', 'cfg-1',
 '{"idle":{"minutes":5,"enabled":true},"layout":{"zones":1},"volume":80,"brightness":80}',
 '2026-10-20 00:00:00'
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM ds_configs WHERE device_id = '__global__');
