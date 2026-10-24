-- Digital Signage layout media (DB `ttpg_digital_signage`) — jalankan manual di produksi.
ALTER TABLE ds_media
  ADD COLUMN name VARCHAR(150) DEFAULT NULL AFTER type,
  ADD COLUMN layout_json TEXT DEFAULT NULL AFTER url;
