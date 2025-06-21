// routes/notification.go
package routes

import (
	"ftrack/controllers"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// SetupNotificationRoutes configures notification related routes
func SetupNotificationRoutes(router *gin.RouterGroup, notificationController *controllers.NotificationController, redis *redis.Client) {
	notifications := router.Group("/notifications")

	// Basic notification operations
	notifications.GET("/", notificationController.GetNotifications)
	notifications.GET("/:notificationId", notificationController.GetNotification)
	notifications.PUT("/:notificationId/read", notificationController.MarkAsRead)
	notifications.PUT("/:notificationId/unread", notificationController.MarkAsUnread)
	notifications.DELETE("/:notificationId", notificationController.DeleteNotification)

	// Bulk notification operations
	bulk := notifications.Group("/bulk")
	{
		bulk.PUT("/read", notificationController.BulkMarkAsRead)
		bulk.PUT("/unread", notificationController.BulkMarkAsUnread)
		bulk.DELETE("/", notificationController.BulkDeleteNotifications)
		bulk.POST("/archive", notificationController.BulkArchiveNotifications)
	}

	// Notification filtering and organization
	filter := notifications.Group("/filter")
	{
		filter.GET("/unread", notificationController.GetUnreadNotifications)
		filter.GET("/read", notificationController.GetReadNotifications)
		filter.GET("/type/:type", notificationController.GetNotificationsByType)
		filter.GET("/priority/:priority", notificationController.GetNotificationsByPriority)
		filter.GET("/circle/:circleId", notificationController.GetCircleNotifications)
		filter.GET("/archived", notificationController.GetArchivedNotifications)
	}

	// Push notification management
	push := notifications.Group("/push")
	{
		push.GET("/settings", notificationController.GetPushSettings)
		push.PUT("/settings", notificationController.UpdatePushSettings)
		push.POST("/test", notificationController.SendTestNotification)
		push.POST("/register-device", notificationController.RegisterPushDevice)
		push.PUT("/device/:deviceId", notificationController.UpdatePushDevice)
		push.DELETE("/device/:deviceId", notificationController.UnregisterPushDevice)
		push.GET("/devices", notificationController.GetPushDevices)
	}

	// Notification preferences and settings
	preferences := notifications.Group("/preferences")
	{
		preferences.GET("/", notificationController.GetNotificationPreferences)
		preferences.PUT("/", notificationController.UpdateNotificationPreferences)
		preferences.GET("/types", notificationController.GetNotificationTypes)
		preferences.PUT("/types/:type", notificationController.UpdateTypePreferences)
		preferences.GET("/schedule", notificationController.GetNotificationSchedule)
		preferences.PUT("/schedule", notificationController.UpdateNotificationSchedule)
	}

	// Email notification settings
	email := notifications.Group("/email")
	{
		email.GET("/settings", notificationController.GetEmailSettings)
		email.PUT("/settings", notificationController.UpdateEmailSettings)
		email.POST("/verify", notificationController.VerifyEmailAddress)
		email.GET("/templates", notificationController.GetEmailTemplates)
		email.PUT("/templates/:templateId", notificationController.UpdateEmailTemplate)
		email.POST("/test", notificationController.SendTestEmail)
	}

	// SMS notification settings
	sms := notifications.Group("/sms")
	{
		sms.GET("/settings", notificationController.GetSMSSettings)
		sms.PUT("/settings", notificationController.UpdateSMSSettings)
		sms.POST("/verify", notificationController.VerifyPhoneNumber)
		sms.POST("/test", notificationController.SendTestSMS)
		sms.GET("/usage", notificationController.GetSMSUsage)
	}

	// In-app notification settings
	inapp := notifications.Group("/in-app")
	{
		inapp.GET("/settings", notificationController.GetInAppSettings)
		inapp.PUT("/settings", notificationController.UpdateInAppSettings)
		inapp.GET("/badges", notificationController.GetNotificationBadges)
		inapp.PUT("/badges/clear", notificationController.ClearNotificationBadges)
		inapp.GET("/sounds", notificationController.GetNotificationSounds)
		inapp.PUT("/sounds", notificationController.UpdateNotificationSounds)
	}

	// Notification channels and delivery
	channels := notifications.Group("/channels")
	{
		channels.GET("/", notificationController.GetNotificationChannels)
		channels.POST("/", notificationController.CreateNotificationChannel)
		channels.PUT("/:channelId", notificationController.UpdateNotificationChannel)
		channels.DELETE("/:channelId", notificationController.DeleteNotificationChannel)
		channels.POST("/:channelId/test", notificationController.TestNotificationChannel)
	}

	// Notification rules and automation
	rules := notifications.Group("/rules")
	{
		rules.GET("/", notificationController.GetNotificationRules)
		rules.POST("/", notificationController.CreateNotificationRule)
		rules.GET("/:ruleId", notificationController.GetNotificationRule)
		rules.PUT("/:ruleId", notificationController.UpdateNotificationRule)
		rules.DELETE("/:ruleId", notificationController.DeleteNotificationRule)
		rules.POST("/:ruleId/test", notificationController.TestNotificationRule)
	}

	// Do Not Disturb and quiet hours
	dnd := notifications.Group("/do-not-disturb")
	{
		dnd.GET("/status", notificationController.GetDoNotDisturbStatus)
		dnd.POST("/enable", notificationController.EnableDoNotDisturb)
		dnd.POST("/disable", notificationController.DisableDoNotDisturb)
		dnd.GET("/schedule", notificationController.GetQuietHours)
		dnd.PUT("/schedule", notificationController.UpdateQuietHours)
		dnd.GET("/exceptions", notificationController.GetDNDExceptions)
		dnd.PUT("/exceptions", notificationController.UpdateDNDExceptions)
	}

	// Notification templates
	templates := notifications.Group("/templates")
	{
		templates.GET("/", notificationController.GetNotificationTemplates)
		templates.POST("/", notificationController.CreateNotificationTemplate)
		templates.GET("/:templateId", notificationController.GetNotificationTemplate)
		templates.PUT("/:templateId", notificationController.UpdateNotificationTemplate)
		templates.DELETE("/:templateId", notificationController.DeleteNotificationTemplate)
		templates.POST("/:templateId/preview", notificationController.PreviewNotificationTemplate)
	}

	// Notification analytics and insights
	analytics := notifications.Group("/analytics")
	{
		analytics.GET("/stats", notificationController.GetNotificationStats)
		analytics.GET("/delivery", notificationController.GetDeliveryStats)
		analytics.GET("/engagement", notificationController.GetEngagementStats)
		analytics.GET("/trends", notificationController.GetNotificationTrends)
		analytics.GET("/performance", notificationController.GetNotificationPerformance)
	}

	// Notification history and audit
	history := notifications.Group("/history")
	{
		history.GET("/", notificationController.GetNotificationHistory)
		history.GET("/:notificationId/delivery", notificationController.GetDeliveryHistory)
		history.POST("/export", notificationController.ExportNotificationHistory)
		history.GET("/export/:exportId/download", notificationController.DownloadNotificationExport)
		history.DELETE("/cleanup", notificationController.CleanupOldNotifications)
	}

	// Notification subscriptions
	subscriptions := notifications.Group("/subscriptions")
	{
		subscriptions.GET("/", notificationController.GetNotificationSubscriptions)
		subscriptions.POST("/", notificationController.CreateNotificationSubscription)
		subscriptions.PUT("/:subscriptionId", notificationController.UpdateNotificationSubscription)
		subscriptions.DELETE("/:subscriptionId", notificationController.DeleteNotificationSubscription)
		subscriptions.GET("/topics", notificationController.GetNotificationTopics)
	}

	// Custom notification actions
	actions := notifications.Group("/actions")
	{
		actions.GET("/", notificationController.GetNotificationActions)
		actions.POST("/:notificationId/action/:actionId", notificationController.ExecuteNotificationAction)
		actions.PUT("/:notificationId/snooze", notificationController.SnoozeNotification)
		actions.PUT("/:notificationId/pin", notificationController.PinNotification)
		actions.PUT("/:notificationId/unpin", notificationController.UnpinNotification)
	}
}
