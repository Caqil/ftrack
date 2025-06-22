package websocket

import (
	"context"
	"encoding/json"
	"ftrack/models"
	"ftrack/utils"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 4096

	// Buffer size for client send channel
	sendBufferSize = 256
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

type Client struct {
	// WebSocket connection
	conn *websocket.Conn

	// User information
	userID    string
	circleIDs []string
	user      *models.User

	// Connection metadata
	connectionID string
	connectedAt  time.Time
	lastPing     time.Time
	lastActivity time.Time
	deviceType   string
	appVersion   string
	ipAddress    string
	userAgent    string

	// Buffered channel of outbound messages
	send chan models.WSMessage

	// Hub reference
	hub *Hub

	rateLimiter  *utils.RateLimiter
	requestCount int
	windowStart  time.Time

	// Subscription management
	subscriptions map[string]bool // event types
	filters       map[string]interface{}

	// Client state
	isActive        bool
	isAuthenticated bool
	pingFailCount   int

	// Context for cleanup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewClient(conn *websocket.Conn, hub *Hub, r *http.Request) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		conn:          conn,
		hub:           hub,
		send:          make(chan models.WSMessage, sendBufferSize),
		connectionID:  utils.GenerateUUID(),
		connectedAt:   time.Now(),
		lastPing:      time.Now(),
		lastActivity:  time.Now(),
		ipAddress:     getClientIP(r),
		userAgent:     r.UserAgent(),
		subscriptions: make(map[string]bool),
		filters:       make(map[string]interface{}),           // 100 requests per minute
		rateLimiter:   utils.NewRateLimiter(100, time.Minute), // 100 requests per minute
		ctx:           ctx,
		cancel:        cancel,
	}

	// Extract device info from headers
	client.deviceType = r.Header.Get("X-Device-Type")
	client.appVersion = r.Header.Get("X-App-Version")

	return client
}

func (c *Client) ReadPump() {
	defer func() {
		c.cleanup()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.handlePong()
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, messageData, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logrus.Errorf("WebSocket error for user %s: %v", c.userID, err)
				}
				return
			}

			c.lastActivity = time.Now()

			// Rate limiting check
			if !c.rateLimiter.Allow() {
				c.sendError(models.WSErrorRateLimit, "Rate limit exceeded")
				continue
			}

			c.handleMessage(messageData)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return

		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				logrus.Errorf("Write error for user %s: %v", c.userID, err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.pingFailCount++
				if c.pingFailCount > 3 {
					logrus.Warnf("Ping failed for user %s, disconnecting", c.userID)
					return
				}
			}
		}
	}
}

func (c *Client) handleMessage(messageData []byte) {
	var wsRequest models.WSRequest
	if err := json.Unmarshal(messageData, &wsRequest); err != nil {
		c.sendError(models.WSErrorInvalidMessage, "Invalid message format")
		return
	}

	// Check authentication for non-auth messages
	if wsRequest.Type != models.WSTypeAuth && !c.isAuthenticated {
		c.sendError(models.WSErrorUnauthorized, "Authentication required")
		return
	}

	switch wsRequest.Type {
	case models.WSTypeAuth:
		c.handleAuth(wsRequest)
	case models.WSRequestLocationUpdate:
		c.handleLocationUpdate(wsRequest)
	case models.WSRequestSendMessage:
		c.handleSendMessage(wsRequest)
	case models.WSRequestEmergencyAlert:
		c.handleEmergencyAlert(wsRequest)
	case models.WSRequestTypingStart:
		c.handleTypingIndicator(wsRequest, true)
	case models.WSRequestTypingStop:
		c.handleTypingIndicator(wsRequest, false)
	case models.WSTypePing:
		c.handlePing(wsRequest)
	default:
		c.sendError(models.WSErrorInvalidMessage, "Unknown message type")
	}
}

func (c *Client) handleAuth(request models.WSRequest) {
	tokenData, ok := request.Data["token"].(string)
	if !ok {
		c.sendError(models.WSErrorUnauthorized, "Token required")
		return
	}

	// Validate JWT token using auth service
	_, err := c.hub.authService.ValidateToken(tokenData)
	if err != nil {
		c.sendError(models.WSErrorUnauthorized, "Invalid token")
		return
	}

	c.isAuthenticated = true

	// Get user's circles
	circles, err := c.hub.circleService.GetUserCircles(context.Background(), c.userID)
	if err != nil {
		logrus.Errorf("Failed to get user circles: %v", err)
	} else {
		for _, circle := range circles {
			c.circleIDs = append(c.circleIDs, circle.ID.Hex())
		}
	}

	// Send authentication success
	response := models.WSAuthResponse{
		Success:   true,
		UserID:    c.userID,
		CircleIDs: c.circleIDs,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	c.sendResponse(models.WSTypeAuth, response, request.RequestID)

	// Register with hub
	c.hub.register <- c

	logrus.Infof("Client authenticated: %s (%s)", c.userID, c.connectionID)
}

func (c *Client) handleLocationUpdate(request models.WSRequest) {
	var locationReq models.WSLocationRequest
	if err := c.unmarshalData(request.Data, &locationReq); err != nil {
		c.sendError(models.WSErrorInvalidMessage, "Invalid location data")
		return
	}

	// Validate coordinates
	if !utils.IsValidCoordinate(locationReq.Latitude, locationReq.Longitude) {
		c.sendError(models.WSErrorInvalidLocation, "Invalid coordinates")
		return
	}

	// Process location update through location service
	go c.hub.processLocationUpdate(c.userID, locationReq)

	c.sendSuccess("Location updated", request.RequestID)
}

func (c *Client) handleSendMessage(request models.WSRequest) {
	var messageReq models.WSMessageRequest
	if err := c.unmarshalData(request.Data, &messageReq); err != nil {
		c.sendError(models.WSErrorInvalidMessage, "Invalid message data")
		return
	}

	// Process message through message service
	go c.hub.processMessage(c.userID, messageReq)

	c.sendSuccess("Message sent", request.RequestID)
}

func (c *Client) handleEmergencyAlert(request models.WSRequest) {
	var emergencyReq models.WSEmergencyRequest
	if err := c.unmarshalData(request.Data, &emergencyReq); err != nil {
		c.sendError(models.WSErrorInvalidMessage, "Invalid emergency data")
		return
	}

	// Process emergency alert through emergency service
	go c.hub.processEmergencyAlert(c.userID, emergencyReq)

	c.sendSuccess("Emergency alert sent", request.RequestID)
}

func (c *Client) handleTypingIndicator(request models.WSRequest, isTyping bool) {
	circleID, ok := request.Data["circleId"].(string)
	if !ok {
		c.sendError(models.WSErrorInvalidMessage, "Circle ID required")
		return
	}

	// Broadcast typing indicator to circle
	c.hub.broadcastTypingIndicator(c.userID, circleID, isTyping)
}

func (c *Client) handlePing(request models.WSRequest) {
	pong := models.WSMessage{
		Type:      models.WSTypePong,
		Timestamp: time.Now(),
	}

	select {
	case c.send <- pong:
	default:
		// Channel full, client likely disconnected
	}
}

func (c *Client) handlePong() {
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.lastPing = time.Now()
	c.pingFailCount = 0
}

func (c *Client) sendError(code, message string) {
	errorMsg := models.WSMessage{
		Type: models.WSTypeError,
		Data: models.WSError{
			Code:      code,
			Message:   message,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	select {
	case c.send <- errorMsg:
	default:
		// Channel full
	}
}

func (c *Client) sendSuccess(message, requestID string) {
	successMsg := models.WSMessage{
		Type: models.WSTypeSuccess,
		Data: map[string]interface{}{
			"message": message,
		},
		Timestamp: time.Now(),
	}

	if requestID != "" {
		successMsg.Data.(map[string]interface{})["requestId"] = requestID
	}

	select {
	case c.send <- successMsg:
	default:
		// Channel full
	}
}

func (c *Client) sendResponse(msgType string, data interface{}, requestID string) {
	response := models.WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}

	if requestID != "" {
		if dataMap, ok := data.(map[string]interface{}); ok {
			dataMap["requestId"] = requestID
		}
	}

	select {
	case c.send <- response:
	default:
		// Channel full
	}
}

func (c *Client) SendMessage(message models.WSMessage) {
	if !c.isActive {
		return
	}

	select {
	case c.send <- message:
	default:
		// Channel full, likely client disconnected
		logrus.Warnf("Send channel full for user %s", c.userID)
	}
}

func (c *Client) cleanup() {
	c.isActive = false
	c.cancel()

	if c.isAuthenticated {
		c.hub.unregister <- c

		// Update user offline status
		if c.hub.userService != nil {
			go c.hub.userService.UpdateOnlineStatus(context.Background(), c.userID, false)
		}
	}

	close(c.send)
	c.conn.Close()

	logrus.Infof("Client disconnected: %s (%s)", c.userID, c.connectionID)
}

func (c *Client) unmarshalData(data map[string]interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
