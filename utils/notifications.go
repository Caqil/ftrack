package utils

import (
	"context"
	"fmt"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"google.golang.org/api/option"
)

type NotificationService struct {
	fcmClient    *messaging.Client
	twilioClient *twilio.RestClient
	twilioNumber string
	emailService EmailService
}

type PushNotification struct {
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Data     map[string]string `json:"data"`
	ImageURL string            `json:"imageUrl,omitempty"`
	Sound    string            `json:"sound,omitempty"`
	Badge    int               `json:"badge,omitempty"`
}

type SMSMessage struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

type EmailMessage struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	IsHTML  bool   `json:"isHtml"`
}

type NotificationResult struct {
	Success   bool   `json:"success"`
	MessageID string `json:"messageId,omitempty"`
	Error     string `json:"error,omitempty"`
}

func NewNotificationService(firebaseCredentials, twilioSID, twilioToken, twilioNumber string) (*NotificationService, error) {
	// Initialize Firebase
	opt := option.WithCredentialsFile(firebaseCredentials)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase: %v", err)
	}

	fcmClient, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize FCM client: %v", err)
	}

	// Initialize Twilio
	twilioClient := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: twilioSID,
		Password: twilioToken,
	})

	return &NotificationService{
		fcmClient:    fcmClient,
		twilioClient: twilioClient,
		twilioNumber: twilioNumber,
		emailService: NewEmailService(),
	}, nil
}

// Push Notifications
func (ns *NotificationService) SendPushNotification(ctx context.Context, deviceToken string, notification PushNotification) (*NotificationResult, error) {
	message := &messaging.Message{
		Token: deviceToken,
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Data: notification.Data,
		Android: &messaging.AndroidConfig{
			Notification: &messaging.AndroidNotification{
				Sound: notification.Sound,
				Icon:  "ic_notification",
				Color: "#FF6B35",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: notification.Title,
						Body:  notification.Body,
					},
					Badge: &notification.Badge,
					Sound: notification.Sound,
				},
			},
		},
	}

	response, err := ns.fcmClient.Send(ctx, message)
	if err != nil {
		return &NotificationResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &NotificationResult{
		Success:   true,
		MessageID: response,
	}, nil
}

func (ns *NotificationService) SendPushToMultipleDevices(ctx context.Context, deviceTokens []string, notification PushNotification) ([]*NotificationResult, error) {
	message := &messaging.MulticastMessage{
		Tokens: deviceTokens,
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Body,
		},
		Data: notification.Data,
	}

	response, err := ns.fcmClient.SendMulticast(ctx, message)
	if err != nil {
		return nil, err
	}

	results := make([]*NotificationResult, len(deviceTokens))
	for i, resp := range response.Responses {
		if resp.Success {
			results[i] = &NotificationResult{
				Success:   true,
				MessageID: resp.MessageID,
			}
		} else {
			results[i] = &NotificationResult{
				Success: false,
				Error:   resp.Error.Error(),
			}
		}
	}

	return results, nil
}

// SMS Notifications
func (ns *NotificationService) SendSMS(ctx context.Context, sms SMSMessage) (*NotificationResult, error) {
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(sms.To)
	params.SetFrom(ns.twilioNumber)
	params.SetBody(sms.Message)

	resp, err := ns.twilioClient.Api.CreateMessage(params)
	if err != nil {
		return &NotificationResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &NotificationResult{
		Success:   true,
		MessageID: *resp.Sid,
	}, nil
}

// Email Notifications
func (ns *NotificationService) SendEmail(ctx context.Context, email EmailMessage) (*NotificationResult, error) {
	return ns.emailService.SendEmail(ctx, email)
}

// Notification Templates
func (ns *NotificationService) CreateLocationNotification(userName, placeName, eventType string) PushNotification {
	var title, body string

	switch eventType {
	case "arrival":
		title = "📍 Location Update"
		body = fmt.Sprintf("%s has arrived at %s", userName, placeName)
	case "departure":
		title = "📍 Location Update"
		body = fmt.Sprintf("%s has left %s", userName, placeName)
	default:
		title = "📍 Location Update"
		body = fmt.Sprintf("%s location updated", userName)
	}

	return PushNotification{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"type":      "location_update",
			"userName":  userName,
			"placeName": placeName,
			"eventType": eventType,
		},
		Sound: "default",
	}
}

func (ns *NotificationService) CreateEmergencyNotification(userName, emergencyType string, lat, lon float64) PushNotification {
	var title, body string

	switch emergencyType {
	case "sos":
		title = "🚨 EMERGENCY ALERT"
		body = fmt.Sprintf("%s has sent an SOS alert!", userName)
	case "crash":
		title = "🚗 CRASH DETECTED"
		body = fmt.Sprintf("Crash detected for %s", userName)
	case "help":
		title = "🆘 Help Needed"
		body = fmt.Sprintf("%s needs help", userName)
	default:
		title = "🚨 Emergency Alert"
		body = fmt.Sprintf("Emergency alert from %s", userName)
	}

	return PushNotification{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"type":          "emergency",
			"userName":      userName,
			"emergencyType": emergencyType,
			"latitude":      fmt.Sprintf("%.6f", lat),
			"longitude":     fmt.Sprintf("%.6f", lon),
			"priority":      "high",
		},
		Sound: "emergency",
	}
}

func (ns *NotificationService) CreateDrivingNotification(userName, eventType string, speed float64) PushNotification {
	var title, body string

	switch eventType {
	case "speeding":
		title = "⚠️ Driving Alert"
		body = fmt.Sprintf("%s is driving %.0f km/h", userName, speed)
	case "hard_brake":
		title = "⚠️ Driving Alert"
		body = fmt.Sprintf("%s had a hard braking event", userName)
	case "phone_usage":
		title = "📱 Driving Alert"
		body = fmt.Sprintf("%s is using phone while driving", userName)
	default:
		title = "🚗 Driving Update"
		body = fmt.Sprintf("Driving update for %s", userName)
	}

	return PushNotification{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"type":      "driving_alert",
			"userName":  userName,
			"eventType": eventType,
			"speed":     fmt.Sprintf("%.2f", speed),
		},
		Sound: "default",
	}
}

func (ns *NotificationService) CreateMessageNotification(senderName, circleName, messagePreview string) PushNotification {
	return PushNotification{
		Title: fmt.Sprintf("💬 %s", circleName),
		Body:  fmt.Sprintf("%s: %s", senderName, messagePreview),
		Data: map[string]string{
			"type":       "new_message",
			"senderName": senderName,
			"circleName": circleName,
		},
		Sound: "default",
		Badge: 1,
	}
}

// Batch notifications
func (ns *NotificationService) SendBatchNotifications(ctx context.Context, notifications []BatchNotification) error {
	for _, notif := range notifications {
		switch notif.Type {
		case "push":
			go ns.SendPushNotification(ctx, notif.DeviceToken, notif.Push)
		case "sms":
			go ns.SendSMS(ctx, notif.SMS)
		case "email":
			go ns.SendEmail(ctx, notif.Email)
		}

		// Small delay to avoid rate limiting
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// CreateCircleInviteNotification creates a notification for circle invitations
func (ns *NotificationService) CreateCircleInviteNotification(inviterName, circleName string) PushNotification {
	return PushNotification{
		Title: "Circle Invitation",
		Body:  fmt.Sprintf("%s invited you to join %s", inviterName, circleName),
		Data: map[string]string{
			"type":        "circle_invite",
			"inviterName": inviterName,
			"circleName":  circleName,
		},
		Sound: "default",
	}
}

// CreatePlaceNotification creates a notification for place events
func (ns *NotificationService) CreatePlaceNotification(userName, placeName, eventType string) PushNotification {
	var title, body string

	switch eventType {
	case "arrival":
		title = "📍 Arrival"
		body = fmt.Sprintf("%s arrived at %s", userName, placeName)
	case "departure":
		title = "📍 Departure"
		body = fmt.Sprintf("%s left %s", userName, placeName)
	case "extended_stay":
		title = "⏰ Extended Stay"
		body = fmt.Sprintf("%s has been at %s for a while", userName, placeName)
	default:
		title = "📍 Location Update"
		body = fmt.Sprintf("%s location updated at %s", userName, placeName)
	}

	return PushNotification{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"type":      "place_event",
			"userName":  userName,
			"placeName": placeName,
			"eventType": eventType,
		},
		Sound: "default",
	}
}

// CreateCheckInNotification creates a notification for check-ins
func (ns *NotificationService) CreateCheckInNotification(userName, message string) PushNotification {
	return PushNotification{
		Title: "✅ Check-in",
		Body:  fmt.Sprintf("%s: %s", userName, message),
		Data: map[string]string{
			"type":     "check_in",
			"userName": userName,
			"message":  message,
		},
		Sound: "default",
	}
}

type BatchNotification struct {
	Type        string           `json:"type"` // push, sms, email
	DeviceToken string           `json:"deviceToken,omitempty"`
	Push        PushNotification `json:"push,omitempty"`
	SMS         SMSMessage       `json:"sms,omitempty"`
	Email       EmailMessage     `json:"email,omitempty"`
}

// Email Service interface
type EmailService interface {
	SendEmail(ctx context.Context, email EmailMessage) (*NotificationResult, error)
}
