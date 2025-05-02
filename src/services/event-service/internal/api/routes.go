package api

import (
    "event-service/internal/db/repos"
    "github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, repo *repos.EventRepository) {
    handler := NewEventHandler(repo)

 
    events := r.Group("/events")
    {
        events.GET("", handler.GetEvents)          
        events.POST("", handler.CreateEvent)       
    }
}
