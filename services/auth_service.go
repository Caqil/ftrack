package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

type AuthService struct {
	userRepo        *repositories.UserRepository
	jwtService      *utils.JWTService
	passwordService *utils.PasswordService
	validator       *utils.ValidationService
}

func NewAuthService(userRepo *repositories.UserRepository, jwtService *utils.JWTService) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		jwtService:      jwtService,
		passwordService: utils.NewPasswordService(),
		validator:       utils.NewValidationService(),
	}
}

func (as *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	// Validate request
	if validationErrors := as.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check if user already exists
	existingUser, _ := as.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	existingUser, _ = as.userRepo.GetByPhone(ctx, req.Phone)
	if existingUser != nil {
		return nil, errors.New("user with this phone number already exists")
	}

	// Hash password
	hashedPassword, err := as.passwordService.HashPassword(req.Password)
	if err != nil {
		logrus.Error("Failed to hash password: ", err)
		return nil, errors.New("failed to create user")
	}

	// Create user
	user := models.User{
		Email:            req.Email,
		Phone:            utils.NormalizePhoneNumber(req.Phone),
		Password:         hashedPassword,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		EmergencyContact: req.EmergencyContact,
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
			Privacy: models.PrivacySettings{
				ShowInDirectory: true,
				AllowInvites:    true,
				ShareDriving:    true,
			},
			Driving: models.DrivingPrefs{
				AutoDetect:  true,
				SpeedLimit:  80, // km/h
				HardBraking: true,
				PhoneUsage:  true,
			},
		},
	}

	err = as.userRepo.Create(ctx, &user)
	if err != nil {
		logrus.Error("Failed to create user: ", err)
		return nil, errors.New("failed to create user")
	}

	// Generate JWT tokens
	tokenPair, err := as.jwtService.GenerateTokenPair(user.ID.Hex(), user.Email, "user")
	if err != nil {
		logrus.Error("Failed to generate tokens: ", err)
		return nil, errors.New("failed to generate authentication tokens")
	}

	// Remove password from response
	user.Password = ""

	return &models.AuthResponse{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
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
		return nil, errors.New("invalid email or password")
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

func (as *AuthService) ValidateToken(ctx context.Context, token string) (*models.User, error) {
	claims, err := as.jwtService.ValidateToken(token)
	if err != nil {
		return nil, errors.New("invalid token")
	}

	user, err := as.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	// Remove password from response
	user.Password = ""
	return user, nil
}

func (as *AuthService) Logout(ctx context.Context, token string) error {
	// In a production environment, you would add the token to a blacklist in Redis
	// For now, just validate the token format
	return as.jwtService.RevokeToken(token)
}

func (as *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify old password
	isValid, err := as.passwordService.ComparePassword(oldPassword, user.Password)
	if err != nil || !isValid {
		return errors.New("invalid current password")
	}

	// Hash new password
	hashedPassword, err := as.passwordService.HashPassword(newPassword)
	if err != nil {
		return errors.New("failed to update password")
	}

	// Update password
	return as.userRepo.Update(ctx, userID, map[string]interface{}{
		"password": hashedPassword,
	})
}

func (as *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := as.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil // Don't reveal if email exists
	}

	resetToken, _ := generateSecureToken(32)
	expiresAt := time.Now().Add(1 * time.Hour)

	update := bson.M{
		"resetToken":        resetToken,
		"resetTokenExpires": expiresAt,
	}

	as.userRepo.Update(ctx, user.ID.Hex(), update)

	// Send email here (implement your email service)
	logrus.Infof("Password reset token generated for: %s", email)
	return nil
}

func (as *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 6 {
		return errors.New("password must be at least 6 characters")
	}

	user, err := as.userRepo.GetByResetToken(ctx, token)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	hashedPassword, _ := as.passwordService.HashPassword(newPassword)

	update := bson.M{
		"password":          hashedPassword,
		"resetToken":        "",
		"resetTokenExpires": time.Time{},
	}

	return as.userRepo.Update(ctx, user.ID.Hex(), update)
}

func (as *AuthService) VerifyEmail(ctx context.Context, token string) error {
	user, err := as.userRepo.GetByVerificationToken(ctx, token)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	update := bson.M{
		"isEmailVerified":   true,
		"verificationToken": "",
	}

	return as.userRepo.Update(ctx, user.ID.Hex(), update)
}

func (as *AuthService) ResendVerification(ctx context.Context, email string) error {
	user, err := as.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil // Don't reveal if email exists
	}

	verificationToken, _ := generateSecureToken(32)

	update := bson.M{
		"verificationToken":        verificationToken,
		"verificationTokenExpires": time.Now().Add(24 * time.Hour),
	}

	as.userRepo.Update(ctx, user.ID.Hex(), update)

	// Send email here
	return nil
}

func (as *AuthService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	user.Password = "" // Remove sensitive data
	return user, nil
}

// Helper function
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes), nil
}
