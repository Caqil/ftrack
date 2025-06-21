package websocket

import (
	"context"
	"ftrack/models"
	"ftrack/services"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Client rooms (circles)
	rooms map[string]*Room

	// User to client mapping for direct messaging
	userClients map[string]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to rooms
	broadcast chan BroadcastMessage

	// Send message to specific user
	sendToUser chan UserMessage

	// Service dependencies
	authService      *services.AuthService
	userService      *services.UserService
	circleService    *services.CircleService
	locationService  *services.LocationService
	messageService   *services.MessageService
	emergencyService *services.EmergencyService

	// Hub statistics
	stats HubStats

	// Mutex for thread safety
	mutex sync.RWMutex

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Background workers
	cleanupTicker *time.Ticker
	metricsTicker *time.Ticker
}

type BroadcastMessage struct {
	RoomID  string
	Message models.WSMessage
	Filter  MessageFilter
}

type UserMessage struct {
	UserID  string
	Message models.WSMessage
}

type MessageFilter struct {
	ExcludeUsers []string
	IncludeUsers []string
	EventTypes   []string
}

type HubStats struct {
	TotalConnections  int64
	ActiveConnections int
	TotalRooms        int
	ActiveRooms       int
	MessagesPerSecond float64
	MessagesSent      int64
	MessagesReceived  int64
	BytesTransferred  int64
	StartTime         time.Time
	LastUpdate        time.Time

	mutex sync.RWMutex
}

func NewHub(
	authService *services.AuthService,
	userService *services.UserService,
	circleService *services.CircleService,
	locationService *services.LocationService,
	messageService *services.MessageService,
	emergencyService *services.EmergencyService,
) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		clients:          make(map[*Client]bool),
		rooms:            make(map[string]*Room),
		userClients:      make(map[string]*Client),
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		broadcast:        make(chan BroadcastMessage),
		sendToUser:       make(chan UserMessage),
		authService:      authService,
		userService:      userService,
		circleService:    circleService,
		locationService:  locationService,
		messageService:   messageService,
		emergencyService: emergencyService,
		stats: HubStats{
			StartTime: time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Start background workers
	hub.cleanupTicker = time.NewTicker(5 * time.Minute)
	hub.metricsTicker = time.NewTicker(1 * time.Minute)

	return hub
}

func (h *Hub) Run() {
	logrus.Info("WebSocket Hub starting...")

	go h.runCleanup()
	go h.runMetrics()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToRoom(message)

		case userMessage := <-h.sendToUser:
			h.sendMessageToUser(userMessage)

		case <-h.ctx.Done():
			logrus.Info("WebSocket Hub shutting down...")
			return
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Register client
	h.clients[client] = true
	h.userClients[client.userID] = client
	h.stats.ActiveConnections++
	h.stats.TotalConnections++

	// Add client to circle rooms
	for _, circleID := range client.circleIDs {
		room := h.getOrCreateRoom(circleID)
		room.AddClient(client)
	}

	// Update user online status
	if h.userService != nil {
		go h.userService.UpdateOnlineStatus(context.Background(), client.userID, true)
	}

	// Notify circle members that user is online
	h.notifyUserStatus(client.userID, client.circleIDs, true)

	logrus.Infof("Client registered: %s (Total: %d)", client.userID, h.stats.ActiveConnections)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.clients[client]; ok {
		// Remove from clients
		delete(h.clients, client)
		delete(h.userClients, client.userID)
		h.stats.ActiveConnections--

		// Remove from rooms
		for _, circleID := range client.circleIDs {
			if room, exists := h.rooms[circleID]; exists {
				room.RemoveClient(client)

				// Remove empty rooms
				if room.IsEmpty() {
					delete(h.rooms, circleID)
				}
			}
		}

		// Notify circle members that user is offline
		h.notifyUserStatus(client.userID, client.circleIDs, false)

		logrus.Infof("Client unregistered: %s (Total: %d)", client.userID, h.stats.ActiveConnections)
	}
}

func (h *Hub) broadcastToRoom(broadcastMsg BroadcastMessage) {
	h.mutex.RLock()
	room := h.rooms[broadcastMsg.RoomID]
	h.mutex.RUnlock()

	if room != nil {
		room.Broadcast(broadcastMsg.Message, broadcastMsg.Filter)
		h.incrementMessagesSent()
	}
}

func (h *Hub) sendMessageToUser(userMessage UserMessage) {
	h.mutex.RLock()
	client := h.userClients[userMessage.UserID]
	h.mutex.RUnlock()

	if client != nil {
		client.SendMessage(userMessage.Message)
		h.incrementMessagesSent()
	}
}

func (h *Hub) getOrCreateRoom(roomID string) *Room {
	if room, exists := h.rooms[roomID]; exists {
		return room
	}

	room := NewRoom(roomID)
	h.rooms[roomID] = room
	return room
}

func (h *Hub) notifyUserStatus(userID string, circleIDs []string, isOnline bool) {
	message := models.WSMessage{
		Type: models.WSTypeUserStatus,
		Data: models.WSUserStatus{
			UserID:    userID,
			IsOnline:  isOnline,
			LastSeen:  time.Now(),
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	for _, circleID := range circleIDs {
		broadcastMsg := BroadcastMessage{
			RoomID:  circleID,
			Message: message,
			Filter: MessageFilter{
				ExcludeUsers: []string{userID}, // Don't send to self
			},
		}

		select {
		case h.broadcast <- broadcastMsg:
		default:
			logrus.Warn("Broadcast channel full, dropping user status message")
		}
	}
}

// Public broadcasting methods
func (h *Hub) BroadcastLocationUpdate(userID string, circleIDs []string, location models.Location) {
	message := models.WSMessage{
		Type: models.WSTypeLocationUpdate,
		Data: models.WSLocationUpdate{
			UserID:    userID,
			Location:  location,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	for _, circleID := range circleIDs {
		broadcastMsg := BroadcastMessage{
			RoomID:  circleID,
			Message: message,
		}

		select {
		case h.broadcast <- broadcastMsg:
		default:
			logrus.Warn("Broadcast channel full, dropping location update")
		}
	}
}

func (h *Hub) BroadcastPlaceEvent(userID string, circleIDs []string, placeEvent models.WSPlaceEvent) {
	message := models.WSMessage{
		Type:      models.WSTypePlaceEvent,
		Data:      placeEvent,
		Timestamp: time.Now(),
	}

	for _, circleID := range circleIDs {
		broadcastMsg := BroadcastMessage{
			RoomID:  circleID,
			Message: message,
		}

		select {
		case h.broadcast <- broadcastMsg:
		default:
			logrus.Warn("Broadcast channel full, dropping place event")
		}
	}
}

func (h *Hub) BroadcastEmergencyAlert(circleIDs []string, alert models.WSEmergencyAlert) {
	message := models.WSMessage{
		Type:      models.WSTypeEmergencyAlert,
		Data:      alert,
		Timestamp: time.Now(),
	}

	for _, circleID := range circleIDs {
		broadcastMsg := BroadcastMessage{
			RoomID:  circleID,
			Message: message,
		}

		select {
		case h.broadcast <- broadcastMsg:
		default:
			logrus.Warn("Broadcast channel full, dropping emergency alert")
		}
	}
}

func (h *Hub) SendNotificationToUser(userID string, notification interface{}) {
	message := models.WSMessage{
		Type:      models.WSTypeNotification,
		Data:      notification,
		Timestamp: time.Now(),
	}

	userMsg := UserMessage{
		UserID:  userID,
		Message: message,
	}

	select {
	case h.sendToUser <- userMsg:
	default:
		logrus.Warn("SendToUser channel full, dropping notification")
	}
}

func (h *Hub) broadcastTypingIndicator(userID, circleID string, isTyping bool) {
	message := models.WSMessage{
		Type: models.WSTypeTypingIndicator,
		Data: models.WSTypingIndicator{
			CircleID:  circleID,
			UserID:    userID,
			IsTyping:  isTyping,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	broadcastMsg := BroadcastMessage{
		RoomID:  circleID,
		Message: message,
		Filter: MessageFilter{
			ExcludeUsers: []string{userID}, // Don't send to sender
		},
	}

	select {
	case h.broadcast <- broadcastMsg:
	default:
		logrus.Warn("Broadcast channel full, dropping typing indicator")
	}
}

// Service integration methods
func (h *Hub) processLocationUpdate(userID string, locationReq models.WSLocationRequest) {
	if h.locationService == nil {
		return
	}

	// Convert to location model
	location := models.Location{
		Latitude:     locationReq.Latitude,
		Longitude:    locationReq.Longitude,
		Accuracy:     locationReq.Accuracy,
		Speed:        locationReq.Speed,
		Bearing:      locationReq.Bearing,
		BatteryLevel: locationReq.BatteryLevel,
		IsCharging:   locationReq.IsCharging,
		IsDriving:    locationReq.IsDriving,
		IsMoving:     locationReq.IsMoving,
		MovementType: locationReq.MovementType,
		NetworkType:  locationReq.NetworkType,
		Source:       locationReq.Source,
	}

	// Process through location service
	_, err := h.locationService.UpdateLocation(context.Background(), userID, location)
	if err != nil {
		logrus.Errorf("Failed to update location for user %s: %v", userID, err)
	}
}

func (h *Hub) processMessage(userID string, messageReq models.WSMessageRequest) {
	if h.messageService == nil {
		return
	}

	// Convert to message request
	req := models.SendMessageRequest{
		CircleID: messageReq.CircleID,
		Type:     messageReq.Type,
		Content:  messageReq.Content,
		Media:    messageReq.Media,
		ReplyTo:  messageReq.ReplyTo,
	}

	// Process through message service
	_, err := h.messageService.SendMessage(context.Background(), userID, req)
	if err != nil {
		logrus.Errorf("Failed to send message for user %s: %v", userID, err)
	}
}

func (h *Hub) processEmergencyAlert(userID string, emergencyReq models.WSEmergencyRequest) {
	if h.emergencyService == nil {
		return
	}

	// Convert to emergency request
	req := models.CreateEmergencyRequest{
		Type:        emergencyReq.Type,
		Description: emergencyReq.Description,
		Location:    emergencyReq.Location,
	}

	// Process through emergency service
	_, err := h.emergencyService.CreateEmergency(context.Background(), userID, req)
	if err != nil {
		logrus.Errorf("Failed to create emergency for user %s: %v", userID, err)
	}
}

// Utility methods
func (h *Hub) GetConnectedUsers() []string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	users := make([]string, 0, len(h.userClients))
	for userID := range h.userClients {
		users = append(users, userID)
	}
	return users
}

func (h *Hub) IsUserOnline(userID string) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	_, exists := h.userClients[userID]
	return exists
}

func (h *Hub) GetStats() models.WSHubStats {
	h.stats.mutex.RLock()
	defer h.stats.mutex.RUnlock()

	h.mutex.RLock()
	roomStats := make(map[string]models.WSRoomStats)
	for roomID, room := range h.rooms {
		roomStats[roomID] = models.WSRoomStats{
			CircleID:      roomID,
			ActiveUsers:   room.GetClientCount(),
			LastActivity:  room.GetLastActivity(),
			TotalMessages: room.GetMessageCount(),
		}
	}
	h.mutex.RUnlock()

	return models.WSHubStats{
		TotalConnections:  int(h.stats.TotalConnections),
		ActiveConnections: h.stats.ActiveConnections,
		TotalRooms:        len(h.rooms),
		ActiveRooms:       len(roomStats),
		MessagesPerSecond: h.stats.MessagesPerSecond,
		RoomStats:         roomStats,
		Uptime:            time.Since(h.stats.StartTime),
		LastUpdate:        time.Now(),
	}
}

func (h *Hub) incrementMessagesSent() {
	h.stats.mutex.Lock()
	h.stats.MessagesSent++
	h.stats.mutex.Unlock()
}

func (h *Hub) runCleanup() {
	for {
		select {
		case <-h.cleanupTicker.C:
			h.performCleanup()
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *Hub) runMetrics() {
	for {
		select {
		case <-h.metricsTicker.C:
			h.updateMetrics()
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *Hub) performCleanup() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Remove inactive clients
	for client := range h.clients {
		if !client.isActive || time.Since(client.lastActivity) > 5*time.Minute {
			logrus.Warnf("Removing inactive client: %s", client.userID)
			go func(c *Client) {
				c.cleanup()
			}(client)
		}
	}

	// Remove empty rooms
	for roomID, room := range h.rooms {
		if room.IsEmpty() {
			delete(h.rooms, roomID)
		}
	}
}

func (h *Hub) updateMetrics() {
	h.stats.mutex.Lock()
	defer h.stats.mutex.Unlock()

	// Calculate messages per second
	now := time.Now()
	if !h.stats.LastUpdate.IsZero() {
		elapsed := now.Sub(h.stats.LastUpdate).Seconds()
		if elapsed > 0 {
			h.stats.MessagesPerSecond = float64(h.stats.MessagesSent) / elapsed
		}
	}

	h.stats.LastUpdate = now
}

func (h *Hub) Shutdown() {
	logrus.Info("Shutting down WebSocket Hub...")

	h.cleanupTicker.Stop()
	h.metricsTicker.Stop()
	h.cancel()

	// Close all client connections
	h.mutex.Lock()
	for client := range h.clients {
		client.cleanup()
	}
	h.mutex.Unlock()

	logrus.Info("WebSocket Hub shutdown complete")
}
