package interfaces

import (
	"context"
	"ftrack/models"
	"ftrack/utils"
)

// Service interfaces that the websocket hub needs
type AuthService interface {
	ValidateToken(token string) (*utils.Claims, error)
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
}

type UserService interface {
	UpdateOnlineStatus(ctx context.Context, userID string, isOnline bool) error
	GetUser(ctx context.Context, userID string) (*models.User, error)
}

type CircleService interface {
	GetUserCircles(ctx context.Context, userID string) ([]models.Circle, error)
	GetCircleMembers(ctx context.Context, circleID string) ([]models.User, error)
}

type LocationService interface {
	UpdateLocation(ctx context.Context, userID string, location models.Location) (*models.Location, error)
	GetUserLocation(ctx context.Context, userID string) (*models.Location, error)
}

type MessageService interface {
	SendMessage(ctx context.Context, userID string, req models.SendMessageRequest) (*models.Message, error)
	GetCircleMessages(ctx context.Context, circleID string, limit int) ([]models.Message, error)
}

type EmergencyService interface {
	CreateEmergency(ctx context.Context, userID string, req models.CreateEmergencyRequest) (*models.Emergency, error)
	ResolveEmergency(ctx context.Context, emergencyID string) error
}

// WebSocket broadcaster interface for services to use
type WebSocketBroadcaster interface {
	BroadcastLocationUpdate(userID string, circleIDs []string, location models.Location)
	BroadcastPlaceEvent(userID string, circleIDs []string, placeEvent models.WSPlaceEvent)
	BroadcastEmergencyAlert(circleIDs []string, alert models.WSEmergencyAlert)
	SendNotificationToUser(userID string, notification interface{})
	BroadcastMessage(roomID string, message models.WSMessage)
	GetConnectedUsers() []string
	IsUserOnline(userID string) bool
}
type SMSService interface {
	SendSMS(ctx context.Context, phone, message string) error
	// Add other SMS methods you need
}
type EmailService interface {
	SendEmail() error
	SendVerificationEmail(email, firstName, token string) error
	SendPasswordResetEmail(email, firstName, token string) error
	SendWelcomeEmail(email, firstName string) error
	Send2FADisabledEmail(email, firstName string) error
	SendPasswordChangedEmail(email, firstName string) error
}
