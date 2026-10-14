package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/andtho89/ttpg/internal/models"
	"github.com/gin-gonic/gin"
)

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

func KeyHandler(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Basic ") {
		c.Header("WWW-Authenticate", `Basic realm="ttpg"`)
		c.JSON(http.StatusUnauthorized, gin.H{"status": false, "data": nil, "message": "gunakan Basic auth"})
		return
	}
	u, err := BasicLogin(header)
	if err != nil || u == nil {
		c.Header("WWW-Authenticate", `Basic realm="ttpg"`)
		msg := "Basic auth tidak valid"
		if err != nil {
			msg = err.Error()
		}
		c.JSON(http.StatusUnauthorized, gin.H{"status": false, "data": nil, "message": msg})
		return
	}
	var poID interface{} = nil
	if poIDs := ParsePOIDs(u.PoIDOrEmpty()); len(poIDs) > 0 {
		poID = poIDs[0]
	}
	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data": gin.H{
			"username":        u.Username,
			"po_id":           poID,
			"public_key":      u.PublicKey,
			"development_key": u.DevelopmentKey,
		},
	})
}
