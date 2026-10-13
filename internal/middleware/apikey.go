package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ApiKeyAuth(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiKey == "" {
			c.Next()
			return
		}
		key := c.GetHeader("X-ttpg-key")
		if key == "" {
			key = c.GetHeader("X-API-Key")
		}
		if key != apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": false, "message": "api key tidak valid"})
			return
		}
		c.Next()
	}
}
