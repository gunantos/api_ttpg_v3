package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/config"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const TokenExpiry = 7 * 24 * time.Hour // 1 minggu

var jwtSecret = []byte("ttpg-secret-key-change-in-production")
var AuthDB *gorm.DB
var UsersDB *gorm.DB

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

func SetUsersDB(db *gorm.DB) {
	UsersDB = db
	logCh = make(chan models.TtpgLog, 2048)
	go logWorker()
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

func RoleFromUser(u *models.TtpgUser) string {
	if u.Level == nil {
		return "admin"
	}
	return "user"
}

func userActive(u *models.TtpgUser) bool {
	if u.IsBlock != 0 {
		return false
	}
	if u.IsApproved != nil && *u.IsApproved == 0 {
		return false
	}
	return true
}

func BasicLogin(header string) (*models.TtpgUser, error) {
	if !strings.HasPrefix(header, "Basic ") {
		return nil, fmt.Errorf("bukan basic auth")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(header, "Basic ")))
	if err != nil {
		return nil, fmt.Errorf("basic auth tidak valid")
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("basic auth tidak valid")
	}
	return Login(parts[0], parts[1])
}

func Login(username, password string) (*models.TtpgUser, error) {
	var u models.TtpgUser
	if err := UsersDB.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, fmt.Errorf("user tidak ditemukan")
	}
	if !userActive(&u) {
		return nil, fmt.Errorf("user tidak aktif")
	}
	if !CheckPassword(u.Password, password) {
		return nil, fmt.Errorf("password salah")
	}
	return &u, nil
}

// ValidateAPIKey mencocokkan key dengan ttpg_users.user.secret_key (aktif).
func ValidateAPIKey(key string) (*models.TtpgUser, error) {
	var u models.TtpgUser
	if err := UsersDB.Where("secret_key = ? AND is_block = 0 AND (is_approved IS NULL OR is_approved = 1)", key).First(&u).Error; err != nil {
		return nil, fmt.Errorf("api key tidak valid")
	}
	return &u, nil
}

func ParsePOIDs(s string) []string {
	out := []string{}
	if strings.TrimSpace(s) == "" {
		return out
	}
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == ';' || r == ',' }) {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		var (
			user       *models.TtpgUser
			scheme     string
			keyUsed    string
			identified bool
		)
		defer func() {
			enqueueLog(c, start, scheme, keyUsed, identified)
		}()

		key := strings.TrimSpace(c.GetHeader("X-ttpg-key"))
		if key == "" {
			key = strings.TrimSpace(c.GetHeader("X-API-Key"))
		}
		if key != "" {
			keyUsed = key
			scheme = "api_key"
			if config.C.APIKey != "" && subtle.ConstantTimeCompare([]byte(key), []byte(config.C.APIKey)) == 1 {
				identified = true
			} else if u, err := ValidateAPIKey(key); err == nil {
				user = u
				identified = true
			} else {
				abortUnauthorized(c, "API key tidak valid")
				return
			}
		} else {
			authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
			if authHeader == "" {
				c.Header("WWW-Authenticate", `Basic realm="ttpg"`)
				abortUnauthorized(c, "Token tidak ditemukan")
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			switch parts[0] {
			case "Bearer":
				if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
					abortUnauthorized(c, "Format token tidak valid")
					return
				}
				cl, err := ValidateToken(parts[1])
				if err != nil {
					abortUnauthorized(c, "Token tidak valid atau sudah kadaluarsa")
					return
				}
				scheme = "bearer"
				var u models.TtpgUser
				if err := UsersDB.First(&u, cl.UserID).Error; err != nil {
					abortUnauthorized(c, "User tidak ditemukan di ttpg_users")
					return
				}
				if !userActive(&u) {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "User diblokir atau belum disetujui"})
					return
				}
				user = &u
				identified = true
			case "Basic":
				u, err := BasicLogin(authHeader)
				if err != nil || u == nil {
					abortUnauthorized(c, errMsg(err, "Basic auth tidak valid"))
					return
				}
				scheme = "basic"
				user = u
				identified = true
			default:
				abortUnauthorized(c, "Skema auth tidak dikenal (dukung: Bearer, Basic, X-ttpg-key)")
				return
			}
		}

		if user != nil {
			if !userActive(user) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "User diblokir atau belum disetujui"})
				return
			}
		}

		if rule, err := findApiURL(c.Request.Method, c.Request.URL.Path, c.FullPath()); err == nil && rule != nil {
			if user == nil {
				// superkey env API_KEY: lewati aturan skema & izin
			} else {
				if msg, ok := schemeMismatch(rule.Authorization, scheme); !ok {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": msg})
					return
				}
				if msg, ok := accessAllowed(user, rule.ID); !ok {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": msg})
					return
				}
			}
		}

		if user != nil {
			poStr := ""
			if user.PoID != nil {
				poStr = *user.PoID
			}
			c.Set("user_id", user.ID)
			c.Set("role", RoleFromUser(user))
			c.Set("po_ids", ParsePOIDs(poStr))
		} else {
			c.Set("user_id", uint(0))
			c.Set("role", "api_key")
			c.Set("po_ids", []string{})
		}
		c.Next()
	}
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.Header("WWW-Authenticate", `Basic realm="ttpg"`)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": msg})
}

func errMsg(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}
