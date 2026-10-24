-- Digital Signage v2: groups, revisions, publish, commands, tombstones.
CREATE TABLE IF NOT EXISTS ds_device_groups (
  id INT NOT NULL AUTO_INCREMENT,
  name VARCHAR(100) NOT NULL,
  description VARCHAR(255) DEFAULT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uq_ds_device_groups_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_device_group_members (
  id INT NOT NULL AUTO_INCREMENT,
  group_id INT NOT NULL,
  device_id VARCHAR(50) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uq_ds_group_device (group_id, device_id),
  KEY idx_ds_group_members_device (device_id),
  CONSTRAINT fk_ds_group_members_group FOREIGN KEY (group_id) REFERENCES ds_device_groups(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_revisions (
  id BIGINT NOT NULL AUTO_INCREMENT,
  version VARCHAR(80) NOT NULL,
  scope_type VARCHAR(20) NOT NULL,
  scope_id VARCHAR(80) NOT NULL DEFAULT '',
  manifest_json MEDIUMTEXT NOT NULL,
  checksum VARCHAR(128) NOT NULL,
  note VARCHAR(255) DEFAULT NULL,
  created_by VARCHAR(100) DEFAULT NULL,
  parent_revision_id BIGINT DEFAULT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uq_ds_revisions_version (version),
  KEY idx_ds_revisions_scope (scope_type, scope_id, id),
  KEY idx_ds_revisions_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_revision_assets (
  id BIGINT NOT NULL AUTO_INCREMENT,
  revision_id BIGINT NOT NULL,
  asset_type VARCHAR(20) NOT NULL,
  asset_id INT NOT NULL,
  checksum VARCHAR(128) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_ds_revision_asset (revision_id, asset_type, asset_id),
  KEY idx_ds_revision_assets_asset (asset_type, asset_id),
  CONSTRAINT fk_ds_revision_assets_revision FOREIGN KEY (revision_id) REFERENCES ds_revisions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_publish_events (
  id BIGINT NOT NULL AUTO_INCREMENT,
  revision_id BIGINT NOT NULL,
  action VARCHAR(20) NOT NULL DEFAULT 'publish',
  scope_type VARCHAR(20) NOT NULL,
  scope_id VARCHAR(80) NOT NULL DEFAULT '',
  note VARCHAR(255) DEFAULT NULL,
  created_by VARCHAR(100) DEFAULT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_ds_publish_events_revision (revision_id),
  CONSTRAINT fk_ds_publish_events_revision FOREIGN KEY (revision_id) REFERENCES ds_revisions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_commands (
  id BIGINT NOT NULL AUTO_INCREMENT,
  device_id VARCHAR(50) NOT NULL,
  action VARCHAR(50) NOT NULL,
  payload_json TEXT DEFAULT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  created_by VARCHAR(100) DEFAULT NULL,
  expires_at DATETIME(3) DEFAULT NULL,
  sent_at DATETIME(3) DEFAULT NULL,
  acked_at DATETIME(3) DEFAULT NULL,
  result_json TEXT DEFAULT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_ds_commands_device_status (device_id, status, id),
  KEY idx_ds_commands_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ds_tombstones (
  id BIGINT NOT NULL AUTO_INCREMENT,
  asset_type VARCHAR(20) NOT NULL,
  asset_id INT NOT NULL,
  deleted_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uq_ds_tombstone_asset (asset_type, asset_id),
  KEY idx_ds_tombstones_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE ds_playlists
  ADD COLUMN selection_mode VARCHAR(20) NOT NULL DEFAULT 'sequential';

ALTER TABLE ds_schedules
  ADD COLUMN group_id INT DEFAULT NULL,
  ADD KEY idx_ds_schedules_group (group_id, enabled);
