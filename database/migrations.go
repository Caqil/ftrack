package database

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	Up          func(*mongo.Database) error
	Down        func(*mongo.Database) error
}

// migrationRecord tracks applied migrations
type migrationRecord struct {
	Version   int       `bson:"version"`
	AppliedAt time.Time `bson:"appliedAt"`
}

// migrations contains all database migrations
var migrations = []Migration{
	{
		Version:     1,
		Description: "Create users collection with indexes",
		Up:          createUsersCollection,
	},
	{
		Version:     2,
		Description: "Create circles collection with indexes",
		Up:          createCirclesCollection,
	},
	{
		Version:     3,
		Description: "Create locations collection with indexes",
		Up:          createLocationsCollection,
	},
	{
		Version:     4,
		Description: "Create messages collection with indexes",
		Up:          createMessagesCollection,
	},
	{
		Version:     5,
		Description: "Create notifications collection with indexes",
		Up:          createNotificationsCollection,
	},
	{
		Version:     6,
		Description: "Create emergencies collection with indexes",
		Up:          createEmergenciesCollection,
	},
	{
		Version:     7,
		Description: "Create places collection with indexes",
		Up:          createPlacesCollection,
	},
	{
		Version:     8,
		Description: "Create invitations collection with indexes",
		Up:          createInvitationsCollection,
	},
	{
		Version:     9,
		Description: "Create emergency settings collection with indexes",
		Up:          createEmergencySettingsCollection,
	},
	{
		Version:     10,
		Description: "Create sessions collection with indexes",
		Up:          createSessionsCollection,
	},
}

// RunMigrations executes all pending migrations
func RunMigrations(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ensure migrations collection exists
	migrationsCol := db.Collection("migrations")

	// Get current migration version
	currentVersion := getCurrentMigrationVersion(ctx, migrationsCol)
	logrus.Infof("📋 Current migration version: %d", currentVersion)

	// Run pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		logrus.Infof("🔄 Running migration %d: %s", migration.Version, migration.Description)

		if err := migration.Up(db); err != nil {
			return fmt.Errorf("migration %d failed: %w", migration.Version, err)
		}

		// Record successful migration
		_, err := migrationsCol.InsertOne(ctx, migrationRecord{
			Version:   migration.Version,
			AppliedAt: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		logrus.Infof("✅ Migration %d completed", migration.Version)
	}

	return nil
}

// getCurrentMigrationVersion returns the current migration version
func getCurrentMigrationVersion(ctx context.Context, col *mongo.Collection) int {
	opts := options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}})
	var record migrationRecord
	err := col.FindOne(ctx, bson.D{}, opts).Decode(&record)
	if err != nil {
		return 0 // No migrations applied yet
	}
	return record.Version
}

// Individual migration functions

func createUsersCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := db.Collection("users")

	// Create indexes
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "firstName", Value: "text"}, {Key: "lastName", Value: "text"}, {Key: "email", Value: "text"}},
			Options: options.Index().SetName("user_search"),
		},
		{
			Keys: bson.D{{Key: "isActive", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "deviceToken", Value: 1}},
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

func createCirclesCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := db.Collection("circles")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "inviteCode", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "members.userId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "createdBy", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "isActive", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "lastActivity", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "name", Value: "text"}},
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

func createLocationsCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := db.Collection("locations")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "timestamp", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "location", Value: "2dsphere"}},
		},
		{
			Keys:    bson.D{{Key: "timestamp", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(30 * 24 * 3600), // 30 days
		},
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "source", Value: 1}},
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

func createMessagesCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := db.Collection("messages")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "circleId", Value: 1}, {Key: "createdAt", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "senderId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "replyTo", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(90 * 24 * 3600), // 90 days
		},
		{
			Keys: bson.D{{Key: "content", Value: "text"}},
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

func createNotificationsCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := db.Collection("notifications")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "priority", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "scheduledFor", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(60 * 24 * 3600), // 60 days
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

func createEmergenciesCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := db.Collection("emergencies")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "location.coordinates", Value: "2dsphere"}},
		},
		{
			Keys: bson.D{{Key: "circleIds", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "priority", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "resolvedAt", Value: 1}},
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

func createPlacesCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := db.Collection("places")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "circleId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "createdBy", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "location.coordinates", Value: "2dsphere"}},
		},
		{
			Keys: bson.D{{Key: "isActive", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "name", Value: "text"}, {Key: "address", Value: "text"}},
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

func createInvitationsCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := db.Collection("invitations")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "circleId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "inviterId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "inviteeId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "email", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0), // Auto-delete expired invitations
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

func createEmergencySettingsCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := db.Collection("emergency_settings")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

func createSessionsCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col := db.Collection("sessions")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "sessionId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "userId", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0), // Auto-delete expired sessions
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: 1}},
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}
