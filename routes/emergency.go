// routes/emergency.go
package routes

import (
	"ftrack/controllers"
	"ftrack/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// SetupEmergencyRoutes configures emergency alert related routes
func SetupEmergencyRoutes(router *gin.RouterGroup, emergencyController *controllers.EmergencyController, redis *redis.Client) {
	emergency := router.Group("/emergency")

	// Emergency alert management
	alerts := emergency.Group("/alerts")
	alerts.Use(middleware.EmergencyRateLimit(redis)) // Special rate limiting for emergency features
	{
		alerts.GET("/", emergencyController.GetEmergencyAlerts)
		alerts.POST("/", emergencyController.CreateEmergencyAlert)
		alerts.GET("/:alertId", emergencyController.GetEmergencyAlert)
		alerts.PUT("/:alertId", emergencyController.UpdateEmergencyAlert)
		alerts.DELETE("/:alertId", emergencyController.DeleteEmergencyAlert)
		alerts.POST("/:alertId/dismiss", emergencyController.DismissEmergencyAlert)
		alerts.POST("/:alertId/resolve", emergencyController.ResolveEmergencyAlert)
	}

	// SOS and panic button functionality
	sos := emergency.Group("/sos")
	{
		sos.POST("/trigger", emergencyController.TriggerSOS)
		sos.POST("/cancel", emergencyController.CancelSOS)
		sos.GET("/status", emergencyController.GetSOSStatus)
		sos.PUT("/settings", emergencyController.UpdateSOSSettings)
		sos.GET("/settings", emergencyController.GetSOSSettings)
		sos.POST("/test", emergencyController.TestSOS)
	}

	// Crash detection
	crash := emergency.Group("/crash")
	{
		crash.POST("/detect", emergencyController.DetectCrash)
		crash.POST("/:detectionId/confirm", emergencyController.ConfirmCrash)
		crash.POST("/:detectionId/false-alarm", emergencyController.MarkFalseAlarm)
		crash.GET("/history", emergencyController.GetCrashHistory)
		crash.GET("/settings", emergencyController.GetCrashDetectionSettings)
		crash.PUT("/settings", emergencyController.UpdateCrashDetectionSettings)
		crash.POST("/calibrate", emergencyController.CalibrateCrashDetection)
	}

	// Emergency contacts management
	contacts := emergency.Group("/contacts")
	{
		contacts.GET("/", emergencyController.GetEmergencyContacts)
		contacts.POST("/", emergencyController.AddEmergencyContact)
		contacts.GET("/:contactId", emergencyController.GetEmergencyContact)
		contacts.PUT("/:contactId", emergencyController.UpdateEmergencyContact)
		contacts.DELETE("/:contactId", emergencyController.DeleteEmergencyContact)
		contacts.POST("/:contactId/verify", emergencyController.VerifyEmergencyContact)
		contacts.POST("/:contactId/notify", emergencyController.NotifyEmergencyContact)
		contacts.GET("/:contactId/history", emergencyController.GetContactHistory)
	}

	// Emergency services integration
	services := emergency.Group("/services")
	{
		services.GET("/", emergencyController.GetNearbyEmergencyServices)
		services.GET("/hospitals", emergencyController.GetNearbyHospitals)
		services.GET("/police", emergencyController.GetNearbyPoliceStations)
		services.GET("/fire", emergencyController.GetNearbyFireStations)
		services.POST("/call/:serviceType", emergencyController.InitiateEmergencyCall)
		services.GET("/numbers", emergencyController.GetEmergencyNumbers)
		services.PUT("/numbers", emergencyController.UpdateEmergencyNumbers)
	}

	// Location sharing during emergencies
	location := emergency.Group("/location")
	{
		location.POST("/share", emergencyController.ShareEmergencyLocation)
		location.GET("/shared", emergencyController.GetSharedEmergencyLocations)
		location.PUT("/:shareId", emergencyController.UpdateLocationShare)
		location.DELETE("/:shareId", emergencyController.StopLocationShare)
		location.GET("/:shareId/track", emergencyController.TrackEmergencyLocation)
	}

	// Emergency response and assistance
	response := emergency.Group("/response")
	{
		response.POST("/:alertId/respond", emergencyController.RespondToEmergency)
		response.GET("/:alertId/responses", emergencyController.GetEmergencyResponses)
		response.PUT("/:alertId/responses/:responseId", emergencyController.UpdateEmergencyResponse)
		response.POST("/:alertId/request-help", emergencyController.RequestHelp)
		response.POST("/:alertId/offer-help", emergencyController.OfferHelp)
	}

	// Check-in safety features
	checkin := emergency.Group("/checkin")
	{
		checkin.POST("/safe", emergencyController.CheckInSafe)
		checkin.POST("/not-safe", emergencyController.CheckInNotSafe)
		checkin.GET("/status", emergencyController.GetCheckInStatus)
		checkin.PUT("/settings", emergencyController.UpdateCheckInSettings)
		checkin.GET("/settings", emergencyController.GetCheckInSettings)
		checkin.POST("/request/:userId", emergencyController.RequestCheckIn)
		checkin.GET("/requests", emergencyController.GetCheckInRequests)
	}

	// Emergency timeline and history
	history := emergency.Group("/history")
	{
		history.GET("/", emergencyController.GetEmergencyHistory)
		history.GET("/:alertId/timeline", emergencyController.GetEmergencyTimeline)
		history.GET("/stats", emergencyController.GetEmergencyStats)
		history.POST("/export", emergencyController.ExportEmergencyHistory)
		history.GET("/export/:exportId/download", emergencyController.DownloadEmergencyExport)
	}

	// Emergency settings and configuration
	settings := emergency.Group("/settings")
	{
		settings.GET("/", emergencyController.GetEmergencySettings)
		settings.PUT("/", emergencyController.UpdateEmergencySettings)
		settings.GET("/notifications", emergencyController.GetEmergencyNotificationSettings)
		settings.PUT("/notifications", emergencyController.UpdateEmergencyNotificationSettings)
		settings.GET("/automation", emergencyController.GetEmergencyAutomationSettings)
		settings.PUT("/automation", emergencyController.UpdateEmergencyAutomationSettings)
	}

	// Emergency drills and testing
	drills := emergency.Group("/drills")
	{
		drills.GET("/", emergencyController.GetEmergencyDrills)
		drills.POST("/", emergencyController.CreateEmergencyDrill)
		drills.POST("/:drillId/start", emergencyController.StartEmergencyDrill)
		drills.POST("/:drillId/complete", emergencyController.CompleteEmergencyDrill)
		drills.GET("/:drillId/results", emergencyController.GetDrillResults)
		drills.DELETE("/:drillId", emergencyController.DeleteEmergencyDrill)
	}

	// Medical information
	medical := emergency.Group("/medical")
	{
		medical.GET("/", emergencyController.GetMedicalInformation)
		medical.PUT("/", emergencyController.UpdateMedicalInformation)
		medical.GET("/allergies", emergencyController.GetAllergies)
		medical.PUT("/allergies", emergencyController.UpdateAllergies)
		medical.GET("/medications", emergencyController.GetMedications)
		medical.PUT("/medications", emergencyController.UpdateMedications)
		medical.GET("/conditions", emergencyController.GetMedicalConditions)
		medical.PUT("/conditions", emergencyController.UpdateMedicalConditions)
	}

	// Emergency broadcast system
	broadcast := emergency.Group("/broadcast")
	{
		broadcast.POST("/", emergencyController.BroadcastEmergency)
		broadcast.GET("/", emergencyController.GetEmergencyBroadcasts)
		broadcast.PUT("/:broadcastId", emergencyController.UpdateEmergencyBroadcast)
		broadcast.DELETE("/:broadcastId", emergencyController.DeleteEmergencyBroadcast)
		broadcast.POST("/:broadcastId/acknowledge", emergencyController.AcknowledgeBroadcast)
	}
}
