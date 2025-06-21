package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/services"
	"ftrack/utils"
	"math/big"
	"net/url"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthService struct {
	userRepo        *repositories.UserRepository
	sessionRepo     *repositories.UserSessionRepository
	jwtService      *utils.JWTService
	passwordService *utils.PasswordService
	emailService    EmailService // Using existing EmailService interface
	smsService      *services.SMSService
	validator       *utils.ValidationService
	redis           *redis.Client
	config          *models.AuthConfig
}

func NewAuthService(
	userRepo *repositories.UserRepository,
	sessionRepo *repositories.UserSessionRepository,
	jwtService *utils.JWTService,
	emailService EmailService, // Using existing EmailService interface
	smsService *services.SMSService,
	redis *redis.Client,
	config *models.AuthConfig,
) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		sessionRepo:     sessionRepo,
		jwtService:      jwtService,
		passwordService: utils.NewPasswordService(),
		emailService:    emailService,
		smsService:      smsService,
		validator:       utils.NewValidationService(),
		redis:           redis,
		config:          config,
	}
}

// ============== BASIC AUTH METHODS ==============

func (as *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	// Validate request
	if validationErrors := as.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check if user already exists
	existingUser, _ := as.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("user already exists")
	}

	existingUser, _ = as.userRepo.GetByPhone(ctx, req.Phone)
	if existingUser != nil {
		return nil, errors.New("phone already exists")
	}

	// Hash password
	hashedPassword, err := as.passwordService.HashPassword(req.Password)
	if err != nil {
		logrus.Error("Failed to hash password: ", err)
		return nil, errors.New("failed to create user")
	}

	// Generate verification token
	verificationToken, err := as.generateSecureToken(32)
	if err != nil {
		logrus.Error("Failed to generate verification token: ", err)
		return nil, errors.New("failed to create user")
	}

	// Create user
	user := models.User{
		Email:             req.Email,
		Phone:             utils.NormalizePhoneNumber(req.Phone),
		Password:          hashedPassword,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		EmergencyContact:  req.EmergencyContact,
		IsVerified:        false,
		VerificationToken: verificationToken,
		TokenExpiresAt:    time.Now().Add(24 * time.Hour),
		LocationSharing: models.LocationSharing{
			Enabled:         true,
			Precision:       "exact",
			UpdateFrequency: 30,
		},
		Preferences: models.UserPreferences{
			Notifications: models.NotificationPrefs{
				PushEnabled:     true,
				SMSEnabled:      true,
				EmailEnabled:    true,
				LocationAlerts:  true,
				DrivingAlerts:   true,
				EmergencyAlerts: true,
			},
		},
	}

	err = as.userRepo.Create(ctx, &user)
	if err != nil {
		logrus.Error("Failed to create user: ", err)
		return nil, errors.New("failed to create user")
	}

	// Send verification email using existing email service
	go as.sendVerificationEmail(user.Email, user.FirstName, verificationToken)

	// Generate JWT tokens
	tokenPair, err := as.jwtService.GenerateTokenPair(user.ID.Hex(), user.Email, "user")
	if err != nil {
		logrus.Error("Failed to generate tokens: ", err)
		return nil, errors.New("failed to generate authentication tokens")
	}

	// Create session
	session := models.UserSession{
		UserID:     user.ID,
		TokenHash:  as.hashToken(tokenPair.AccessToken),
		DeviceType: req.DeviceType,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		IsActive:   true,
	}
	as.sessionRepo.Create(ctx, &session)

	// Remove password from response
	user.Password = ""

	return &models.AuthResponse{
		User:                 user,
		AccessToken:          tokenPair.AccessToken,
		RefreshToken:         tokenPair.RefreshToken,
		TokenType:            tokenPair.TokenType,
		ExpiresIn:            tokenPair.ExpiresIn,
		RequiresVerification: !user.IsVerified,
	}, nil
}

func (as *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	// Validate request
	if validationErrors := as.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Get user by email
	user, err := as.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	// Verify password
	isValid, err := as.passwordService.ComparePassword(req.Password, user.Password)
	if err != nil || !isValid {
		// Log failed login attempt
		as.logSecurityEvent(ctx, user.ID.Hex(), "failed_login", map[string]interface{}{
			"email": req.Email,
			"ip":    req.IPAddress,
		})
		return nil, errors.New("invalid email or password")
	}

	// Check if email is verified (optional based on config)
	if as.config.RequireEmailVerification && !user.IsVerified {
		return nil, errors.New("email not verified")
	}

	// Check if 2FA is enabled
	if user.TwoFactorEnabled && req.TwoFactorCode == "" {
		return nil, errors.New("2fa required")
	}

	// Verify 2FA if provided
	if user.TwoFactorEnabled && req.TwoFactorCode != "" {
		valid, err := as.verify2FACode(user.TwoFactorSecret, req.TwoFactorCode)
		if err != nil || !valid {
			as.logSecurityEvent(ctx, user.ID.Hex(), "failed_2fa", map[string]interface{}{
				"code": req.TwoFactorCode,
			})
			return nil, errors.New("invalid 2fa code")
		}
	}

	// Update last seen
	err = as.userRepo.UpdateLastSeen(ctx, user.ID.Hex())
	if err != nil {
		logrus.Warn("Failed to update last seen: ", err)
	}

	// Generate JWT tokens
	tokenPair, err := as.jwtService.GenerateTokenPair(user.ID.Hex(), user.Email, "user")
	if err != nil {
		logrus.Error("Failed to generate tokens: ", err)
		return nil, errors.New("failed to generate authentication tokens")
	}

	// Create session
	session := models.UserSession{
		UserID:     user.ID,
		TokenHash:  as.hashToken(tokenPair.AccessToken),
		DeviceType: req.DeviceType,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		IsActive:   true,
	}
	as.sessionRepo.Create(ctx, &session)

	// Log successful login
	as.logSecurityEvent(ctx, user.ID.Hex(), "successful_login", map[string]interface{}{
		"device": req.DeviceType,
		"ip":     req.IPAddress,
	})

	// Remove password from response
	user.Password = ""

	return &models.AuthResponse{
		User:         *user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// ============== EMAIL VERIFICATION ==============

func (as *AuthService) VerifyEmail(ctx context.Context, token string) error {
	user, err := as.userRepo.GetByVerificationToken(ctx, token)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	// Check if token is expired
	if time.Now().After(user.TokenExpiresAt) {
		return errors.New("invalid or expired token")
	}

	// Check if already verified
	if user.IsVerified {
		return errors.New("already verified")
	}

	// Update user as verified
	updateFields := bson.M{
		"isVerified":        true,
		"verificationToken": "",
		"tokenExpiresAt":    time.Time{},
		"updatedAt":         time.Now(),
	}

	err = as.userRepo.Update(ctx, user.ID.Hex(), updateFields)
	if err != nil {
		return err
	}

	// Send welcome email
	go as.sendWelcomeEmail(user.Email, user.FirstName)

	// Log email verification
	as.logSecurityEvent(ctx, user.ID.Hex(), "email_verified", nil)

	return nil
}

func (as *AuthService) ResendVerification(ctx context.Context, email string) error {
	user, err := as.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	if user.IsVerified {
		return nil // Don't reveal that user is already verified
	}

	// Generate new verification token
	verificationToken, err := as.generateSecureToken(32)
	if err != nil {
		return err
	}

	// Update user with new token
	updateFields := bson.M{
		"verificationToken": verificationToken,
		"tokenExpiresAt":    time.Now().Add(24 * time.Hour),
		"updatedAt":         time.Now(),
	}

	err = as.userRepo.Update(ctx, user.ID.Hex(), updateFields)
	if err != nil {
		return err
	}

	// Send verification email
	go as.sendVerificationEmail(user.Email, user.FirstName, verificationToken)

	return nil
}

// ============== PASSWORD RESET ==============

func (as *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := as.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	// Generate reset token
	resetToken, err := as.generateSecureToken(32)
	if err != nil {
		return err
	}

	// Update user with reset token
	updateFields := bson.M{
		"resetToken":     resetToken,
		"tokenExpiresAt": time.Now().Add(time.Hour), // 1 hour expiry
		"updatedAt":      time.Now(),
	}

	err = as.userRepo.Update(ctx, user.ID.Hex(), updateFields)
	if err != nil {
		return err
	}

	// Send reset email using existing email service
	go as.sendPasswordResetEmail(user.Email, user.FirstName, resetToken)

	// Log password reset request
	as.logSecurityEvent(ctx, user.ID.Hex(), "password_reset_requested", nil)

	return nil
}

func (as *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, err := as.userRepo.GetByResetToken(ctx, token)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	// Check if token is expired
	if time.Now().After(user.TokenExpiresAt) {
		return errors.New("invalid or expired token")
	}

	// Validate new password
	if len(newPassword) < 6 {
		return errors.New("validation failed")
	}

	// Hash new password
	hashedPassword, err := as.passwordService.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update user password and clear reset token
	updateFields := bson.M{
		"password":       hashedPassword,
		"resetToken":     "",
		"tokenExpiresAt": time.Time{},
		"updatedAt":      time.Now(),
	}

	err = as.userRepo.Update(ctx, user.ID.Hex(), updateFields)
	if err != nil {
		return err
	}

	// Invalidate all user sessions for security
	as.sessionRepo.InvalidateAllUserSessions(ctx, user.ID)

	// Send password changed notification
	go as.sendPasswordChangedEmail(user.Email, user.FirstName)

	// Log password reset
	as.logSecurityEvent(ctx, user.ID.Hex(), "password_reset_completed", nil)

	return nil
}

func (as *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	isValid, err := as.passwordService.ComparePassword(currentPassword, user.Password)
	if err != nil || !isValid {
		return errors.New("invalid current password")
	}

	// Validate new password
	if len(newPassword) < 6 {
		return errors.New("validation failed")
	}

	// Check if new password is different
	isSame, _ := as.passwordService.ComparePassword(newPassword, user.Password)
	if isSame {
		return errors.New("same password")
	}

	// Hash new password
	hashedPassword, err := as.passwordService.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	updateFields := bson.M{
		"password":  hashedPassword,
		"updatedAt": time.Now(),
	}

	err = as.userRepo.Update(ctx, userID, updateFields)
	if err != nil {
		return err
	}

	// Send password changed notification
	go as.sendPasswordChangedEmail(user.Email, user.FirstName)

	// Log password change
	as.logSecurityEvent(ctx, userID, "password_changed", nil)

	return nil
}

// ============== 2FA METHODS ==============

func (as *AuthService) Disable2FA(ctx context.Context, userID, password, code string) error {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.TwoFactorEnabled {
		return errors.New("not enabled")
	}

	// Verify password
	isValid, err := as.passwordService.ComparePassword(password, user.Password)
	if err != nil || !isValid {
		return errors.New("invalid password")
	}

	// Verify 2FA code
	valid, err := as.verify2FACode(user.TwoFactorSecret, code)
	if err != nil || !valid {
		return errors.New("invalid code")
	}

	// Disable 2FA
	updateFields := bson.M{
		"twoFactorEnabled": false,
		"twoFactorSecret":  "",
		"backupCodes":      []string{},
		"updatedAt":        time.Now(),
	}

	err = as.userRepo.Update(ctx, userID, updateFields)
	if err != nil {
		return err
	}

	// Send 2FA disabled email notification
	go as.send2FADisabledEmail(user.Email, user.FirstName)

	// Log 2FA disabled
	as.logSecurityEvent(ctx, userID, "2fa_disabled", nil)

	return nil
}

// ============== EMAIL SENDING METHODS (using existing EmailService) ==============

func (as *AuthService) sendVerificationEmail(email, firstName, token string) {
	if as.emailService != nil {
		err := as.emailService.SendVerificationEmail(email, firstName, token)
		if err != nil {
			logrus.Errorf("Failed to send verification email: %v", err)
		}
	}
}

func (as *AuthService) sendPasswordResetEmail(email, firstName, token string) {
	if as.emailService != nil {
		err := as.emailService.SendPasswordResetEmail(email, firstName, token)
		if err != nil {
			logrus.Errorf("Failed to send password reset email: %v", err)
		}
	}
}

func (as *AuthService) sendWelcomeEmail(email, firstName string) {
	if as.emailService != nil {
		err := as.emailService.SendWelcomeEmail(email, firstName)
		if err != nil {
			logrus.Errorf("Failed to send welcome email: %v", err)
		}
	}
}

func (as *AuthService) send2FADisabledEmail(email, firstName string) {
	if as.emailService != nil {
		err := as.emailService.Send2FADisabledEmail(email, firstName)
		if err != nil {
			logrus.Errorf("Failed to send 2FA disabled email: %v", err)
		}
	}
}

func (as *AuthService) sendPasswordChangedEmail(email, firstName string) {
	if as.emailService != nil {
		err := as.emailService.SendPasswordChangedEmail(email, firstName)
		if err != nil {
			logrus.Errorf("Failed to send password changed email: %v", err)
		}
	}
}

// ============== ALL OTHER METHODS REMAIN THE SAME ==============

func (as *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	tokenPair, err := as.jwtService.RefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Get user info from token
	claims, err := as.jwtService.ValidateToken(tokenPair.AccessToken)
	if err != nil {
		return nil, errors.New("invalid token")
	}

	user, err := as.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Update session with new token
	as.sessionRepo.UpdateTokenHash(ctx, claims.UserID, as.hashToken(tokenPair.AccessToken))

	// Remove password from response
	user.Password = ""

	return &models.AuthResponse{
		User:         *user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

func (as *AuthService) Logout(ctx context.Context, userID, token string) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	// Invalidate session
	tokenHash := as.hashToken(token)
	err = as.sessionRepo.InvalidateByTokenHash(ctx, userObjectID, tokenHash)
	if err != nil {
		logrus.Warn("Failed to invalidate session: ", err)
	}

	// Add token to blacklist in Redis
	if as.redis != nil {
		key := fmt.Sprintf("blacklist:token:%s", tokenHash)
		as.redis.Set(ctx, key, "1", 24*time.Hour)
	}

	// Log logout
	as.logSecurityEvent(ctx, userID, "logout", nil)

	return nil
}

func (as *AuthService) VerifyPassword(ctx context.Context, userID, password string) (bool, error) {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}

	return as.passwordService.ComparePassword(password, user.Password)
}

func (as *AuthService) LogoutAllDevices(ctx context.Context, userID string) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	// Invalidate all user sessions
	err = as.sessionRepo.InvalidateAllUserSessions(ctx, userObjectID)
	if err != nil {
		return err
	}

	// Log logout all devices
	as.logSecurityEvent(ctx, userID, "logout_all_devices", nil)

	return nil
}

func (as *AuthService) GetActiveSessions(ctx context.Context, userID string) ([]models.UserSession, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	return as.sessionRepo.GetActiveSessions(ctx, userObjectID)
}

func (as *AuthService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	sessionObjectID, err := primitive.ObjectIDFromHex(sessionID)
	if err != nil {
		return errors.New("invalid session ID")
	}

	// Verify session belongs to user
	session, err := as.sessionRepo.GetByID(ctx, sessionObjectID)
	if err != nil {
		return errors.New("session not found")
	}

	if session.UserID != userObjectID {
		return errors.New("unauthorized")
	}

	// Revoke session
	err = as.sessionRepo.InvalidateSession(ctx, sessionObjectID)
	if err != nil {
		return err
	}

	// Log session revocation
	as.logSecurityEvent(ctx, userID, "session_revoked", map[string]interface{}{
		"sessionId": sessionID,
	})

	return nil
}

func (as *AuthService) Setup2FA(ctx context.Context, userID string) (*models.TwoFactorSetup, error) {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.TwoFactorEnabled {
		return nil, errors.New("already enabled")
	}

	// Generate secret
	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "FTrack",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, err
	}

	// Generate QR code URL
	qrCodeURL := secret.URL()

	return &models.TwoFactorSetup{
		Secret:      secret.Secret(),
		QRCodeURL:   qrCodeURL,
		BackupCodes: nil, // Will be generated after verification
	}, nil
}

func (as *AuthService) Verify2FA(ctx context.Context, userID, code, secret string) ([]string, error) {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.TwoFactorEnabled {
		return nil, errors.New("already enabled")
	}

	// Verify the code
	valid, err := as.verify2FACode(secret, code)
	if err != nil || !valid {
		return nil, errors.New("invalid code")
	}

	// Generate backup codes
	backupCodes := as.generateBackupCodes(8)
	hashedBackupCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		hash, _ := as.passwordService.HashPassword(code)
		hashedBackupCodes[i] = hash
	}

	// Enable 2FA for user
	updateFields := bson.M{
		"twoFactorEnabled": true,
		"twoFactorSecret":  secret,
		"backupCodes":      hashedBackupCodes,
		"updatedAt":        time.Now(),
	}

	err = as.userRepo.Update(ctx, userID, updateFields)
	if err != nil {
		return nil, err
	}

	// Log 2FA enabled
	as.logSecurityEvent(ctx, userID, "2fa_enabled", nil)

	return backupCodes, nil
}

func (as *AuthService) GetBackupCodes(ctx context.Context, userID, password string) ([]string, error) {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !user.TwoFactorEnabled {
		return nil, errors.New("2fa not enabled")
	}

	// Verify password
	isValid, err := as.passwordService.ComparePassword(password, user.Password)
	if err != nil || !isValid {
		return nil, errors.New("invalid password")
	}

	// Return masked backup codes for security
	maskedCodes := make([]string, len(user.BackupCodes))
	for i := range user.BackupCodes {
		maskedCodes[i] = "****-****" // Show format but mask actual codes
	}

	return maskedCodes, nil
}

func (as *AuthService) RegenerateBackupCodes(ctx context.Context, userID, password, code string) ([]string, error) {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !user.TwoFactorEnabled {
		return nil, errors.New("2fa not enabled")
	}

	// Verify password
	isValid, err := as.passwordService.ComparePassword(password, user.Password)
	if err != nil || !isValid {
		return nil, errors.New("invalid password")
	}

	// Verify 2FA code
	valid, err := as.verify2FACode(user.TwoFactorSecret, code)
	if err != nil || !valid {
		return nil, errors.New("invalid 2fa code")
	}

	// Generate new backup codes
	backupCodes := as.generateBackupCodes(8)
	hashedBackupCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		hash, _ := as.passwordService.HashPassword(code)
		hashedBackupCodes[i] = hash
	}

	// Update backup codes
	updateFields := bson.M{
		"backupCodes": hashedBackupCodes,
		"updatedAt":   time.Now(),
	}

	err = as.userRepo.Update(ctx, userID, updateFields)
	if err != nil {
		return nil, err
	}

	// Log backup codes regenerated
	as.logSecurityEvent(ctx, userID, "backup_codes_regenerated", nil)

	return backupCodes, nil
}

// ============== OAUTH METHODS ==============

func (as *AuthService) GetGoogleOAuthURL(ctx context.Context) (string, error) {
	state, err := as.generateSecureToken(32)
	if err != nil {
		return "", err
	}

	// Store state in Redis for verification
	key := fmt.Sprintf("oauth:state:%s", state)
	as.redis.Set(ctx, key, "google", 10*time.Minute)

	params := url.Values{}
	params.Add("client_id", as.config.Google.ClientID)
	params.Add("redirect_uri", as.config.Google.RedirectURL)
	params.Add("scope", "openid email profile")
	params.Add("response_type", "code")
	params.Add("state", state)

	return fmt.Sprintf("https://accounts.google.com/o/oauth2/auth?%s", params.Encode()), nil
}

func (as *AuthService) HandleGoogleOAuthCallback(ctx context.Context, code, state string) (*models.AuthResponse, error) {
	// Verify state
	key := fmt.Sprintf("oauth:state:%s", state)
	provider, err := as.redis.Get(ctx, key).Result()
	if err != nil || provider != "google" {
		return nil, errors.New("invalid state")
	}

	// Delete state from Redis
	as.redis.Del(ctx, key)

	// Exchange code for token (implement actual OAuth flow)
	googleUser, err := as.exchangeGoogleCode(ctx, code)
	if err != nil {
		return nil, errors.New("oauth failed")
	}

	// Check if user exists
	user, err := as.userRepo.GetByEmail(ctx, googleUser.Email)
	if err != nil {
		// Create new user
		user = &models.User{
			Email:          googleUser.Email,
			FirstName:      googleUser.FirstName,
			LastName:       googleUser.LastName,
			IsVerified:     true, // OAuth emails are considered verified
			AuthProvider:   "google",
			AuthProviderID: googleUser.ID,
		}
		err = as.userRepo.Create(ctx, user)
		if err != nil {
			return nil, err
		}
	}

	// Generate tokens and return auth response
	return as.generateAuthResponse(ctx, user)
}

func (as *AuthService) GetAppleOAuthURL(ctx context.Context) (string, error) {
	state, err := as.generateSecureToken(32)
	if err != nil {
		return "", err
	}

	// Store state in Redis for verification
	key := fmt.Sprintf("oauth:state:%s", state)
	as.redis.Set(ctx, key, "apple", 10*time.Minute)

	params := url.Values{}
	params.Add("client_id", as.config.Apple.ClientID)
	params.Add("redirect_uri", as.config.Apple.RedirectURL)
	params.Add("scope", "name email")
	params.Add("response_type", "code")
	params.Add("state", state)

	return fmt.Sprintf("https://appleid.apple.com/auth/authorize?%s", params.Encode()), nil
}

func (as *AuthService) HandleAppleOAuthCallback(ctx context.Context, code, state string) (*models.AuthResponse, error) {
	// Similar implementation to Google OAuth
	// Verify state, exchange code, create/get user, return auth response
	return nil, errors.New("apple oauth not implemented")
}

func (as *AuthService) GetFacebookOAuthURL(ctx context.Context) (string, error) {
	state, err := as.generateSecureToken(32)
	if err != nil {
		return "", err
	}

	// Store state in Redis for verification
	key := fmt.Sprintf("oauth:state:%s", state)
	as.redis.Set(ctx, key, "facebook", 10*time.Minute)

	params := url.Values{}
	params.Add("client_id", as.config.Facebook.ClientID)
	params.Add("redirect_uri", as.config.Facebook.RedirectURL)
	params.Add("scope", "email")
	params.Add("response_type", "code")
	params.Add("state", state)

	return fmt.Sprintf("https://www.facebook.com/v18.0/dialog/oauth?%s", params.Encode()), nil
}

func (as *AuthService) HandleFacebookOAuthCallback(ctx context.Context, code, state string) (*models.AuthResponse, error) {
	// Similar implementation to Google OAuth
	// Verify state, exchange code, create/get user, return auth response
	return nil, errors.New("facebook oauth not implemented")
}

// ============== SECURITY METHODS ==============

func (as *AuthService) GetSecurityOverview(ctx context.Context, userID string) (*models.SecurityOverview, error) {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	activeSessions, err := as.GetActiveSessions(ctx, userID)
	if err != nil {
		activeSessions = []models.UserSession{}
	}

	overview := &models.SecurityOverview{
		EmailVerified:      user.IsVerified,
		TwoFactorEnabled:   user.TwoFactorEnabled,
		ActiveSessions:     len(activeSessions),
		LastPasswordChange: user.UpdatedAt,              // Would need separate field for password changes
		RecentActivity:     []models.SecurityActivity{}, // Would get from audit log
		SecurityScore:      as.calculateSecurityScore(user),
	}

	return overview, nil
}

func (as *AuthService) GetAuditLog(ctx context.Context, userID string, page, pageSize int) ([]models.AuditLogEntry, int64, error) {
	// Implementation would depend on how audit logs are stored
	// For now, return empty slice
	return []models.AuditLogEntry{}, 0, nil
}

func (as *AuthService) ReportSuspiciousActivity(ctx context.Context, userID, description, category string, details map[string]interface{}) error {
	// Log the suspicious activity report
	as.logSecurityEvent(ctx, userID, "suspicious_activity_reported", map[string]interface{}{
		"description": description,
		"category":    category,
		"details":     details,
	})

	// Could also send to security team, create ticket, etc.
	logrus.Warnf("Suspicious activity reported by user %s: %s", userID, description)

	return nil
}

// ============== UTILITY METHODS ==============

func (as *AuthService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return as.userRepo.GetByID(ctx, userID)
}

func (as *AuthService) ValidateToken(ctx context.Context, token string) (*models.User, error) {
	claims, err := as.jwtService.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	// Check if token is blacklisted
	if as.redis != nil {
		tokenHash := as.hashToken(token)
		key := fmt.Sprintf("blacklist:token:%s", tokenHash)
		exists := as.redis.Exists(ctx, key).Val()
		if exists > 0 {
			return nil, errors.New("token is blacklisted")
		}
	}

	return as.userRepo.GetByID(ctx, claims.UserID)
}

func (as *AuthService) UpdateLastSeen(ctx context.Context, userID string) error {
	return as.userRepo.UpdateLastSeen(ctx, userID)
}

func (as *AuthService) SetUserOffline(ctx context.Context, userID string) error {
	return as.userRepo.SetOffline(ctx, userID)
}

func (as *AuthService) IsAccountLocked(ctx context.Context, userID string) (bool, error) {
	return as.userRepo.IsAccountLocked(ctx, userID)
}

func (as *AuthService) LockAccount(ctx context.Context, userID string, duration time.Duration) error {
	return as.userRepo.LockAccount(ctx, userID, duration)
}

func (as *AuthService) UnlockAccount(ctx context.Context, userID string) error {
	return as.userRepo.UnlockAccount(ctx, userID)
}

func (as *AuthService) IncrementLoginAttempts(ctx context.Context, userID string) error {
	return as.userRepo.IncrementLoginAttempts(ctx, userID)
}

func (as *AuthService) ResetLoginAttempts(ctx context.Context, userID string) error {
	return as.userRepo.ResetLoginAttempts(ctx, userID)
}

// ============== HELPER METHODS ==============

func (as *AuthService) generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (as *AuthService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (as *AuthService) verify2FACode(secret, code string) (bool, error) {
	return totp.Validate(code, secret), nil
}

func (as *AuthService) generateBackupCodes(count int) []string {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		// Generate 8-digit backup code
		code := ""
		for j := 0; j < 8; j++ {
			digit, _ := rand.Int(rand.Reader, big.NewInt(10))
			code += digit.String()
		}
		// Format as XXXX-XXXX
		codes[i] = code[:4] + "-" + code[4:]
	}
	return codes
}

func (as *AuthService) calculateSecurityScore(user *models.User) int {
	score := 0
	if user.IsVerified {
		score += 30
	}
	if user.TwoFactorEnabled {
		score += 40
	}
	if user.Password != "" { // Has password (not OAuth only)
		score += 20
	}
	// Add more criteria as needed
	score += 10 // Base score
	return score
}

func (as *AuthService) generateAuthResponse(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
	// Generate JWT tokens
	tokenPair, err := as.jwtService.GenerateTokenPair(user.ID.Hex(), user.Email, "user")
	if err != nil {
		return nil, err
	}

	// Create session
	session := models.UserSession{
		UserID:    user.ID,
		TokenHash: as.hashToken(tokenPair.AccessToken),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsActive:  true,
	}
	as.sessionRepo.Create(ctx, &session)

	// Remove password from response
	user.Password = ""

	return &models.AuthResponse{
		User:         *user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

func (as *AuthService) logSecurityEvent(ctx context.Context, userID, eventType string, data map[string]interface{}) {
	// Implementation would log to audit system
	logrus.Infof("Security event for user %s: %s - %v", userID, eventType, data)
}

func (as *AuthService) exchangeGoogleCode(ctx context.Context, code string) (*models.OAuthUser, error) {
	// Implementation would exchange OAuth code for user info
	// This is a placeholder
	return &models.OAuthUser{
		ID:        "google_user_id",
		Email:     "user@gmail.com",
		FirstName: "John",
		LastName:  "Doe",
	}, nil
}
