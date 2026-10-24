-- Digital Signage: layout main per device + slot media.
ALTER TABLE ds_devices
  ADD COLUMN main_layout_id INT DEFAULT NULL,
  ADD KEY idx_ds_devices_main_layout (main_layout_id);
