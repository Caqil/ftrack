// routes/auth.go
package routes

import (
	"ftrack/controllers"
	"ftrack/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// SetupAuthRoutes configures authentication-related routes
func SetupAuthRoutes(router *gin.RouterGroup, authController *controllers.AuthController, authMiddleware *middleware.AuthMiddleware) {
	auth := router.Group("/auth")

	// Public authentication endpoints
	auth.POST("/register", authController.Register)
	auth.POST("/login", authController.Login)
	auth.POST("/forgot-password", authController.ForgotPassword)
	auth.POST("/reset-password", authController.ResetPassword)
	auth.POST("/verify-email", authController.VerifyEmail)
	auth.POST("/resend-verification", authController.ResendVerification)

	// Token management
	auth.POST("/refresh", authController.RefreshToken)
	auth.POST("/logout", authController.Logout)

	// OAuth endpoints
	oauth := auth.Group("/oauth")
	{
		oauth.GET("/google", authController.GoogleOAuth)
		oauth.GET("/google/callback", authController.GoogleOAuthCallback)
		oauth.GET("/apple", authController.AppleOAuth)
		oauth.GET("/apple/callback", authController.AppleOAuthCallback)
		oauth.GET("/facebook", authController.FacebookOAuth)
		oauth.GET("/facebook/callback", authController.FacebookOAuthCallback)
	}

	// Protected authentication endpoints (require valid token)
	protected := auth.Group("/")
	protected.Use(authMiddleware.RequireAuth()) // ✅ Now authMiddleware is available
	{
		// Current user token operations
		protected.POST("/validate", authController.ValidateToken)
		protected.POST("/logout-all", authController.LogoutAllDevices)
		protected.GET("/sessions", authController.GetActiveSessions)
		protected.DELETE("/sessions/:sessionId", authController.RevokeSession)

		// Password management
		protected.POST("/change-password", authController.ChangePassword)
		protected.POST("/verify-password", authController.VerifyPassword)

		// Two-factor authentication
		protected.POST("/2fa/setup", authController.Setup2FA)
		protected.POST("/2fa/verify", authController.Verify2FA)
		protected.POST("/2fa/disable", authController.Disable2FA)
		protected.GET("/2fa/backup-codes", authController.GetBackupCodes)
		protected.POST("/2fa/regenerate-codes", authController.RegenerateBackupCodes)

		// Account security
		protected.GET("/security/overview", authController.SecurityOverview)
		protected.GET("/security/audit-log", authController.GetAuditLog)
		protected.POST("/security/report-suspicious", authController.ReportSuspiciousActivity)
	}
}

// In routes/auth.go
func SetupAuthMiddleware(router *gin.RouterGroup, redis *redis.Client) {
	// Rate limiting for authentication endpoints
	router.Use(middleware.AuthRateLimit(redis))

	// Additional security for sensitive auth operations
	sensitiveOps := router.Group("/")
	sensitiveOps.Use(middleware.CustomRateLimit(redis, 3, 5*time.Minute, middleware.StrategyIP, "strict_auth")) // ✅ Use CustomRateLimit
	{
		sensitiveOps.POST("/forgot-password")
		sensitiveOps.POST("/reset-password")
		sensitiveOps.POST("/change-password")
		sensitiveOps.POST("/2fa/setup")
	}
}
