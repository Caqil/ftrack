// controllers/auth_controller.go
package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"
	"time"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AuthController struct {
	authService *services.AuthService
}

func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// ============== PUBLIC AUTHENTICATION ENDPOINTS ==============

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration data"
// @Success 201 {object} models.APIResponse{data=models.AuthResponse}
// @Failure 400 {object} models.APIResponse
// @Failure 409 {object} models.APIResponse
// @Router /auth/register [post]
func (ac *AuthController) Register(c *gin.Context) {
	var req models.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	response, err := ac.authService.Register(c.Request.Context(), req)
	if err != nil {
		logrus.Errorf("Registration failed: %v", err)

		switch err.Error() {
		case "user already exists":
			utils.ConflictResponse(c, "User with this email already exists")
		case "phone already exists":
			utils.ConflictResponse(c, "User with this phone number already exists")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid input data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create account")
		}
		return
	}

	utils.CreatedResponse(c, "Account created successfully", response)
}

// Login handles user authentication
// @Summary Login user
// @Description Authenticate user and return tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.APIResponse{data=models.AuthResponse}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/login [post]
func (ac *AuthController) Login(c *gin.Context) {
	var req models.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	response, err := ac.authService.Login(c.Request.Context(), req)
	if err != nil {
		logrus.Errorf("Login failed: %v", err)

		switch err.Error() {
		case "invalid email or password":
			utils.UnauthorizedResponse(c, "Invalid email or password")
		case "account is deactivated":
			utils.UnauthorizedResponse(c, "Account is deactivated")
		case "email not verified":
			utils.UnauthorizedResponse(c, "Please verify your email address")
		case "2fa required":
			utils.UnauthorizedResponse(c, "Two-factor authentication required")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid input data")
		default:
			utils.InternalServerErrorResponse(c, "Authentication failed")
		}
		return
	}

	utils.SuccessResponse(c, "Login successful", response)
}

// ForgotPassword handles password reset requests
// @Summary Request password reset
// @Description Send password reset email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body object{email=string} true "Email address"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Router /auth/forgot-password [post]
func (ac *AuthController) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Valid email address is required")
		return
	}

	err := ac.authService.ForgotPassword(c.Request.Context(), req.Email)
	if err != nil {
		logrus.Errorf("Forgot password failed: %v", err)
		// Don't reveal if email exists or not for security
	}

	utils.SuccessResponse(c, "If the email exists, a password reset link has been sent", nil)
}

// ResetPassword handles password reset
// @Summary Reset password
// @Description Reset password using token from email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body object{token=string,password=string} true "Reset token and new password"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/reset-password [post]
func (ac *AuthController) ResetPassword(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Token and password are required")
		return
	}

	err := ac.authService.ResetPassword(c.Request.Context(), req.Token, req.Password)
	if err != nil {
		logrus.Errorf("Password reset failed: %v", err)

		switch err.Error() {
		case "invalid or expired token":
			utils.UnauthorizedResponse(c, "Invalid or expired reset token")
		case "validation failed":
			utils.BadRequestResponse(c, "Password must be at least 6 characters")
		default:
			utils.InternalServerErrorResponse(c, "Failed to reset password")
		}
		return
	}

	utils.SuccessResponse(c, "Password reset successfully", nil)
}

// VerifyEmail handles email verification
// @Summary Verify email address
// @Description Verify user email using verification token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body object{token=string} true "Verification token"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Router /auth/verify-email [post]
func (ac *AuthController) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Verification token is required")
		return
	}

	err := ac.authService.VerifyEmail(c.Request.Context(), req.Token)
	if err != nil {
		logrus.Errorf("Email verification failed: %v", err)

		switch err.Error() {
		case "invalid or expired token":
			utils.BadRequestResponse(c, "Invalid or expired verification token")
		case "already verified":
			utils.BadRequestResponse(c, "Email is already verified")
		default:
			utils.InternalServerErrorResponse(c, "Failed to verify email")
		}
		return
	}

	utils.SuccessResponse(c, "Email verified successfully", nil)
}

// ResendVerification sends verification email again
// @Summary Resend verification email
// @Description Send verification email again
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body object{email=string} true "Email address"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Router /auth/resend-verification [post]
func (ac *AuthController) ResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Valid email address is required")
		return
	}

	err := ac.authService.ResendVerification(c.Request.Context(), req.Email)
	if err != nil {
		logrus.Errorf("Resend verification failed: %v", err)
		// Don't reveal if email exists or not for security
	}

	utils.SuccessResponse(c, "If the email exists, a verification link has been sent", nil)
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Get new access token using refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body object{refreshToken=string} true "Refresh token"
// @Success 200 {object} models.APIResponse{data=models.AuthResponse}
// @Failure 401 {object} models.APIResponse
// @Router /auth/refresh [post]
func (ac *AuthController) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Refresh token is required")
		return
	}

	response, err := ac.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		logrus.Errorf("Token refresh failed: %v", err)
		utils.UnauthorizedResponse(c, "Invalid refresh token")
		return
	}

	utils.SuccessResponse(c, "Token refreshed successfully", response)
}

// Logout handles user logout
// @Summary Logout user
// @Description Logout user and invalidate tokens
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/logout [post]
func (ac *AuthController) Logout(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Get token from header to invalidate specific session
	token := c.GetHeader("Authorization")
	if token != "" && len(token) > 7 {
		token = token[7:] // Remove "Bearer " prefix
	}

	err := ac.authService.Logout(c.Request.Context(), userID, token)
	if err != nil {
		logrus.Errorf("Logout failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to logout")
		return
	}

	utils.SuccessResponse(c, "Logged out successfully", nil)
}

// ============== OAUTH ENDPOINTS ==============

// GoogleOAuth initiates Google OAuth flow
// @Summary Google OAuth
// @Description Initiate Google OAuth authentication
// @Tags Authentication
// @Produce json
// @Success 302 "Redirect to Google OAuth"
// @Router /auth/oauth/google [get]
func (ac *AuthController) GoogleOAuth(c *gin.Context) {
	authURL, err := ac.authService.GetGoogleOAuthURL(c.Request.Context())
	if err != nil {
		logrus.Errorf("Google OAuth URL generation failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to initiate Google OAuth")
		return
	}

	c.Redirect(302, authURL)
}

// GoogleOAuthCallback handles Google OAuth callback
// @Summary Google OAuth Callback
// @Description Handle Google OAuth callback
// @Tags Authentication
// @Param code query string true "OAuth authorization code"
// @Param state query string false "OAuth state parameter"
// @Success 200 {object} models.APIResponse{data=models.AuthResponse}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/oauth/google/callback [get]
func (ac *AuthController) GoogleOAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		utils.BadRequestResponse(c, "Authorization code is required")
		return
	}

	response, err := ac.authService.HandleGoogleOAuthCallback(c.Request.Context(), code, state)
	if err != nil {
		logrus.Errorf("Google OAuth callback failed: %v", err)

		switch err.Error() {
		case "invalid state":
			utils.BadRequestResponse(c, "Invalid OAuth state")
		case "oauth failed":
			utils.UnauthorizedResponse(c, "Google OAuth authentication failed")
		default:
			utils.InternalServerErrorResponse(c, "OAuth authentication failed")
		}
		return
	}

	utils.SuccessResponse(c, "Google OAuth authentication successful", response)
}

// AppleOAuth initiates Apple OAuth flow
// @Summary Apple OAuth
// @Description Initiate Apple OAuth authentication
// @Tags Authentication
// @Produce json
// @Success 302 "Redirect to Apple OAuth"
// @Router /auth/oauth/apple [get]
func (ac *AuthController) AppleOAuth(c *gin.Context) {
	authURL, err := ac.authService.GetAppleOAuthURL(c.Request.Context())
	if err != nil {
		logrus.Errorf("Apple OAuth URL generation failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to initiate Apple OAuth")
		return
	}

	c.Redirect(302, authURL)
}

// AppleOAuthCallback handles Apple OAuth callback
// @Summary Apple OAuth Callback
// @Description Handle Apple OAuth callback
// @Tags Authentication
// @Param code query string true "OAuth authorization code"
// @Param state query string false "OAuth state parameter"
// @Success 200 {object} models.APIResponse{data=models.AuthResponse}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/oauth/apple/callback [get]
func (ac *AuthController) AppleOAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		utils.BadRequestResponse(c, "Authorization code is required")
		return
	}

	response, err := ac.authService.HandleAppleOAuthCallback(c.Request.Context(), code, state)
	if err != nil {
		logrus.Errorf("Apple OAuth callback failed: %v", err)

		switch err.Error() {
		case "invalid state":
			utils.BadRequestResponse(c, "Invalid OAuth state")
		case "oauth failed":
			utils.UnauthorizedResponse(c, "Apple OAuth authentication failed")
		default:
			utils.InternalServerErrorResponse(c, "OAuth authentication failed")
		}
		return
	}

	utils.SuccessResponse(c, "Apple OAuth authentication successful", response)
}

// FacebookOAuth initiates Facebook OAuth flow
// @Summary Facebook OAuth
// @Description Initiate Facebook OAuth authentication
// @Tags Authentication
// @Produce json
// @Success 302 "Redirect to Facebook OAuth"
// @Router /auth/oauth/facebook [get]
func (ac *AuthController) FacebookOAuth(c *gin.Context) {
	authURL, err := ac.authService.GetFacebookOAuthURL(c.Request.Context())
	if err != nil {
		logrus.Errorf("Facebook OAuth URL generation failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to initiate Facebook OAuth")
		return
	}

	c.Redirect(302, authURL)
}

// FacebookOAuthCallback handles Facebook OAuth callback
// @Summary Facebook OAuth Callback
// @Description Handle Facebook OAuth callback
// @Tags Authentication
// @Param code query string true "OAuth authorization code"
// @Param state query string false "OAuth state parameter"
// @Success 200 {object} models.APIResponse{data=models.AuthResponse}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/oauth/facebook/callback [get]
func (ac *AuthController) FacebookOAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		utils.BadRequestResponse(c, "Authorization code is required")
		return
	}

	response, err := ac.authService.HandleFacebookOAuthCallback(c.Request.Context(), code, state)
	if err != nil {
		logrus.Errorf("Facebook OAuth callback failed: %v", err)

		switch err.Error() {
		case "invalid state":
			utils.BadRequestResponse(c, "Invalid OAuth state")
		case "oauth failed":
			utils.UnauthorizedResponse(c, "Facebook OAuth authentication failed")
		default:
			utils.InternalServerErrorResponse(c, "OAuth authentication failed")
		}
		return
	}

	utils.SuccessResponse(c, "Facebook OAuth authentication successful", response)
}

// ============== PROTECTED AUTHENTICATION ENDPOINTS ==============

// ValidateToken validates access token
// @Summary Validate token
// @Description Validate access token
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/validate [post]
func (ac *AuthController) ValidateToken(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "Invalid token")
		return
	}

	user, err := ac.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get user failed: %v", err)
		utils.UnauthorizedResponse(c, "Invalid token")
		return
	}

	utils.SuccessResponse(c, "Token is valid", map[string]interface{}{
		"valid":     true,
		"userID":    userID,
		"email":     user.Email,
		"verified":  user.IsVerified,
		"expiresAt": time.Now().Add(24 * time.Hour), // Would get from actual token
	})
}

// LogoutAllDevices logs out user from all devices
// @Summary Logout from all devices
// @Description Logout user from all devices and invalidate all tokens
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/logout-all [post]
func (ac *AuthController) LogoutAllDevices(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	err := ac.authService.LogoutAllDevices(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Logout all devices failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to logout from all devices")
		return
	}

	utils.SuccessResponse(c, "Logged out from all devices successfully", nil)
}

// GetActiveSessions gets user's active sessions
// @Summary Get active sessions
// @Description Get list of user's active sessions
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.UserSession}
// @Failure 401 {object} models.APIResponse
// @Router /auth/sessions [get]
func (ac *AuthController) GetActiveSessions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	sessions, err := ac.authService.GetActiveSessions(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get active sessions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get active sessions")
		return
	}

	utils.SuccessResponse(c, "Active sessions retrieved successfully", sessions)
}

// RevokeSession revokes a specific session
// @Summary Revoke session
// @Description Revoke a specific user session
// @Tags Authentication
// @Security BearerAuth
// @Param sessionId path string true "Session ID"
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /auth/sessions/{sessionId} [delete]
func (ac *AuthController) RevokeSession(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	sessionID := c.Param("sessionId")
	if sessionID == "" {
		utils.BadRequestResponse(c, "Session ID is required")
		return
	}

	err := ac.authService.RevokeSession(c.Request.Context(), userID, sessionID)
	if err != nil {
		logrus.Errorf("Revoke session failed: %v", err)

		switch err.Error() {
		case "session not found":
			utils.NotFoundResponse(c, "Session")
		case "unauthorized":
			utils.ForbiddenResponse(c, "Cannot revoke session belonging to another user")
		default:
			utils.InternalServerErrorResponse(c, "Failed to revoke session")
		}
		return
	}

	utils.SuccessResponse(c, "Session revoked successfully", nil)
}

// ChangePassword handles password change for authenticated users
// @Summary Change password
// @Description Change user password
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{currentPassword=string,newPassword=string} true "Current and new passwords"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/change-password [post]
func (ac *AuthController) ChangePassword(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Current password and new password are required")
		return
	}

	err := ac.authService.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		logrus.Errorf("Change password failed: %v", err)

		switch err.Error() {
		case "invalid current password":
			utils.UnauthorizedResponse(c, "Current password is incorrect")
		case "validation failed":
			utils.BadRequestResponse(c, "New password must be at least 6 characters")
		case "same password":
			utils.BadRequestResponse(c, "New password must be different from current password")
		default:
			utils.InternalServerErrorResponse(c, "Failed to change password")
		}
		return
	}

	utils.SuccessResponse(c, "Password changed successfully", nil)
}

// VerifyPassword verifies user's current password
// @Summary Verify password
// @Description Verify user's current password
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{password=string} true "Password to verify"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/verify-password [post]
func (ac *AuthController) VerifyPassword(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Password is required")
		return
	}

	isValid, err := ac.authService.VerifyPassword(c.Request.Context(), userID, req.Password)
	if err != nil {
		logrus.Errorf("Verify password failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to verify password")
		return
	}

	if !isValid {
		utils.UnauthorizedResponse(c, "Invalid password")
		return
	}

	utils.SuccessResponse(c, "Password verified successfully", map[string]interface{}{
		"valid": true,
	})
}

// ============== TWO-FACTOR AUTHENTICATION ==============

// Setup2FA sets up two-factor authentication
// @Summary Setup 2FA
// @Description Setup two-factor authentication for user
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.TwoFactorSetup}
// @Failure 401 {object} models.APIResponse
// @Router /auth/2fa/setup [post]
func (ac *AuthController) Setup2FA(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	setupData, err := ac.authService.Setup2FA(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Setup 2FA failed: %v", err)

		switch err.Error() {
		case "already enabled":
			utils.BadRequestResponse(c, "Two-factor authentication is already enabled")
		default:
			utils.InternalServerErrorResponse(c, "Failed to setup two-factor authentication")
		}
		return
	}

	utils.SuccessResponse(c, "Two-factor authentication setup initiated", setupData)
}

// Verify2FA verifies and enables two-factor authentication
// @Summary Verify 2FA
// @Description Verify and enable two-factor authentication
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{code=string,secret=string} true "2FA code and secret"
// @Success 200 {object} models.APIResponse{data=models.TwoFactorBackupCodes}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/2fa/verify [post]
func (ac *AuthController) Verify2FA(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Code   string `json:"code" binding:"required"`
		Secret string `json:"secret" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "2FA code and secret are required")
		return
	}

	backupCodes, err := ac.authService.Verify2FA(c.Request.Context(), userID, req.Code, req.Secret)
	if err != nil {
		logrus.Errorf("Verify 2FA failed: %v", err)

		switch err.Error() {
		case "invalid code":
			utils.BadRequestResponse(c, "Invalid 2FA code")
		case "invalid secret":
			utils.BadRequestResponse(c, "Invalid 2FA secret")
		case "already enabled":
			utils.BadRequestResponse(c, "Two-factor authentication is already enabled")
		default:
			utils.InternalServerErrorResponse(c, "Failed to verify two-factor authentication")
		}
		return
	}

	utils.SuccessResponse(c, "Two-factor authentication enabled successfully", map[string]interface{}{
		"backupCodes": backupCodes,
		"message":     "Save these backup codes in a secure location",
	})
}

// Disable2FA disables two-factor authentication
// @Summary Disable 2FA
// @Description Disable two-factor authentication
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{password=string,code=string} true "Password and 2FA code"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/2fa/disable [post]
func (ac *AuthController) Disable2FA(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Password and 2FA code are required")
		return
	}

	err := ac.authService.Disable2FA(c.Request.Context(), userID, req.Password, req.Code)
	if err != nil {
		logrus.Errorf("Disable 2FA failed: %v", err)

		switch err.Error() {
		case "invalid password":
			utils.UnauthorizedResponse(c, "Invalid password")
		case "invalid code":
			utils.BadRequestResponse(c, "Invalid 2FA code")
		case "not enabled":
			utils.BadRequestResponse(c, "Two-factor authentication is not enabled")
		default:
			utils.InternalServerErrorResponse(c, "Failed to disable two-factor authentication")
		}
		return
	}

	utils.SuccessResponse(c, "Two-factor authentication disabled successfully", nil)
}

// GetBackupCodes gets user's 2FA backup codes
// @Summary Get backup codes
// @Description Get user's two-factor authentication backup codes
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{password=string} true "User password"
// @Success 200 {object} models.APIResponse{data=[]string}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/2fa/backup-codes [get]
func (ac *AuthController) GetBackupCodes(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Password is required")
		return
	}

	backupCodes, err := ac.authService.GetBackupCodes(c.Request.Context(), userID, req.Password)
	if err != nil {
		logrus.Errorf("Get backup codes failed: %v", err)

		switch err.Error() {
		case "invalid password":
			utils.UnauthorizedResponse(c, "Invalid password")
		case "2fa not enabled":
			utils.BadRequestResponse(c, "Two-factor authentication is not enabled")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get backup codes")
		}
		return
	}

	utils.SuccessResponse(c, "Backup codes retrieved successfully", backupCodes)
}

// RegenerateBackupCodes regenerates user's 2FA backup codes
// @Summary Regenerate backup codes
// @Description Regenerate two-factor authentication backup codes
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{password=string,code=string} true "Password and 2FA code"
// @Success 200 {object} models.APIResponse{data=[]string}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/2fa/regenerate-codes [post]
func (ac *AuthController) RegenerateBackupCodes(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Password and 2FA code are required")
		return
	}

	newBackupCodes, err := ac.authService.RegenerateBackupCodes(c.Request.Context(), userID, req.Password, req.Code)
	if err != nil {
		logrus.Errorf("Regenerate backup codes failed: %v", err)

		switch err.Error() {
		case "invalid password":
			utils.UnauthorizedResponse(c, "Invalid password")
		case "invalid code":
			utils.BadRequestResponse(c, "Invalid 2FA code")
		case "2fa not enabled":
			utils.BadRequestResponse(c, "Two-factor authentication is not enabled")
		default:
			utils.InternalServerErrorResponse(c, "Failed to regenerate backup codes")
		}
		return
	}

	utils.SuccessResponse(c, "Backup codes regenerated successfully", map[string]interface{}{
		"backupCodes": newBackupCodes,
		"message":     "Old backup codes are now invalid. Save these new codes in a secure location",
	})
}

// ============== ACCOUNT SECURITY ==============

// SecurityOverview gets user's security overview
// @Summary Security overview
// @Description Get user's account security overview
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.SecurityOverview}
// @Failure 401 {object} models.APIResponse
// @Router /auth/security/overview [get]
func (ac *AuthController) SecurityOverview(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	overview, err := ac.authService.GetSecurityOverview(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get security overview failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get security overview")
		return
	}

	utils.SuccessResponse(c, "Security overview retrieved successfully", overview)
}

// GetAuditLog gets user's security audit log
// @Summary Get audit log
// @Description Get user's security and activity audit log
// @Tags Authentication
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.AuditLogEntry}
// @Failure 401 {object} models.APIResponse
// @Router /auth/security/audit-log [get]
func (ac *AuthController) GetAuditLog(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			page = v
		}
	}
	if ps := c.Query("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil {
			pageSize = v
		}
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	auditLog, total, err := ac.authService.GetAuditLog(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get audit log failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get audit log")
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Audit log retrieved successfully", auditLog, meta)
}

// ReportSuspiciousActivity reports suspicious activity
// @Summary Report suspicious activity
// @Description Report suspicious activity on user account
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{description=string,category=string,details=object} true "Suspicious activity details"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/security/report-suspicious [post]
func (ac *AuthController) ReportSuspiciousActivity(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Description string                 `json:"description" binding:"required"`
		Category    string                 `json:"category" binding:"required"`
		Details     map[string]interface{} `json:"details"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Description and category are required")
		return
	}

	err := ac.authService.ReportSuspiciousActivity(c.Request.Context(), userID, req.Description, req.Category, req.Details)
	if err != nil {
		logrus.Errorf("Report suspicious activity failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to report suspicious activity")
		return
	}

	utils.SuccessResponse(c, "Suspicious activity reported successfully. Our security team will investigate.", nil)
}
