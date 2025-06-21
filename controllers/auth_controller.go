package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"

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
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid input data")
		default:
			utils.InternalServerErrorResponse(c, "Authentication failed")
		}
		return
	}

	utils.SuccessResponse(c, "Login successful", response)
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

	err := ac.authService.Logout(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Logout failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to logout")
		return
	}

	utils.SuccessResponse(c, "Logged out successfully", nil)
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
		default:
			utils.InternalServerErrorResponse(c, "Failed to change password")
		}
		return
	}

	utils.SuccessResponse(c, "Password changed successfully", nil)
}

// VerifyEmail handles email verification
// @Summary Verify email address
// @Description Verify user email using verification token
// @Tags Authentication
// @Param token query string true "Verification token"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Router /auth/verify-email [get]
func (ac *AuthController) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		utils.BadRequestResponse(c, "Verification token is required")
		return
	}

	err := ac.authService.VerifyEmail(c.Request.Context(), token)
	if err != nil {
		logrus.Errorf("Email verification failed: %v", err)

		switch err.Error() {
		case "invalid or expired token":
			utils.BadRequestResponse(c, "Invalid or expired verification token")
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

// GetMe returns current user information
// @Summary Get current user
// @Description Get current authenticated user information
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.User}
// @Failure 401 {object} models.APIResponse
// @Router /auth/me [get]
func (ac *AuthController) GetMe(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	user, err := ac.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get user failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get user information")
		return
	}

	utils.SuccessResponse(c, "User information retrieved successfully", user)
}

// ValidateToken validates access token
// @Summary Validate token
// @Description Validate access token
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /auth/validate [get]
func (ac *AuthController) ValidateToken(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "Invalid token")
		return
	}

	utils.SuccessResponse(c, "Token is valid", map[string]interface{}{
		"valid":  true,
		"userID": userID,
	})
}
