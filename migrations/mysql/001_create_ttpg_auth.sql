-- Database auth baru untuk pengguna & API key
CREATE DATABASE IF NOT EXISTS ttpg_auth CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Tabel dibuat otomatis oleh GORM AutoMigrate di main.go:
--   users(id, username, password, role, po_ids, active, created_at, updated_at, deleted_at)
--   api_keys(id, user_id, key_hash, active, created_at, deleted_at)
