package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"
	"ftrack/websocket"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type WebSocketController struct {
	hub         *websocket.Hub
	authService *services.AuthService
	upgrader    websocket.Upgrader
}

func NewWebSocketController(hub *websocket.Hub, authService *services.AuthService) *WebSocketController {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// In production, implement proper origin checking
			return true
		},
	}

	return &WebSocketController{
		hub:         hub,
		authService: authService,
		upgrader:    upgrader,
	}
}

// HandleWebSocket handles WebSocket connections
// @Summary WebSocket endpoint
// @Description Establish WebSocket connection for real-time communication
// @Tags WebSocket
// @Param token query string true "Authentication token"
// @Success 101 "Switching Protocols"
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /ws [get]
func (wsc *WebSocketController) HandleWebSocket(c *gin.Context) {
	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		utils.UnauthorizedResponse(c, "Authentication token is required")
		return
	}

	// Validate token
	claims, err := wsc.authService.ValidateToken(token)
	if err != nil {
		logrus.Errorf("WebSocket authentication failed: %v", err)
		utils.UnauthorizedResponse(c, "Invalid authentication token")
		return
	}

	// Get user information
	user, err := wsc.authService.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		logrus.Errorf("Failed to get user for WebSocket: %v", err)
		utils.UnauthorizedResponse(c, "User not found")
		return
	}

	// Check if user is active
	if !user.IsActive {
		utils.UnauthorizedResponse(c, "Account is deactivated")
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := wsc.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade WebSocket connection: %v", err)
		utils.BadRequestResponse(c, "Failed to establish WebSocket connection")
		return
	}

	// Create new client
	client := websocket.NewClient(wsc.hub, conn, user)

	// Register client with hub
	wsc.hub.Register <- client

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()

	logrus.Infof("WebSocket connection established for user: %s", user.ID.Hex())
}

// GetConnectedUsers gets currently connected users (admin only)
// @Summary Get connected users
// @Description Get list of currently connected users (admin endpoint)
// @Tags WebSocket
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.ConnectedUser}
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /ws/connected [get]
func (wsc *WebSocketController) GetConnectedUsers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Check if user has admin privileges
	role := c.GetString("role")
	if role != "admin" {
		utils.ForbiddenResponse(c, "Admin access required")
		return
	}

	connectedUsers := wsc.hub.GetConnectedUsers()
	utils.SuccessResponse(c, "Connected users retrieved successfully", connectedUsers)
}

// GetConnectionStats gets WebSocket connection statistics (admin only)
// @Summary Get connection statistics
// @Description Get WebSocket connection statistics (admin endpoint)
// @Tags WebSocket
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.ConnectionStats}
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /ws/stats [get]
func (wsc *WebSocketController) GetConnectionStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Check if user has admin privileges
	role := c.GetString("role")
	if role != "admin" {
		utils.ForbiddenResponse(c, "Admin access required")
		return
	}

	stats := wsc.hub.GetConnectionStats()
	utils.SuccessResponse(c, "Connection statistics retrieved successfully", stats)
}

// BroadcastMessage sends a message to all connected users (admin only)
// @Summary Broadcast message
// @Description Send a message to all connected users (admin endpoint)
// @Tags WebSocket
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.BroadcastMessageRequest true "Broadcast message data"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /ws/broadcast [post]
func (wsc *WebSocketController) BroadcastMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Check if user has admin privileges
	role := c.GetString("role")
	if role != "admin" {
		utils.ForbiddenResponse(c, "Admin access required")
		return
	}

	var req models.BroadcastMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	// Create broadcast message
	message := models.WSMessage{
		Type: req.Type,
		Data: req.Data,
	}

	// Send to hub for broadcasting
	broadcastMsg := websocket.BroadcastMessage{
		Message: message,
		Filter: websocket.MessageFilter{
			CircleIDs:    req.CircleIDs,
			UserIDs:      req.UserIDs,
			ExcludeUsers: req.ExcludeUsers,
		},
	}

	select {
	case wsc.hub.Broadcast <- broadcastMsg:
		utils.SuccessResponse(c, "Message broadcasted successfully", nil)
	default:
		utils.InternalServerErrorResponse(c, "Failed to broadcast message - channel full")
	}
}

// SendDirectMessage sends a direct message to specific users (admin only)
// @Summary Send direct message
// @Description Send a direct message to specific users (admin endpoint)
// @Tags WebSocket
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.DirectMessageRequest true "Direct message data"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /ws/direct-message [post]
func (wsc *WebSocketController) SendDirectMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Check if user has admin privileges
	role := c.GetString("role")
	if role != "admin" {
		utils.ForbiddenResponse(c, "Admin access required")
		return
	}

	var req models.DirectMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	if len(req.UserIDs) == 0 {
		utils.BadRequestResponse(c, "At least one user ID is required")
		return
	}

	// Create message
	message := models.WSMessage{
		Type: req.Type,
		Data: req.Data,
	}

	// Send direct messages
	for _, targetUserID := range req.UserIDs {
		userMsg := websocket.UserMessage{
			UserID:  targetUserID,
			Message: message,
		}

		select {
		case wsc.hub.SendToUser <- userMsg:
		default:
			logrus.Warnf("Failed to send direct message to user %s - channel full", targetUserID)
		}
	}

	utils.SuccessResponse(c, "Direct messages sent successfully", nil)
}

// DisconnectUser forcefully disconnects a user (admin only)
// @Summary Disconnect user
// @Description Forcefully disconnect a user's WebSocket connection (admin endpoint)
// @Tags WebSocket
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID to disconnect"
// @Param reason query string false "Reason for disconnection"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /ws/disconnect/{userId} [post]
func (wsc *WebSocketController) DisconnectUser(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Check if user has admin privileges
	role := c.GetString("role")
	if role != "admin" {
		utils.ForbiddenResponse(c, "Admin access required")
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		utils.BadRequestResponse(c, "User ID is required")
		return
	}

	reason := c.Query("reason")
	if reason == "" {
		reason = "Disconnected by administrator"
	}

	// Find and disconnect user
	disconnected := wsc.hub.DisconnectUser(targetUserID, reason)
	if !disconnected {
		utils.NotFoundResponse(c, "User connection")
		return
	}

	logrus.Infof("User %s disconnected by admin %s. Reason: %s", targetUserID, userID, reason)
	utils.SuccessResponse(c, "User disconnected successfully", nil)
}

// GetUserConnection gets connection info for a specific user (admin only)
// @Summary Get user connection
// @Description Get connection information for a specific user (admin endpoint)
// @Tags WebSocket
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} models.APIResponse{data=models.UserConnection}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /ws/users/{userId}/connection [get]
func (wsc *WebSocketController) GetUserConnection(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Check if user has admin privileges
	role := c.GetString("role")
	if role != "admin" {
		utils.ForbiddenResponse(c, "Admin access required")
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		utils.BadRequestResponse(c, "User ID is required")
		return
	}

	connection := wsc.hub.GetUserConnection(targetUserID)
	if connection == nil {
		utils.NotFoundResponse(c, "User connection")
		return
	}

	utils.SuccessResponse(c, "User connection retrieved successfully", connection)
}

// TestConnection tests WebSocket functionality
// @Summary Test WebSocket connection
// @Description Test WebSocket connection and send a test message
// @Tags WebSocket
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{message=string} false "Test message"
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /ws/test [post]
func (wsc *WebSocketController) TestConnection(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Message string `json:"message"`
	}

	c.ShouldBindJSON(&req)

	if req.Message == "" {
		req.Message = "WebSocket test message"
	}

	// Create test message
	testMessage := models.WSMessage{
		Type: models.WSTypeNotification,
		Data: models.WSNotification{
			Type:   "test",
			Title:  "Test Message",
			Body:   req.Message,
			UserID: userID,
		},
	}

	// Send test message to user
	userMsg := websocket.UserMessage{
		UserID:  userID,
		Message: testMessage,
	}

	select {
	case wsc.hub.SendToUser <- userMsg:
		utils.SuccessResponse(c, "Test message sent successfully", nil)
	default:
		utils.InternalServerErrorResponse(c, "Failed to send test message - user not connected")
	}
}

// GetActiveConnections gets list of active connections with basic info
// @Summary Get active connections
// @Description Get list of active WebSocket connections
// @Tags WebSocket
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.ActiveConnection}
// @Failure 401 {object} models.APIResponse
// @Router /ws/active [get]
func (wsc *WebSocketController) GetActiveConnections(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// For regular users, only show their own connections
	// For admins, show all connections
	role := c.GetString("role")
	var connections []models.ActiveConnection

	if role == "admin" {
		connections = wsc.hub.GetAllActiveConnections()
	} else {
		connections = wsc.hub.GetUserActiveConnections(userID)
	}

	utils.SuccessResponse(c, "Active connections retrieved successfully", connections)
}

// PingConnection sends a ping to test connection
// @Summary Ping connection
// @Description Send a ping message to test WebSocket connection
// @Tags WebSocket
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /ws/ping [post]
func (wsc *WebSocketController) PingConnection(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Create ping message
	pingMessage := models.WSMessage{
		Type: models.WSTypePing,
		Data: map[string]interface{}{
			"timestamp": utils.TimeNow(),
		},
	}

	// Send ping to user
	userMsg := websocket.UserMessage{
		UserID:  userID,
		Message: pingMessage,
	}

	select {
	case wsc.hub.SendToUser <- userMsg:
		utils.SuccessResponse(c, "Ping sent successfully", nil)
	default:
		utils.InternalServerErrorResponse(c, "Failed to send ping - user not connected")
	}
}

// GetMessageTypes gets available WebSocket message types
// @Summary Get message types
// @Description Get list of available WebSocket message types and their descriptions
// @Tags WebSocket
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.MessageType}
// @Failure 401 {object} models.APIResponse
// @Router /ws/message-types [get]
func (wsc *WebSocketController) GetMessageTypes(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageTypes := []models.MessageType{
		{
			Type:        models.WSTypeLocationUpdate,
			Description: "Location update from a user",
			Example:     map[string]interface{}{"latitude": 37.7749, "longitude": -122.4194},
		},
		{
			Type:        models.WSTypePlaceEvent,
			Description: "Place arrival/departure event",
			Example:     map[string]interface{}{"eventType": "arrival", "placeName": "Home"},
		},
		{
			Type:        models.WSTypeEmergencyAlert,
			Description: "Emergency alert notification",
			Example:     map[string]interface{}{"type": "sos", "message": "Emergency help needed"},
		},
		{
			Type:        models.WSTypeCircleUpdate,
			Description: "Circle membership or settings update",
			Example:     map[string]interface{}{"type": "member_joined", "circleId": "..."},
		},
		{
			Type:        models.WSTypeMessage,
			Description: "Chat message in a circle",
			Example:     map[string]interface{}{"content": "Hello everyone!", "circleId": "..."},
		},
		{
			Type:        models.WSTypeNotification,
			Description: "Push notification",
			Example:     map[string]interface{}{"title": "New Message", "body": "You have a new message"},
		},
		{
			Type:        models.WSTypeUserStatus,
			Description: "User online/offline status update",
			Example:     map[string]interface{}{"isOnline": true, "userId": "..."},
		},
		{
			Type:        models.WSTypeTypingIndicator,
			Description: "User typing indicator",
			Example:     map[string]interface{}{"isTyping": true, "circleId": "..."},
		},
	}

	utils.SuccessResponse(c, "Message types retrieved successfully", messageTypes)
}
