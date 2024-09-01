package main

import (
	"log"

	"github.com/Vadim-Karpenko/golang-json-sync-service/handlers"
	"github.com/Vadim-Karpenko/golang-json-sync-service/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// Uncomment the line below to run the application in release mode
	// gin.SetMode(gin.ReleaseMode)
	// The following lines disable logging to stdout and stderr. In case of extremely high traffic, it can be useful to disable logging to reduce the load on the server.
	// gin.DefaultWriter = io.Discard
	// gin.DefaultErrorWriter = io.Discard

	// Initialize Redis
	rdb := utils.InitializeRedis()

	// Initialize Gin router
	router := gin.Default()

	// WebSocket route
	router.GET("/ws/:uuid", func(c *gin.Context) {
		handlers.HandleWebSocket(c, rdb)
	})

	// API routes
	router.POST("/upload", func(c *gin.Context) {
		handlers.UploadJSON(c, rdb)
	})
	router.GET("/json/:uuid", func(c *gin.Context) {
		handlers.GetJSON(c, rdb)
	})

	// Start the server
	log.Fatal(router.Run("localhost:8080"))
}
