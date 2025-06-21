// services/email_service.go - Extended with auth methods
package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/sirupsen/logrus"
)

// EmailService interface - keeping your existing interface
type EmailService interface {
	SendEmail(data EmailData) error
	// Adding auth-specific methods
	SendVerificationEmail(email, firstName, token string) error
	SendPasswordResetEmail(email, firstName, token string) error
	SendWelcomeEmail(email, firstName string) error
	Send2FADisabledEmail(email, firstName string) error
	SendPasswordChangedEmail(email, firstName string) error
}

// EmailData structure for email content - keeping your existing structure
type EmailData struct {
	To       string                 `json:"to"`
	Subject  string                 `json:"subject"`
	Template string                 `json:"template"`
	Data     map[string]interface{} `json:"data"`
}

// SMTPEmailService implements EmailService using SMTP - extending your existing service
type SMTPEmailService struct {
	host     string
	port     string
	username string
	password string
	from     string
	baseURL  string // Adding base URL for auth links
}

// NewSMTPEmailService creates a new SMTP email service - updated constructor
func NewSMTPEmailService(host, port, username, password, from, baseURL string) EmailService {
	return &SMTPEmailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		baseURL:  baseURL,
	}
}

// SendEmail sends an email using SMTP - keeping your existing method
func (es *SMTPEmailService) SendEmail(data EmailData) error {
	// Build email content
	htmlBody, err := es.buildHTMLTemplate(data.Template, data.Data)
	if err != nil {
		logrus.Errorf("Failed to build email template: %v", err)
		return err
	}

	textBody := es.buildTextVersion(data.Template, data.Data)

	// Create email message
	message := es.buildMessage(data.To, data.Subject, htmlBody, textBody)

	// Send email
	auth := smtp.PlainAuth("", es.username, es.password, es.host)
	addr := fmt.Sprintf("%s:%s", es.host, es.port)

	err = smtp.SendMail(addr, auth, es.from, []string{data.To}, []byte(message))
	if err != nil {
		logrus.Errorf("Failed to send email to %s: %v", data.To, err)
		return err
	}

	logrus.Infof("Email sent successfully to %s", data.To)
	return nil
}

// ============== NEW AUTH-SPECIFIC METHODS ==============

// SendVerificationEmail sends email verification email
func (es *SMTPEmailService) SendVerificationEmail(email, firstName, token string) error {
	verificationURL := fmt.Sprintf("%s/auth/verify-email?token=%s", es.baseURL, token)

	return es.SendEmail(EmailData{
		To:       email,
		Subject:  "Verify Your Email Address - FTrack",
		Template: "email_verification",
		Data: map[string]interface{}{
			"Name":            firstName,
			"VerificationURL": verificationURL,
			"Token":           token,
		},
	})
}

// SendPasswordResetEmail sends password reset email
func (es *SMTPEmailService) SendPasswordResetEmail(email, firstName, token string) error {
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", es.baseURL, token)

	return es.SendEmail(EmailData{
		To:       email,
		Subject:  "Reset Your Password - FTrack",
		Template: "password_reset",
		Data: map[string]interface{}{
			"Name":     firstName,
			"ResetURL": resetURL,
			"Token":    token,
		},
	})
}

// SendWelcomeEmail sends welcome email after successful verification
func (es *SMTPEmailService) SendWelcomeEmail(email, firstName string) error {
	return es.SendEmail(EmailData{
		To:       email,
		Subject:  "Welcome to FTrack! 🎉",
		Template: "welcome",
		Data: map[string]interface{}{
			"Name": firstName,
		},
	})
}

// Send2FADisabledEmail sends notification when 2FA is disabled
func (es *SMTPEmailService) Send2FADisabledEmail(email, firstName string) error {
	return es.SendEmail(EmailData{
		To:       email,
		Subject:  "Two-Factor Authentication Disabled - FTrack",
		Template: "2fa_disabled",
		Data: map[string]interface{}{
			"Name": firstName,
		},
	})
}

// SendPasswordChangedEmail sends notification when password is changed
func (es *SMTPEmailService) SendPasswordChangedEmail(email, firstName string) error {
	return es.SendEmail(EmailData{
		To:       email,
		Subject:  "Password Changed - FTrack",
		Template: "password_changed",
		Data: map[string]interface{}{
			"Name": firstName,
		},
	})
}

// buildHTMLTemplate builds HTML email content from template - extending your existing method
func (es *SMTPEmailService) buildHTMLTemplate(templateName string, data map[string]interface{}) (string, error) {
	// Your existing templates plus new auth templates
	templates := map[string]string{
		// Keep your existing password_reset template
		"password_reset": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #007bff; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
        .button { display: inline-block; padding: 12px 24px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Reset Request</h1>
        </div>
        <div class="content">
            <p>Hi {{.Name}},</p>
            <p>You requested to reset your password for your FTrack account.</p>
            <p>Click the button below to reset your password:</p>
            <p><a href="{{.ResetURL}}" class="button">Reset Password</a></p>
            <p>If the button doesn't work, copy and paste this link into your browser:</p>
            <p>{{.ResetURL}}</p>
            <p>This link will expire in 1 hour.</p>
            <p>If you didn't request this password reset, please ignore this email.</p>
        </div>
        <div class="footer">
            <p>&copy; 2024 FTrack. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		// New email verification template
		"email_verification": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Email Verification</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #28a745; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
        .button { display: inline-block; padding: 12px 24px; background: #28a745; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to FTrack!</h1>
        </div>
        <div class="content">
            <p>Hi {{.Name}},</p>
            <p>Thank you for creating an account with FTrack. Please verify your email address by clicking the link below:</p>
            <p><a href="{{.VerificationURL}}" class="button">Verify Email</a></p>
            <p>If the button doesn't work, copy and paste this link into your browser:</p>
            <p>{{.VerificationURL}}</p>
            <p>This link will expire in 24 hours.</p>
            <p>If you didn't create this account, please ignore this email.</p>
        </div>
        <div class="footer">
            <p>&copy; 2024 FTrack. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		// Welcome email template
		"welcome": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to FTrack</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #6f42c1; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
        .feature { padding: 10px 0; border-bottom: 1px solid #dee2e6; }
        .footer { padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 Welcome to FTrack!</h1>
        </div>
        <div class="content">
            <p>Hi {{.Name}},</p>
            <p>Your email has been verified and you're all set to start using FTrack!</p>
            <h3>Getting Started:</h3>
            <div class="feature">📱 Download our mobile app for the best experience</div>
            <div class="feature">👨‍👩‍👧‍👦 Create or join your first family circle</div>
            <div class="feature">🌍 Set up your location sharing preferences</div>
            <div class="feature">🚨 Add emergency contacts for safety</div>
            <div class="feature">📍 Create important places with geofences</div>
            <p>If you have any questions, feel free to reach out to our support team.</p>
        </div>
        <div class="footer">
            <p>&copy; 2024 FTrack. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		// 2FA disabled notification
		"2fa_disabled": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Two-Factor Authentication Disabled</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #dc3545; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
        .warning { background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; margin: 15px 0; border-radius: 4px; }
        .footer { padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔒 Security Alert</h1>
        </div>
        <div class="content">
            <p>Hi {{.Name}},</p>
            <p>Two-factor authentication has been <strong>disabled</strong> on your FTrack account.</p>
            <div class="warning">
                <strong>⚠️ Security Notice:</strong> If you didn't make this change, please contact our support team immediately and consider changing your password.
            </div>
            <p>For your security, we recommend keeping two-factor authentication enabled.</p>
            <p>You can re-enable 2FA anytime in your account security settings.</p>
        </div>
        <div class="footer">
            <p>&copy; 2024 FTrack. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		// Password changed notification
		"password_changed": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Changed</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #28a745; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
        .warning { background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; margin: 15px 0; border-radius: 4px; }
        .footer { padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>✅ Password Changed</h1>
        </div>
        <div class="content">
            <p>Hi {{.Name}},</p>
            <p>Your FTrack account password has been successfully changed.</p>
            <div class="warning">
                <strong>⚠️ Security Notice:</strong> If you didn't make this change, please contact our support team immediately.
            </div>
            <p>For your security:</p>
            <ul>
                <li>Your password is encrypted and secure</li>
                <li>All active sessions have been logged out</li>
                <li>Consider enabling two-factor authentication</li>
            </ul>
        </div>
        <div class="footer">
            <p>&copy; 2024 FTrack. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		// Keep your existing templates...
		"verification": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Email Verification</title>
</head>
<body>
    <h2>Email Verification</h2>
    <p>Hi {{.Name}},</p>
    <p>Please click the link below to verify your email:</p>
    <p><a href="{{.Link}}">Verify Email</a></p>
    <p>If you can't click the link, copy and paste this URL into your browser:</p>
    <p>{{.Link}}</p>
    <p>This link will expire in 24 hours.</p>
    <p>Best regards,<br>FTrack Team</p>
</body>
</html>`,

		"welcome_user": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to FTrack</title>
</head>
<body>
    <h2>Welcome to FTrack, {{.Name}}!</h2>
    <p>Your account has been successfully created and verified.</p>
    <p>You can now start using all features of FTrack:</p>
    <ul>
        <li>Create family circles</li>
        <li>Track location safely</li>
        <li>Send emergency alerts</li>
        <li>Stay connected with family</li>
    </ul>
    <p>Best regards,<br>FTrack Team</p>
</body>
</html>`,
	}

	tmplStr, exists := templates[templateName]
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	tmpl, err := template.New(templateName).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// buildTextVersion builds text version of email - keeping your existing method and extending it
func (es *SMTPEmailService) buildTextVersion(templateName string, data map[string]interface{}) string {
	name, ok := data["Name"].(string)
	if !ok {
		name = "User"
	}

	switch templateName {
	case "email_verification":
		verificationURL, _ := data["VerificationURL"].(string)
		return fmt.Sprintf(`Hi %s,

Thank you for creating an account with FTrack. Please verify your email address by visiting this link:

%s

This link will expire in 24 hours.

If you didn't create this account, please ignore this email.

© 2024 FTrack. All rights reserved.`, name, verificationURL)

	case "password_reset":
		resetURL, _ := data["ResetURL"].(string)
		return fmt.Sprintf(`Hi %s,

You requested to reset your password for your FTrack account.

Reset your password by visiting this link:

%s

This link will expire in 1 hour.

If you didn't request this password reset, please ignore this email.

© 2024 FTrack. All rights reserved.`, name, resetURL)

	case "welcome":
		return fmt.Sprintf(`Hi %s,

Welcome to FTrack! Your email has been verified and you're all set to start using FTrack.

Getting Started:
- Download our mobile app for the best experience
- Create or join your first family circle
- Set up your location sharing preferences
- Add emergency contacts for safety
- Create important places with geofences

If you have any questions, feel free to reach out to our support team.

© 2024 FTrack. All rights reserved.`, name)

	case "2fa_disabled":
		return fmt.Sprintf(`Hi %s,

Two-factor authentication has been disabled on your FTrack account.

If you didn't make this change, please contact our support team immediately and consider changing your password.

For your security, we recommend keeping two-factor authentication enabled.

© 2024 FTrack. All rights reserved.`, name)

	case "password_changed":
		return fmt.Sprintf(`Hi %s,

Your FTrack account password has been successfully changed.

If you didn't make this change, please contact our support team immediately.

For your security:
- Your password is encrypted and secure
- All active sessions have been logged out
- Consider enabling two-factor authentication

© 2024 FTrack. All rights reserved.`, name)

	// Keep your existing cases...
	case "verification":
		link, _ := data["Link"].(string)
		return fmt.Sprintf(`Hi %s,

Please visit this link to verify your email:

%s

This link will expire in 24 hours.

Best regards,
FTrack Team`, name, link)

	case "welcome_user":
		return fmt.Sprintf(`Hi %s,

Welcome to FTrack!

You can now enjoy all the features of our family tracking app.

Thanks for joining us!

© 2024 FTrack. All rights reserved.`, name)

	case "password_reset_success":
		return fmt.Sprintf(`Hi %s,

Your password has been successfully reset.

If you didn't make this change, please contact our support team immediately.

© 2024 FTrack. All rights reserved.`, name)

	default:
		return "Email notification from FTrack"
	}
}

// buildMessage creates the full email message - keeping your existing method
func (es *SMTPEmailService) buildMessage(to, subject, htmlBody, textBody string) string {
	boundary := "boundary-ftrack-email"

	message := fmt.Sprintf(`From: %s
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="%s"

--%s
Content-Type: text/plain; charset=UTF-8

%s

--%s
Content-Type: text/html; charset=UTF-8

%s

--%s--`, es.from, to, subject, boundary, boundary, textBody, boundary, htmlBody, boundary)

	return message
}

// MockEmailService for testing/development - keeping your existing mock service
type MockEmailService struct{}

func NewMockEmailService() EmailService {
	return &MockEmailService{}
}

func (es *MockEmailService) SendEmail(data EmailData) error {
	logrus.Infof("[MOCK EMAIL] To: %s, Subject: %s, Template: %s",
		data.To, data.Subject, data.Template)

	// Log template data for debugging
	if data.Data != nil {
		for key, value := range data.Data {
			logrus.Infof("[MOCK EMAIL] Data - %s: %v", key, value)
		}
	}

	return nil
}

// Mock implementations for auth methods
func (es *MockEmailService) SendVerificationEmail(email, firstName, token string) error {
	logrus.Infof("[MOCK EMAIL] Verification email to %s for %s with token %s", email, firstName, token)
	return nil
}

func (es *MockEmailService) SendPasswordResetEmail(email, firstName, token string) error {
	logrus.Infof("[MOCK EMAIL] Password reset email to %s for %s with token %s", email, firstName, token)
	return nil
}

func (es *MockEmailService) SendWelcomeEmail(email, firstName string) error {
	logrus.Infof("[MOCK EMAIL] Welcome email to %s for %s", email, firstName)
	return nil
}

func (es *MockEmailService) Send2FADisabledEmail(email, firstName string) error {
	logrus.Infof("[MOCK EMAIL] 2FA disabled email to %s for %s", email, firstName)
	return nil
}

func (es *MockEmailService) SendPasswordChangedEmail(email, firstName string) error {
	logrus.Infof("[MOCK EMAIL] Password changed email to %s for %s", email, firstName)
	return nil
}
