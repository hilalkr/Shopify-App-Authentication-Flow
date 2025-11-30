package httpapi

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok": true,
	})
}

var shopRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-]*\.myshopify\.com$`)

func Login(c *gin.Context) {
	shop := c.Query("shop")

	if shop == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing shop query parameter. Example: /login?shop=your-store.myshopify.com",
		})
		return
	}
	if !shopRe.MatchString(shop) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid shop domain. Must match *.myshopify.com",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "shop is valid",
		"shop":    shop,
	})
}
