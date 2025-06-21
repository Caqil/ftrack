package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// Seeder represents a database seeder
type Seeder struct {
	Name        string
	Description string
	Seed        func(*mongo.Database) error
}

// seeders contains all database seeders
var seeders = []Seeder{
	{
		Name:        "demo_users",
		Description: "Create demo users for development",
		Seed:        seedDemoUsers,
	},
	{
		Name:        "demo_circles",
		Description: "Create demo circles for development",
		Seed:        seedDemoCircles,
	},
	{
		Name:        "demo_places",
		Description: "Create demo places for development",
		Seed:        seedDemoPlaces,
	},
}

// RunSeeders executes all database seeders
func RunSeeders(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if seeders have already been run
	seedersCol := db.Collection("seeders")
	count, err := seedersCol.CountDocuments(ctx, bson.M{})
	if err == nil && count > 0 {
		logrus.Info("🌱 Seeders already run, skipping...")
		return nil
	}

	logrus.Info("🌱 Running database seeders...")

	for _, seeder := range seeders {
		logrus.Infof("🔄 Running seeder: %s", seeder.Name)

		if err := seeder.Seed(db); err != nil {
			logrus.Errorf("❌ Seeder %s failed: %v", seeder.Name, err)
			continue // Continue with other seeders
		}

		// Record successful seeder
		_, err := seedersCol.InsertOne(ctx, bson.M{
			"name":      seeder.Name,
			"createdAt": time.Now(),
		})
		if err != nil {
			logrus.Warnf("Failed to record seeder %s: %v", seeder.Name, err)
		}

		logrus.Infof("✅ Seeder %s completed", seeder.Name)
	}

	logrus.Info("🌱 All seeders completed")
	return nil
}

// seedDemoUsers creates demo users for development
func seedDemoUsers(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	usersCol := db.Collection("users")

	// Check if demo users already exist
	count, err := usersCol.CountDocuments(ctx, bson.M{"email": bson.M{"$regex": "@demo.com$"}})
	if err == nil && count > 0 {
		return nil // Demo users already exist
	}

	// Hash password for demo users
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	demoUsers := []interface{}{
		bson.M{
			"_id":        primitive.NewObjectID(),
			"firstName":  "John",
			"lastName":   "Doe",
			"email":      "john.doe@demo.com",
			"password":   string(hashedPassword),
			"isVerified": true,
			"isActive":   true,
			"locationSharing": bson.M{
				"enabled":    true,
				"precision":  "exact",
				"circles":    []string{},
				"exceptions": []string{},
			},
			"preferences": bson.M{
				"notifications": bson.M{
					"push":        true,
					"email":       true,
					"sms":         false,
					"emergencies": true,
					"places":      true,
					"circles":     true,
				},
				"privacy": bson.M{
					"shareLocation": true,
					"shareStatus":   true,
					"allowInvites":  true,
				},
				"theme":    "light",
				"language": "en",
			},
			"emergencyContact": bson.M{
				"name":         "Jane Doe",
				"phone":        "+1234567890",
				"relationship": "spouse",
			},
			"deviceInfo": bson.M{
				"platform": "iOS",
				"version":  "15.0",
				"model":    "iPhone 13",
			},
			"createdAt": time.Now(),
			"updatedAt": time.Now(),
		},
		bson.M{
			"_id":        primitive.NewObjectID(),
			"firstName":  "Jane",
			"lastName":   "Smith",
			"email":      "jane.smith@demo.com",
			"password":   string(hashedPassword),
			"isVerified": true,
			"isActive":   true,
			"locationSharing": bson.M{
				"enabled":    true,
				"precision":  "exact",
				"circles":    []string{},
				"exceptions": []string{},
			},
			"preferences": bson.M{
				"notifications": bson.M{
					"push":        true,
					"email":       true,
					"sms":         true,
					"emergencies": true,
					"places":      true,
					"circles":     true,
				},
				"privacy": bson.M{
					"shareLocation": true,
					"shareStatus":   true,
					"allowInvites":  true,
				},
				"theme":    "dark",
				"language": "en",
			},
			"emergencyContact": bson.M{
				"name":         "John Smith",
				"phone":        "+1234567891",
				"relationship": "spouse",
			},
			"deviceInfo": bson.M{
				"platform": "Android",
				"version":  "12.0",
				"model":    "Samsung Galaxy S21",
			},
			"createdAt": time.Now(),
			"updatedAt": time.Now(),
		},
		bson.M{
			"_id":        primitive.NewObjectID(),
			"firstName":  "Mike",
			"lastName":   "Johnson",
			"email":      "mike.johnson@demo.com",
			"password":   string(hashedPassword),
			"isVerified": true,
			"isActive":   true,
			"locationSharing": bson.M{
				"enabled":    true,
				"precision":  "approximate",
				"circles":    []string{},
				"exceptions": []string{},
			},
			"preferences": bson.M{
				"notifications": bson.M{
					"push":        true,
					"email":       false,
					"sms":         false,
					"emergencies": true,
					"places":      false,
					"circles":     true,
				},
				"privacy": bson.M{
					"shareLocation": true,
					"shareStatus":   false,
					"allowInvites":  true,
				},
				"theme":    "light",
				"language": "en",
			},
			"emergencyContact": bson.M{
				"name":         "Sarah Johnson",
				"phone":        "+1234567892",
				"relationship": "mother",
			},
			"deviceInfo": bson.M{
				"platform": "iOS",
				"version":  "16.0",
				"model":    "iPhone 14 Pro",
			},
			"createdAt": time.Now(),
			"updatedAt": time.Now(),
		},
	}

	_, err = usersCol.InsertMany(ctx, demoUsers)
	return err
}

// seedDemoCircles creates demo circles for development
func seedDemoCircles(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	circlesCol := db.Collection("circles")
	usersCol := db.Collection("users")

	// Check if demo circles already exist
	count, err := circlesCol.CountDocuments(ctx, bson.M{"name": bson.M{"$regex": "Demo"}})
	if err == nil && count > 0 {
		return nil // Demo circles already exist
	}

	// Get demo users
	cursor, err := usersCol.Find(ctx, bson.M{"email": bson.M{"$regex": "@demo.com$"}})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var users []bson.M
	if err := cursor.All(ctx, &users); err != nil {
		return err
	}

	if len(users) < 2 {
		return fmt.Errorf("need at least 2 demo users to create circles")
	}

	// Create demo family circle
	familyCircleID := primitive.NewObjectID()
	demoCircles := []interface{}{
		bson.M{
			"_id":        familyCircleID,
			"name":       "Demo Family",
			"inviteCode": generateInviteCode(),
			"createdBy":  users[0]["_id"],
			"isActive":   true,
			"members": []bson.M{
				{
					"userId":      users[0]["_id"],
					"role":        "admin",
					"status":      "active",
					"joinedAt":    time.Now(),
					"permissions": getDefaultAdminPermissions(),
				},
				{
					"userId":      users[1]["_id"],
					"role":        "member",
					"status":      "active",
					"joinedAt":    time.Now().Add(-24 * time.Hour),
					"permissions": getDefaultMemberPermissions(),
				},
			},
			"settings": bson.M{
				"autoAcceptInvites":  false,
				"requireApproval":    true,
				"maxMembers":         20,
				"locationSharing":    true,
				"drivingReports":     true,
				"emergencyAlerts":    true,
				"autoCheckIn":        true,
				"placeNotifications": true,
			},
			"stats": bson.M{
				"totalMembers":  2,
				"activeMembers": 2,
				"totalMessages": 0,
				"totalPlaces":   0,
				"lastActivity":  time.Now(),
			},
			"createdAt": time.Now().Add(-48 * time.Hour),
			"updatedAt": time.Now(),
		},
	}

	// Add third user to a separate circle if available
	if len(users) >= 3 {
		friendsCircleID := primitive.NewObjectID()
		demoCircles = append(demoCircles, bson.M{
			"_id":        friendsCircleID,
			"name":       "Demo Friends",
			"inviteCode": generateInviteCode(),
			"createdBy":  users[2]["_id"],
			"isActive":   true,
			"members": []bson.M{
				{
					"userId":      users[2]["_id"],
					"role":        "admin",
					"status":      "active",
					"joinedAt":    time.Now(),
					"permissions": getDefaultAdminPermissions(),
				},
				{
					"userId":      users[1]["_id"],
					"role":        "member",
					"status":      "active",
					"joinedAt":    time.Now().Add(-12 * time.Hour),
					"permissions": getDefaultMemberPermissions(),
				},
			},
			"settings": bson.M{
				"autoAcceptInvites":  true,
				"requireApproval":    false,
				"maxMembers":         15,
				"locationSharing":    true,
				"drivingReports":     false,
				"emergencyAlerts":    true,
				"autoCheckIn":        false,
				"placeNotifications": false,
			},
			"stats": bson.M{
				"totalMembers":  2,
				"activeMembers": 2,
				"totalMessages": 0,
				"totalPlaces":   0,
				"lastActivity":  time.Now(),
			},
			"createdAt": time.Now().Add(-24 * time.Hour),
			"updatedAt": time.Now(),
		})
	}

	_, err = circlesCol.InsertMany(ctx, demoCircles)
	return err
}

// seedDemoPlaces creates demo places for development
func seedDemoPlaces(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	placesCol := db.Collection("places")
	circlesCol := db.Collection("circles")

	// Check if demo places already exist
	count, err := placesCol.CountDocuments(ctx, bson.M{"name": bson.M{"$regex": "Demo"}})
	if err == nil && count > 0 {
		return nil // Demo places already exist
	}

	// Get demo circles
	cursor, err := circlesCol.Find(ctx, bson.M{"name": bson.M{"$regex": "Demo"}})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var circles []bson.M
	if err := cursor.All(ctx, &circles); err != nil {
		return err
	}

	if len(circles) == 0 {
		return fmt.Errorf("no demo circles found to create places")
	}

	demoPlaces := []interface{}{
		bson.M{
			"_id":      primitive.NewObjectID(),
			"circleId": circles[0]["_id"],
			"name":     "Demo Home",
			"address":  "123 Main Street, Demo City, DC 12345",
			"location": bson.M{
				"type":        "Point",
				"coordinates": []float64{-122.4194, 37.7749}, // San Francisco coordinates
			},
			"radius":    100.0,
			"type":      "home",
			"isActive":  true,
			"createdBy": circles[0]["createdBy"],
			"settings": bson.M{
				"notifyOnEntry": true,
				"notifyOnExit":  true,
				"quietHours": bson.M{
					"enabled": true,
					"start":   "22:00",
					"end":     "07:00",
				},
			},
			"createdAt": time.Now().Add(-24 * time.Hour),
			"updatedAt": time.Now(),
		},
		bson.M{
			"_id":      primitive.NewObjectID(),
			"circleId": circles[0]["_id"],
			"name":     "Demo Office",
			"address":  "456 Business Ave, Demo City, DC 12346",
			"location": bson.M{
				"type":        "Point",
				"coordinates": []float64{-122.4094, 37.7849},
			},
			"radius":    150.0,
			"type":      "work",
			"isActive":  true,
			"createdBy": circles[0]["createdBy"],
			"settings": bson.M{
				"notifyOnEntry": false,
				"notifyOnExit":  true,
				"quietHours": bson.M{
					"enabled": false,
				},
			},
			"createdAt": time.Now().Add(-18 * time.Hour),
			"updatedAt": time.Now(),
		},
	}

	// Add school place if we have multiple circles
	if len(circles) > 1 {
		demoPlaces = append(demoPlaces, bson.M{
			"_id":      primitive.NewObjectID(),
			"circleId": circles[0]["_id"],
			"name":     "Demo School",
			"address":  "789 Education Blvd, Demo City, DC 12347",
			"location": bson.M{
				"type":        "Point",
				"coordinates": []float64{-122.3994, 37.7949},
			},
			"radius":    200.0,
			"type":      "school",
			"isActive":  true,
			"createdBy": circles[0]["createdBy"],
			"settings": bson.M{
				"notifyOnEntry": true,
				"notifyOnExit":  true,
				"quietHours": bson.M{
					"enabled": true,
					"start":   "08:00",
					"end":     "15:30",
				},
			},
			"createdAt": time.Now().Add(-12 * time.Hour),
			"updatedAt": time.Now(),
		})
	}

	_, err = placesCol.InsertMany(ctx, demoPlaces)
	return err
}

// Helper functions

func generateInviteCode() string {
	return uuid.New().String()[:8] // First 8 characters of UUID
}

func getDefaultAdminPermissions() bson.M {
	return bson.M{
		"canViewLocations": true,
		"canManageMembers": true,
		"canEditSettings":  true,
		"canSendMessages":  true,
		"canManagePlaces":  true,
		"canReceiveAlerts": true,
		"canSendEmergency": true,
	}
}

func getDefaultMemberPermissions() bson.M {
	return bson.M{
		"canViewLocations": true,
		"canManageMembers": false,
		"canEditSettings":  false,
		"canSendMessages":  true,
		"canManagePlaces":  false,
		"canReceiveAlerts": true,
		"canSendEmergency": true,
	}
}
