package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const TokenExpiry = 7 * 24 * time.Hour // 1 minggu

var jwtSecret = []byte("ttpg-secret-key-change-in-production")
var AuthDB *gorm.DB

type Claims struct {
	UserID uint     `json:"user_id"`
	Role   string   `json:"role"`
	POIDs  []string `json:"po_ids"`
	jwt.RegisteredClaims
}

func SetSecret(secret string) {
	jwtSecret = []byte(secret)
}

func SetDB(db *gorm.DB) {
	AuthDB = db
}

func GenerateToken(userID uint, role string, poIDs []string) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		POIDs:  poIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token invalid: %w", err)
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("token invalid")
}

func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func Login(username, password string) (*models.User, error) {
	var u models.User
	if err := AuthDB.Where("username = ? AND active = true", username).First(&u).Error; err != nil {
		return nil, fmt.Errorf("user tidak ditemukan")
	}
	if !CheckPassword(u.Password, password) {
		return nil, fmt.Errorf("password salah")
	}
	return &u, nil
}

func ValidateAPIKey(key string) (*models.APIKey, *models.User, error) {
	var k models.APIKey
	if err := AuthDB.Where("key_hash = ? AND active = true", HashKey(key)).First(&k).Error; err != nil {
		return nil, nil, fmt.Errorf("api key tidak valid")
	}
	var u models.User
	if err := AuthDB.Where("id = ? AND active = true", k.UserID).First(&u).Error; err != nil {
		return nil, nil, fmt.Errorf("user tidak ditemukan")
	}
	return &k, &u, nil
}

func ParsePOIDs(s string) []string {
	out := []string{}
	if strings.TrimSpace(s) == "" {
		return out
	}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-ttpg-key")
		if apiKey != "" {
			_, u, err := ValidateAPIKey(apiKey)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "API key tidak valid"})
				return
			}
			c.Set("user_id", u.ID)
			c.Set("role", u.Role)
			c.Set("po_ids", ParsePOIDs(u.POIDs))
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Token tidak ditemukan"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Format token tidak valid"})
			return
		}

		claims, err := ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Token tidak valid atau sudah kadaluarsa"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("po_ids", claims.POIDs)
		c.Next()
	}
}
