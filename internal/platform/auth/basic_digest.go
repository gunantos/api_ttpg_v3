package auth

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/andtho89/ttpg/internal/models"
	"github.com/gin-gonic/gin"
)

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func BasicLogin(header string) (*models.User, error) {
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

func VerifyDigest(header, method, uri string) (*models.User, error) {
	if !strings.HasPrefix(header, "Digest ") {
		return nil, fmt.Errorf("bukan digest auth")
	}
	params := map[string]string{}
	for _, part := range strings.Split(header[len("Digest "):], ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			params[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
		}
	}
	username := params["username"]
	realm := params["realm"]
	nonce := params["nonce"]
	qop := params["qop"]
	nc := params["nc"]
	cnonce := params["cnonce"]
	response := params["response"]
	if username == "" || nonce == "" || uri == "" || response == "" {
		return nil, fmt.Errorf("digest auth tidak lengkap")
	}
	var u models.User
	if err := AuthDB.Where("username = ? AND active = true", username).First(&u).Error; err != nil {
		return nil, fmt.Errorf("user tidak ditemukan")
	}
	if u.DigestHA1 == "" {
		return nil, fmt.Errorf("digest auth belum di-set untuk user ini")
	}
	h2 := md5Hex(method + ":" + uri)
	var expected string
	if qop != "" {
		expected = md5Hex(u.DigestHA1 + ":" + nonce + ":" + nc + ":" + cnonce + ":" + qop + ":" + h2)
	} else {
		expected = md5Hex(u.DigestHA1 + ":" + nonce + ":" + h2)
	}
	if subtle.ConstantTimeCompare([]byte(expected), []byte(response)) != 1 {
		return nil, fmt.Errorf("digest auth gagal")
	}
	_ = realm
	return &u, nil
}

func KeyHandler(c *gin.Context) {
	header := c.GetHeader("Authorization")
	var u *models.User
	var err error
	switch {
	case strings.HasPrefix(header, "Basic "):
		u, err = BasicLogin(header)
	case strings.HasPrefix(header, "Digest "):
		u, err = VerifyDigest(header, c.Request.Method, c.Request.RequestURI)
	default:
		err = fmt.Errorf("gunakan Basic atau Digest auth")
	}
	if err != nil || u == nil {
		c.Header("WWW-Authenticate", `Basic realm="ttpg", Digest realm="ttpg", qop="auth"`)
		c.JSON(http.StatusUnauthorized, gin.H{"status": false, "data": nil, "message": err.Error()})
		return
	}
	var poID interface{} = nil
	if poIDs := ParsePOIDs(u.POIDs); len(poIDs) > 0 {
		poID = poIDs[0]
	}
	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data": gin.H{
			"username":        u.Username,
			"po_id":           poID,
			"production_key":  u.ProductionKey,
			"development_key": u.DevelopmentKey,
		},
	})
}
