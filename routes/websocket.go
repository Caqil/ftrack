// routes/websocket.go
package routes

import (
	"ftrack/controllers"
	"ftrack/middleware"

	"github.com/gin-gonic/gin"
)

// SetupWebSocketRoutes configures WebSocket related routes
func SetupWebSocketRoutes(router *gin.Engine, wsController *controllers.WebSocketController) {
	// Main WebSocket connection endpoint
	router.GET("/ws", wsController.HandleWebSocket)

	// WebSocket API endpoints for management and info
	ws := router.Group("/api/v1/ws")
	ws.Use(middleware.AuthMiddleware()) // Require authentication for WebSocket management

	// WebSocket connection management
	connections := ws.Group("/connections")
	{
		connections.GET("/", wsController.GetConnections)
		connections.GET("/me", wsController.GetMyConnections)
		connections.DELETE("/:connectionId", wsController.DisconnectConnection)
		connections.GET("/:connectionId/info", wsController.GetConnectionInfo)
		connections.POST("/:connectionId/ping", wsController.PingConnection)
	}

	// WebSocket message types and documentation
	messageTypes := ws.Group("/message-types")
	{
		messageTypes.GET("/", wsController.GetMessageTypes)
		messageTypes.GET("/:type", wsController.GetMessageType)
		messageTypes.GET("/:type/schema", wsController.GetMessageSchema)
		messageTypes.GET("/:type/examples", wsController.GetMessageExamples)
	}

	// WebSocket room/channel management
	rooms := ws.Group("/rooms")
	{
		rooms.GET("/", wsController.GetRooms)
		rooms.GET("/:roomId", wsController.GetRoom)
		rooms.GET("/:roomId/members", wsController.GetRoomMembers)
		rooms.POST("/:roomId/join", wsController.JoinRoom)
		rooms.POST("/:roomId/leave", wsController.LeaveRoom)
		rooms.GET("/:roomId/activity", wsController.GetRoomActivity)
	}

	// WebSocket broadcasting and messaging
	broadcast := ws.Group("/broadcast")
	{
		broadcast.POST("/message", wsController.BroadcastMessage)
		broadcast.POST("/notification", wsController.BroadcastNotification)
		broadcast.POST("/alert", wsController.BroadcastAlert)
		broadcast.POST("/room/:roomId", wsController.BroadcastToRoom)
		broadcast.POST("/user/:userId", wsController.BroadcastToUser)
		broadcast.POST("/circle/:circleId", wsController.BroadcastToCircle)
	}

	// WebSocket event handling and hooks
	events := ws.Group("/events")
	{
		events.GET("/", wsController.GetEventTypes)
		events.POST("/simulate/:eventType", wsController.SimulateEvent)
		events.GET("/history", wsController.GetEventHistory)
		events.GET("/stats", wsController.GetEventStats)
	}

	// WebSocket health and monitoring
	health := ws.Group("/health")
	{
		health.GET("/", wsController.GetWebSocketHealth)
		health.GET("/connections/count", wsController.GetConnectionCount)
		health.GET("/metrics", wsController.GetWebSocketMetrics)
		health.GET("/performance", wsController.GetPerformanceMetrics)
		health.POST("/diagnostics", wsController.RunDiagnostics)
	}

	// WebSocket authentication and security
	auth := ws.Group("/auth")
	{
		auth.POST("/validate-token", wsController.ValidateWebSocketToken)
		auth.GET("/sessions", wsController.GetWebSocketSessions)
		auth.DELETE("/sessions/:sessionId", wsController.RevokeWebSocketSession)
		auth.POST("/refresh-token", wsController.RefreshWebSocketToken)
	}

	// WebSocket rate limiting and throttling
	limits := ws.Group("/limits")
	{
		limits.GET("/", wsController.GetRateLimits)
		limits.PUT("/", wsController.UpdateRateLimits)
		limits.GET("/status/:connectionId", wsController.GetRateLimitStatus)
		limits.POST("/reset/:connectionId", wsController.ResetRateLimit)
	}

	// WebSocket configuration and settings
	config := ws.Group("/config")
	{
		config.GET("/", wsController.GetWebSocketConfig)
		config.PUT("/", wsController.UpdateWebSocketConfig)
		config.GET("/features", wsController.GetWebSocketFeatures)
		config.PUT("/features", wsController.UpdateWebSocketFeatures)
	}

	// WebSocket presence and user status
	presence := ws.Group("/presence")
	{
		presence.GET("/", wsController.GetOnlineUsers)
		presence.GET("/circle/:circleId", wsController.GetCirclePresence)
		presence.PUT("/status", wsController.UpdatePresenceStatus)
		presence.GET("/status/:userId", wsController.GetUserPresence)
		presence.POST("/heartbeat", wsController.SendHeartbeat)
	}

	// WebSocket typing indicators
	typing := ws.Group("/typing")
	{
		typing.POST("/start", wsController.StartTyping)
		typing.POST("/stop", wsController.StopTyping)
		typing.GET("/status/:circleId", wsController.GetTypingStatus)
		typing.GET("/indicators/:circleId", wsController.GetTypingIndicators)
	}

	// WebSocket push notifications integration
	push := ws.Group("/push")
	{
		push.POST("/register", wsController.RegisterForPush)
		push.DELETE("/unregister", wsController.UnregisterFromPush)
		push.GET("/settings", wsController.GetPushSettings)
		push.PUT("/settings", wsController.UpdatePushSettings)
	}

	// WebSocket debugging and development tools
	debug := ws.Group("/debug")
	{
		debug.GET("/logs", wsController.GetWebSocketLogs)
		debug.POST("/echo", wsController.EchoMessage)
		debug.GET("/client-info/:connectionId", wsController.GetClientInfo)
		debug.POST("/force-disconnect/:connectionId", wsController.ForceDisconnect)
		debug.GET("/message-queue/:connectionId", wsController.GetMessageQueue)
	}

	// WebSocket backup and recovery
	backup := ws.Group("/backup")
	{
		backup.POST("/save-state", wsController.SaveWebSocketState)
		backup.POST("/restore-state", wsController.RestoreWebSocketState)
		backup.GET("/state-info", wsController.GetStateInfo)
		backup.POST("/export-messages", wsController.ExportWebSocketMessages)
	}
}
