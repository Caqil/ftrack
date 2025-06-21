// routes/location.go
package routes

import (
	"ftrack/controllers"
	"ftrack/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// SetupLocationRoutes configures location tracking related routes
func SetupLocationRoutes(router *gin.RouterGroup, locationController *controllers.LocationController, redis *redis.Client) {
	location := router.Group("/location")

	// Location updates and tracking
	tracking := location.Group("/tracking")
	tracking.Use(middleware.MessageRateLimit(redis)) // Rate limit for frequent updates
	{
		tracking.POST("/update", locationController.UpdateLocation)
		tracking.POST("/bulk-update", locationController.BulkUpdateLocation)
		tracking.GET("/current", locationController.GetCurrentLocation)
		tracking.GET("/history", locationController.GetLocationHistory)
		tracking.DELETE("/history", locationController.ClearLocationHistory)
	}

	// Location sharing and privacy
	sharing := location.Group("/sharing")
	{
		sharing.GET("/settings", locationController.GetLocationSettings)
		sharing.PUT("/settings", locationController.UpdateLocationSettings)
		sharing.GET("/permissions", locationController.GetSharingPermissions)
		sharing.PUT("/permissions", locationController.UpdateSharingPermissions)
		sharing.POST("/temporary-share", locationController.CreateTemporaryShare)
		sharing.GET("/temporary-shares", locationController.GetTemporaryShares)
		sharing.DELETE("/temporary-shares/:shareId", locationController.DeleteTemporaryShare)
	}

	// Nearby users and proximity
	proximity := location.Group("/proximity")
	{
		proximity.GET("/nearby", locationController.GetNearbyUsers)
		proximity.GET("/nearby/circles/:circleId", locationController.GetNearbyCircleMembers)
		proximity.POST("/alerts", locationController.CreateProximityAlert)
		proximity.GET("/alerts", locationController.GetProximityAlerts)
		proximity.PUT("/alerts/:alertId", locationController.UpdateProximityAlert)
		proximity.DELETE("/alerts/:alertId", locationController.DeleteProximityAlert)
	}

	// Trip tracking and management
	trips := location.Group("/trips")
	{
		trips.GET("/", locationController.GetTrips)
		trips.POST("/start", locationController.StartTrip)
		trips.PUT("/:tripId/end", locationController.EndTrip)
		trips.GET("/:tripId", locationController.GetTrip)
		trips.PUT("/:tripId", locationController.UpdateTrip)
		trips.DELETE("/:tripId", locationController.DeleteTrip)
		trips.GET("/:tripId/route", locationController.GetTripRoute)
		trips.GET("/:tripId/stats", locationController.GetTripStats)
		trips.POST("/:tripId/share", locationController.ShareTrip)
	}

	// Driving detection and analysis
	driving := location.Group("/driving")
	{
		driving.GET("/status", locationController.GetDrivingStatus)
		driving.POST("/start", locationController.StartDriving)
		driving.POST("/stop", locationController.StopDriving)
		driving.GET("/sessions", locationController.GetDrivingSessions)
		driving.GET("/sessions/:sessionId", locationController.GetDrivingSession)
		driving.GET("/reports", locationController.GetDrivingReports)
		driving.GET("/reports/:reportId", locationController.GetDrivingReport)
		driving.GET("/score", locationController.GetDrivingScore)
		driving.GET("/events", locationController.GetDrivingEvents)
		driving.POST("/events", locationController.ReportDrivingEvent)
	}

	// Location analytics and statistics
	analytics := location.Group("/analytics")
	{
		analytics.GET("/stats", locationController.GetLocationStats)
		analytics.GET("/heatmap", locationController.GetLocationHeatmap)
		analytics.GET("/patterns", locationController.GetLocationPatterns)
		analytics.GET("/insights", locationController.GetLocationInsights)
		analytics.GET("/timeline", locationController.GetLocationTimeline)
		analytics.GET("/summary/:period", locationController.GetLocationSummary)
	}

	// Geofencing and place detection
	geofencing := location.Group("/geofencing")
	{
		geofencing.GET("/events", locationController.GetGeofenceEvents)
		geofencing.GET("/events/:eventId", locationController.GetGeofenceEvent)
		geofencing.POST("/test", locationController.TestGeofence)
		geofencing.GET("/status", locationController.GetGeofenceStatus)
	}

	// Location data management
	data := location.Group("/data")
	{
		data.POST("/export", locationController.ExportLocationData)
		data.GET("/export/status", locationController.GetExportStatus)
		data.GET("/export/:exportId/download", locationController.DownloadLocationExport)
		data.DELETE("/purge", locationController.PurgeLocationData)
		data.GET("/usage", locationController.GetDataUsage)
	}

	// Emergency location features
	emergency := location.Group("/emergency")
	{
		emergency.POST("/share", locationController.ShareEmergencyLocation)
		emergency.GET("/last-known", locationController.GetLastKnownLocation)
		emergency.POST("/ping", locationController.SendLocationPing)
		emergency.GET("/ping/:pingId", locationController.GetLocationPing)
	}

	// Location accuracy and calibration
	calibration := location.Group("/calibration")
	{
		calibration.GET("/accuracy", locationController.GetLocationAccuracy)
		calibration.POST("/calibrate", locationController.CalibrateLocation)
		calibration.GET("/providers", locationController.GetLocationProviders)
		calibration.PUT("/providers", locationController.UpdateLocationProviders)
	}

	// Battery optimization for location tracking
	battery := location.Group("/battery")
	{
		battery.GET("/optimization", locationController.GetBatteryOptimization)
		battery.PUT("/optimization", locationController.UpdateBatteryOptimization)
		battery.GET("/usage", locationController.GetBatteryUsage)
		battery.POST("/mode/:mode", locationController.SetPowerMode)
	}
}
