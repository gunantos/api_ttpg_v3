-- Vending (mesin OTOMAT) — DB utama `ttpg`; jalankan manual di produksi (seperti 005).
CREATE TABLE IF NOT EXISTS vending_machines (
  id INT NOT NULL AUTO_INCREMENT,
  machine_id VARCHAR(50) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  name VARCHAR(100) DEFAULT NULL,
  app_version VARCHAR(30) DEFAULT NULL,
  config_version VARCHAR(50) DEFAULT NULL,
  last_sync_at DATETIME DEFAULT NULL,
  last_seen_at DATETIME DEFAULT NULL,
  extra TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_vending_machines_machine_id (machine_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vending_configs (
  id INT NOT NULL AUTO_INCREMENT,
  machine_id VARCHAR(50) NOT NULL DEFAULT '__global__',
  version VARCHAR(50) NOT NULL,
  payload TEXT NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_vending_configs_machine_id (machine_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS vending_media (
  id INT NOT NULL AUTO_INCREMENT,
  type VARCHAR(20) NOT NULL DEFAULT 'video',
  url VARCHAR(500) NOT NULL,
  `order` INT NOT NULL DEFAULT 0,
  checksum VARCHAR(100) DEFAULT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_vending_media_order (`order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO vending_configs (machine_id, version, payload, updated_at)
SELECT '__global__', 'cfg-20261015-1',
 '{"cameras":{"ktp":null,"person":null},"ocr":{"crop":null,"language":"eng+ind","confidence_min":0.7},"idle":{"minutes":3,"enabled":true},"attract":{"mode":"hybrid","sources":[]},"speech":{"enabled":true,"model":"whisper-small","language":"id"},"media_cache_limit_mb":500}',
 '2026-10-15 10:00:00'
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM vending_configs WHERE machine_id = '__global__');
