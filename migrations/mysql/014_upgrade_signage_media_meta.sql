-- Digital Signage v2: media metadata for cache/download.
ALTER TABLE ds_media
  ADD COLUMN mime_type VARCHAR(100) DEFAULT NULL,
  ADD COLUMN size_bytes BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN width INT NOT NULL DEFAULT 0,
  ADD COLUMN height INT NOT NULL DEFAULT 0,
  ADD COLUMN thumbnail_url VARCHAR(500) DEFAULT NULL,
  ADD COLUMN origin_url VARCHAR(500) DEFAULT NULL,
  ADD KEY idx_ds_media_checksum (checksum),
  ADD KEY idx_ds_media_type_enabled (type, enabled);
