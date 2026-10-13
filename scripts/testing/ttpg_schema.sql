CREATE TABLE tbl_tujuan (
  id_tujuan INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  provinsi VARCHAR(50) NOT NULL DEFAULT '',
  tujuan VARCHAR(100) NOT NULL,
  kawasan VARCHAR(50) NOT NULL DEFAULT '',
  kode_tujuan VARCHAR(20) NOT NULL DEFAULT '',
  nama_terminal VARCHAR(100) NOT NULL DEFAULT '',
  create_date DATETIME NULL,
  update_date DATETIME NULL,
  UNIQUE KEY uq_tujuan (tujuan)
) ENGINE=InnoDB;

CREATE TABLE tbl_po (
  id_po INT NOT NULL PRIMARY KEY,
  nama_po VARCHAR(100) NOT NULL DEFAULT '',
  nama_singkat VARCHAR(50) NOT NULL DEFAULT '',
  pt_po VARCHAR(100) NOT NULL DEFAULT '',
  almt_po VARCHAR(255) NOT NULL DEFAULT '',
  tlp_po VARCHAR(30) NOT NULL DEFAULT '',
  gambar_po VARCHAR(255) NOT NULL DEFAULT '',
  email VARCHAR(100) NOT NULL DEFAULT '',
  kode_perusahaan VARCHAR(30) NOT NULL DEFAULT '',
  no_sk VARCHAR(50) NOT NULL DEFAULT '',
  exp_sk VARCHAR(30) NOT NULL DEFAULT '',
  is_active INT NOT NULL DEFAULT 1,
  create_date DATETIME NULL,
  update_date DATETIME NULL
) ENGINE=InnoDB;

CREATE TABLE terminal_boardingpass (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  noreg VARCHAR(50) NOT NULL,
  ticket_id VARCHAR(50) NULL,
  id_po VARCHAR(20) NOT NULL DEFAULT '',
  id_tujuan INT NULL,
  nopol VARCHAR(30) NOT NULL DEFAULT '',
  tanggal_berangkat DATE NOT NULL,
  jam_berangkat TIME NOT NULL,
  nama_pnp VARCHAR(150) NOT NULL DEFAULT '',
  sex_pnp VARCHAR(10) NULL,
  fungsi VARCHAR(30) NULL,
  gate_id INT NULL,
  device_id VARCHAR(100) NULL,
  time_tab DATETIME NULL,
  id_user VARCHAR(50) NULL,
  retribusi INT NOT NULL DEFAULT 0,
  count_scan INT NULL,
  pembelian_dari VARCHAR(100) NULL,
  created DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated DATETIME NULL,
  deleted DATETIME NULL,
  UNIQUE KEY uq_noreg (noreg),
  KEY idx_ticket (ticket_id),
  KEY idx_fungsi_tgl (fungsi, tanggal_berangkat)
) ENGINE=InnoDB;

INSERT INTO tbl_tujuan (provinsi, tujuan, kawasan, kode_tujuan, nama_terminal) VALUES
  ('JK', 'Jakarta', 'JK', 'JKT', 'Terminal Pulogebang'),
  ('BT', 'Bekasi', 'BT', 'BKS', 'Terminal Bekasi'),
  ('JB', 'Bandung', 'JB', 'BDG', 'Terminal Cicaheum'),
  ('JT', 'Semarang', 'JT', 'SMG', 'Terminal Terboyo');

INSERT INTO tbl_po (id_po, nama_po, nama_singkat) VALUES
  (1, 'PO Contoh Utama', 'PO001'),
  (28, 'Damri', 'PO028'),
  (122, 'PO Loket Eksternal', 'PO122');

INSERT INTO terminal_boardingpass
  (noreg, ticket_id, id_po, id_tujuan, nopol, tanggal_berangkat, jam_berangkat,
   nama_pnp, sex_pnp, fungsi, gate_id, device_id, time_tab, retribusi, count_scan,
   pembelian_dari, created, updated)
VALUES
  ('SCAN001', 'TKT-001', 'PO001', 1, 'B 1111 XY', CURDATE(), '08:00:00',
   'Budi Santoso', 'L', 'checkin', NULL, NULL, NULL, 0, 0,
   'loket', NOW(), NOW()),
  ('EXPIRE01', 'TKT-002', 'PO001', 1, 'B 2222 XY', CURDATE(), '09:00:00',
   'Siti Aminah', 'P', 'gatein', 1, '10.0.0.1', DATE_SUB(NOW(), INTERVAL 10 MINUTE), 0, 1,
   'loket', NOW(), NOW()),
  ('MAXSCAN1', 'TKT-003', 'PO001', 1, 'B 3333 XY', CURDATE(), '10:00:00',
   'Andi Wijaya', 'L', 'gatein', 1, '10.0.0.1', NOW(), 0, 5,
   'loket', NOW(), NOW()),
  ('OLDDATE1', 'TKT-004', 'PO001', 1, 'B 4444 XY', DATE_SUB(CURDATE(), INTERVAL 1 DAY), '11:00:00',
   'Rina Marlina', 'P', 'checkin', NULL, NULL, NULL, 0, 0,
   'loket', NOW(), NOW());
