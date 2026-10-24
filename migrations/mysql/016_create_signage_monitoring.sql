-- Digital Signage v2: telemetry samples, rollups, events.
CREATE TABLE IF NOT EXISTS ds_device_metric_samples (
  id BIGINT NOT NULL AUTO_INCREMENT,
  device_id VARCHAR(50) NOT NULL,
  bucket_at DATETIME(3) NOT NULL,
  cpu_percent DECIMAL(6,2) DEFAULT NULL,
  memory_percent DECIMAL(6,2) DEFAULT NULL,
  disk_percent DECIMAL(6,2) DEFAULT NULL,
  cache_bytes BIGINT DEFAULT NULL,
  metrics_json MEDIUMTEXT DEFAULT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uq_ds_metric_sample (device_id, bucket_at),
  KEY idx_ds_metric_samples_bucket (bucket_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_device_metric_rollups (
  id BIGINT NOT NULL AUTO_INCREMENT,
  device_id VARCHAR(50) NOT NULL,
  bucket_at DATETIME(3) NOT NULL,
  cpu_percent DECIMAL(6,2) DEFAULT NULL,
  memory_percent DECIMAL(6,2) DEFAULT NULL,
  disk_percent DECIMAL(6,2) DEFAULT NULL,
  cache_bytes BIGINT DEFAULT NULL,
  samples INT NOT NULL DEFAULT 0,
  metrics_json MEDIUMTEXT DEFAULT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uq_ds_metric_rollup (device_id, bucket_at),
  KEY idx_ds_metric_rollups_bucket (bucket_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_device_events (
  id BIGINT NOT NULL AUTO_INCREMENT,
  device_id VARCHAR(50) NOT NULL,
  type VARCHAR(40) NOT NULL,
  severity VARCHAR(20) NOT NULL DEFAULT 'info',
  message VARCHAR(255) DEFAULT NULL,
  payload_json TEXT DEFAULT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_ds_device_events_device (device_id, created_at),
  KEY idx_ds_device_events_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
