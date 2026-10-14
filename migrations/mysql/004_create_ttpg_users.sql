-- 004: skema ttpg_users (auth produksi) — disalin dari 192.168.1.3/ttpg_users (lihat docs/api.md).
-- Lokal: dimuat reset.sh. Produksi: tabel sudah ada, TIDAK dijalankan di sana.
CREATE DATABASE IF NOT EXISTS `ttpg_users` DEFAULT CHARACTER SET latin1;
USE `ttpg_users`;

DROP TABLE IF EXISTS `logs`;
DROP TABLE IF EXISTS `limits`;
DROP TABLE IF EXISTS `ip_white_list`;
DROP TABLE IF EXISTS `ip_black_list`;
DROP TABLE IF EXISTS `user_detail`;
DROP TABLE IF EXISTS `access`;
DROP TABLE IF EXISTS `api_url`;
DROP TABLE IF EXISTS `user`;

CREATE TABLE `logs` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uri` varchar(250) NOT NULL,
  `method` varchar(6) NOT NULL,
  `params` text DEFAULT NULL,
  `api_key` varchar(40) NOT NULL,
  `ip_address` varchar(45) NOT NULL,
  `time` int(11) NOT NULL,
  `rtime` float DEFAULT NULL,
  `authorized` varchar(1) NOT NULL,
  `response_code` smallint(3) NOT NULL,
  `id_user` int(11) NOT NULL,
  `po_id` varchar(50) DEFAULT NULL,
  `access_by` varchar(20) NOT NULL,
  `date` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 ROW_FORMAT=COMPACT;

CREATE TABLE `limits` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uri` varchar(255) NOT NULL,
  `count` int(10) NOT NULL,
  `hour_started` int(11) NOT NULL,
  `api_key` varchar(40) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

CREATE TABLE `ip_white_list` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) NOT NULL,
  `ip_address` varchar(20) NOT NULL,
  `create_date` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

CREATE TABLE `ip_black_list` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `ip_address` varchar(20) NOT NULL,
  `notes` text NOT NULL,
  `create_date` datetime NOT NULL,
  `modife_date` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

CREATE TABLE `user_detail` (
  `id` int(11) NOT NULL,
  `nama_user` varchar(45) DEFAULT NULL,
  `email_user` varchar(55) DEFAULT NULL,
  `gambar_user` varchar(45) DEFAULT NULL,
  `telp_user` varchar(45) DEFAULT NULL,
  `website_user` varchar(45) DEFAULT NULL,
  `nama_contact` varchar(45) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

CREATE TABLE `access` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) NOT NULL,
  `id_api` varchar(250) NOT NULL,
  `block` enum('y','n') NOT NULL DEFAULT 'n',
  `allow` enum('y','n') NOT NULL DEFAULT 'n',
  `create_time` datetime NOT NULL,
  `update_time` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

CREATE TABLE `api_url` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `authorization` enum('api_key','bearer_token','basic_auth','digest_auth','oauth1','oauth2','none') NOT NULL DEFAULT 'api_key',
  `ajax` enum('0','1','2') NOT NULL DEFAULT '0',
  `controller` varchar(250) NOT NULL,
  `method` enum('*','get','post','put','patch','delete','options','head') NOT NULL DEFAULT 'get',
  `ignore_limits` int(11) NOT NULL DEFAULT 0,
  `help` text DEFAULT NULL,
  `create_date` timestamp NOT NULL DEFAULT current_timestamp(),
  `update_date` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

CREATE TABLE `user` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `access_id` int(11) NOT NULL DEFAULT 2,
  `po_id` varchar(50) DEFAULT NULL,
  `username` varchar(150) NOT NULL,
  `password` varchar(250) NOT NULL,
  `public_key` varchar(150) NOT NULL,
  `secret_key` varchar(150) NOT NULL,
  `development_key` varchar(150) NOT NULL,
  `level` int(11) DEFAULT NULL,
  `all_access` enum('y','n') NOT NULL DEFAULT 'n',
  `allowed` varchar(250) DEFAULT NULL,
  `is_block` int(4) NOT NULL DEFAULT 0,
  `create_date` datetime NOT NULL DEFAULT current_timestamp(),
  `update_date` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `is_approved` int(11) DEFAULT 1,
  PRIMARY KEY (`id`),
  UNIQUE KEY `username_UNIQUE` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- seed lokal
INSERT INTO `user` (`access_id`,`po_id`,`username`,`password`,`public_key`,`secret_key`,`development_key`,`level`,`all_access`,`allowed`,`is_block`,`is_approved`) VALUES
(1,NULL,'admin','$2a$10$ddOHD26AzTrJeUtfSNHDd.O8Dexx3gnDFkULGUiTK/QJzMTdWdhyy','pub-admin-123','dev-key-123','dev-key-123',NULL,'y',NULL,0,1),
(2,NULL,'limited','$2a$10$YwmO7dr0f.DiRA5pmoMDxOKr8P3mcNVulYWUBU8n7BDUJ.x/1102G','pub-lim-123','limited-key','lim-dev-key',1,'n',NULL,0,1);

INSERT INTO `access` (`user_id`,`id_api`,`block`,`allow`,`create_time`) VALUES
(1,'1;2','n','y',NOW()),(2,'2','n','y',NOW());

-- aturan endpoint: contoh proteksi ketat (Bearer only)
INSERT INTO `api_url` (`authorization`,`ajax`,`controller`,`method`,`ignore_limits`,`help`) VALUES
('bearer_token','0','/boarding','post',0,'contoh: POST /boarding wajib Bearer');
