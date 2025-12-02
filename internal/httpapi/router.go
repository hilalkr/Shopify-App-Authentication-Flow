package httpapi

import "github.com/gin-gonic/gin"

func NewRouter(h *Handlers) *gin.Engine {
	r := gin.New()

	//Logger and Recovery global middlewares
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	r.GET("/health", h.Health)
	r.GET("/login", h.Login)
	r.GET("/auth/callback", h.OAuthCallback)
	r.GET("/dashboard", h.Dashboard)

	return r
}
