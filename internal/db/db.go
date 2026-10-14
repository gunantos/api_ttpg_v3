package db

import (
	"database/sql"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"
)

var (
	DB           *sql.DB
	MongoClient  *mongo.Client
	PenginapanDB *gorm.DB
	UsersDB      *gorm.DB
)
