package main

import (
	"log"
	"net/http"

	"bugyou-backend/internal/config"
	"bugyou-backend/internal/routes"

	"github.com/gin-gonic/gin"
)

func corsMiddleware() gin.HandlerFunc {
	allowedOrigins := map[string]bool{
		"https://bugs.anandkv.in": true,
		"http://localhost:5173":   true,
		"http://localhost:3000":   true,
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length,Content-Type")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Vary", "Origin")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func main() {
	cfg := config.Load()

	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20

	// CORS must be before routes
	router.Use(corsMiddleware())

	router.Static("/uploads", cfg.UploadDir)

	if err := config.ConnectMongo(cfg.MongoURI, cfg.DatabaseName); err != nil {
		log.Fatalf("mongo connection failed: %v", err)
	}

	router.MaxMultipartMemory = 8 << 20

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "BugYou backend running",
		})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Your app routes
	routes.Register(router)

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
