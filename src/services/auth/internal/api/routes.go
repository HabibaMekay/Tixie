package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.POST("/login", Login)
	r.GET("/oauth2-login", OAuth2Login)
	r.GET("/callback", OAuth2Callback)
	r.GET("/protected", Protected)
}
