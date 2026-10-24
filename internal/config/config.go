package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AuthDSN       string
	MainDSN       string
	PenginapanDSN string
	UsersDSN      string
	MongoURI      string
	JWTSecret     string
	Port          string

	Bypass                      bool
	APIKey                      string
	MongoDatabase               string
	MongoBoardingCollection     string
	MongoBoardingpassCollection string
	EnableDamriFallback         bool
	DamriUsername               string
	DamriPassword               string
	DamriBaseURL                string
	DamriTimeoutSeconds         int
	SignageOfflineSeconds	int		`gorm:"column:signage_offline_seconds" json:"signage_offline_seconds"`
	IgnoredMongoUserIDs         []int
	LogRetentionDays            int
	UploadDir                   string
}

var C *Config

func Load() *Config {
	c := &Config{
		AuthDSN:       getEnv("AUTH_DSN", "api2:andtho1989@tcp(192.168.1.3:3306)/ttpg_auth?parseTime=true"),
		MainDSN:       getEnv("MAIN_DSN", "api2:andtho1989@tcp(192.168.1.3:3306)/ttpg?parseTime=true"),
		PenginapanDSN: getEnv("PENGINAPAN_DSN", "api2:andtho1989@tcp(192.168.1.3:3306)/ttpg_penginapan?parseTime=true"),
		UsersDSN:      getEnv("USERS_DSN", "api2:andtho1989@tcp(192.168.1.3:3306)/ttpg_users?parseTime=true"),
		MongoURI:      getEnv("MONGO_URI", "mongodb://admin:Andtho1989@192.168.1.3:27017/?authSource=admin&directConnection=true"),
		JWTSecret:     getEnv("JWT_SECRET", "ttpg-secret-key-change-in-production"),
		Port:          getEnv("PORT", "3080"),
		UploadDir:     getEnv("UPLOAD_DIR", "uploads"),

		Bypass:                      getEnv("BYPASS", "false") == "true",
		APIKey:                      getEnv("API_KEY", "12345"),
		MongoDatabase:               getEnv("MONGO_DATABASE", "ttpg"),
		MongoBoardingCollection:     getEnv("MONGO_BOARDING_COLLECTION", "boarding"),
		MongoBoardingpassCollection: getEnv("MONGO_BOARDINGPASS_COLLECTION", "boardingpass"),
		EnableDamriFallback:         getEnv("ENABLE_DAMRI_FALLBACK", "false") == "true",
		DamriUsername:               getEnv("DAMRI_USERNAME", ""),
		DamriPassword:               getEnv("DAMRI_PASSWORD", ""),
		DamriBaseURL:                getEnv("DAMRI_BASE_URL", ""),
		DamriTimeoutSeconds:         10,
		IgnoredMongoUserIDs:         []int{},
		LogRetentionDays:            getEnvInt("LOG_RETENTION_DAYS", 30),
	}
	C = c
	return c
}

func (c *Config) IsIDCardAllowed(noreg string) bool {
	list := getEnv("LIST_ID_CARD", "")
	for _, id := range strings.Split(list, ",") {
		if strings.TrimSpace(id) == noreg {
			return true
		}
	}
	return false
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
