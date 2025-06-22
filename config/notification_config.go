// config/notification_config.go
package config

import (
	"context"
	"ftrack/repositories"
	"ftrack/services"
	"ftrack/websocket"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/api/option"
)

// NotificationConfig holds configuration for notification services
type NotificationConfig struct {
	// Email configuration
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string

	// SMS configuration (Twilio)
	TwilioAccountSID  string
	TwilioAuthToken   string
	TwilioPhoneNumber string

	// Firebase configuration
	FirebaseCredentialsPath string
	FirebaseProjectID       string
}

// LoadNotificationConfig loads notification configuration from environment
func LoadNotificationConfig() *NotificationConfig {
	return &NotificationConfig{
		// Email settings
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     getEnvOrDefault("SMTP_PORT", "587"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		FromEmail:    os.Getenv("FROM_EMAIL"),
		FromName:     getEnvOrDefault("FROM_NAME", "Family Tracker"),

		// SMS settings
		TwilioAccountSID:  os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:   os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioPhoneNumber: os.Getenv("TWILIO_PHONE_NUMBER"),

		// Firebase settings
		FirebaseCredentialsPath: os.Getenv("FIREBASE_CREDENTIALS_PATH"),
		FirebaseProjectID:       os.Getenv("FIREBASE_PROJECT_ID"),
	}
}

// InitializeNotificationServices initializes all notification-related services
func InitializeNotificationServices(
	db *mongo.Database,
	redis *redis.Client,
	hub *websocket.Hub,
) (*services.NotificationService, error) {

	config := LoadNotificationConfig()

	// Initialize repositories
	notificationRepo := repositories.NewNotificationRepository(db)
	userRepo := repositories.NewUserRepository(db)
	circleRepo := repositories.NewCircleRepository(db)

	// Create database indexes
	if err := notificationRepo.CreateIndexes(context.Background()); err != nil {
		logrus.Errorf("Failed to create notification indexes: %v", err)
	}

	// Initialize Firebase/FCM client
	var fcmClient *messaging.Client
	if config.FirebaseCredentialsPath != "" {
		app, err := initializeFirebase(config)
		if err != nil {
			logrus.Errorf("Failed to initialize Firebase: %v", err)
		} else {
			fcmClient, err = app.Messaging(context.Background())
			if err != nil {
				logrus.Errorf("Failed to get FCM client: %v", err)
			}
		}
	}

	// Initialize external services
	pushService := services.NewPushService(fcmClient, notificationRepo)
	emailService := services.NewEmailService(
		config.SMTPHost,
		config.SMTPPort,
		config.SMTPUsername,
		config.SMTPPassword,
		config.FromEmail,
		config.FromName,
		notificationRepo,
	)
	smsService := services.NewSMSService(
		config.TwilioAccountSID,
		config.TwilioAuthToken,
		config.TwilioPhoneNumber,
		notificationRepo,
	)

	// Initialize main notification service
	notificationService := services.NewNotificationService(
		notificationRepo,
		userRepo,
		circleRepo,
		redis,
		hub,
		emailService,
		smsService,
		pushService,
	)

	return notificationService, nil
}

// initializeFirebase initializes Firebase app
func initializeFirebase(config *NotificationConfig) (*firebase.App, error) {
	ctx := context.Background()

	var app *firebase.App
	var err error

	if config.FirebaseCredentialsPath != "" {
		// Initialize with service account credentials
		opt := option.WithCredentialsFile(config.FirebaseCredentialsPath)
		app, err = firebase.NewApp(ctx, nil, opt)
	} else {
		// Initialize with default credentials (for cloud environments)
		app, err = firebase.NewApp(ctx, nil)
	}

	if err != nil {
		return nil, err
	}

	return app, nil
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetupNotificationWorkers sets up background workers for notifications
func SetupNotificationWorkers(notificationService *services.NotificationService) {
	// You can add background workers here for:
	// - Processing scheduled notifications
	// - Sending digest emails
	// - Cleaning up old notifications
	// - Processing notification rules

	logrus.Info("Notification workers initialized")
}

// Example environment configuration file (.env)
/*
# Email Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
FROM_EMAIL=noreply@yourapp.com
FROM_NAME=Family Tracker

# SMS Configuration (Twilio)
TWILIO_ACCOUNT_SID=your-twilio-account-sid
TWILIO_AUTH_TOKEN=your-twilio-auth-token
TWILIO_PHONE_NUMBER=+1234567890

# Firebase Configuration
FIREBASE_CREDENTIALS_PATH=path/to/firebase-service-account.json
FIREBASE_PROJECT_ID=your-firebase-project-id
*/
