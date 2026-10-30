CREATE TABLE IF NOT EXISTS `vending_media_devices` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `media_id` INT(11) NOT NULL,
  `device_id` VARCHAR(50) NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_vending_media_device` (`media_id`, `device_id`),
  KEY `idx_vending_media_devices_device` (`device_id`),
  CONSTRAINT `fk_vending_media_devices_media` FOREIGN KEY (`media_id`) REFERENCES `vending_media` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
