// routes/circle.go
package routes

import (
	"ftrack/controllers"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// SetupCircleRoutes configures circle (family group) related routes
func SetupCircleRoutes(router *gin.RouterGroup, circleController *controllers.CircleController, redis *redis.Client) {
	circles := router.Group("/circles")

	// Circle CRUD operations
	circles.GET("/", circleController.GetUserCircles)
	circles.POST("/", circleController.CreateCircle)
	circles.GET("/:circleId", circleController.GetCircle)
	circles.PUT("/:circleId", circleController.UpdateCircle)
	circles.DELETE("/:circleId", circleController.DeleteCircle)

	// Circle invitation and joining
	invitations := circles.Group("/:circleId/invitations")
	{
		invitations.GET("/", circleController.GetCircleInvitations)
		invitations.POST("/", circleController.CreateInvitation)
		invitations.GET("/:invitationId", circleController.GetInvitation)
		invitations.PUT("/:invitationId", circleController.UpdateInvitation)
		invitations.DELETE("/:invitationId", circleController.DeleteInvitation)
		invitations.POST("/:invitationId/resend", circleController.ResendInvitation)
	}

	// Join circle operations
	join := circles.Group("/join")
	{
		join.POST("/by-code", circleController.JoinByInviteCode)
		join.POST("/by-invitation/:invitationId", circleController.JoinByInvitation)
		join.POST("/request/:circleId", circleController.RequestToJoin)
	}

	// Member management
	members := circles.Group("/:circleId/members")
	{
		members.GET("/", circleController.GetCircleMembers)
		members.GET("/:userId", circleController.GetCircleMember)
		members.PUT("/:userId", circleController.UpdateCircleMember)
		members.DELETE("/:userId", circleController.RemoveCircleMember)
		members.POST("/:userId/promote", circleController.PromoteMember)
		members.POST("/:userId/demote", circleController.DemoteMember)
		members.PUT("/:userId/permissions", circleController.UpdateMemberPermissions)
		members.GET("/:userId/activity", circleController.GetMemberActivity)
	}

	// Join requests management
	requests := circles.Group("/:circleId/requests")
	{
		requests.GET("/", circleController.GetJoinRequests)
		requests.PUT("/:requestId/approve", circleController.ApproveJoinRequest)
		requests.PUT("/:requestId/decline", circleController.DeclineJoinRequest)
		requests.DELETE("/:requestId", circleController.DeleteJoinRequest)
	}

	// Circle settings and configuration
	settings := circles.Group("/:circleId/settings")
	{
		settings.GET("/", circleController.GetCircleSettings)
		settings.PUT("/", circleController.UpdateCircleSettings)
		settings.GET("/privacy", circleController.GetPrivacySettings)
		settings.PUT("/privacy", circleController.UpdatePrivacySettings)
		settings.GET("/permissions", circleController.GetPermissionSettings)
		settings.PUT("/permissions", circleController.UpdatePermissionSettings)
		settings.GET("/notifications", circleController.GetNotificationSettings)
		settings.PUT("/notifications", circleController.UpdateNotificationSettings)
	}

	// Circle activity and monitoring
	activity := circles.Group("/:circleId/activity")
	{
		activity.GET("/", circleController.GetCircleActivity)
		activity.GET("/feed", circleController.GetActivityFeed)
		activity.GET("/locations", circleController.GetMemberLocations)
		activity.GET("/timeline", circleController.GetActivityTimeline)
		activity.GET("/events", circleController.GetCircleEvents)
	}

	// Circle statistics and analytics
	stats := circles.Group("/:circleId/stats")
	{
		stats.GET("/", circleController.GetCircleStats)
		stats.GET("/overview", circleController.GetStatsOverview)
		stats.GET("/location", circleController.GetLocationStats)
		stats.GET("/driving", circleController.GetDrivingStats)
		stats.GET("/places", circleController.GetPlaceStats)
		stats.GET("/safety", circleController.GetSafetyStats)
	}

	// Circle places and geofences
	places := circles.Group("/:circleId/places")
	{
		places.GET("/", circleController.GetCirclePlaces)
		places.POST("/", circleController.CreateCirclePlace)
		places.GET("/:placeId", circleController.GetCirclePlace)
		places.PUT("/:placeId", circleController.UpdateCirclePlace)
		places.DELETE("/:placeId", circleController.DeleteCirclePlace)
		places.GET("/:placeId/activity", circleController.GetPlaceActivity)
	}

	// Circle communication
	communication := circles.Group("/:circleId/communication")
	{
		communication.GET("/announcements", circleController.GetAnnouncements)
		communication.POST("/announcements", circleController.CreateAnnouncement)
		communication.PUT("/announcements/:announcementId", circleController.UpdateAnnouncement)
		communication.DELETE("/announcements/:announcementId", circleController.DeleteAnnouncement)
		communication.POST("/broadcast", circleController.BroadcastMessage)
	}

	// Circle backup and export
	backup := circles.Group("/:circleId/backup")
	{
		backup.POST("/export", circleController.ExportCircleData)
		backup.GET("/export/status", circleController.GetExportStatus)
		backup.GET("/export/:exportId/download", circleController.DownloadExport)
	}

	// Leave circle
	circles.POST("/:circleId/leave", circleController.LeaveCircle)

	// Circle discovery and public features
	discovery := circles.Group("/discovery")
	{
		discovery.GET("/public", circleController.GetPublicCircles)
		discovery.GET("/recommended", circleController.GetRecommendedCircles)
		discovery.POST("/search", circleController.SearchPublicCircles)
	}
}
