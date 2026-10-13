package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Username       string         `gorm:"uniqueIndex;size:50" json:"username"`
	Password       string         `json:"-"`
	Role           string         `json:"role"`
	POIDs          string         `json:"po_ids"` // format: "1,2,3" or empty
	Active         bool           `json:"active"`
	ProductionKey  string         `json:"production_key,omitempty"`
	DevelopmentKey string         `json:"development_key,omitempty"`
	DigestHA1      string         `json:"-"` // md5(username:realm:password) untuk Digest auth
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type APIKey struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `json:"user_id"`
	KeyHash   string         `json:"-"`
	Active    bool           `json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Boarding struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	POID      int            `json:"po_id"`
	UserID    uint           `json:"user_id"`
	Status    string         `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Penginapan struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	POID      int            `json:"po_id"`
	UserID    uint           `json:"user_id"`
	Nama      string         `json:"nama"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Permission maps roles to allowed endpoint operations
type Permission struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Role     string `json:"role"`
	Endpoint string `json:"endpoint"` // e.g. "/boarding"
	Method   string `json:"method"`   // GET, POST, PUT, DELETE
}
