package main

import (
	"context"
	"ftrack/config"
	"ftrack/database"
	"ftrack/routes"
	"ftrack/websocket"
	"ftrack/workers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize configuration
	cfg := config.Load()

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize logger
	setupLogger(cfg)

	// Initialize database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logrus.Fatal("Failed to connect to database: ", err)
	}
	defer database.Disconnect()

	// Initialize Redis
	redis := config.InitRedis(cfg)
	defer redis.Close()

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize workers
	workers.StartLocationWorker(db, redis, hub)
	workers.StartNotificationWorker(db, redis)
	workers.StartGeofenceWorker(db, redis, hub)

	// Setup routes
	router := routes.SetupRoutes(db, redis, hub)

	// Create HTTP server
	server := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in goroutine
	go func() {
		logrus.Info("🚀 Life360 Backend Server starting on port ", cfg.Port)
		logrus.Info("📱 WebSocket endpoint: /ws")
		logrus.Info("📚 API Documentation: /docs")
		logrus.Info("💖 Health Check: /health")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatal("Failed to start server: ", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatal("Server forced to shutdown: ", err)
	}

	logrus.Info("✅ Server shutdown complete")
}

func setupLogger(cfg *config.Config) {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	if cfg.Environment == "development" {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}
