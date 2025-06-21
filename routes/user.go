// routes/user.go
package routes

import (
	"ftrack/controllers"
	"ftrack/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// SetupUserRoutes configures user-related routes
func SetupUserRoutes(router *gin.RouterGroup, userController *controllers.UserController, redis *redis.Client) {
	users := router.Group("/users")

	// Current user endpoints
	users.GET("/me", userController.GetCurrentUser)
	users.PUT("/me", userController.UpdateCurrentUser)
	users.DELETE("/me", userController.DeleteCurrentUser)
	users.GET("/me/profile", userController.GetProfile)
	users.PUT("/me/profile", userController.UpdateProfile)

	// Profile picture management
	profilePictures := users.Group("/me/profile-picture")
	profilePictures.Use(middleware.UploadRateLimit(redis))
	{
		profilePictures.POST("/", userController.UploadProfilePicture)
		profilePictures.DELETE("/", userController.DeleteProfilePicture)
		profilePictures.GET("/", userController.GetProfilePicture)
	}

	// User preferences and settings
	settings := users.Group("/me/settings")
	{
		settings.GET("/", userController.GetUserSettings)
		settings.PUT("/", userController.UpdateUserSettings)
		settings.GET("/privacy", userController.GetPrivacySettings)
		settings.PUT("/privacy", userController.UpdatePrivacySettings)
		settings.GET("/notifications", userController.GetNotificationSettings)
		settings.PUT("/notifications", userController.UpdateNotificationSettings)
		settings.GET("/location", userController.GetLocationSettings)
		settings.PUT("/location", userController.UpdateLocationSettings)
		settings.GET("/driving", userController.GetDrivingSettings)
		settings.PUT("/driving", userController.UpdateDrivingSettings)
	}

	// Emergency contacts
	emergency := users.Group("/me/emergency-contacts")
	{
		emergency.GET("/", userController.GetEmergencyContacts)
		emergency.POST("/", userController.AddEmergencyContact)
		emergency.PUT("/:contactId", userController.UpdateEmergencyContact)
		emergency.DELETE("/:contactId", userController.DeleteEmergencyContact)
		emergency.POST("/:contactId/verify", userController.VerifyEmergencyContact)
	}

	// Device management
	devices := users.Group("/me/devices")
	{
		devices.GET("/", userController.GetUserDevices)
		devices.POST("/register", userController.RegisterDevice)
		devices.PUT("/:deviceId", userController.UpdateDevice)
		devices.DELETE("/:deviceId", userController.UnregisterDevice)
		devices.POST("/:deviceId/push-test", userController.TestPushNotification)
	}

	// Social features
	social := users.Group("/")
	{
		social.GET("/search", userController.SearchUsers)
		social.GET("/:userId", userController.GetUserByID)
		social.GET("/:userId/profile", userController.GetPublicProfile)

		// Friend requests (if implemented)
		social.POST("/:userId/friend-request", userController.SendFriendRequest)
		social.PUT("/friend-requests/:requestId/accept", userController.AcceptFriendRequest)
		social.PUT("/friend-requests/:requestId/decline", userController.DeclineFriendRequest)
		social.GET("/friend-requests", userController.GetFriendRequests)
		social.GET("/friends", userController.GetFriends)
		social.DELETE("/friends/:userId", userController.RemoveFriend)
	}

	// Blocking and reporting
	moderation := users.Group("/me/moderation")
	{
		moderation.GET("/blocked", userController.GetBlockedUsers)
		moderation.POST("/block/:userId", userController.BlockUser)
		moderation.DELETE("/block/:userId", userController.UnblockUser)
		moderation.POST("/report/:userId", userController.ReportUser)
	}

	// Data export and privacy
	data := users.Group("/me/data")
	{
		data.GET("/export", userController.ExportUserData)
		data.GET("/export/status", userController.GetExportStatus)
		data.POST("/download/:exportId", userController.DownloadExport)
		data.DELETE("/purge", userController.RequestDataPurge)
	}

	// Account statistics
	stats := users.Group("/me/stats")
	{
		stats.GET("/", userController.GetUserStats)
		stats.GET("/activity", userController.GetActivityStats)
		stats.GET("/location", userController.GetLocationStats)
		stats.GET("/driving", userController.GetDrivingStats)
		stats.GET("/circles", userController.GetCircleStats)
	}
}
