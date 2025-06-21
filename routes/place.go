// routes/place.go
package routes

import (
	"ftrack/controllers"
	"ftrack/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// SetupPlaceRoutes configures place and geofence related routes
func SetupPlaceRoutes(router *gin.RouterGroup, placeController *controllers.PlaceController, redis *redis.Client) {
	places := router.Group("/places")

	// Basic place operations
	places.GET("/", placeController.GetPlaces)
	places.POST("/", placeController.CreatePlace)
	places.GET("/:placeId", placeController.GetPlace)
	places.PUT("/:placeId", placeController.UpdatePlace)
	places.DELETE("/:placeId", placeController.DeletePlace)

	// Place categories and organization
	categories := places.Group("/categories")
	{
		categories.GET("/", placeController.GetPlaceCategories)
		categories.POST("/", placeController.CreatePlaceCategory)
		categories.PUT("/:categoryId", placeController.UpdatePlaceCategory)
		categories.DELETE("/:categoryId", placeController.DeletePlaceCategory)
		categories.GET("/:categoryId/places", placeController.GetPlacesByCategory)
	}

	// Place search and discovery
	search := places.Group("/search")
	{
		search.GET("/", placeController.SearchPlaces)
		search.GET("/nearby", placeController.SearchNearbyPlaces)
		search.GET("/popular", placeController.GetPopularPlaces)
		search.GET("/recent", placeController.GetRecentPlaces)
		search.GET("/favorites", placeController.GetFavoritePlaces)
		search.POST("/advanced", placeController.AdvancedPlaceSearch)
	}

	// Geofencing and automation
	geofencing := places.Group("/:placeId/geofencing")
	{
		geofencing.GET("/settings", placeController.GetGeofenceSettings)
		geofencing.PUT("/settings", placeController.UpdateGeofenceSettings)
		geofencing.POST("/test", placeController.TestGeofence)
		geofencing.GET("/events", placeController.GetGeofenceEvents)
		geofencing.GET("/activity", placeController.GetGeofenceActivity)
	}

	// Place notifications and alerts
	notifications := places.Group("/:placeId/notifications")
	{
		notifications.GET("/", placeController.GetPlaceNotifications)
		notifications.PUT("/", placeController.UpdatePlaceNotifications)
		notifications.POST("/test", placeController.TestPlaceNotification)
		notifications.GET("/history", placeController.GetNotificationHistory)
	}

	// Place sharing and permissions
	sharing := places.Group("/:placeId/sharing")
	{
		sharing.GET("/", placeController.GetPlaceSharing)
		sharing.PUT("/", placeController.UpdatePlaceSharing)
		sharing.POST("/invite", placeController.InviteToPlace)
		sharing.GET("/members", placeController.GetPlaceMembers)
		sharing.PUT("/members/:userId", placeController.UpdatePlaceMember)
		sharing.DELETE("/members/:userId", placeController.RemovePlaceMember)
	}

	// Place visit tracking and analytics
	visits := places.Group("/:placeId/visits")
	{
		visits.GET("/", placeController.GetPlaceVisits)
		visits.POST("/", placeController.RecordPlaceVisit)
		visits.GET("/:visitId", placeController.GetPlaceVisit)
		visits.PUT("/:visitId", placeController.UpdatePlaceVisit)
		visits.DELETE("/:visitId", placeController.DeletePlaceVisit)
		visits.GET("/stats", placeController.GetVisitStats)
	}

	// Place hours and availability
	hours := places.Group("/:placeId/hours")
	{
		hours.GET("/", placeController.GetPlaceHours)
		hours.PUT("/", placeController.UpdatePlaceHours)
		hours.GET("/current", placeController.GetCurrentStatus)
		hours.POST("/override", placeController.CreateHoursOverride)
		hours.DELETE("/override/:overrideId", placeController.DeleteHoursOverride)
	}

	// Place automation and rules
	automation := places.Group("/:placeId/automation")
	{
		automation.GET("/rules", placeController.GetAutomationRules)
		automation.POST("/rules", placeController.CreateAutomationRule)
		automation.PUT("/rules/:ruleId", placeController.UpdateAutomationRule)
		automation.DELETE("/rules/:ruleId", placeController.DeleteAutomationRule)
		automation.POST("/rules/:ruleId/test", placeController.TestAutomationRule)
		automation.GET("/triggers", placeController.GetAvailableTriggers)
		automation.GET("/actions", placeController.GetAvailableActions)
	}

	// Place photos and media
	media := places.Group("/:placeId/media")
	media.Use(middleware.UploadRateLimit(redis))
	{
		media.GET("/", placeController.GetPlaceMedia)
		media.POST("/", placeController.UploadPlaceMedia)
		media.DELETE("/:mediaId", placeController.DeletePlaceMedia)
		media.PUT("/:mediaId", placeController.UpdatePlaceMedia)
		media.GET("/:mediaId/thumbnail", placeController.GetMediaThumbnail)
	}

	// Place reviews and ratings
	reviews := places.Group("/:placeId/reviews")
	{
		reviews.GET("/", placeController.GetPlaceReviews)
		reviews.POST("/", placeController.CreatePlaceReview)
		reviews.PUT("/:reviewId", placeController.UpdatePlaceReview)
		reviews.DELETE("/:reviewId", placeController.DeletePlaceReview)
		reviews.GET("/stats", placeController.GetReviewStats)
		reviews.POST("/:reviewId/helpful", placeController.MarkReviewHelpful)
	}

	// Place check-ins and social features
	checkins := places.Group("/:placeId/checkins")
	{
		checkins.GET("/", placeController.GetPlaceCheckins)
		checkins.POST("/", placeController.CheckInToPlace)
		checkins.GET("/:checkinId", placeController.GetCheckin)
		checkins.PUT("/:checkinId", placeController.UpdateCheckin)
		checkins.DELETE("/:checkinId", placeController.DeleteCheckin)
		checkins.GET("/leaderboard", placeController.GetCheckinLeaderboard)
	}

	// Place recommendations and suggestions
	recommendations := places.Group("/recommendations")
	{
		recommendations.GET("/", placeController.GetPlaceRecommendations)
		recommendations.GET("/nearby", placeController.GetNearbyRecommendations)
		recommendations.GET("/trending", placeController.GetTrendingPlaces)
		recommendations.GET("/similar/:placeId", placeController.GetSimilarPlaces)
		recommendations.POST("/feedback", placeController.ProvideFeedback)
	}

	// Place collections and lists
	collections := places.Group("/collections")
	{
		collections.GET("/", placeController.GetPlaceCollections)
		collections.POST("/", placeController.CreatePlaceCollection)
		collections.GET("/:collectionId", placeController.GetPlaceCollection)
		collections.PUT("/:collectionId", placeController.UpdatePlaceCollection)
		collections.DELETE("/:collectionId", placeController.DeletePlaceCollection)
		collections.POST("/:collectionId/places/:placeId", placeController.AddPlaceToCollection)
		collections.DELETE("/:collectionId/places/:placeId", placeController.RemovePlaceFromCollection)
	}

	// Place import and export
	data := places.Group("/data")
	{
		data.POST("/import", placeController.ImportPlaces)
		data.POST("/export", placeController.ExportPlaces)
		data.GET("/export/:exportId/download", placeController.DownloadPlaceExport)
		data.GET("/templates", placeController.GetImportTemplates)
		data.POST("/bulk-create", placeController.BulkCreatePlaces)
	}

	// Place templates and presets
	templates := places.Group("/templates")
	{
		templates.GET("/", placeController.GetPlaceTemplates)
		templates.POST("/", placeController.CreatePlaceTemplate)
		templates.GET("/:templateId", placeController.GetPlaceTemplate)
		templates.PUT("/:templateId", placeController.UpdatePlaceTemplate)
		templates.DELETE("/:templateId", placeController.DeletePlaceTemplate)
		templates.POST("/:templateId/use", placeController.UsePlaceTemplate)
	}

	// Place statistics and analytics
	analytics := places.Group("/analytics")
	{
		analytics.GET("/stats", placeController.GetPlaceStats)
		analytics.GET("/usage", placeController.GetPlaceUsageStats)
		analytics.GET("/trends", placeController.GetPlaceTrends)
		analytics.GET("/heatmap", placeController.GetPlaceHeatmap)
		analytics.GET("/insights", placeController.GetPlaceInsights)
	}
}
