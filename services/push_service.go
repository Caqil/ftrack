package services

import (
	"context"
	"ftrack/utils"
)

type PushService struct {
	notificationService *utils.NotificationService
}

func NewPushService(firebaseCredentials, twilioSID, twilioToken, twilioNumber string) (*PushService, error) {
	notificationService, err := utils.NewNotificationService(firebaseCredentials, twilioSID, twilioToken, twilioNumber)
	if err != nil {
		return nil, err
	}

	return &PushService{
		notificationService: notificationService,
	}, nil
}

func (ps *PushService) SendPushNotification(ctx context.Context, deviceToken string, notification utils.PushNotification) (*utils.NotificationResult, error) {
	return ps.notificationService.SendPushNotification(ctx, deviceToken, notification)
}

func (ps *PushService) SendPushToMultipleDevices(ctx context.Context, deviceTokens []string, notification utils.PushNotification) ([]*utils.NotificationResult, error) {
	return ps.notificationService.SendPushToMultipleDevices(ctx, deviceTokens, notification)
}

func (ps *PushService) SendSMS(ctx context.Context, sms utils.SMSMessage) (*utils.NotificationResult, error) {
	return ps.notificationService.SendSMS(ctx, sms)
}

func (ps *PushService) SendEmail(ctx context.Context, email utils.EmailMessage) (*utils.NotificationResult, error) {
	return ps.notificationService.SendEmail(ctx, email)
}

func (ps *PushService) SendBatchNotifications(ctx context.Context, notifications []utils.BatchNotification) error {
	return ps.notificationService.SendBatchNotifications(ctx, notifications)
}
