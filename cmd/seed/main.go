package main

import (
	"crypto/md5"
	"encoding/hex"
	"log"

	"github.com/andtho89/ttpg/internal/config"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/platform/auth"
	"github.com/andtho89/ttpg/internal/platform/database"
)

func main() {
	cfg := config.Load()
	db, err := database.ConnectMySQL(cfg.AuthDSN)
	if err != nil {
		log.Fatalf("authDB: %v", err)
	}
	db.AutoMigrate(&models.User{}, &models.APIKey{})

	hash, _ := auth.HashPassword("admin123")
	realm := "ttpg"
	ha1sum := md5.Sum([]byte("admin:" + realm + ":admin123"))
	digestHA1 := hex.EncodeToString(ha1sum[:])
	admin := models.User{Username: "admin", Password: hash, Role: "admin", Active: true,
		ProductionKey: "prod-key-123", DevelopmentKey: "dev-key-123", DigestHA1: digestHA1}
	if err := db.FirstOrCreate(&admin, models.User{Username: "admin"}).Error; err != nil {
		log.Fatalf("seed: %v", err)
	}
	db.Model(&admin).Updates(map[string]interface{}{"production_key": admin.ProductionKey, "development_key": admin.DevelopmentKey, "digest_ha1": admin.DigestHA1})

	rawKey := "dev-key-123"
	apiKey := models.APIKey{UserID: admin.ID, KeyHash: auth.HashKey(rawKey), Active: true}
	if err := db.Where("key_hash = ?", apiKey.KeyHash).FirstOrCreate(&apiKey).Error; err != nil {
		log.Fatalf("seed key: %v", err)
	}

	log.Printf("Seed OK: admin/admin123, X-ttpg-key: %s", rawKey)
}
