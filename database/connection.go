package database

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client   *mongo.Client
	database *mongo.Database
)

// Connect establishes connection to MongoDB
func Connect(databaseURL string) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set client options
	clientOptions := options.Client().ApplyURI(databaseURL)

	// Configure connection pool
	clientOptions.SetMaxPoolSize(100)
	clientOptions.SetMinPoolSize(5)
	clientOptions.SetMaxConnIdleTime(30 * time.Second)
	clientOptions.SetRetryWrites(true)
	clientOptions.SetRetryReads(true)

	// Set read preference to primary preferred for better consistency
	clientOptions.SetReadPreference(readpref.PrimaryPreferred())

	// Create a new client and connect to the server
	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the primary to verify connection
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Extract database name from URL or use default
	dbName := extractDatabaseName(databaseURL)
	database = client.Database(dbName)

	logrus.Info("✅ Connected to MongoDB successfully")
	logrus.Infof("📊 Database: %s", dbName)

	// Run migrations
	if err := RunMigrations(database); err != nil {
		logrus.Warnf("Migration warning: %v", err)
	}

	// Run seeders if in development
	if shouldRunSeeders() {
		if err := RunSeeders(database); err != nil {
			logrus.Warnf("Seeder warning: %v", err)
		}
	}

	return database, nil
}

// Disconnect closes the MongoDB connection
func Disconnect() error {
	if client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.Disconnect(ctx)
	if err != nil {
		logrus.Errorf("Error disconnecting from MongoDB: %v", err)
		return err
	}

	logrus.Info("🔌 Disconnected from MongoDB")
	return nil
}

// GetDatabase returns the database instance
func GetDatabase() *mongo.Database {
	return database
}

// GetClient returns the MongoDB client
func GetClient() *mongo.Client {
	return client
}

// GetCollection returns a collection from the database
func GetCollection(name string) *mongo.Collection {
	if database == nil {
		logrus.Fatal("Database not initialized")
	}
	return database.Collection(name)
}

// IsConnected checks if the database connection is alive
func IsConnected() bool {
	if client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Ping(ctx, readpref.Primary())
	return err == nil
}

// extractDatabaseName extracts database name from MongoDB URI
func extractDatabaseName(uri string) string {
	// Default database name
	defaultDB := "ftrack"

	// Parse URI to extract database name
	clientOptions := options.Client().ApplyURI(uri)
	if clientOptions.Auth != nil && clientOptions.Auth.AuthSource != "" {
		return clientOptions.Auth.AuthSource
	}

	// Try to extract from URI path
	if len(uri) > 0 {
		// Look for database name after last slash
		for i := len(uri) - 1; i >= 0; i-- {
			if uri[i] == '/' {
				if i < len(uri)-1 {
					dbName := uri[i+1:]
					// Remove query parameters if any
					for j, char := range dbName {
						if char == '?' || char == '&' {
							dbName = dbName[:j]
							break
						}
					}
					if dbName != "" && dbName != "admin" {
						return dbName
					}
				}
				break
			}
		}
	}

	return defaultDB
}

// shouldRunSeeders determines if seeders should run
func shouldRunSeeders() bool {
	// Check environment or use development as default
	// This should match your config setup
	return true // Modify based on your config.Environment logic
}

// Health check for the database
func HealthCheck() map[string]interface{} {
	result := map[string]interface{}{
		"status": "unhealthy",
		"error":  nil,
	}

	if !IsConnected() {
		result["error"] = "database connection lost"
		return result
	}

	// Check database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get server status
	var serverStatus map[string]interface{}
	err := database.RunCommand(ctx, map[string]interface{}{"serverStatus": 1}).Decode(&serverStatus)
	if err != nil {
		result["error"] = fmt.Sprintf("server status check failed: %v", err)
		return result
	}

	result["status"] = "healthy"
	result["server_status"] = map[string]interface{}{
		"uptime":      serverStatus["uptime"],
		"version":     serverStatus["version"],
		"connections": serverStatus["connections"],
	}

	return result
}
