package middleware

import (
	"context"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthMiddleware struct {
	jwtService *utils.JWTService
	userRepo   *repositories.UserRepository
}

func NewAuthMiddleware(jwtService *utils.JWTService, userRepo *repositories.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
		userRepo:   userRepo,
	}
}

// RequireAuth validates JWT token and sets user context
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		token := am.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "UNAUTHORIZED",
				Message: "Authentication token required",
				Code:    "AUTH_TOKEN_REQUIRED",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := am.jwtService.ValidateToken(token)
		if err != nil {
			logrus.Warnf("Invalid token: %v", err)
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "UNAUTHORIZED",
				Message: "Invalid authentication token",
				Code:    "AUTH_TOKEN_INVALID",
			})
			c.Abort()
			return
		}

		// Check if token is not expired
		if claims.ExpiresAt.Before(time.Now()) {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "UNAUTHORIZED",
				Message: "Authentication token expired",
				Code:    "AUTH_TOKEN_EXPIRED",
			})
			c.Abort()
			return
		}

		// Verify token type
		if claims.TokenType != "access" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "UNAUTHORIZED",
				Message: "Invalid token type",
				Code:    "AUTH_TOKEN_INVALID_TYPE",
			})
			c.Abort()
			return
		}

		// Get user from database to ensure account is still active
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		user, err := am.userRepo.GetByID(ctx, claims.UserID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "UNAUTHORIZED",
					Message: "User account not found",
					Code:    "AUTH_USER_NOT_FOUND",
				})
			} else {
				logrus.Errorf("Error fetching user %s: %v", claims.UserID, err)
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error:   "INTERNAL_ERROR",
					Message: "Failed to validate authentication",
					Code:    "AUTH_VALIDATION_ERROR",
				})
			}
			c.Abort()
			return
		}

		// Check if user account is active
		if !user.IsActive {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "UNAUTHORIZED",
				Message: "User account is deactivated",
				Code:    "AUTH_USER_INACTIVE",
			})
			c.Abort()
			return
		}

		// Set user context
		c.Set("user", user)
		c.Set("userID", user.ID.Hex())
		c.Set("userEmail", user.Email)
		c.Set("userRole", claims.Role)

		// Update user last seen
		go am.updateUserLastSeen(user.ID.Hex())

		c.Next()
	})
}

// RequireRole validates user has specific role
func (am *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		userRole, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "UNAUTHORIZED",
				Message: "User role not found in context",
				Code:    "AUTH_ROLE_MISSING",
			})
			c.Abort()
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "INTERNAL_ERROR",
				Message: "Invalid role data type",
				Code:    "AUTH_ROLE_INVALID_TYPE",
			})
			c.Abort()
			return
		}

		// Check if user has required role
		hasRole := false
		for _, role := range roles {
			if roleStr == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "FORBIDDEN",
				Message: "Insufficient permissions",
				Code:    "AUTH_INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Next()
	})
}

// OptionalAuth validates token if present but doesn't require it
func (am *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		token := am.extractToken(c)
		if token == "" {
			c.Next()
			return
		}

		// Validate token
		claims, err := am.jwtService.ValidateToken(token)
		if err != nil {
			// Log but don't abort for optional auth
			logrus.Debugf("Optional auth - invalid token: %v", err)
			c.Next()
			return
		}

		// Check if token is not expired
		if claims.ExpiresAt.Before(time.Now()) {
			logrus.Debug("Optional auth - token expired")
			c.Next()
			return
		}

		// Get user from database
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		user, err := am.userRepo.GetByID(ctx, claims.UserID)
		if err != nil {
			logrus.Debugf("Optional auth - user not found: %v", err)
			c.Next()
			return
		}

		// Check if user account is active
		if !user.IsActive {
			logrus.Debug("Optional auth - user inactive")
			c.Next()
			return
		}

		// Set user context
		c.Set("user", user)
		c.Set("userID", user.ID.Hex())
		c.Set("userEmail", user.Email)
		c.Set("userRole", claims.Role)

		c.Next()
	})
}

// RequireVerification checks if user's email is verified
func (am *AuthMiddleware) RequireVerification() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "UNAUTHORIZED",
				Message: "User not authenticated",
				Code:    "AUTH_USER_NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		userModel, ok := user.(*models.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "INTERNAL_ERROR",
				Message: "Invalid user data type",
				Code:    "AUTH_USER_INVALID_TYPE",
			})
			c.Abort()
			return
		}

		if !userModel.IsVerified {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "FORBIDDEN",
				Message: "Email verification required",
				Code:    "AUTH_EMAIL_NOT_VERIFIED",
			})
			c.Abort()
			return
		}

		c.Next()
	})
}

// WebSocketAuth validates token for WebSocket connections
func (am *AuthMiddleware) WebSocketAuth(token string) (*models.User, error) {
	if token == "" {
		return nil, utils.NewValidationError("Authentication token required")
	}

	// Validate token
	claims, err := am.jwtService.ValidateToken(token)
	if err != nil {
		return nil, utils.NewValidationError("Invalid authentication token")
	}

	// Check if token is not expired
	if claims.ExpiresAt.Before(time.Now()) {
		return nil, utils.NewValidationError("Authentication token expired")
	}

	// Verify token type
	if claims.TokenType != "access" {
		return nil, utils.NewValidationError("Invalid token type")
	}

	// Get user from database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := am.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, utils.NewValidationError("User account not found")
		}
		return nil, err
	}

	// Check if user account is active
	if !user.IsActive {
		return nil, utils.NewValidationError("User account is deactivated")
	}

	// Update user last seen
	go am.updateUserLastSeen(user.ID.Hex())

	return user, nil
}

// extractToken extracts JWT token from request
func (am *AuthMiddleware) extractToken(c *gin.Context) string {
	// Check Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Check query parameter
	if token := c.Query("token"); token != "" {
		return token
	}

	// Check cookie
	if token, err := c.Cookie("auth_token"); err == nil {
		return token
	}

	return ""
}

// updateUserLastSeen updates user's last seen timestamp
func (am *AuthMiddleware) updateUserLastSeen(userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := am.userRepo.UpdateLastSeen(ctx, userID)
	if err != nil {
		logrus.Debugf("Failed to update last seen for user %s: %v", userID, err)
	}
}

// Helper functions for getting user data from context

// GetCurrentUser returns the current authenticated user from context
func GetCurrentUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	userModel, ok := user.(*models.User)
	return userModel, ok
}

// GetCurrentUserID returns the current authenticated user ID from context
func GetCurrentUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return "", false
	}

	userIDStr, ok := userID.(string)
	return userIDStr, ok
}

// MustGetCurrentUserID returns the current user ID or panics
func MustGetCurrentUserID(c *gin.Context) string {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		panic("user ID not found in context")
	}
	return userID
}
