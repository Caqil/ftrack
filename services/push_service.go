// services/push_service.go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"ftrack/models"
	"ftrack/repositories"

	"firebase.google.com/go/v4/messaging"
	"github.com/sirupsen/logrus"
)

type PushService struct {
	fcmClient        *messaging.Client
	notificationRepo *repositories.NotificationRepository
}

func NewPushService(fcmClient *messaging.Client, notificationRepo *repositories.NotificationRepository) *PushService {
	return &PushService{
		fcmClient:        fcmClient,
		notificationRepo: notificationRepo,
	}
}

// SendNotification sends a push notification to user's devices
func (ps *PushService) SendNotification(ctx context.Context, notification *models.Notification) error {
	// Get user's active push devices
	devices, err := ps.notificationRepo.GetUserPushDevices(ctx, notification.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user devices: %w", err)
	}

	if len(devices) == 0 {
		logrus.Warnf("No active devices found for user %s", notification.UserID)
		return nil
	}

	// Get user's push settings
	pushSettings, err := ps.notificationRepo.GetPushSettings(ctx, notification.UserID)
	if err != nil {
		logrus.Warnf("Failed to get push settings for user %s: %v", notification.UserID, err)
		// Continue with default settings
		pushSettings = &models.PushSettings{
			Enabled:   true,
			Sound:     true,
			Vibration: true,
			Badge:     true,
			Preview:   true,
		}
	}

	// Check if push notifications are enabled
	if !pushSettings.Enabled {
		logrus.Infof("Push notifications disabled for user %s", notification.UserID)
		return nil
	}

	// Check type-specific settings
	if typeSettings, exists := pushSettings.TypeSettings[notification.Type]; exists {
		if !typeSettings.Enabled {
			logrus.Infof("Push notifications disabled for type %s for user %s", notification.Type, notification.UserID)
			return nil
		}
	}

	// Check quiet hours
	if ps.isQuietHours(pushSettings.QuietHours) {
		logrus.Infof("Notification suppressed due to quiet hours for user %s", notification.UserID)
		return nil
	}

	// Prepare FCM messages for each device
	var messages []*messaging.Message
	for _, device := range devices {
		message := ps.buildFCMMessage(notification, device, pushSettings)
		if message != nil {
			messages = append(messages, message)
		}
	}

	if len(messages) == 0 {
		return fmt.Errorf("no valid messages to send")
	}

	// Send messages
	return ps.sendFCMMessages(ctx, messages, notification)
}

// buildFCMMessage creates an FCM message from notification data
func (ps *PushService) buildFCMMessage(notification *models.Notification, device models.PushDevice, settings *models.PushSettings) *messaging.Message {
	// Build notification payload
	fcmNotification := &messaging.Notification{
		Title: notification.Title,
		Body:  notification.Message,
	}

	// Add image if present
	if notification.ImageURL != "" {
		fcmNotification.ImageURL = notification.ImageURL
	}

	// Build data payload
	data := map[string]string{
		"notification_id": notification.ID.Hex(),
		"type":            notification.Type,
		"priority":        notification.Priority,
		"user_id":         notification.UserID,
	}

	// Add custom data if present
	if notification.Data != nil {
		if dataBytes, err := json.Marshal(notification.Data); err == nil {
			data["custom_data"] = string(dataBytes)
		}
	}

	// Add deep link if present
	if notification.DeepLink != "" {
		data["deep_link"] = notification.DeepLink
	}

	// Add circle ID if present
	if notification.CircleID != "" {
		data["circle_id"] = notification.CircleID
	}

	// Add action buttons if present
	if len(notification.ActionButtons) > 0 {
		if actionsBytes, err := json.Marshal(notification.ActionButtons); err == nil {
			data["action_buttons"] = string(actionsBytes)
		}
	}

	// Build Android config
	androidConfig := &messaging.AndroidConfig{
		Priority: "high",
		Notification: &messaging.AndroidNotification{
			Title:       notification.Title,
			Body:        notification.Message,
			Icon:        "ic_notification",
			Color:       "#FF5722",
			ClickAction: "FLUTTER_NOTIFICATION_CLICK",
		},
	}

	// Configure sound and vibration
	if settings.Sound {
		if typeSettings, exists := settings.TypeSettings[notification.Type]; exists && typeSettings.Sound != "" {
			androidConfig.Notification.Sound = typeSettings.Sound
		} else {
			androidConfig.Notification.Sound = "default"
		}
	}

	if settings.Vibration {
		androidConfig.Notification.DefaultVibrateTimings = true
	}

	// Set priority based on notification priority
	switch notification.Priority {
	case "high", "urgent":
		androidConfig.Priority = "high"
		androidConfig.Notification.Priority = messaging.PriorityHigh
	case "low":
		androidConfig.Priority = "normal"
		androidConfig.Notification.Priority = messaging.PriorityLow
	default:
		androidConfig.Priority = "normal"
		androidConfig.Notification.Priority = messaging.PriorityDefault
	}

	// Build iOS config
	iosConfig := &messaging.APNSConfig{
		Headers: map[string]string{
			"apns-priority": "10",
		},
		Payload: &messaging.APNSPayload{
			Aps: &messaging.Aps{
				Alert: &messaging.ApsAlert{
					Title: notification.Title,
					Body:  notification.Message,
				},
				Sound: "default",
			},
		},
	}

	// Configure iOS sound
	if settings.Sound {
		if typeSettings, exists := settings.TypeSettings[notification.Type]; exists && typeSettings.Sound != "" {
			iosConfig.Payload.Aps.Sound = typeSettings.Sound
		}
	} else {
		iosConfig.Payload.Aps.Sound = ""
	}

	// Configure badge if enabled
	if settings.Badge {
		// You might want to get the current badge count here
		badge := 1
		iosConfig.Payload.Aps.Badge = &badge
	}

	// Set priority for iOS
	switch notification.Priority {
	case "high", "urgent":
		iosConfig.Headers["apns-priority"] = "10"
	default:
		iosConfig.Headers["apns-priority"] = "5"
	}

	// Build the final message
	message := &messaging.Message{
		Token:        device.DeviceToken,
		Notification: fcmNotification,
		Data:         data,
		Android:      androidConfig,
		APNS:         iosConfig,
	}

	return message
}

// sendFCMMessages sends the prepared FCM messages
func (ps *PushService) sendFCMMessages(ctx context.Context, messages []*messaging.Message, notification *models.Notification) error {
	if ps.fcmClient == nil {
		return fmt.Errorf("FCM client not initialized")
	}

	// For single message
	if len(messages) == 1 {
		response, err := ps.fcmClient.Send(ctx, messages[0])
		if err != nil {
			logrus.Errorf("Failed to send FCM message: %v", err)
			return fmt.Errorf("failed to send push notification: %w", err)
		}
		logrus.Infof("Successfully sent FCM message: %s", response)
		return nil
	}

	// For multiple messages
	batchResponse, err := ps.fcmClient.SendAll(ctx, messages)
	if err != nil {
		logrus.Errorf("Failed to send FCM batch: %v", err)
		return fmt.Errorf("failed to send push notifications: %w", err)
	}

	// Log results
	logrus.Infof("Successfully sent %d/%d FCM messages", batchResponse.SuccessCount, len(messages))

	// Handle failures
	if batchResponse.FailureCount > 0 {
		for i, response := range batchResponse.Responses {
			if !response.Success {
				logrus.Errorf("Failed to send to device %s: %v", messages[i].Token, response.Error)

				// Handle invalid tokens
				if messaging.IsRegistrationTokenNotRegistered(response.Error) {
					// Mark device as inactive or delete it
					ps.handleInvalidToken(ctx, messages[i].Token)
				}
			}
		}
	}

	return nil
}

// handleInvalidToken handles invalid or unregistered device tokens
func (ps *PushService) handleInvalidToken(ctx context.Context, token string) {
	device, err := ps.notificationRepo.GetDeviceByToken(ctx, token)
	if err != nil {
		logrus.Errorf("Failed to get device by token: %v", err)
		return
	}

	// Mark device as inactive
	device.IsActive = false
	if err := ps.notificationRepo.UpdatePushDevice(ctx, device); err != nil {
		logrus.Errorf("Failed to mark device as inactive: %v", err)
	}
}

// isQuietHours checks if current time is within quiet hours
func (ps *PushService) isQuietHours(quietHours models.QuietHours) bool {
	if !quietHours.Enabled {
		return false
	}

	// This is a simplified check - you'd want to implement proper timezone handling
	// and day-of-week checking based on the quietHours configuration

	// For now, return false (not in quiet hours)
	// TODO: Implement proper quiet hours logic with timezone support
	return false
}

// SendTestNotification sends a test notification
func (ps *PushService) SendTestNotification(ctx context.Context, userID string, title, message string) error {
	notification := &models.Notification{
		UserID:   userID,
		Title:    title,
		Message:  message,
		Type:     "test",
		Priority: "normal",
	}

	return ps.SendNotification(ctx, notification)
}

// GetDeliveryStatus gets the delivery status of a notification
func (ps *PushService) GetDeliveryStatus(ctx context.Context, notificationID string) (*models.DeliveryHistory, error) {
	// This would require storing delivery attempts in the database
	// For now, return a placeholder
	return &models.DeliveryHistory{
		NotificationID: notificationID,
		Attempts:       []models.DeliveryAttempt{},
		Summary: models.DeliverySummary{
			TotalAttempts:   1,
			SuccessfulCount: 1,
			FailedCount:     0,
			DeliveryRate:    100.0,
		},
	}, nil
}

// ValidateDeviceToken validates a device token format
func (ps *PushService) ValidateDeviceToken(deviceType, token string) error {
	if token == "" {
		return fmt.Errorf("device token cannot be empty")
	}

	switch deviceType {
	case "ios":
		// iOS tokens are 64 hex characters
		if len(token) != 64 {
			return fmt.Errorf("invalid iOS device token length")
		}
	case "android":
		// Android tokens are variable length but typically much longer
		if len(token) < 100 {
			return fmt.Errorf("invalid Android device token length")
		}
	default:
		return fmt.Errorf("unsupported device type: %s", deviceType)
	}

	return nil
}

// UpdateBadgeCount updates the badge count for a user's devices
func (ps *PushService) UpdateBadgeCount(ctx context.Context, userID string, count int) error {
	devices, err := ps.notificationRepo.GetUserPushDevices(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user devices: %w", err)
	}

	for _, device := range devices {
		if device.DeviceType == "ios" {
			// Send a silent push to update badge count
			message := &messaging.Message{
				Token: device.DeviceToken,
				APNS: &messaging.APNSConfig{
					Headers: map[string]string{
						"apns-priority":  "5",
						"apns-push-type": "background",
					},
					Payload: &messaging.APNSPayload{
						Aps: &messaging.Aps{
							Badge:            &count,
							ContentAvailable: true,
						},
					},
				},
			}

			if ps.fcmClient != nil {
				_, err := ps.fcmClient.Send(ctx, message)
				if err != nil {
					logrus.Errorf("Failed to update badge count for device %s: %v", device.ID.Hex(), err)
				}
			}
		}
	}

	return nil
}
