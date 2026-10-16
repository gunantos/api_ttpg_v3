-- Upgrade terminal_boardingpass: kolom gambar KTP & foto tiket (opsional).
-- Jalankan manual di produksi; lokal otomatis via ttpg_schema.sql / reset.
ALTER TABLE terminal_boardingpass
  ADD COLUMN ktp_photo VARCHAR(250) NULL AFTER pembelian_dari,
  ADD COLUMN ticket_photo VARCHAR(250) NULL AFTER ktp_photo;
