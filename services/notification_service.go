package services

import (
	"context"
	"errors"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

type NotificationService struct {
	notificationRepo *repositories.NotificationRepository
	userRepo         *repositories.UserRepository
	pushService      *PushService
	validator        *utils.ValidationService
}

func NewNotificationService(
	notificationRepo *repositories.NotificationRepository,
	userRepo *repositories.UserRepository,
	pushService *PushService,
) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		pushService:      pushService,
		validator:        utils.NewValidationService(),
	}
}

func (ns *NotificationService) CreateNotification(ctx context.Context, notification *models.Notification) error {
	// Set default values
	if notification.Priority == "" {
		notification.Priority = "normal"
	}
	if notification.Category == "" {
		notification.Category = "system"
	}

	// Set expiration (default 30 days)
	if notification.ExpiresAt.IsZero() {
		notification.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	}

	return ns.notificationRepo.Create(ctx, notification)
}

func (ns *NotificationService) SendNotification(ctx context.Context, req models.SendNotificationRequest) error {
	// Validate request
	if validationErrors := ns.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return errors.New("validation failed")
	}

	// Get users
	users, err := ns.userRepo.GetUsersByIDs(ctx, req.UserIDs)
	if err != nil {
		return err
	}

	// Create notifications for each user
	for _, user := range users {
		// Check user preferences
		prefs, err := ns.notificationRepo.GetUserPreferences(ctx, user.ID.Hex())
		if err != nil {
			continue
		}

		if !prefs.GlobalEnabled {
			continue
		}

		// Create notification record
		notification := models.Notification{
			UserID:   user.ID,
			Type:     req.Type,
			Title:    req.Title,
			Body:     req.Body,
			Data:     req.Data,
			Priority: req.Priority,
			Channels: req.Channels,
		}

		if req.ScheduleFor != nil {
			notification.ScheduledFor = *req.ScheduleFor
		}

		err = ns.notificationRepo.Create(ctx, &notification)
		if err != nil {
			continue
		}

		// Send immediately if not scheduled
		if req.ScheduleFor == nil || req.ScheduleFor.Before(time.Now()) {
			go ns.processNotification(ctx, &notification, &user)
		}
	}

	return nil
}

func (ns *NotificationService) GetUserNotifications(ctx context.Context, userID string, page, pageSize int) ([]models.Notification, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return ns.notificationRepo.GetUserNotifications(ctx, userID, page, pageSize)
}

func (ns *NotificationService) MarkAsRead(ctx context.Context, userID, notificationID string) error {
	// Get notification to verify ownership
	notification, err := ns.notificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return err
	}

	if notification.UserID.Hex() != userID {
		return errors.New("permission denied")
	}

	return ns.notificationRepo.MarkAsRead(ctx, notificationID)
}

func (ns *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	return ns.notificationRepo.MarkAllAsRead(ctx, userID)
}

func (ns *NotificationService) DeleteNotification(ctx context.Context, userID, notificationID string) error {
	// Get notification to verify ownership
	notification, err := ns.notificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return err
	}

	if notification.UserID.Hex() != userID {
		return errors.New("permission denied")
	}

	return ns.notificationRepo.Delete(ctx, notificationID)
}

func (ns *NotificationService) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	return ns.notificationRepo.GetUnreadCount(ctx, userID)
}

func (ns *NotificationService) UpdateUserPreferences(ctx context.Context, userID string, req models.UpdateNotificationPrefsRequest) error {
	prefs := models.NotificationPreference{
		GlobalEnabled: req.GlobalEnabled,
		QuietHours:    *req.QuietHours,
		Categories:    req.Categories,
		CirclePrefs:   req.CirclePrefs,
	}

	return ns.notificationRepo.UpdateUserPreferences(ctx, userID, &prefs)
}

func (ns *NotificationService) GetUserPreferences(ctx context.Context, userID string) (*models.NotificationPreference, error) {
	return ns.notificationRepo.GetUserPreferences(ctx, userID)
}

func (ns *NotificationService) ProcessPendingNotifications(ctx context.Context) error {
	notifications, err := ns.notificationRepo.GetPendingNotifications(ctx, 100)
	if err != nil {
		return err
	}

	for _, notification := range notifications {
		// Get user info
		user, err := ns.userRepo.GetByID(ctx, notification.UserID.Hex())
		if err != nil {
			continue
		}

		go ns.processNotification(ctx, &notification, user)
	}

	return nil
}

func (ns *NotificationService) processNotification(ctx context.Context, notification *models.Notification, user *models.User) {
	// Update status to sending
	ns.notificationRepo.Update(ctx, notification.ID.Hex(), bson.M{
		"status": "sending",
	})

	var success bool

	// Send push notification
	if notification.Channels.Push && user.DeviceToken != "" {
		pushNotif := utils.PushNotification{
			Title: notification.Title,
			Body:  notification.Body,
			Data:  make(map[string]string),
		}

		// Convert data map to string map
		for k, v := range notification.Data {
			if str, ok := v.(string); ok {
				pushNotif.Data[k] = str
			}
		}

		_, err := ns.pushService.SendPushNotification(ctx, user.DeviceToken, pushNotif)
		if err == nil {
			success = true
		}
	}

	// Send SMS notification
	if notification.Channels.SMS && user.Phone != "" {
		sms := utils.SMSMessage{
			To:      user.Phone,
			Message: notification.Title + ": " + notification.Body,
		}

		_, err := ns.pushService.SendSMS(ctx, sms)
		if err == nil {
			success = true
		}
	}

	// Send email notification
	if notification.Channels.Email && user.Email != "" {
		email := utils.EmailMessage{
			To:      user.Email,
			Subject: notification.Title,
			Body:    notification.Body,
			IsHTML:  false,
		}

		_, err := ns.pushService.SendEmail(ctx, email)
		if err == nil {
			success = true
		}
	}

	// Update notification status
	status := "failed"
	if success {
		status = "sent"
	}

	updateData := bson.M{
		"status": status,
		"sentAt": time.Now(),
	}

	if !success {
		updateData["retryCount"] = notification.RetryCount + 1
		updateData["lastRetry"] = time.Now()
	}

	ns.notificationRepo.Update(ctx, notification.ID.Hex(), updateData)
}

func (ns *NotificationService) CleanupExpiredNotifications(ctx context.Context) error {
	deletedCount, err := ns.notificationRepo.DeleteExpired(ctx)
	if err != nil {
		return err
	}

	logrus.Info("Cleaned up ", deletedCount, " expired notifications")
	return nil
}
