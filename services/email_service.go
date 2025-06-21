package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/sirupsen/logrus"
)

// EmailService interface
type EmailService interface {
	SendEmail(data EmailData) error
}

// EmailData structure for email content
type EmailData struct {
	To       string                 `json:"to"`
	Subject  string                 `json:"subject"`
	Template string                 `json:"template"`
	Data     map[string]interface{} `json:"data"`
}

// SMTPEmailService implements EmailService using SMTP
type SMTPEmailService struct {
	host     string
	port     string
	username string
	password string
	from     string
}

// NewSMTPEmailService creates a new SMTP email service
func NewSMTPEmailService(host, port, username, password, from string) EmailService {
	return &SMTPEmailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

// SendEmail sends an email using SMTP
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

// buildHTMLTemplate builds HTML email content from template
func (es *SMTPEmailService) buildHTMLTemplate(templateName string, data map[string]interface{}) (string, error) {
	templates := map[string]string{
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
            <p>You requested to reset your password for your FTrack account. Click the button below to reset your password:</p>
            <a href="{{.ResetURL}}" class="button">Reset Password</a>
            <p>This link will expire in {{.ExpiresIn}}.</p>
            <p>If you didn't request this, please ignore this email.</p>
        </div>
        <div class="footer">
            <p>© 2024 FTrack. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		"email_verification": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verify Your Email</title>
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
            <h1>Verify Your Email Address</h1>
        </div>
        <div class="content">
            <p>Hi {{.Name}},</p>
            <p>Thanks for signing up for FTrack! Please verify your email address by clicking the button below:</p>
            <a href="{{.VerificationURL}}" class="button">Verify Email</a>
            <p>This link will expire in {{.ExpiresIn}}.</p>
            <p>If you didn't create an account, please ignore this email.</p>
        </div>
        <div class="footer">
            <p>© 2024 FTrack. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		"email_verified": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to FTrack</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #28a745; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
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
            <p>Your email has been successfully verified! Welcome to FTrack.</p>
            <p>You can now enjoy all the features of our family tracking app:</p>
            <ul>
                <li>Real-time location sharing</li>
                <li>Family circles and groups</li>
                <li>Emergency alerts</li>
                <li>Places and geofences</li>
            </ul>
            <p>Thanks for joining us!</p>
        </div>
        <div class="footer">
            <p>© 2024 FTrack. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		"password_reset_success": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset Successful</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #28a745; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
        .footer { padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Reset Successful</h1>
        </div>
        <div class="content">
            <p>Hi {{.Name}},</p>
            <p>Your password has been successfully reset.</p>
            <p>If you didn't make this change, please contact our support team immediately.</p>
        </div>
        <div class="footer">
            <p>© 2024 FTrack. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		"password_changed": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Changed</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #17a2b8; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
        .footer { padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Changed</h1>
        </div>
        <div class="content">
            <p>Hi {{.Name}},</p>
            <p>Your account password has been successfully changed.</p>
            <p>If you didn't make this change, please contact our support team immediately.</p>
        </div>
        <div class="footer">
            <p>© 2024 FTrack. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,
	}

	templateStr, exists := templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	tmpl, err := template.New(templateName).Parse(templateStr)
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

// buildTextVersion creates a plain text version of the email
func (es *SMTPEmailService) buildTextVersion(templateName string, data map[string]interface{}) string {
	name := "User"
	if n, ok := data["Name"].(string); ok {
		name = n
	}

	switch templateName {
	case "password_reset":
		resetURL := data["ResetURL"].(string)
		expiresIn := data["ExpiresIn"].(string)
		return fmt.Sprintf(`Hi %s,

You requested to reset your password for your FTrack account.

Reset your password: %s

This link will expire in %s.

If you didn't request this, please ignore this email.

© 2024 FTrack. All rights reserved.`, name, resetURL, expiresIn)

	case "email_verification":
		verificationURL := data["VerificationURL"].(string)
		expiresIn := data["ExpiresIn"].(string)
		return fmt.Sprintf(`Hi %s,

Thanks for signing up for FTrack! Please verify your email address:

%s

This link will expire in %s.

If you didn't create an account, please ignore this email.

© 2024 FTrack. All rights reserved.`, name, verificationURL, expiresIn)

	case "email_verified":
		return fmt.Sprintf(`Hi %s,

Your email has been successfully verified! Welcome to FTrack.

You can now enjoy all the features of our family tracking app.

Thanks for joining us!

© 2024 FTrack. All rights reserved.`, name)

	case "password_reset_success":
		return fmt.Sprintf(`Hi %s,

Your password has been successfully reset.

If you didn't make this change, please contact our support team immediately.

© 2024 FTrack. All rights reserved.`, name)

	case "password_changed":
		return fmt.Sprintf(`Hi %s,

Your account password has been successfully changed.

If you didn't make this change, please contact our support team immediately.

© 2024 FTrack. All rights reserved.`, name)

	default:
		return "Email notification from FTrack"
	}
}

// buildMessage creates the full email message
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

// MockEmailService for testing/development
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
