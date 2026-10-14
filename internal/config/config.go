package config

import (
	"os"
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
	IgnoredMongoUserIDs         []int
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
