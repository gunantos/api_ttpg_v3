-- Upgrade guests: kolom foto KTP (nama file di $UPLOAD_DIR/ktp).
-- Jalankan manual di produksi bila kolom belum ada (AutoMigrate server juga
-- menambahkannya saat start — jalankan sebelum start binari baru agar deterministik).
ALTER TABLE guests
  ADD COLUMN ktp_photo VARCHAR(250) NULL AFTER nik;
