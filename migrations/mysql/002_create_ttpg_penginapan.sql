-- Database penginapan (DDL final, RESET: tabel lama di-drop)
CREATE DATABASE IF NOT EXISTS ttpg_penginapan CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE ttpg_penginapan;

DROP TABLE IF EXISTS payment_guests;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS room_histories;
DROP TABLE IF EXISTS room_layouts;
DROP TABLE IF EXISTS guests;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS payment_methods;
DROP TABLE IF EXISTS purchase_sources;
DROP TABLE IF EXISTS configs;

CREATE TABLE configs (
  id_configs INT(11) NOT NULL AUTO_INCREMENT,
  key_configs VARCHAR(250) NOT NULL,
  value_configs TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id_configs),
  UNIQUE KEY uq_configs_key (key_configs)
) ENGINE=InnoDB;

CREATE TABLE payment_methods (
  id INT(11) NOT NULL AUTO_INCREMENT,
  code VARCHAR(30) NOT NULL,
  name VARCHAR(100) NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  sort_order INT(11) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_payment_methods_code (code)
) ENGINE=InnoDB;

CREATE TABLE purchase_sources (
  id INT(11) NOT NULL AUTO_INCREMENT,
  code VARCHAR(30) NOT NULL,
  name VARCHAR(100) NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  sort_order INT(11) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_purchase_sources_code (code)
) ENGINE=InnoDB;

CREATE TABLE locations (
  id INT(11) NOT NULL AUTO_INCREMENT,
  name VARCHAR(100) NOT NULL,
  keterangan VARCHAR(250) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE rooms (
  id INT(11) NOT NULL AUTO_INCREMENT,
  location_id INT(11) NULL,
  room_number VARCHAR(100) NOT NULL,
  room_type ENUM('pria','wanita') NOT NULL,
  capacity INT(11) NOT NULL DEFAULT 1,
  current_occupants INT(11) NOT NULL DEFAULT 0,
  status ENUM('kosong','terisi','maintenance') NOT NULL DEFAULT 'kosong',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_rooms_loc_number (location_id, room_number),
  KEY idx_rooms_location (location_id),
  CONSTRAINT fk_rooms_location FOREIGN KEY (location_id) REFERENCES locations (id) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE guests (
  id INT(11) NOT NULL AUTO_INCREMENT,
  room_id INT(11) NOT NULL,
  name VARCHAR(250) NOT NULL,
  nik VARCHAR(50) NULL,
  gender ENUM('pria','wanita') NOT NULL,
  harga INT(11) NOT NULL DEFAULT 0,
  checkin_date DATE NULL,
  checkin_time TIME NULL,
  checkout_date DATE NULL,
  checkout_time TIME NULL,
  user_id INT(11) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_guests_room (room_id),
  KEY idx_guests_checkin (checkin_date),
  CONSTRAINT fk_guests_room FOREIGN KEY (room_id) REFERENCES rooms (id)
) ENGINE=InnoDB;

CREATE TABLE payments (
  id INT(11) NOT NULL AUTO_INCREMENT,
  amount INT(11) NOT NULL DEFAULT 0,
  payment_method_id INT(11) NOT NULL,
  source_id INT(11) NOT NULL,
  status ENUM('pending','lunas','batal') NOT NULL DEFAULT 'pending',
  note VARCHAR(250) NULL,
  paid_at TIMESTAMP NULL,
  user_id INT(11) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_payments_method (payment_method_id),
  KEY idx_payments_source (source_id),
  CONSTRAINT fk_payments_method FOREIGN KEY (payment_method_id) REFERENCES payment_methods (id),
  CONSTRAINT fk_payments_source FOREIGN KEY (source_id) REFERENCES purchase_sources (id)
) ENGINE=InnoDB;

CREATE TABLE payment_guests (
  id INT(11) NOT NULL AUTO_INCREMENT,
  payment_id INT(11) NOT NULL,
  guest_id INT(11) NOT NULL,
  amount INT(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uq_payment_guest (payment_id, guest_id),
  KEY idx_payment_guests_guest (guest_id),
  CONSTRAINT fk_pg_payment FOREIGN KEY (payment_id) REFERENCES payments (id) ON DELETE CASCADE,
  CONSTRAINT fk_pg_guest FOREIGN KEY (guest_id) REFERENCES guests (id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE room_histories (
  id INT(11) NOT NULL AUTO_INCREMENT,
  guest_id INT(11) NOT NULL,
  room_id INT(11) NOT NULL,
  status ENUM('check-in','check-out','pindah') NOT NULL,
  occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  user_id INT(11) NULL,
  note VARCHAR(250) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_rh_guest (guest_id),
  KEY idx_rh_room (room_id),
  CONSTRAINT fk_rh_guest FOREIGN KEY (guest_id) REFERENCES guests (id),
  CONSTRAINT fk_rh_room FOREIGN KEY (room_id) REFERENCES rooms (id)
) ENGINE=InnoDB;

CREATE TABLE room_layouts (
  id INT(11) NOT NULL AUTO_INCREMENT,
  location_id INT(11) NULL,
  room_id INT(11) NULL,
  `row_number` INT(11) NULL,
  urutan INT(11) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_rl_location (location_id, `row_number`, urutan),
  KEY idx_rl_room (room_id),
  CONSTRAINT fk_rl_location FOREIGN KEY (location_id) REFERENCES locations (id) ON DELETE SET NULL,
  CONSTRAINT fk_rl_room FOREIGN KEY (room_id) REFERENCES rooms (id) ON DELETE SET NULL
) ENGINE=InnoDB;

INSERT INTO payment_methods (code, name, is_active, sort_order) VALUES
  ('cash', 'Cash', 1, 1),
  ('qris', 'QRIS', 1, 2),
  ('transfer', 'Transfer Bank', 1, 3),
  ('debit', 'Kartu Debit', 1, 4),
  ('lainnya', 'Lainnya', 1, 99);

INSERT INTO purchase_sources (code, name, is_active, sort_order) VALUES
  ('langsung', 'Langsung (Loket)', 1, 1),
  ('mesin', 'Mesin (Pending)', 1, 2);
