// models/auth.go - Auth-related models
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ============== AUTH REQUESTS ==============

type LoginRequest struct {
	Email         string `json:"email" validate:"required,email"`
	Password      string `json:"password" validate:"required,min=6"`
	TwoFactorCode string `json:"twoFactorCode,omitempty"`
	DeviceType    string `json:"deviceType,omitempty"`
	IPAddress     string `json:"ipAddress,omitempty"`
	UserAgent     string `json:"userAgent,omitempty"`
	RememberMe    bool   `json:"rememberMe,omitempty"`
}

type RegisterRequest struct {
	Email            string           `json:"email" validate:"required,email"`
	Phone            string           `json:"phone" validate:"required"`
	Password         string           `json:"password" validate:"required,min=6"`
	FirstName        string           `json:"firstName" validate:"required"`
	LastName         string           `json:"lastName" validate:"required"`
	EmergencyContact EmergencyContact `json:"emergencyContact"`
	AcceptTerms      bool             `json:"acceptTerms" validate:"required"`
	DeviceType       string           `json:"deviceType,omitempty"`
	IPAddress        string           `json:"ipAddress,omitempty"`
	UserAgent        string           `json:"userAgent,omitempty"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=6"`
}

type VerifyEmailRequest struct {
	EmailAddress     string `json:"email_address"`
	VerificationCode string `json:"verification_code"`
}
type Setup2FARequest struct {
	Password string `json:"password" validate:"required"`
}

type Verify2FARequest struct {
	Code   string `json:"code" validate:"required"`
	Secret string `json:"secret" validate:"required"`
}

type Disable2FARequest struct {
	Password string `json:"password" validate:"required"`
	Code     string `json:"code" validate:"required"`
}

// ============== AUTH RESPONSES ==============

type AuthResponse struct {
	User                 User   `json:"user"`
	AccessToken          string `json:"accessToken"`
	RefreshToken         string `json:"refreshToken"`
	TokenType            string `json:"tokenType"`
	ExpiresIn            int64  `json:"expiresIn"`
	RequiresVerification bool   `json:"requiresVerification,omitempty"`
	Requires2FA          bool   `json:"requires2FA,omitempty"`
}

type TokenValidationResponse struct {
	Valid     bool   `json:"valid"`
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Verified  bool   `json:"verified"`
	ExpiresAt int64  `json:"expiresAt"`
}

type TwoFactorSetup struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qrCodeUrl"`
	BackupCodes []string `json:"backupCodes,omitempty"`
}

type TwoFactorBackupCodes struct {
	BackupCodes []string  `json:"backupCodes"`
	Generated   time.Time `json:"generated"`
}

type SecurityOverview struct {
	EmailVerified      bool               `json:"emailVerified"`
	TwoFactorEnabled   bool               `json:"twoFactorEnabled"`
	ActiveSessions     int                `json:"activeSessions"`
	LastPasswordChange time.Time          `json:"lastPasswordChange"`
	RecentActivity     []SecurityActivity `json:"recentActivity"`
	SecurityScore      int                `json:"securityScore"`
	Recommendations    []string           `json:"recommendations"`
}

type SecurityActivity struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	IPAddress   string                 `json:"ipAddress,omitempty"`
	DeviceType  string                 `json:"deviceType,omitempty"`
	Location    string                 `json:"location,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ============== USER SESSION MODEL ==============

type UserSession struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID     primitive.ObjectID `json:"userId" bson:"userId"`
	TokenHash  string             `json:"-" bson:"tokenHash"` // Don't expose in JSON
	DeviceType string             `json:"deviceType" bson:"deviceType"`
	DeviceName string             `json:"deviceName" bson:"deviceName"`
	IPAddress  string             `json:"ipAddress" bson:"ipAddress"`
	UserAgent  string             `json:"userAgent" bson:"userAgent"`
	Location   string             `json:"location,omitempty" bson:"location,omitempty"`
	IsActive   bool               `json:"isActive" bson:"isActive"`
	IsCurrent  bool               `json:"isCurrent" bson:"-"` // Computed field
	ExpiresAt  time.Time          `json:"expiresAt" bson:"expiresAt"`
	LastUsed   time.Time          `json:"lastUsed" bson:"lastUsed"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt  time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ============== AUDIT LOG MODEL ==============

type AuditLogEntry struct {
	ID          primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID     `json:"userId" bson:"userId"`
	EventType   string                 `json:"eventType" bson:"eventType"`
	Description string                 `json:"description" bson:"description"`
	IPAddress   string                 `json:"ipAddress" bson:"ipAddress"`
	UserAgent   string                 `json:"userAgent" bson:"userAgent"`
	DeviceType  string                 `json:"deviceType" bson:"deviceType"`
	Location    string                 `json:"location,omitempty" bson:"location,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty" bson:"details,omitempty"`
	Severity    string                 `json:"severity" bson:"severity"` // info, warning, critical
	CreatedAt   time.Time              `json:"createdAt" bson:"createdAt"`
}

// ============== OAUTH MODELS ==============

type OAuthUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Picture   string `json:"picture"`
	Verified  bool   `json:"verified"`
}

type OAuthConfig struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RedirectURL  string `json:"redirectUrl"`
}

// ============== AUTH CONFIG ==============

type AuthConfig struct {
	RequireEmailVerification bool `json:"requireEmailVerification"`
	AllowRegistration        bool `json:"allowRegistration"`
	PasswordMinLength        int  `json:"passwordMinLength"`
	SessionTimeout           int  `json:"sessionTimeout"` // minutes
	MaxActiveSessions        int  `json:"maxActiveSessions"`

	// Rate limiting
	LoginRateLimit  int `json:"loginRateLimit"`  // attempts per window
	LoginRateWindow int `json:"loginRateWindow"` // minutes

	// OAuth configurations
	Google   OAuthConfig `json:"google"`
	Apple    OAuthConfig `json:"apple"`
	Facebook OAuthConfig `json:"facebook"`

	// JWT settings
	JWTSecret          string `json:"-"`                  // Don't expose in JSON
	AccessTokenExpiry  int    `json:"accessTokenExpiry"`  // minutes
	RefreshTokenExpiry int    `json:"refreshTokenExpiry"` // days

	// Email settings
	EmailVerificationExpiry int `json:"emailVerificationExpiry"` // hours
	PasswordResetExpiry     int `json:"passwordResetExpiry"`     // hours

	// 2FA settings
	Enable2FA        bool `json:"enable2FA"`
	Require2FA       bool `json:"require2FA"`
	BackupCodesCount int  `json:"backupCodesCount"`
	TOTPWindow       int  `json:"totpWindow"` // seconds
}

// ============== SECURITY MODELS ==============

type SecurityEvent struct {
	Type        string                 `json:"type"`
	UserID      string                 `json:"userId"`
	Description string                 `json:"description"`
	IPAddress   string                 `json:"ipAddress"`
	UserAgent   string                 `json:"userAgent"`
	DeviceType  string                 `json:"deviceType"`
	Details     map[string]interface{} `json:"details"`
	Severity    string                 `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
}

type SuspiciousActivityReport struct {
	ID          primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID     `json:"userId" bson:"userId"`
	Description string                 `json:"description" bson:"description"`
	Category    string                 `json:"category" bson:"category"`
	Details     map[string]interface{} `json:"details" bson:"details"`
	Status      string                 `json:"status" bson:"status"`     // pending, investigating, resolved
	Priority    string                 `json:"priority" bson:"priority"` // low, medium, high, critical
	CreatedAt   time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" bson:"updatedAt"`
}

// ============== USER VERIFICATION MODELS ==============

type EmailVerification struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId"`
	Email     string             `json:"email" bson:"email"`
	Token     string             `json:"token" bson:"token"`
	ExpiresAt time.Time          `json:"expiresAt" bson:"expiresAt"`
	Verified  bool               `json:"verified" bson:"verified"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}

type PasswordReset struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId"`
	Token     string             `json:"token" bson:"token"`
	ExpiresAt time.Time          `json:"expiresAt" bson:"expiresAt"`
	Used      bool               `json:"used" bson:"used"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UsedAt    time.Time          `json:"usedAt,omitempty" bson:"usedAt,omitempty"`
}

// ============== LOGIN ATTEMPT TRACKING ==============

type LoginAttempt struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email      string             `json:"email" bson:"email"`
	IPAddress  string             `json:"ipAddress" bson:"ipAddress"`
	UserAgent  string             `json:"userAgent" bson:"userAgent"`
	Successful bool               `json:"successful" bson:"successful"`
	FailReason string             `json:"failReason,omitempty" bson:"failReason,omitempty"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
}

// ============== API KEY MANAGEMENT ==============

type APIKey struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"userId" bson:"userId"`
	Name        string             `json:"name" bson:"name"`
	KeyHash     string             `json:"-" bson:"keyHash"`           // Don't expose
	KeyPrefix   string             `json:"keyPrefix" bson:"keyPrefix"` // First 8 chars for identification
	Permissions []string           `json:"permissions" bson:"permissions"`
	IsActive    bool               `json:"isActive" bson:"isActive"`
	ExpiresAt   time.Time          `json:"expiresAt,omitempty" bson:"expiresAt,omitempty"`
	LastUsed    time.Time          `json:"lastUsed,omitempty" bson:"lastUsed,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ============== MESSAGE TYPES FOR VARIOUS FEATURES ==============

type MessageType struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Example     map[string]interface{} `json:"example"`
}

// ============== UTILITY TYPES ==============

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int64  `json:"expiresIn"`
}

// ============== CONSTANTS ==============

const (
	// Event types for audit logging
	EventLogin              = "login"
	EventLogout             = "logout"
	EventPasswordChange     = "password_change"
	EventPasswordReset      = "password_reset"
	EventEmailVerification  = "email_verification"
	Event2FAEnabled         = "2fa_enabled"
	Event2FADisabled        = "2fa_disabled"
	EventSuspiciousActivity = "suspicious_activity"
	EventAccountLocked      = "account_locked"
	EventAccountUnlocked    = "account_unlocked"

	// Security severity levels
	SeverityInfo    = "info"
	SeverityWarning = "warning"

	// Device types
	DeviceTypeIOS     = "ios"
	DeviceTypeAndroid = "android"
	DeviceTypeWeb     = "web"
	DeviceTypeDesktop = "desktop"

	// Session statuses
	SessionActive  = "active"
	SessionExpired = "expired"
	SessionRevoked = "revoked"

	// OAuth providers
	ProviderGoogle   = "google"
	ProviderApple    = "apple"
	ProviderFacebook = "facebook"
)
