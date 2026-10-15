-- Upgrade payments: kolom payment gateway (midtrans/xendit) untuk DB lama.
-- Jalankan manual di ttpg_penginapan produksi bila tabel payments sudah ada tanpa kolom gateway.
ALTER TABLE payments
  ADD COLUMN gateway VARCHAR(20) NULL AFTER user_id,
  ADD COLUMN external_id VARCHAR(100) NULL AFTER gateway,
  ADD KEY idx_payments_external (external_id),
  ADD COLUMN qr_string VARCHAR(1000) NULL AFTER external_id,
  ADD COLUMN invoice_url VARCHAR(500) NULL AFTER qr_string,
  ADD COLUMN snap_token VARCHAR(255) NULL AFTER invoice_url,
  ADD COLUMN gateway_status VARCHAR(30) NULL AFTER snap_token,
  ADD COLUMN gateway_expires_at DATETIME(3) NULL AFTER gateway_status;
