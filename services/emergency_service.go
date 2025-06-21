package services

import (
	"context"
	"errors"
	"fmt"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"life360-backend/websocket"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmergencyService struct {
	emergencyRepo   *repositories.EmergencyRepository
	circleRepo      *repositories.CircleRepository
	userRepo        *repositories.UserRepository
	notificationSvc *NotificationService
	websocketHub    *websocket.Hub
	validator       *utils.ValidationService
}

func NewEmergencyService(
	emergencyRepo *repositories.EmergencyRepository,
	circleRepo *repositories.CircleRepository,
	userRepo *repositories.UserRepository,
	notificationSvc *NotificationService,
	websocketHub *websocket.Hub,
) *EmergencyService {
	return &EmergencyService{
		emergencyRepo:   emergencyRepo,
		circleRepo:      circleRepo,
		userRepo:        userRepo,
		notificationSvc: notificationSvc,
		websocketHub:    websocketHub,
		validator:       utils.NewValidationService(),
	}
}

func (es *EmergencyService) CreateEmergency(ctx context.Context, userID string, req models.CreateEmergencyRequest) (*models.Emergency, error) {
	// Validate request
	if validationErrors := es.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Get user's circles
	circles, err := es.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Set priority based on emergency type
	priority := "high"
	if req.Type == models.EmergencyTypeSOS || req.Type == models.EmergencyTypeCrash {
		priority = "critical"
	}

	// Create emergency
	emergency := models.Emergency{
		UserID:      userObjectID,
		Type:        req.Type,
		Priority:    priority,
		Title:       es.getEmergencyTitle(req.Type),
		Description: req.Description,
		Location:    req.Location,
		Detection: models.EmergencyDetection{
			Method:     "manual",
			Confidence: 1.0,
		},
		Response: models.EmergencyResponse{
			AutoSent: false,
		},
	}

	// Set circle ID if user has circles
	if len(circles) > 0 {
		emergency.CircleID = circles[0].ID // Use first circle
	}

	err = es.emergencyRepo.Create(ctx, &emergency)
	if err != nil {
		return nil, err
	}

	// Get emergency contacts and notify
	go es.notifyEmergencyContacts(ctx, &emergency, circles)

	// Broadcast to circle members
	go es.broadcastEmergencyAlert(userID, &emergency, circles)

	return &emergency, nil
}

func (es *EmergencyService) GetEmergency(ctx context.Context, userID, emergencyID string) (*models.Emergency, error) {
	emergency, err := es.emergencyRepo.GetByID(ctx, emergencyID)
	if err != nil {
		return nil, err
	}

	// Check permission (user is owner or in same circle)
	hasPermission := emergency.UserID.Hex() == userID

	if !hasPermission && !emergency.CircleID.IsZero() {
		isMember, err := es.circleRepo.IsMember(ctx, emergency.CircleID.Hex(), userID)
		if err == nil && isMember {
			hasPermission = true
		}
	}

	if !hasPermission {
		return nil, errors.New("access denied")
	}

	return emergency, nil
}

func (es *EmergencyService) GetUserEmergencies(ctx context.Context, userID string) ([]models.Emergency, error) {
	return es.emergencyRepo.GetUserEmergencies(ctx, userID)
}

func (es *EmergencyService) GetCircleEmergencies(ctx context.Context, userID, circleID string) ([]models.Emergency, error) {
	// Check if user is a member of the circle
	isMember, err := es.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	return es.emergencyRepo.GetCircleEmergencies(ctx, circleID)
}

func (es *EmergencyService) UpdateEmergency(ctx context.Context, userID, emergencyID string, req models.UpdateEmergencyRequest) (*models.Emergency, error) {
	emergency, err := es.emergencyRepo.GetByID(ctx, emergencyID)
	if err != nil {
		return nil, err
	}

	// Check permission (user is owner or circle admin)
	hasPermission := emergency.UserID.Hex() == userID

	if !hasPermission && !emergency.CircleID.IsZero() {
		role, err := es.circleRepo.GetMemberRole(ctx, emergency.CircleID.Hex(), userID)
		if err == nil && role == "admin" {
			hasPermission = true
		}
	}

	if !hasPermission {
		return nil, errors.New("access denied")
	}

	// Build update document
	update := bson.M{}

	if req.Status != "" {
		update["status"] = req.Status
	}
	if req.Description != "" {
		update["description"] = req.Description
	}
	if req.Resolution != "" {
		update["resolution"] = req.Resolution
	}

	if len(update) == 0 {
		return nil, errors.New("no fields to update")
	}

	err = es.emergencyRepo.Update(ctx, emergencyID, update)
	if err != nil {
		return nil, err
	}

	// Add timeline event
	event := models.EmergencyEvent{
		Type:        "updated",
		Description: "Emergency updated",
		Actor:       primitive.ObjectID{},
	}
	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	event.Actor = userObjectID

	es.emergencyRepo.AddTimelineEvent(ctx, emergencyID, event)

	return es.emergencyRepo.GetByID(ctx, emergencyID)
}

func (es *EmergencyService) ResolveEmergency(ctx context.Context, userID, emergencyID, resolution string) error {
	emergency, err := es.emergencyRepo.GetByID(ctx, emergencyID)
	if err != nil {
		return err
	}

	// Check permission
	hasPermission := emergency.UserID.Hex() == userID

	if !hasPermission && !emergency.CircleID.IsZero() {
		role, err := es.circleRepo.GetMemberRole(ctx, emergency.CircleID.Hex(), userID)
		if err == nil && role == "admin" {
			hasPermission = true
		}
	}

	if !hasPermission {
		return errors.New("access denied")
	}

	return es.emergencyRepo.ResolveEmergency(ctx, emergencyID, userID, resolution)
}

func (es *EmergencyService) AddMedia(ctx context.Context, userID, emergencyID string, media models.EmergencyMedia) error {
	emergency, err := es.emergencyRepo.GetByID(ctx, emergencyID)
	if err != nil {
		return err
	}

	// Check permission (user is owner or in same circle)
	hasPermission := emergency.UserID.Hex() == userID

	if !hasPermission && !emergency.CircleID.IsZero() {
		isMember, err := es.circleRepo.IsMember(ctx, emergency.CircleID.Hex(), userID)
		if err == nil && isMember {
			hasPermission = true
		}
	}

	if !hasPermission {
		return errors.New("access denied")
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	media.UploadedBy = userObjectID

	return es.emergencyRepo.AddMedia(ctx, emergencyID, media)
}

func (es *EmergencyService) GetEmergencySettings(ctx context.Context, userID string) (*models.EmergencySettings, error) {
	return es.emergencyRepo.GetUserSettings(ctx, userID)
}

func (es *EmergencyService) UpdateEmergencySettings(ctx context.Context, userID string, settings *models.EmergencySettings) error {
	return es.emergencyRepo.UpdateUserSettings(ctx, userID, settings)
}

func (es *EmergencyService) GetEmergencyStats(ctx context.Context, userID string) (*models.EmergencyStats, error) {
	// Check if user is admin of any circle or system admin
	// For now, return basic stats
	startTime := time.Now().AddDate(0, -1, 0) // Last month
	endTime := time.Now()

	return es.emergencyRepo.GetEmergencyStats(ctx, startTime, endTime)
}

// Helper methods
func (es *EmergencyService) getEmergencyTitle(emergencyType string) string {
	switch emergencyType {
	case models.EmergencyTypeSOS:
		return "SOS Emergency Alert"
	case models.EmergencyTypeCrash:
		return "Crash Detected"
	case models.EmergencyTypeHelp:
		return "Help Needed"
	case models.EmergencyTypeMedical:
		return "Medical Emergency"
	case models.EmergencyTypeFire:
		return "Fire Emergency"
	case models.EmergencyTypePolice:
		return "Police Emergency"
	case models.EmergencyTypeRoadside:
		return "Roadside Assistance"
	default:
		return "Emergency Alert"
	}
}

func (es *EmergencyService) notifyEmergencyContacts(ctx context.Context, emergency *models.Emergency, circles []models.Circle) {
	// Get user details
	user, err := es.userRepo.GetByID(ctx, emergency.UserID.Hex())
	if err != nil {
		return
	}

	// Notify emergency contact
	if user.EmergencyContact.Phone != "" {
		sms := utils.SMSMessage{
			To: user.EmergencyContact.Phone,
			Message: fmt.Sprintf("EMERGENCY: %s %s needs help. Location: %.6f,%.6f",
				user.FirstName, user.LastName,
				emergency.Location.Latitude, emergency.Location.Longitude),
		}

		// Send SMS (would integrate with actual SMS service)
		logrus.Info("Sending emergency SMS to: ", user.EmergencyContact.Phone)
	}

	// Notify circle members
	for _, circle := range circles {
		if circle.Settings.EmergencyAlerts {
			var userIDs []string
			for _, member := range circle.Members {
				if member.Status == "active" && member.UserID != emergency.UserID {
					userIDs = append(userIDs, member.UserID.Hex())
				}
			}

			if len(userIDs) > 0 {
				// Send notifications to circle members
				notifReq := models.SendNotificationRequest{
					UserIDs:  userIDs,
					Type:     models.NotificationEmergencySOS,
					Title:    emergency.Title,
					Body:     fmt.Sprintf("%s %s needs help", user.FirstName, user.LastName),
					Priority: "urgent",
					Channels: models.NotificationChannels{
						Push:  true,
						SMS:   true,
						InApp: true,
					},
				}

				es.notificationSvc.SendNotification(ctx, notifReq)
			}
		}
	}
}

func (es *EmergencyService) broadcastEmergencyAlert(userID string, emergency *models.Emergency, circles []models.Circle) {
	alert := models.WSEmergencyAlert{
		UserID:    userID,
		Location:  emergency.Location,
		Type:      emergency.Type,
		Message:   emergency.Description,
		Timestamp: emergency.CreatedAt,
	}

	var circleIDs []string
	for _, circle := range circles {
		if circle.Settings.EmergencyAlerts {
			circleIDs = append(circleIDs, circle.ID.Hex())
		}
	}

	if len(circleIDs) > 0 {
		es.websocketHub.BroadcastEmergencyAlert(circleIDs, alert)
	}
}
