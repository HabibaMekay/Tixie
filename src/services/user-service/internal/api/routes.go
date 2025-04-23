package api

import (
	"user-service/internal/db/repos"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, repo *repos.UserRepository) {
	handler := NewHandler(repo)

	users := r.Group("")
	{
		users.POST("", handler.CreateUser)
		users.GET("", handler.GetUsers)
		users.GET("/:id", handler.GetUserByID)
		users.PUT("/:id", handler.UpdateUser)
		users.DELETE("/:id", handler.DeleteUser)
		users.POST("/authenticate", handler.AuthenticateUser)
	}
}
