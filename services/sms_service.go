// services/sms_service.go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"ftrack/models"
	"ftrack/repositories"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type SMSService struct {
	twilioAccountSID  string
	twilioAuthToken   string
	twilioPhoneNumber string
	notificationRepo  *repositories.NotificationRepository
	httpClient        *http.Client
}

func NewSMSService(
	twilioAccountSID, twilioAuthToken, twilioPhoneNumber string,
	notificationRepo *repositories.NotificationRepository,
) *SMSService {
	return &SMSService{
		twilioAccountSID:  twilioAccountSID,
		twilioAuthToken:   twilioAuthToken,
		twilioPhoneNumber: twilioPhoneNumber,
		notificationRepo:  notificationRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendNotification sends an SMS notification
func (ss *SMSService) SendNotification(ctx context.Context, notification *models.Notification) error {
	// Get user's SMS settings
	smsSettings, err := ss.notificationRepo.GetSMSSettings(ctx, notification.UserID)
	if err != nil {
		logrus.Warnf("Failed to get SMS settings for user %s: %v", notification.UserID, err)
		return nil // Don't fail the entire notification
	}

	// Check if SMS notifications are enabled
	if !smsSettings.Enabled {
		logrus.Infof("SMS notifications disabled for user %s", notification.UserID)
		return nil
	}

	// Check if phone number is verified
	if !smsSettings.IsVerified {
		logrus.Infof("Phone number not verified for user %s", notification.UserID)
		return nil
	}

	// Check type-specific settings
	if typeEnabled, exists := smsSettings.TypeSettings[notification.Type]; exists && !typeEnabled {
		logrus.Infof("SMS notifications disabled for type %s for user %s", notification.Type, notification.UserID)
		return nil
	}

	// Check usage limits
	if err := ss.checkUsageLimits(ctx, notification.UserID, smsSettings); err != nil {
		logrus.Warnf("SMS usage limit exceeded for user %s: %v", notification.UserID, err)
		return nil
	}

	// Format SMS content
	smsContent := ss.formatSMSContent(notification)

	// Send SMS
	err = ss.sendSMS(smsSettings.PhoneNumber, smsContent)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	// Update usage tracking
	ss.updateUsageTracking(ctx, notification.UserID)

	logrus.Infof("Successfully sent SMS to user %s", notification.UserID)
	return nil
}

// SendTestSMS sends a test SMS
func (ss *SMSService) SendTestSMS(ctx context.Context, phoneNumber, message string) error {
	if ss.twilioAccountSID == "" {
		return fmt.Errorf("Twilio not configured")
	}

	return ss.sendSMS(phoneNumber, fmt.Sprintf("Test SMS: %s", message))
}

// SendVerificationSMS sends an SMS verification code
func (ss *SMSService) SendVerificationSMS(ctx context.Context, phoneNumber, verificationCode string) error {
	message := fmt.Sprintf("Your Family Tracker verification code is: %s. This code expires in 10 minutes.", verificationCode)
	return ss.sendSMS(phoneNumber, message)
}

// checkUsageLimits checks if the user has exceeded their SMS limits
func (ss *SMSService) checkUsageLimits(ctx context.Context, userID string, settings *models.SMSSettings) error {
	usage, err := ss.getSMSUsage(ctx, userID)
	if err != nil {
		// If we can't get usage, allow the SMS (graceful degradation)
		logrus.Warnf("Failed to get SMS usage for user %s: %v", userID, err)
		return nil
	}

	// Check daily limit
	if settings.DailyLimit > 0 && usage.DailyTotal >= settings.DailyLimit {
		return fmt.Errorf("daily SMS limit exceeded (%d/%d)", usage.DailyTotal, settings.DailyLimit)
	}

	// Check monthly limit
	if settings.MonthlyLimit > 0 && usage.MonthlyTotal >= settings.MonthlyLimit {
		return fmt.Errorf("monthly SMS limit exceeded (%d/%d)", usage.MonthlyTotal, settings.MonthlyLimit)
	}

	return nil
}

// formatSMSContent formats the notification content for SMS
func (ss *SMSService) formatSMSContent(notification *models.Notification) string {
	content := fmt.Sprintf("%s: %s", notification.Title, notification.Message)

	// Add priority indicator for high/urgent notifications
	if notification.Priority == "high" || notification.Priority == "urgent" {
		content = fmt.Sprintf("🚨 %s", content)
	}

	// Truncate if too long (SMS limit is 160 characters for single message)
	if len(content) > 150 {
		content = content[:147] + "..."
	}

	// Add app signature
	content += " - Family Tracker"

	return content
}

// sendSMS sends an SMS using Twilio API
func (ss *SMSService) sendSMS(phoneNumber, message string) error {
	if ss.twilioAccountSID == "" {
		logrus.Warn("Twilio not configured, skipping SMS send")
		return nil
	}

	// Twilio API endpoint
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", ss.twilioAccountSID)

	// Prepare request data
	data := url.Values{}
	data.Set("From", ss.twilioPhoneNumber)
	data.Set("To", phoneNumber)
	data.Set("Body", message)

	// Create request
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(ss.twilioAccountSID, ss.twilioAuthToken)

	// Send request
	resp, err := ss.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logrus.Errorf("Twilio API error: %s", string(body))
		return fmt.Errorf("SMS API error: %s", resp.Status)
	}

	// Parse response
	var twilioResponse struct {
		SID          string      `json:"sid"`
		Status       string      `json:"status"`
		ErrorCode    interface{} `json:"error_code"`
		ErrorMessage string      `json:"error_message"`
	}

	if err := json.Unmarshal(body, &twilioResponse); err != nil {
		logrus.Warnf("Failed to parse Twilio response: %v", err)
		// Don't fail if we can't parse response - SMS might still have been sent
		return nil
	}

	// Check for errors in response
	if twilioResponse.ErrorCode != nil {
		return fmt.Errorf("SMS error: %s", twilioResponse.ErrorMessage)
	}

	logrus.Infof("SMS sent successfully - SID: %s, Status: %s", twilioResponse.SID, twilioResponse.Status)
	return nil
}

// getSMSUsage gets the current SMS usage for a user
func (ss *SMSService) getSMSUsage(ctx context.Context, userID string) (*models.SMSUsage, error) {
	// This would typically query a usage tracking collection
	// For now, return mock data
	now := time.Now()

	return &models.SMSUsage{
		UserID:           userID,
		Date:             now,
		Count:            1,
		DailyTotal:       5,   // Mock daily usage
		MonthlyTotal:     45,  // Mock monthly usage
		DailyLimit:       50,  // Default daily limit
		MonthlyLimit:     500, // Default monthly limit
		RemainingDaily:   45,
		RemainingMonthly: 455,
	}, nil
}

// updateUsageTracking updates the SMS usage tracking
func (ss *SMSService) updateUsageTracking(ctx context.Context, userID string) {
	// This would typically update a usage tracking collection
	// For now, just log the usage
	logrus.Infof("SMS usage updated for user %s", userID)
}

// ValidatePhoneNumber validates a phone number format
func (ss *SMSService) ValidatePhoneNumber(phoneNumber string) error {
	if phoneNumber == "" {
		return fmt.Errorf("phone number cannot be empty")
	}

	// Remove common formatting characters
	cleaned := strings.ReplaceAll(phoneNumber, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")

	// Check if it starts with + (international format)
	if !strings.HasPrefix(cleaned, "+") {
		return fmt.Errorf("phone number must be in international format (+1234567890)")
	}

	// Basic length check (international numbers are typically 10-15 digits)
	if len(cleaned) < 10 || len(cleaned) > 16 {
		return fmt.Errorf("invalid phone number length")
	}

	// Check if remaining characters are digits
	for _, char := range cleaned[1:] {
		if char < '0' || char > '9' {
			return fmt.Errorf("phone number contains invalid characters")
		}
	}

	return nil
}

// GetSMSDeliveryStatus gets the delivery status of an SMS
func (ss *SMSService) GetSMSDeliveryStatus(ctx context.Context, notificationID string) (*models.DeliveryHistory, error) {
	// This would require storing SMS SIDs and querying Twilio's status API
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

// GetUsageStatistics gets SMS usage statistics for a user
func (ss *SMSService) GetUsageStatistics(ctx context.Context, userID string, days int) (*models.SMSUsage, error) {
	// This would query usage tracking data over the specified period
	return ss.getSMSUsage(ctx, userID)
}

// UpdateSMSSettings updates SMS settings for a user
func (ss *SMSService) UpdateSMSSettings(ctx context.Context, userID string, settings *models.SMSSettings) error {
	// Validate phone number if provided
	if settings.PhoneNumber != "" {
		if err := ss.ValidatePhoneNumber(settings.PhoneNumber); err != nil {
			return fmt.Errorf("invalid phone number: %w", err)
		}
	}

	// Update settings in repository
	return ss.notificationRepo.UpdateSMSSettings(ctx, settings)
}

// SendBulkSMS sends SMS to multiple recipients (admin feature)
func (ss *SMSService) SendBulkSMS(ctx context.Context, phoneNumbers []string, message string) error {
	if ss.twilioAccountSID == "" {
		return fmt.Errorf("Twilio not configured")
	}

	var errors []string
	successCount := 0

	for _, phoneNumber := range phoneNumbers {
		if err := ss.sendSMS(phoneNumber, message); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", phoneNumber, err))
			logrus.Errorf("Failed to send SMS to %s: %v", phoneNumber, err)
		} else {
			successCount++
		}
	}

	logrus.Infof("Bulk SMS completed: %d/%d successful", successCount, len(phoneNumbers))

	if len(errors) > 0 {
		return fmt.Errorf("some SMS messages failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// CheckTwilioWebhook processes Twilio delivery status webhooks
func (ss *SMSService) CheckTwilioWebhook(ctx context.Context, webhookData map[string]string) error {
	// Extract webhook data
	messageSID := webhookData["MessageSid"]
	messageStatus := webhookData["MessageStatus"]
	errorCode := webhookData["ErrorCode"]

	logrus.Infof("Received Twilio webhook - SID: %s, Status: %s", messageSID, messageStatus)

	// Handle different status types
	switch messageStatus {
	case "delivered":
		logrus.Infof("SMS delivered successfully: %s", messageSID)
	case "failed", "undelivered":
		logrus.Errorf("SMS delivery failed: %s, Error: %s", messageSID, errorCode)
	case "sent", "queued":
		logrus.Infof("SMS in transit: %s, Status: %s", messageSID, messageStatus)
	}

	// You would typically update delivery status in database here
	return nil
}

// EstimateSMSCost estimates the cost of sending an SMS
func (ss *SMSService) EstimateSMSCost(message string, phoneNumber string) (float64, error) {
	// Basic cost estimation (Twilio pricing varies by destination)
	messageLength := len(message)
	segments := (messageLength + 159) / 160 // SMS segments

	// Basic US pricing (you'd want to implement proper pricing logic)
	costPerSegment := 0.0075 // $0.0075 per segment for US numbers

	return float64(segments) * costPerSegment, nil
}

// FormatPhoneNumber formats a phone number to international format
func (ss *SMSService) FormatPhoneNumber(phoneNumber, countryCode string) string {
	// Remove all non-digit characters
	cleaned := ""
	for _, char := range phoneNumber {
		if char >= '0' && char <= '9' {
			cleaned += string(char)
		}
	}

	// Add country code if not present
	if !strings.HasPrefix(phoneNumber, "+") {
		if countryCode == "" {
			countryCode = "1" // Default to US
		}
		cleaned = "+" + countryCode + cleaned
	}

	return cleaned
}
