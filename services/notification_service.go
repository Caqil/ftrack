// services/notification_service.go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/websocket"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationService struct {
	notificationRepo *repositories.NotificationRepository
	userRepo         *repositories.UserRepository
	circleRepo       *repositories.CircleRepository
	redis            *redis.Client
	hub              *websocket.Hub
	emailService     EmailService // Remove the pointer (*) for interface
	smsService       *SMSService
	pushService      *PushService
}

func NewNotificationService(
	notificationRepo *repositories.NotificationRepository,
	userRepo *repositories.UserRepository,
	circleRepo *repositories.CircleRepository,
	redis *redis.Client,
	hub *websocket.Hub,
	emailService EmailService, // Remove the pointer (*) for interface
	smsService *SMSService,
	pushService *PushService,
) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		circleRepo:       circleRepo,
		redis:            redis,
		hub:              hub,
		emailService:     emailService,
		smsService:       smsService,
		pushService:      pushService,
	}
}

// ========================
// Basic Notification Operations
// ========================

func (ns *NotificationService) GetNotifications(ctx context.Context, req models.GetNotificationsRequest) (*models.PaginatedNotifications, error) {
	notifications, total, err := ns.notificationRepo.GetUserNotifications(ctx, req.UserID, req.Page, req.PageSize, req.Type, req.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &models.PaginatedNotifications{
		Notifications: notifications,
		Page:          req.Page,
		PageSize:      req.PageSize,
		Total:         total,
		TotalPages:    totalPages,
		HasNext:       req.Page < totalPages,
		HasPrev:       req.Page > 1,
	}, nil
}

func (ns *NotificationService) GetNotification(ctx context.Context, userID, notificationID string) (*models.Notification, error) {
	notification, err := ns.notificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return nil, fmt.Errorf("notification not found")
	}

	if notification.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return notification, nil
}

// SendNotification sends an email notification
func (es *SMTPEmailService) SendNotification(ctx context.Context, notification *models.Notification) error {
	// Get user's email settings
	emailSettings, err := es.notificationRepo.GetEmailSettings(ctx, notification.UserID)
	if err != nil {
		logrus.Warnf("Failed to get email settings for user %s: %v", notification.UserID, err)
		return nil // Don't fail the entire notification
	}

	// Check if email notifications are enabled
	if !emailSettings.Enabled {
		logrus.Infof("Email notifications disabled for user %s", notification.UserID)
		return nil
	}

	// Check if email is verified
	if !emailSettings.IsVerified {
		logrus.Infof("Email not verified for user %s", notification.UserID)
		return nil
	}

	// Check type-specific settings
	if typeEnabled, exists := emailSettings.TypeSettings[notification.Type]; exists && !typeEnabled {
		logrus.Infof("Email notifications disabled for type %s for user %s", notification.Type, notification.UserID)
		return nil
	}

	// Get email template for this notification type
	subject := notification.Title
	if subject == "" {
		subject = "New Notification from FTrack"
	}

	return es.SendEmail(EmailData{
		To:       emailSettings.EmailAddress,
		Subject:  subject,
		Template: "notification",
		Data: map[string]interface{}{
			"Title":         notification.Title,
			"Message":       notification.Message,
			"Type":          notification.Type,
			"Priority":      notification.Priority,
			"ImageURL":      notification.ImageURL,
			"ActionButtons": notification.ActionButtons,
			"DeepLink":      notification.DeepLink,
			"CreatedAt":     notification.CreatedAt.Format(time.RFC3339),
			"Data":          notification.Data,
		},
	})
}

// SendTestEmail sends a test email
func (es *SMTPEmailService) SendTestEmail(ctx context.Context, toEmail, subject, content string) error {
	return es.SendEmail(EmailData{
		To:       toEmail,
		Subject:  subject,
		Template: "test_notification",
		Data: map[string]interface{}{
			"Content": content,
		},
	})
}

func (ns *NotificationService) MarkAsRead(ctx context.Context, userID, notificationID string) error {
	notification, err := ns.GetNotification(ctx, userID, notificationID)
	if err != nil {
		return err
	}

	now := time.Now()
	notification.Status = "read"
	notification.ReadAt = &now
	notification.UpdatedAt = now

	if err := ns.notificationRepo.Update(ctx, notification); err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	// Update badge count in real-time
	ns.updateBadgeCount(ctx, userID)

	return nil
}

func (ns *NotificationService) MarkAsUnread(ctx context.Context, userID, notificationID string) error {
	notification, err := ns.GetNotification(ctx, userID, notificationID)
	if err != nil {
		return err
	}

	notification.Status = "unread"
	notification.ReadAt = nil
	notification.UpdatedAt = time.Now()

	if err := ns.notificationRepo.Update(ctx, notification); err != nil {
		return fmt.Errorf("failed to mark notification as unread: %w", err)
	}

	// Update badge count in real-time
	ns.updateBadgeCount(ctx, userID)

	return nil
}

func (ns *NotificationService) DeleteNotification(ctx context.Context, userID, notificationID string) error {
	notification, err := ns.GetNotification(ctx, userID, notificationID)
	if err != nil {
		return err
	}

	if err := ns.notificationRepo.Delete(ctx, notification.ID.Hex()); err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	// Update badge count in real-time
	ns.updateBadgeCount(ctx, userID)

	return nil
}

// ========================
// Bulk Operations
// ========================

func (ns *NotificationService) BulkMarkAsRead(ctx context.Context, userID string, notificationIDs []string) (*models.BulkOperationResult, error) {
	result := &models.BulkOperationResult{}

	for _, id := range notificationIDs {
		if err := ns.MarkAsRead(ctx, userID, id); err != nil {
			result.FailedCount++
			result.FailedIDs = append(result.FailedIDs, id)
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

func (ns *NotificationService) BulkMarkAsUnread(ctx context.Context, userID string, notificationIDs []string) (*models.BulkOperationResult, error) {
	result := &models.BulkOperationResult{}

	for _, id := range notificationIDs {
		if err := ns.MarkAsUnread(ctx, userID, id); err != nil {
			result.FailedCount++
			result.FailedIDs = append(result.FailedIDs, id)
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

func (ns *NotificationService) BulkDeleteNotifications(ctx context.Context, userID string, notificationIDs []string) (*models.BulkOperationResult, error) {
	result := &models.BulkOperationResult{}

	for _, id := range notificationIDs {
		if err := ns.DeleteNotification(ctx, userID, id); err != nil {
			result.FailedCount++
			result.FailedIDs = append(result.FailedIDs, id)
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

func (ns *NotificationService) BulkArchiveNotifications(ctx context.Context, userID string, notificationIDs []string) (*models.BulkOperationResult, error) {
	result := &models.BulkOperationResult{}

	for _, id := range notificationIDs {
		notification, err := ns.GetNotification(ctx, userID, id)
		if err != nil {
			result.FailedCount++
			result.FailedIDs = append(result.FailedIDs, id)
			continue
		}

		notification.IsArchived = true
		notification.UpdatedAt = time.Now()

		if err := ns.notificationRepo.Update(ctx, notification); err != nil {
			result.FailedCount++
			result.FailedIDs = append(result.FailedIDs, id)
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

// ========================
// Filtering Operations
// ========================

func (ns *NotificationService) GetUnreadNotifications(ctx context.Context, userID string, page, pageSize int) (*models.PaginatedNotifications, error) {
	req := models.GetNotificationsRequest{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		Status:   "unread",
	}
	return ns.GetNotifications(ctx, req)
}

func (ns *NotificationService) GetReadNotifications(ctx context.Context, userID string, page, pageSize int) (*models.PaginatedNotifications, error) {
	req := models.GetNotificationsRequest{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		Status:   "read",
	}
	return ns.GetNotifications(ctx, req)
}

func (ns *NotificationService) GetNotificationsByType(ctx context.Context, userID, notificationType string, page, pageSize int) (*models.PaginatedNotifications, error) {
	req := models.GetNotificationsRequest{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		Type:     notificationType,
	}
	return ns.GetNotifications(ctx, req)
}

func (ns *NotificationService) GetNotificationsByPriority(ctx context.Context, userID, priority string, page, pageSize int) (*models.PaginatedNotifications, error) {
	notifications, total, err := ns.notificationRepo.GetNotificationsByPriority(ctx, userID, priority, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications by priority: %w", err)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.PaginatedNotifications{
		Notifications: notifications,
		Page:          page,
		PageSize:      pageSize,
		Total:         total,
		TotalPages:    totalPages,
		HasNext:       page < totalPages,
		HasPrev:       page > 1,
	}, nil
}

func (ns *NotificationService) GetCircleNotifications(ctx context.Context, userID, circleID string, page, pageSize int) (*models.PaginatedNotifications, error) {
	// Verify user has access to circle
	circle, err := ns.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, fmt.Errorf("circle not found")
	}

	// Convert userID string to ObjectID for comparisons
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	hasAccess := false

	// Check if user is the admin (owner) of the circle
	if circle.AdminID == userObjectID {
		hasAccess = true
	} else {
		// Check if user is a member of the circle
		for _, member := range circle.Members {
			if member.UserID == userObjectID {
				hasAccess = true
				break
			}
		}
	}

	if !hasAccess {
		return nil, fmt.Errorf("access denied")
	}

	notifications, total, err := ns.notificationRepo.GetCircleNotifications(ctx, userID, circleID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get circle notifications: %w", err)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.PaginatedNotifications{
		Notifications: notifications,
		Page:          page,
		PageSize:      pageSize,
		Total:         total,
		TotalPages:    totalPages,
		HasNext:       page < totalPages,
		HasPrev:       page > 1,
	}, nil
}

func (ns *NotificationService) GetArchivedNotifications(ctx context.Context, userID string, page, pageSize int) (*models.PaginatedNotifications, error) {
	notifications, total, err := ns.notificationRepo.GetArchivedNotifications(ctx, userID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get archived notifications: %w", err)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.PaginatedNotifications{
		Notifications: notifications,
		Page:          page,
		PageSize:      pageSize,
		Total:         total,
		TotalPages:    totalPages,
		HasNext:       page < totalPages,
		HasPrev:       page > 1,
	}, nil
}

// ========================
// Push Notification Management
// ========================

func (ns *NotificationService) GetPushSettings(ctx context.Context, userID string) (*models.PushSettings, error) {
	settings, err := ns.notificationRepo.GetPushSettings(ctx, userID)
	if err != nil {
		// Create default settings if not found
		if err.Error() == "not found" {
			defaultSettings := &models.PushSettings{
				UserID:       userID,
				Enabled:      true,
				Sound:        true,
				Vibration:    true,
				Badge:        true,
				Preview:      true,
				TypeSettings: make(map[string]models.TypePreference),
				QuietHours: models.QuietHours{
					Enabled: false,
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := ns.notificationRepo.CreatePushSettings(ctx, defaultSettings); err != nil {
				return nil, fmt.Errorf("failed to create default push settings: %w", err)
			}

			return defaultSettings, nil
		}
		return nil, fmt.Errorf("failed to get push settings: %w", err)
	}

	return settings, nil
}

func (ns *NotificationService) UpdatePushSettings(ctx context.Context, userID string, req models.UpdatePushSettingsRequest) (*models.PushSettings, error) {
	settings, err := ns.GetPushSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields that are provided
	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.Sound != nil {
		settings.Sound = *req.Sound
	}
	if req.Vibration != nil {
		settings.Vibration = *req.Vibration
	}
	if req.Badge != nil {
		settings.Badge = *req.Badge
	}
	if req.Preview != nil {
		settings.Preview = *req.Preview
	}
	if req.TypeSettings != nil {
		settings.TypeSettings = req.TypeSettings
	}
	if req.QuietHours != nil {
		settings.QuietHours = *req.QuietHours
	}

	settings.UpdatedAt = time.Now()

	if err := ns.notificationRepo.UpdatePushSettings(ctx, settings); err != nil {
		return nil, fmt.Errorf("failed to update push settings: %w", err)
	}

	return settings, nil
}

func (ns *NotificationService) SendTestNotification(ctx context.Context, userID string, req models.TestNotificationRequest) error {
	notification := &models.Notification{
		ID:               primitive.NewObjectID(),
		UserID:           userID,
		Title:            req.Title,
		Message:          req.Message,
		Type:             req.Type,
		Priority:         "normal",
		Status:           "unread",
		Data:             req.Data,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		DeliveryChannels: []string{"push"},
	}

	// Send via push service
	if err := ns.pushService.SendNotification(ctx, notification); err != nil {
		logrus.Errorf("Failed to send test push notification: %v", err)
	}

	return nil
}

func (ns *NotificationService) RegisterPushDevice(ctx context.Context, userID string, req models.RegisterDeviceRequest) (*models.PushDevice, error) {
	// Check if device already exists
	existingDevice, err := ns.notificationRepo.GetDeviceByToken(ctx, req.DeviceToken)
	if err == nil && existingDevice.UserID == userID {
		return nil, fmt.Errorf("device already registered")
	}

	device := &models.PushDevice{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		DeviceToken: req.DeviceToken,
		DeviceType:  req.DeviceType,
		AppVersion:  req.AppVersion,
		DeviceModel: req.DeviceModel,
		OS:          req.OS,
		OSVersion:   req.OSVersion,
		IsActive:    true,
		LastUsed:    time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := ns.notificationRepo.CreatePushDevice(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to register device: %w", err)
	}

	return device, nil
}

func (ns *NotificationService) UpdatePushDevice(ctx context.Context, userID, deviceID string, req models.UpdateDeviceRequest) (*models.PushDevice, error) {
	device, err := ns.notificationRepo.GetPushDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found")
	}

	if device.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Update fields that are provided
	if req.AppVersion != "" {
		device.AppVersion = req.AppVersion
	}
	if req.DeviceModel != "" {
		device.DeviceModel = req.DeviceModel
	}
	if req.OS != "" {
		device.OS = req.OS
	}
	if req.OSVersion != "" {
		device.OSVersion = req.OSVersion
	}
	if req.IsActive != nil {
		device.IsActive = *req.IsActive
	}

	device.LastUsed = time.Now()
	device.UpdatedAt = time.Now()

	if err := ns.notificationRepo.UpdatePushDevice(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	return device, nil
}

func (ns *NotificationService) UnregisterPushDevice(ctx context.Context, userID, deviceID string) error {
	device, err := ns.notificationRepo.GetPushDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("device not found")
	}

	if device.UserID != userID {
		return fmt.Errorf("access denied")
	}

	if err := ns.notificationRepo.DeletePushDevice(ctx, deviceID); err != nil {
		return fmt.Errorf("failed to unregister device: %w", err)
	}

	return nil
}

func (ns *NotificationService) GetPushDevices(ctx context.Context, userID string) ([]models.PushDevice, error) {
	devices, err := ns.notificationRepo.GetUserPushDevices(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get push devices: %w", err)
	}

	return devices, nil
}

// ========================
// Notification Preferences
// ========================

func (ns *NotificationService) GetNotificationPreferences(ctx context.Context, userID string) (*models.NotificationPreferences, error) {
	preferences, err := ns.notificationRepo.GetNotificationPreferences(ctx, userID)
	if err != nil {
		// Create default preferences if not found
		if err.Error() == "not found" {
			defaultPrefs := &models.NotificationPreferences{
				UserID:          userID,
				GlobalEnabled:   true,
				EmailEnabled:    true,
				SMSEnabled:      true,
				PushEnabled:     true,
				InAppEnabled:    true,
				TypePreferences: make(map[string]models.TypePreference),
				Schedule: models.NotificationSchedule{
					Enabled: false,
				},
				Language:  "en",
				Timezone:  "UTC",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := ns.notificationRepo.CreateNotificationPreferences(ctx, defaultPrefs); err != nil {
				return nil, fmt.Errorf("failed to create default preferences: %w", err)
			}

			return defaultPrefs, nil
		}
		return nil, fmt.Errorf("failed to get preferences: %w", err)
	}

	return preferences, nil
}

func (ns *NotificationService) UpdateNotificationPreferences(ctx context.Context, userID string, req models.UpdateNotificationPreferencesRequest) (*models.NotificationPreferences, error) {
	preferences, err := ns.GetNotificationPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields that are provided
	if req.GlobalEnabled != nil {
		preferences.GlobalEnabled = *req.GlobalEnabled
	}
	if req.EmailEnabled != nil {
		preferences.EmailEnabled = *req.EmailEnabled
	}
	if req.SMSEnabled != nil {
		preferences.SMSEnabled = *req.SMSEnabled
	}
	if req.PushEnabled != nil {
		preferences.PushEnabled = *req.PushEnabled
	}
	if req.InAppEnabled != nil {
		preferences.InAppEnabled = *req.InAppEnabled
	}
	if req.TypePreferences != nil {
		preferences.TypePreferences = req.TypePreferences
	}
	if req.Schedule != nil {
		preferences.Schedule = *req.Schedule
	}
	if req.Language != "" {
		preferences.Language = req.Language
	}
	if req.Timezone != "" {
		preferences.Timezone = req.Timezone
	}

	preferences.UpdatedAt = time.Now()

	if err := ns.notificationRepo.UpdateNotificationPreferences(ctx, preferences); err != nil {
		return nil, fmt.Errorf("failed to update preferences: %w", err)
	}

	return preferences, nil
}

func (ns *NotificationService) GetNotificationTypes(ctx context.Context) ([]models.NotificationType, error) {
	// Return predefined notification types
	types := []models.NotificationType{
		{
			ID:          "emergency",
			Name:        "Emergency",
			Description: "Emergency alerts and notifications",
			Category:    "safety",
			Channels:    []string{"push", "sms", "email", "in-app"},
			IsSystem:    true,
		},
		{
			ID:          "location_update",
			Name:        "Location Update",
			Description: "Location sharing notifications",
			Category:    "location",
			Channels:    []string{"push", "in-app"},
			IsSystem:    true,
		},
		{
			ID:          "circle_invite",
			Name:        "Circle Invitation",
			Description: "Circle invitation notifications",
			Category:    "social",
			Channels:    []string{"push", "email", "in-app"},
			IsSystem:    true,
		},
		{
			ID:          "message",
			Name:        "Message",
			Description: "Chat message notifications",
			Category:    "communication",
			Channels:    []string{"push", "in-app"},
			IsSystem:    true,
		},
		{
			ID:          "geofence",
			Name:        "Geofence",
			Description: "Geofence entry/exit notifications",
			Category:    "location",
			Channels:    []string{"push", "in-app"},
			IsSystem:    true,
		},
	}

	return types, nil
}

func (ns *NotificationService) UpdateTypePreferences(ctx context.Context, userID, notificationType string, req models.UpdateTypePreferencesRequest) (*models.TypePreference, error) {
	preferences, err := ns.GetNotificationPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Validate notification type
	types, _ := ns.GetNotificationTypes(ctx)
	validType := false
	for _, t := range types {
		if t.ID == notificationType {
			validType = true
			break
		}
	}

	if !validType {
		return nil, fmt.Errorf("invalid type")
	}

	// Get existing preference or create default
	typePreference := preferences.TypePreferences[notificationType]

	// Update fields that are provided
	if req.Enabled != nil {
		typePreference.Enabled = *req.Enabled
	}
	if req.Sound != "" {
		typePreference.Sound = req.Sound
	}
	if req.Vibration != nil {
		typePreference.Vibration = *req.Vibration
	}

	preferences.TypePreferences[notificationType] = typePreference
	preferences.UpdatedAt = time.Now()

	if err := ns.notificationRepo.UpdateNotificationPreferences(ctx, preferences); err != nil {
		return nil, fmt.Errorf("failed to update type preferences: %w", err)
	}

	return &typePreference, nil
}

func (ns *NotificationService) GetNotificationSchedule(ctx context.Context, userID string) (*models.NotificationSchedule, error) {
	preferences, err := ns.GetNotificationPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &preferences.Schedule, nil
}

func (ns *NotificationService) UpdateNotificationSchedule(ctx context.Context, userID string, req models.UpdateNotificationScheduleRequest) (*models.NotificationSchedule, error) {
	preferences, err := ns.GetNotificationPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update schedule fields that are provided
	if req.Enabled != nil {
		preferences.Schedule.Enabled = *req.Enabled
	}
	if req.AllowedTimeRanges != nil {
		preferences.Schedule.AllowedTimeRanges = req.AllowedTimeRanges
	}
	if req.Timezone != "" {
		preferences.Schedule.Timezone = req.Timezone
	}
	if req.ExceptionDates != nil {
		preferences.Schedule.ExceptionDates = req.ExceptionDates
	}
	if req.SpecialSchedules != nil {
		preferences.Schedule.SpecialSchedules = req.SpecialSchedules
	}

	preferences.UpdatedAt = time.Now()

	if err := ns.notificationRepo.UpdateNotificationPreferences(ctx, preferences); err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	return &preferences.Schedule, nil
}

// ========================
// Email Notifications
// ========================

func (ns *NotificationService) GetEmailSettings(ctx context.Context, userID string) (*models.EmailSettings, error) {
	settings, err := ns.notificationRepo.GetEmailSettings(ctx, userID)
	if err != nil {
		// Create default settings if not found
		if err.Error() == "not found" {
			defaultSettings := &models.EmailSettings{
				UserID:          userID,
				Enabled:         true,
				DigestEnabled:   false,
				DigestFrequency: "daily",
				DigestTime:      "09:00",
				TypeSettings:    make(map[string]bool),
				Templates:       make(map[string]string),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			if err := ns.notificationRepo.CreateEmailSettings(ctx, defaultSettings); err != nil {
				return nil, fmt.Errorf("failed to create default email settings: %w", err)
			}

			return defaultSettings, nil
		}
		return nil, fmt.Errorf("failed to get email settings: %w", err)
	}

	return settings, nil
}

func (ns *NotificationService) UpdateEmailSettings(ctx context.Context, userID string, req models.UpdateEmailSettingsRequest) (*models.EmailSettings, error) {
	settings, err := ns.GetEmailSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields that are provided
	if req.EmailAddress != "" {
		settings.EmailAddress = req.EmailAddress
		settings.IsVerified = false // Reset verification when email changes
	}
	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.DigestEnabled != nil {
		settings.DigestEnabled = *req.DigestEnabled
	}
	if req.DigestFrequency != "" {
		settings.DigestFrequency = req.DigestFrequency
	}
	if req.DigestTime != "" {
		settings.DigestTime = req.DigestTime
	}
	if req.TypeSettings != nil {
		settings.TypeSettings = req.TypeSettings
	}
	if req.Templates != nil {
		settings.Templates = req.Templates
	}

	settings.UpdatedAt = time.Now()

	if err := ns.notificationRepo.UpdateEmailSettings(ctx, settings); err != nil {
		return nil, fmt.Errorf("failed to update email settings: %w", err)
	}

	return settings, nil
}

func (ns *NotificationService) VerifyEmailAddress(ctx context.Context, userID string, req models.VerifyEmailRequest) (*models.EmailSettings, error) {
	// Get verification code from Redis
	cacheKey := fmt.Sprintf("email_verification:%s:%s", userID, req.EmailAddress)
	storedCode, err := ns.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("verification code expired")
		}
		return nil, fmt.Errorf("failed to get verification code: %w", err)
	}

	if storedCode != req.VerificationCode {
		return nil, fmt.Errorf("invalid verification code")
	}

	// Update email settings to mark as verified
	settings, err := ns.GetEmailSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	settings.EmailAddress = req.EmailAddress
	settings.IsVerified = true
	settings.UpdatedAt = time.Now()

	if err := ns.notificationRepo.UpdateEmailSettings(ctx, settings); err != nil {
		return nil, fmt.Errorf("failed to update email settings: %w", err)
	}

	// Remove verification code from Redis
	ns.redis.Del(ctx, cacheKey)

	return settings, nil
}

func (ns *NotificationService) GetEmailTemplates(ctx context.Context, userID string) ([]models.EmailTemplate, error) {
	templates, err := ns.notificationRepo.GetEmailTemplates(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email templates: %w", err)
	}

	return templates, nil
}

func (ns *NotificationService) UpdateEmailTemplate(ctx context.Context, userID, templateID string, req models.UpdateEmailTemplateRequest) (*models.EmailTemplate, error) {
	template, err := ns.notificationRepo.GetEmailTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found")
	}

	if template.UserID != userID && !template.IsDefault {
		return nil, fmt.Errorf("access denied")
	}

	// Update fields that are provided
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Subject != "" {
		template.Subject = req.Subject
	}
	if req.HTMLContent != "" {
		template.HTMLContent = req.HTMLContent
	}
	if req.TextContent != "" {
		template.TextContent = req.TextContent
	}
	if req.Variables != nil {
		template.Variables = req.Variables
	}

	template.UpdatedAt = time.Now()

	if err := ns.notificationRepo.UpdateEmailTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return template, nil
}

// ========================
// Helper Methods
// ========================

func (ns *NotificationService) updateBadgeCount(ctx context.Context, userID string) {
	// Get current badge counts
	badges, err := ns.GetNotificationBadges(ctx, userID)
	if err != nil {
		logrus.Errorf("Failed to get notification badges: %v", err)
		return
	}

	// Send real-time update via WebSocket
	if ns.hub != nil {
		message := map[string]interface{}{
			"type": "badge_update",
			"data": badges,
		}

		msgBytes, _ := json.Marshal(message)
		ns.hub.SendNotificationToUser(userID, msgBytes)
	}

	// Cache badge count in Redis
	cacheKey := fmt.Sprintf("user:badges:%s", userID)
	badgeData, _ := json.Marshal(badges)
	ns.redis.Set(ctx, cacheKey, badgeData, time.Hour).Err()
}

func (ns *NotificationService) GetNotificationBadges(ctx context.Context, userID string) (*models.NotificationBadges, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("user:badges:%s", userID)
	cachedData, err := ns.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var badges models.NotificationBadges
		if json.Unmarshal([]byte(cachedData), &badges) == nil {
			return &badges, nil
		}
	}

	// Calculate badge counts
	total, err := ns.notificationRepo.GetNotificationCount(ctx, userID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	unread, err := ns.notificationRepo.GetNotificationCount(ctx, userID, "unread")
	if err != nil {
		return nil, fmt.Errorf("failed to get unread count: %w", err)
	}

	byType, err := ns.notificationRepo.GetNotificationCountsByType(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get counts by type: %w", err)
	}

	byCircle, err := ns.notificationRepo.GetNotificationCountsByCircle(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get counts by circle: %w", err)
	}

	byPriority, err := ns.notificationRepo.GetNotificationCountsByPriority(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get counts by priority: %w", err)
	}

	badges := &models.NotificationBadges{
		UserID:      userID,
		Total:       int(total),
		Unread:      int(unread),
		ByType:      byType,
		ByCircle:    byCircle,
		ByPriority:  byPriority,
		LastUpdated: time.Now(),
	}

	// Cache the result
	badgeData, _ := json.Marshal(badges)
	ns.redis.Set(ctx, cacheKey, badgeData, time.Hour).Err()

	return badges, nil
}

// Placeholder methods for other operations that would need full implementation
// These are simplified versions to show the structure

func (ns *NotificationService) GetSMSSettings(ctx context.Context, userID string) (*models.SMSSettings, error) {
	// Implementation would be similar to email settings
	return &models.SMSSettings{}, nil
}

func (ns *NotificationService) UpdateSMSSettings(ctx context.Context, userID string, req models.UpdateSMSSettingsRequest) (*models.SMSSettings, error) {
	return &models.SMSSettings{}, nil
}

func (ns *NotificationService) VerifyPhoneNumber(ctx context.Context, userID string, req models.VerifyPhoneRequest) (*models.SMSSettings, error) {
	return &models.SMSSettings{}, nil
}

func (ns *NotificationService) SendTestSMS(ctx context.Context, userID string, req models.TestSMSRequest) error {
	return nil
}

func (ns *NotificationService) GetSMSUsage(ctx context.Context, userID string, days int) (*models.SMSUsage, error) {
	return &models.SMSUsage{}, nil
}

func (ns *NotificationService) GetInAppSettings(ctx context.Context, userID string) (*models.InAppSettings, error) {
	return &models.InAppSettings{}, nil
}

func (ns *NotificationService) UpdateInAppSettings(ctx context.Context, userID string, req models.UpdateInAppSettingsRequest) (*models.InAppSettings, error) {
	return &models.InAppSettings{}, nil
}

func (ns *NotificationService) ClearNotificationBadges(ctx context.Context, userID string, badgeTypes []string) error {
	return nil
}

func (ns *NotificationService) GetNotificationSounds(ctx context.Context) ([]models.NotificationSound, error) {
	return []models.NotificationSound{}, nil
}

func (ns *NotificationService) UpdateNotificationSounds(ctx context.Context, userID string, req models.UpdateSoundPreferencesRequest) (*models.UpdateSoundPreferencesRequest, error) {
	return &req, nil
}

func (ns *NotificationService) GetNotificationChannels(ctx context.Context, userID string) ([]models.NotificationChannel, error) {
	return []models.NotificationChannel{}, nil
}

func (ns *NotificationService) CreateNotificationChannel(ctx context.Context, userID string, req models.CreateChannelRequest) (*models.NotificationChannel, error) {
	return &models.NotificationChannel{}, nil
}

func (ns *NotificationService) UpdateNotificationChannel(ctx context.Context, userID, channelID string, req models.UpdateChannelRequest) (*models.NotificationChannel, error) {
	return &models.NotificationChannel{}, nil
}

func (ns *NotificationService) DeleteNotificationChannel(ctx context.Context, userID, channelID string) error {
	return nil
}

func (ns *NotificationService) TestNotificationChannel(ctx context.Context, userID, channelID string) (*models.ChannelTestResult, error) {
	return &models.ChannelTestResult{}, nil
}

func (ns *NotificationService) GetNotificationRules(ctx context.Context, userID string) ([]models.NotificationRule, error) {
	return []models.NotificationRule{}, nil
}

func (ns *NotificationService) CreateNotificationRule(ctx context.Context, userID string, req models.CreateRuleRequest) (*models.NotificationRule, error) {
	return &models.NotificationRule{}, nil
}

func (ns *NotificationService) GetNotificationRule(ctx context.Context, userID, ruleID string) (*models.NotificationRule, error) {
	return &models.NotificationRule{}, nil
}

func (ns *NotificationService) UpdateNotificationRule(ctx context.Context, userID, ruleID string, req models.UpdateRuleRequest) (*models.NotificationRule, error) {
	return &models.NotificationRule{}, nil
}

func (ns *NotificationService) DeleteNotificationRule(ctx context.Context, userID, ruleID string) error {
	return nil
}

func (ns *NotificationService) TestNotificationRule(ctx context.Context, userID, ruleID string) (*models.RuleTestResult, error) {
	return &models.RuleTestResult{}, nil
}

func (ns *NotificationService) GetDoNotDisturbStatus(ctx context.Context, userID string) (*models.DoNotDisturbStatus, error) {
	return &models.DoNotDisturbStatus{}, nil
}

func (ns *NotificationService) EnableDoNotDisturb(ctx context.Context, userID string, req models.EnableDNDRequest) (*models.DoNotDisturbStatus, error) {
	return &models.DoNotDisturbStatus{}, nil
}

func (ns *NotificationService) DisableDoNotDisturb(ctx context.Context, userID string) (*models.DoNotDisturbStatus, error) {
	return &models.DoNotDisturbStatus{}, nil
}

func (ns *NotificationService) GetQuietHours(ctx context.Context, userID string) (*models.QuietHours, error) {
	return &models.QuietHours{}, nil
}

func (ns *NotificationService) UpdateQuietHours(ctx context.Context, userID string, req models.UpdateQuietHoursRequest) (*models.QuietHours, error) {
	return &models.QuietHours{}, nil
}

func (ns *NotificationService) GetDNDExceptions(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}

func (ns *NotificationService) UpdateDNDExceptions(ctx context.Context, userID string, req models.UpdateDNDExceptionsRequest) ([]string, error) {
	return req.Exceptions, nil
}

func (ns *NotificationService) GetNotificationTemplates(ctx context.Context, userID, templateType, category string) ([]models.NotificationTemplate, error) {
	return []models.NotificationTemplate{}, nil
}

func (ns *NotificationService) CreateNotificationTemplate(ctx context.Context, userID string, req models.CreateTemplateRequest) (*models.NotificationTemplate, error) {
	return &models.NotificationTemplate{}, nil
}

func (ns *NotificationService) GetNotificationTemplate(ctx context.Context, userID, templateID string) (*models.NotificationTemplate, error) {
	return &models.NotificationTemplate{}, nil
}

func (ns *NotificationService) UpdateNotificationTemplate(ctx context.Context, userID, templateID string, req models.UpdateTemplateRequest) (*models.NotificationTemplate, error) {
	return &models.NotificationTemplate{}, nil
}

func (ns *NotificationService) DeleteNotificationTemplate(ctx context.Context, userID, templateID string) error {
	return nil
}

func (ns *NotificationService) PreviewNotificationTemplate(ctx context.Context, userID, templateID string, req models.PreviewTemplateRequest) (*models.TemplatePreview, error) {
	return &models.TemplatePreview{}, nil
}

func (ns *NotificationService) GetNotificationStats(ctx context.Context, userID string, days int, groupBy string) (*models.NotificationStats, error) {
	return &models.NotificationStats{}, nil
}

func (ns *NotificationService) GetDeliveryStats(ctx context.Context, userID string, days int, channel string) (*models.DeliveryStats, error) {
	return &models.DeliveryStats{}, nil
}

func (ns *NotificationService) GetEngagementStats(ctx context.Context, userID string, days int, notificationType string) (*models.EngagementStats, error) {
	return &models.EngagementStats{}, nil
}

func (ns *NotificationService) GetNotificationTrends(ctx context.Context, userID string, days int, metric string) (*models.NotificationTrends, error) {
	return &models.NotificationTrends{}, nil
}

func (ns *NotificationService) GetNotificationPerformance(ctx context.Context, userID string, days, compareWith int) (*models.NotificationPerformance, error) {
	return &models.NotificationPerformance{}, nil
}

func (ns *NotificationService) GetNotificationHistory(ctx context.Context, req models.GetHistoryRequest) (*models.NotificationHistory, error) {
	return &models.NotificationHistory{}, nil
}

func (ns *NotificationService) GetDeliveryHistory(ctx context.Context, userID, notificationID string) (*models.DeliveryHistory, error) {
	return &models.DeliveryHistory{}, nil
}

func (ns *NotificationService) ExportNotificationHistory(ctx context.Context, req models.ExportHistoryRequest) (*models.ExportResult, error) {
	return &models.ExportResult{}, nil
}

func (ns *NotificationService) DownloadNotificationExport(ctx context.Context, userID, exportID string) ([]byte, string, error) {
	return []byte{}, "export.csv", nil
}

func (ns *NotificationService) CleanupOldNotifications(ctx context.Context, userID, role string, days int, dryRun bool) (*models.CleanupResult, error) {
	return &models.CleanupResult{}, nil
}

func (ns *NotificationService) GetNotificationSubscriptions(ctx context.Context, userID string) ([]models.NotificationSubscription, error) {
	return []models.NotificationSubscription{}, nil
}

func (ns *NotificationService) CreateNotificationSubscription(ctx context.Context, userID string, req models.CreateSubscriptionRequest) (*models.NotificationSubscription, error) {
	return &models.NotificationSubscription{}, nil
}

func (ns *NotificationService) UpdateNotificationSubscription(ctx context.Context, userID, subscriptionID string, req models.UpdateSubscriptionRequest) (*models.NotificationSubscription, error) {
	return &models.NotificationSubscription{}, nil
}

func (ns *NotificationService) DeleteNotificationSubscription(ctx context.Context, userID, subscriptionID string) error {
	return nil
}

func (ns *NotificationService) GetNotificationTopics(ctx context.Context) ([]models.NotificationTopic, error) {
	return []models.NotificationTopic{}, nil
}

func (ns *NotificationService) GetNotificationActions(ctx context.Context, userID, notificationType string) ([]models.NotificationAction, error) {
	return []models.NotificationAction{}, nil
}

func (ns *NotificationService) ExecuteNotificationAction(ctx context.Context, userID, notificationID, actionID string, req models.ExecuteActionRequest) (*models.ActionResult, error) {
	return &models.ActionResult{}, nil
}

func (ns *NotificationService) SnoozeNotification(ctx context.Context, userID, notificationID string, req models.SnoozeRequest) (*models.SnoozeResult, error) {
	return &models.SnoozeResult{}, nil
}

func (ns *NotificationService) PinNotification(ctx context.Context, userID, notificationID string) error {
	notification, err := ns.GetNotification(ctx, userID, notificationID)
	if err != nil {
		return err
	}

	if notification.IsPinned {
		return fmt.Errorf("already pinned")
	}

	notification.IsPinned = true
	notification.UpdatedAt = time.Now()

	if err := ns.notificationRepo.Update(ctx, notification); err != nil {
		return fmt.Errorf("failed to pin notification: %w", err)
	}

	return nil
}

func (ns *NotificationService) UnpinNotification(ctx context.Context, userID, notificationID string) error {
	notification, err := ns.GetNotification(ctx, userID, notificationID)
	if err != nil {
		return err
	}

	if !notification.IsPinned {
		return fmt.Errorf("not pinned")
	}

	notification.IsPinned = false
	notification.UpdatedAt = time.Now()

	if err := ns.notificationRepo.Update(ctx, notification); err != nil {
		return fmt.Errorf("failed to unpin notification: %w", err)
	}

	return nil
}

// SendNotification is the main method for sending notifications
func (ns *NotificationService) SendNotification(ctx context.Context, req models.SendNotificationRequest) error {
	// Validate request
	if len(req.Recipients) == 0 {
		return fmt.Errorf("no recipients")
	}

	// Send notification to each recipient
	for _, recipientID := range req.Recipients {
		notification := &models.Notification{
			ID:               primitive.NewObjectID(),
			UserID:           recipientID,
			Title:            req.Title,
			Message:          req.Message,
			Type:             req.Type,
			Priority:         req.Priority,
			Category:         req.Category,
			Status:           "unread",
			Data:             req.Data,
			ActionButtons:    req.ActionButtons,
			ImageURL:         req.ImageURL,
			DeepLink:         req.DeepLink,
			ScheduledAt:      req.ScheduledAt,
			ExpiresAt:        req.ExpiresAt,
			DeliveryChannels: req.DeliveryChannels,
			Metadata:         req.Metadata,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		// Save notification to database
		if err := ns.notificationRepo.Create(ctx, notification); err != nil {
			logrus.Errorf("Failed to save notification for user %s: %v", recipientID, err)
			continue
		}

		// Send via configured channels
		for _, channel := range req.DeliveryChannels {
			switch channel {
			case "push":
				if err := ns.pushService.SendNotification(ctx, notification); err != nil {
					logrus.Errorf("Failed to send push notification: %v", err)
				}
			case "email":
				if err := ns.emailService.SendNotification(ctx, notification); err != nil {
					logrus.Errorf("Failed to send email notification: %v", err)
				}
			case "sms":
				if err := ns.smsService.SendNotification(ctx, notification); err != nil {
					logrus.Errorf("Failed to send SMS notification: %v", err)
				}
			case "in-app":
				// Send real-time notification via WebSocket
				if ns.hub != nil {
					ns.hub.SendNotificationToUser(recipientID, notification)
				}
			}
		}

		// Update badge count
		ns.updateBadgeCount(ctx, recipientID)
	}

	return nil
}
