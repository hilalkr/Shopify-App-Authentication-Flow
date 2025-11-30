package httpapi

import "github.com/gin-gonic/gin"

func NewRouter() *gin.Engine {
	r := gin.New()

	//Logger and Recovery global middlewares
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/health", Health)
	r.GET("/login", Login)

	return r
}
