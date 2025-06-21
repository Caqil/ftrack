// services/sms_service.go - New SMS service that wraps existing notification service
package services

import (
	"context"
	"fmt"
	"ftrack/utils"
	"time"

	"github.com/sirupsen/logrus"
)

// SMSService interface for auth-specific SMS operations
type SMSService interface {
	// Basic SMS sending (wrapping existing functionality)
	SendSMS(ctx context.Context, sms utils.SMSMessage) (*utils.NotificationResult, error)

	// Auth-specific SMS methods
	SendVerificationCode(phone, code string) error
	SendPasswordResetCode(phone, code string) error
	Send2FACode(phone, code string) error
	SendLoginAlert(phone, deviceType, location string) error
	SendPasswordChangedAlert(phone string) error
	SendAccountLockedAlert(phone string) error
	SendEmergencyAlert(phone, message string) error
}

// TwilioSMSService implements SMSService using existing notification service
type TwilioSMSService struct {
	notificationService *utils.NotificationService
	fromNumber          string
	appName             string
}

// NewTwilioSMSService creates a new SMS service using existing notification infrastructure
func NewTwilioSMSService(notificationService *utils.NotificationService, fromNumber, appName string) SMSService {
	return &TwilioSMSService{
		notificationService: notificationService,
		fromNumber:          fromNumber,
		appName:             appName,
	}
}

// SendSMS sends a basic SMS using existing notification service
func (sms *TwilioSMSService) SendSMS(ctx context.Context, message utils.SMSMessage) (*utils.NotificationResult, error) {
	return sms.notificationService.SendSMS(ctx, message)
}

// ============== AUTH-SPECIFIC SMS METHODS ==============

// SendVerificationCode sends SMS with email verification code
func (sms *TwilioSMSService) SendVerificationCode(phone, code string) error {
	message := fmt.Sprintf("Your %s verification code is: %s. This code will expire in 10 minutes. Don't share this code with anyone.", sms.appName, code)

	return sms.sendAuthSMS(phone, message)
}

// SendPasswordResetCode sends SMS with password reset code
func (sms *TwilioSMSService) SendPasswordResetCode(phone, code string) error {
	message := fmt.Sprintf("Your %s password reset code is: %s. This code will expire in 10 minutes. If you didn't request this, please ignore.", sms.appName, code)

	return sms.sendAuthSMS(phone, message)
}

// Send2FACode sends SMS with two-factor authentication code
func (sms *TwilioSMSService) Send2FACode(phone, code string) error {
	message := fmt.Sprintf("Your %s 2FA code is: %s. This code will expire in 5 minutes. Don't share this code.", sms.appName, code)

	return sms.sendAuthSMS(phone, message)
}

// SendLoginAlert sends SMS notification for new login
func (sms *TwilioSMSService) SendLoginAlert(phone, deviceType, location string) error {
	var message string
	if location != "" {
		message = fmt.Sprintf("🔐 New %s login detected on %s from %s. If this wasn't you, secure your account immediately.", sms.appName, deviceType, location)
	} else {
		message = fmt.Sprintf("🔐 New %s login detected on %s. If this wasn't you, secure your account immediately.", sms.appName, deviceType)
	}

	return sms.sendSecuritySMS(phone, message)
}

// SendPasswordChangedAlert sends SMS notification when password is changed
func (sms *TwilioSMSService) SendPasswordChangedAlert(phone string) error {
	message := fmt.Sprintf("✅ Your %s password has been successfully changed. If you didn't make this change, contact support immediately.", sms.appName)

	return sms.sendSecuritySMS(phone, message)
}

// SendAccountLockedAlert sends SMS when account is locked
func (sms *TwilioSMSService) SendAccountLockedAlert(phone string) error {
	message := fmt.Sprintf("🔒 Your %s account has been temporarily locked due to multiple failed login attempts. Try again in 15 minutes or contact support.", sms.appName)

	return sms.sendSecuritySMS(phone, message)
}

// SendEmergencyAlert sends emergency SMS (high priority)
func (sms *TwilioSMSService) SendEmergencyAlert(phone, alertMessage string) error {
	message := fmt.Sprintf("🚨 EMERGENCY ALERT from %s: %s", sms.appName, alertMessage)

	return sms.sendEmergencySMS(phone, message)
}

// ============== HELPER METHODS ==============

// sendAuthSMS sends authentication-related SMS
func (sms *TwilioSMSService) sendAuthSMS(phone, message string) error {
	ctx := context.Background()

	smsMessage := utils.SMSMessage{
		To:      phone,
		Message: message,
	}

	result, err := sms.notificationService.SendSMS(ctx, smsMessage)
	if err != nil {
		logrus.Errorf("Failed to send auth SMS to %s: %v", phone, err)
		return err
	}

	if !result.Success {
		logrus.Errorf("Auth SMS failed to %s: %s", phone, result.Error)
		return fmt.Errorf("SMS sending failed: %s", result.Error)
	}

	logrus.Infof("Auth SMS sent successfully to %s (MessageID: %s)", phone, result.MessageID)
	return nil
}

// sendSecuritySMS sends security-related SMS notifications
func (sms *TwilioSMSService) sendSecuritySMS(phone, message string) error {
	ctx := context.Background()

	smsMessage := utils.SMSMessage{
		To:      phone,
		Message: message,
	}

	result, err := sms.notificationService.SendSMS(ctx, smsMessage)
	if err != nil {
		logrus.Errorf("Failed to send security SMS to %s: %v", phone, err)
		return err
	}

	if !result.Success {
		logrus.Errorf("Security SMS failed to %s: %s", phone, result.Error)
		return fmt.Errorf("SMS sending failed: %s", result.Error)
	}

	logrus.Infof("Security SMS sent successfully to %s (MessageID: %s)", phone, result.MessageID)
	return nil
}

// sendEmergencySMS sends emergency SMS (highest priority)
func (sms *TwilioSMSService) sendEmergencySMS(phone, message string) error {
	ctx := context.Background()

	smsMessage := utils.SMSMessage{
		To:      phone,
		Message: message,
	}

	// For emergency SMS, we might want to add retry logic or use priority sending
	result, err := sms.notificationService.SendSMS(ctx, smsMessage)
	if err != nil {
		logrus.Errorf("Failed to send emergency SMS to %s: %v", phone, err)

		// For emergency SMS, try once more after a short delay
		time.Sleep(2 * time.Second)
		result, err = sms.notificationService.SendSMS(ctx, smsMessage)
		if err != nil {
			logrus.Errorf("Emergency SMS retry failed to %s: %v", phone, err)
			return err
		}
	}

	if !result.Success {
		logrus.Errorf("Emergency SMS failed to %s: %s", phone, result.Error)
		return fmt.Errorf("Emergency SMS sending failed: %s", result.Error)
	}

	logrus.Infof("🚨 Emergency SMS sent successfully to %s (MessageID: %s)", phone, result.MessageID)
	return nil
}

// ============== MOCK SMS SERVICE FOR DEVELOPMENT ==============

// MockSMSService for testing/development
type MockSMSService struct {
	appName string
}

func NewMockSMSService(appName string) SMSService {
	return &MockSMSService{
		appName: appName,
	}
}

func (sms *MockSMSService) SendSMS(ctx context.Context, message utils.SMSMessage) (*utils.NotificationResult, error) {
	logrus.Infof("[MOCK SMS] To: %s, Message: %s", message.To, message.Message)
	return &utils.NotificationResult{
		Success:   true,
		MessageID: "mock-sms-" + generateMockID(),
	}, nil
}

func (sms *MockSMSService) SendVerificationCode(phone, code string) error {
	logrus.Infof("[MOCK SMS] Verification code to %s: %s", phone, code)
	return nil
}

func (sms *MockSMSService) SendPasswordResetCode(phone, code string) error {
	logrus.Infof("[MOCK SMS] Password reset code to %s: %s", phone, code)
	return nil
}

func (sms *MockSMSService) Send2FACode(phone, code string) error {
	logrus.Infof("[MOCK SMS] 2FA code to %s: %s", phone, code)
	return nil
}

func (sms *MockSMSService) SendLoginAlert(phone, deviceType, location string) error {
	logrus.Infof("[MOCK SMS] Login alert to %s: device=%s, location=%s", phone, deviceType, location)
	return nil
}

func (sms *MockSMSService) SendPasswordChangedAlert(phone string) error {
	logrus.Infof("[MOCK SMS] Password changed alert to %s", phone)
	return nil
}

func (sms *MockSMSService) SendAccountLockedAlert(phone string) error {
	logrus.Infof("[MOCK SMS] Account locked alert to %s", phone)
	return nil
}

func (sms *MockSMSService) SendEmergencyAlert(phone, message string) error {
	logrus.Infof("[MOCK SMS] 🚨 Emergency alert to %s: %s", phone, message)
	return nil
}

// ============== HELPER FUNCTIONS ==============

func generateMockID() string {
	return fmt.Sprintf("mock_%d", time.Now().UnixNano())
}

// ============== SMS TEMPLATE FUNCTIONS ==============

// SMSTemplates provides pre-built SMS message templates
type SMSTemplates struct {
	AppName string
}

func NewSMSTemplates(appName string) *SMSTemplates {
	return &SMSTemplates{AppName: appName}
}

// GetVerificationMessage returns formatted verification SMS
func (t *SMSTemplates) GetVerificationMessage(code string) string {
	return fmt.Sprintf("Your %s verification code is: %s\n\nThis code expires in 10 minutes.\nDon't share this code with anyone.", t.AppName, code)
}

// GetPasswordResetMessage returns formatted password reset SMS
func (t *SMSTemplates) GetPasswordResetMessage(code string) string {
	return fmt.Sprintf("Your %s password reset code is: %s\n\nThis code expires in 10 minutes.\nIf you didn't request this, please ignore.", t.AppName, code)
}

// Get2FAMessage returns formatted 2FA SMS
func (t *SMSTemplates) Get2FAMessage(code string) string {
	return fmt.Sprintf("Your %s 2FA code is: %s\n\nThis code expires in 5 minutes.\nDon't share this code.", t.AppName, code)
}

// GetLoginAlertMessage returns formatted login alert SMS
func (t *SMSTemplates) GetLoginAlertMessage(deviceType, location string) string {
	if location != "" {
		return fmt.Sprintf("🔐 New %s login detected\nDevice: %s\nLocation: %s\n\nIf this wasn't you, secure your account immediately.", t.AppName, deviceType, location)
	}
	return fmt.Sprintf("🔐 New %s login detected on %s\n\nIf this wasn't you, secure your account immediately.", t.AppName, deviceType)
}

// GetPasswordChangedMessage returns formatted password changed SMS
func (t *SMSTemplates) GetPasswordChangedMessage() string {
	return fmt.Sprintf("✅ Your %s password has been successfully changed.\n\nIf you didn't make this change, contact support immediately.", t.AppName)
}

// GetAccountLockedMessage returns formatted account locked SMS
func (t *SMSTemplates) GetAccountLockedMessage() string {
	return fmt.Sprintf("🔒 Your %s account has been temporarily locked due to multiple failed login attempts.\n\nTry again in 15 minutes or contact support.", t.AppName)
}

// GetEmergencyMessage returns formatted emergency SMS
func (t *SMSTemplates) GetEmergencyMessage(alertMessage string) string {
	return fmt.Sprintf("🚨 EMERGENCY ALERT from %s:\n\n%s", t.AppName, alertMessage)
}

// ============== SMS VALIDATION ==============

// ValidatePhoneNumber validates phone number format for SMS
func ValidatePhoneNumber(phone string) bool {
	// Basic validation - should be enhanced based on requirements
	if len(phone) < 10 || len(phone) > 15 {
		return false
	}

	// Must start with + or digit
	if phone[0] != '+' && (phone[0] < '0' || phone[0] > '9') {
		return false
	}

	return true
}

// NormalizePhoneNumber normalizes phone number for SMS sending
func NormalizePhoneNumber(phone string) string {
	// Remove all non-digit characters except +
	normalized := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' || char == '+' {
			normalized += string(char)
		}
	}

	// Add + if not present and looks like international number
	if len(normalized) > 10 && normalized[0] != '+' {
		normalized = "+" + normalized
	}

	return normalized
}

// ============== SMS RATE LIMITING ==============

// SMSRateLimiter provides rate limiting for SMS sending
type SMSRateLimiter struct {
	// This could be implemented with Redis or in-memory store
	// For now, it's a placeholder
}

func NewSMSRateLimiter() *SMSRateLimiter {
	return &SMSRateLimiter{}
}

func (rl *SMSRateLimiter) CanSendSMS(phone string) bool {
	// Implementation would check rate limits
	// For now, always allow
	return true
}

func (rl *SMSRateLimiter) RecordSMS(phone string) {
	// Implementation would record SMS sent for rate limiting
	logrus.Debugf("SMS rate limit recorded for %s", phone)
}
