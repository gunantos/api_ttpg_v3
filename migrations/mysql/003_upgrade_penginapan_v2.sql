-- 003_upgrade_penginapan_v2.sql
-- Upgrade ttpg_penginapan dari skema lama ke skema baru (kode aplikasi saat ini).
-- AMAN: tidak ada DROP data — rename tabel + rename kolom + tambah kolom + buat tabel baru + seed.
-- Jalankan dengan user yang punya akses penuh ke ttpg_penginapan.
--
-- Skema lama (produksi):  configs, locations, quests, room, room_history, room_layouts
-- Skema baru (aplikasi):  configs, payment_methods, purchase_sources, locations,
--                         rooms, guests, payments, payment_guests, room_histories, room_layouts

-- ========== 1. Tabel baru ==========

CREATE TABLE IF NOT EXISTS payment_methods (
  id INT(11) NOT NULL AUTO_INCREMENT,
  code VARCHAR(30) NOT NULL,
  name VARCHAR(100) NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  sort_order INT(11) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp(),
  updated_at TIMESTAMP NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (id),
  UNIQUE KEY idx_pm_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS purchase_sources (
  id INT(11) NOT NULL AUTO_INCREMENT,
  code VARCHAR(30) NOT NULL,
  name VARCHAR(100) NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  sort_order INT(11) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp(),
  updated_at TIMESTAMP NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (id),
  UNIQUE KEY idx_ps_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS payments (
  id INT(11) NOT NULL AUTO_INCREMENT,
  amount INT(11) NOT NULL DEFAULT 0,
  payment_method_id INT(11) NOT NULL,
  source_id INT(11) NOT NULL,
  status ENUM('pending','lunas','batal') NOT NULL DEFAULT 'pending',
  note VARCHAR(250) DEFAULT NULL,
  paid_at TIMESTAMP NULL DEFAULT NULL,
  user_id INT(11) DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp(),
  updated_at TIMESTAMP NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (id),
  KEY idx_pay_method (payment_method_id),
  KEY idx_pay_source (source_id),
  CONSTRAINT fk_pay_method FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id),
  CONSTRAINT fk_pay_source FOREIGN KEY (source_id) REFERENCES purchase_sources(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- FK ke guests dibuat SETELAH rename (lihat blok 5)

-- ========== 2. Seed lookup ==========

INSERT IGNORE INTO payment_methods (code, name, is_active, sort_order) VALUES
  ('cash','Cash',1,1), ('qris','QRIS',1,2), ('transfer','Transfer',1,3),
  ('debit','Debit',1,4), ('lainnya','Lainnya',1,5);

INSERT IGNORE INTO purchase_sources (code, name, is_active, sort_order) VALUES
  ('langsung','Langsung (Loket)',1,1), ('mesin','Mesin (Pending)',1,2);

-- ========== 3. Rename tabel (idempoten — hanya jika nama lama masih ada) ==========

SET @has := (SELECT COUNT(*) FROM information_schema.tables
  WHERE table_schema='ttpg_penginapan' AND table_name='quests');
SET @sql := IF(@has > 0, 'RENAME TABLE quests TO guests', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @has := (SELECT COUNT(*) FROM information_schema.tables
  WHERE table_schema='ttpg_penginapan' AND table_name='room');
SET @sql := IF(@has > 0, 'RENAME TABLE room TO rooms', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @has := (SELECT COUNT(*) FROM information_schema.tables
  WHERE table_schema='ttpg_penginapan' AND table_name='room_history');
SET @sql := IF(@has > 0, 'RENAME TABLE room_history TO room_histories', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- ========== 4. guests: tambah kolom user_id ==========

SET @col := (SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema='ttpg_penginapan' AND table_name='guests' AND column_name='user_id');
SET @sql := IF(@col = 0,
  'ALTER TABLE guests ADD COLUMN user_id INT(11) DEFAULT NULL AFTER checkout_time',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- ========== 5. room_histories: sesuaikan kolom ==========

-- guests_id → guest_id
SET @col := (SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema='ttpg_penginapan' AND table_name='room_histories' AND column_name='guests_id');
SET @sql := IF(@col > 0,
  'ALTER TABLE room_histories CHANGE COLUMN guests_id guest_id INT(11) NOT NULL',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- timestamp → occurred_at
SET @col := (SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema='ttpg_penginapan' AND table_name='room_histories' AND column_name='timestamp');
SET @sql := IF(@col > 0,
  'ALTER TABLE room_histories CHANGE COLUMN timestamp occurred_at TIMESTAMP NOT NULL DEFAULT current_timestamp()',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- user_id
SET @col := (SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema='ttpg_penginapan' AND table_name='room_histories' AND column_name='user_id');
SET @sql := IF(@col = 0,
  'ALTER TABLE room_histories ADD COLUMN user_id INT(11) DEFAULT NULL AFTER occurred_at',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- note
SET @col := (SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema='ttpg_penginapan' AND table_name='room_histories' AND column_name='note');
SET @sql := IF(@col = 0,
  'ALTER TABLE room_histories ADD COLUMN note VARCHAR(250) DEFAULT NULL AFTER user_id',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- ========== 6. payment_guests: FK ke guests (setelah rename) ==========

CREATE TABLE IF NOT EXISTS payment_guests (
  id INT(11) NOT NULL AUTO_INCREMENT,
  payment_id INT(11) NOT NULL,
  guest_id INT(11) NOT NULL,
  amount INT(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY idx_payment_guest (payment_id, guest_id),
  KEY idx_pg_guest (guest_id),
  CONSTRAINT fk_pg_payment FOREIGN KEY (payment_id) REFERENCES payments(id),
  CONSTRAINT fk_pg_guest FOREIGN KEY (guest_id) REFERENCES guests(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ========== 7. Sinkronkan current_occupants & status ==========

UPDATE rooms r
  SET r.current_occupants = (
    SELECT COUNT(*) FROM guests g
    WHERE g.room_id = r.id AND g.checkout_date IS NULL
  );

UPDATE rooms r
  SET r.status = CASE
    WHEN r.status = 'maintenance' THEN 'maintenance'
    WHEN r.current_occupants = 0 THEN 'kosong'
    ELSE 'terisi'
  END;

-- ========== 8. Ringkasan ==========

SELECT 'MIGRASI SELESAI' AS status,
  (SELECT COUNT(*) FROM guests) AS total_guests,
  (SELECT COUNT(*) FROM rooms) AS total_rooms,
  (SELECT COUNT(*) FROM room_histories) AS total_histories,
  (SELECT COUNT(*) FROM payment_methods) AS total_methods,
  (SELECT COUNT(*) FROM purchase_sources) AS total_sources,
  (SELECT COUNT(*) FROM payments) AS total_payments;
