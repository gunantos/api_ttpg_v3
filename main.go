package main

import (
	"context"
	"log"

	"github.com/andtho89/ttpg/internal/config"
	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/platform/auth"
	"github.com/andtho89/ttpg/internal/platform/database"
	"github.com/andtho89/ttpg/internal/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	auth.SetSecret(cfg.JWTSecret)

	authDB, err := database.ConnectMySQL(cfg.AuthDSN)
	if err != nil {
		log.Fatalf("authDB: %v", err)
	}
	auth.SetDB(authDB)
	if err := authDB.AutoMigrate(&models.User{}, &models.APIKey{}); err != nil {
		log.Fatalf("migrate authDB: %v", err)
	}

	usersDB, err := database.ConnectMySQL(cfg.UsersDSN)
	if err != nil {
		log.Fatalf("usersDB: %v", err)
	}
	db.UsersDB = usersDB
	auth.SetLogRetention(cfg.LogRetentionDays)
	auth.SetUsersDB(usersDB)

	mainDB, err := database.ConnectMySQL(cfg.MainDSN)
	if err != nil {
		log.Fatalf("mainDB: %v", err)
	}

	penginapanDB, err := database.ConnectMySQL(cfg.PenginapanDSN)
	if err != nil {
		log.Fatalf("penginapanDB: %v", err)
	}
	if err := penginapanDB.AutoMigrate(
		&models.PenginapanConfig{},
		&models.PaymentMethod{},
		&models.PurchaseSource{},
		&models.PenginapanLocation{},
		&models.PenginapanRoom{},
		&models.PenginapanGuest{},
		&models.Payment{},
		&models.PaymentGuest{},
		&models.RoomHistory{},
		&models.RoomLayout{},
	); err != nil {
		log.Fatalf("migrate penginapanDB: %v", err)
	}

	mongoClient, err := database.ConnectMongo(cfg.MongoURI)
	if err != nil {
		log.Fatalf("mongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	sqlDB, err := mainDB.DB()
	if err != nil {
		log.Fatalf("mainDB sql: %v", err)
	}
	db.DB = sqlDB
	db.MongoClient = mongoClient
	db.PenginapanDB = penginapanDB

	r := gin.Default()
	routes.RegisterRoutes(r, mainDB, mongoClient)

	log.Printf("Server running on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
