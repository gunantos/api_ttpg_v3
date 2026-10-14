package docs

import (
	_ "embed"

	"github.com/gin-gonic/gin"
)

//go:embed api.html
var page []byte

func Serve(c *gin.Context) {
	c.Header("Cache-Control", "no-store, max-age=0")
	c.Data(200, "text/html; charset=utf-8", page)
}
