package websocket

import (
	"ftrack/models"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Room represents a WebSocket room (typically a circle)
type Room struct {
	// Room identification
	ID       string
	CircleID string

	// Clients in this room
	clients map[*Client]bool
	mutex   sync.RWMutex

	// Room statistics
	stats RoomStats

	// Room metadata
	createdAt    time.Time
	lastActivity time.Time

	// Background cleanup
	cleanupTicker *time.Ticker
	done          chan bool
}

// RoomStats tracks room activity and performance
type RoomStats struct {
	TotalClients     int64
	ActiveClients    int
	MessagesSent     int64
	MessagesDropped  int64
	BytesTransferred int64
	CreatedAt        time.Time
	LastActivity     time.Time
	mutex            sync.RWMutex
}

// NewRoom creates a new room for a circle
func NewRoom(circleID string) *Room {
	room := &Room{
		ID:            circleID,
		CircleID:      circleID,
		clients:       make(map[*Client]bool),
		createdAt:     time.Now(),
		lastActivity:  time.Now(),
		cleanupTicker: time.NewTicker(5 * time.Minute),
		done:          make(chan bool),
		stats: RoomStats{
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		},
	}

	// Start background cleanup
	go room.runCleanup()

	logrus.Infof("Created new room: %s", circleID)
	return room
}

// AddClient adds a client to the room
func (r *Room) AddClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if client == nil {
		return
	}

	// Check if client is already in room
	if r.clients[client] {
		return
	}

	r.clients[client] = true
	r.stats.ActiveClients++
	r.stats.TotalClients++
	r.lastActivity = time.Now()

	logrus.Debugf("Client %s joined room %s (Total: %d)", client.userID, r.ID, len(r.clients))

	// Send room join confirmation to client
	r.sendRoomEvent(client, models.WSTypeConnectionStatus, models.WSConnectionStatus{
		UserID:       client.userID,
		ConnectionID: client.connectionID,
		Status:       "joined_room",
		Timestamp:    time.Now(),
	})

	// Notify other clients in room about new member
	r.broadcastUserJoined(client)
}

// RemoveClient removes a client from the room
func (r *Room) RemoveClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if client == nil {
		return
	}

	if !r.clients[client] {
		return
	}

	delete(r.clients, client)
	r.stats.ActiveClients--
	r.lastActivity = time.Now()

	logrus.Debugf("Client %s left room %s (Remaining: %d)", client.userID, r.ID, len(r.clients))

	// Notify other clients in room about member leaving
	r.broadcastUserLeft(client)
}

// Broadcast sends a message to all clients in the room with optional filtering
func (r *Room) Broadcast(message models.WSMessage, filter MessageFilter) {
	r.mutex.RLock()
	clients := make([]*Client, 0, len(r.clients))
	for client := range r.clients {
		if r.shouldSendToClient(client, filter) {
			clients = append(clients, client)
		}
	}
	r.mutex.RUnlock()

	// Send to filtered clients
	successCount := 0
	for _, client := range clients {
		client.SendMessage(message)
		successCount++
	}

	r.incrementMessagesSent(int64(successCount))
	r.updateLastActivity()

	logrus.Debugf("Broadcasted message to %d/%d clients in room %s", successCount, len(clients), r.ID)
}

// BroadcastToUser sends a message to a specific user in the room
func (r *Room) BroadcastToUser(userID string, message models.WSMessage) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for client := range r.clients {
		if client.userID == userID {
			client.SendMessage(message)
			r.incrementMessagesSent(1)
			r.updateLastActivity()
			return true
		}
	}

	return false
}

// IsEmpty returns true if the room has no clients
func (r *Room) IsEmpty() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.clients) == 0
}

// GetClientCount returns the number of active clients
func (r *Room) GetClientCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.clients)
}

// GetClients returns a list of all clients in the room
func (r *Room) GetClients() []*Client {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	clients := make([]*Client, 0, len(r.clients))
	for client := range r.clients {
		clients = append(clients, client)
	}
	return clients
}

// GetUserIDs returns a list of user IDs in the room
func (r *Room) GetUserIDs() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	userIDs := make([]string, 0, len(r.clients))
	for client := range r.clients {
		userIDs = append(userIDs, client.userID)
	}
	return userIDs
}

// HasUser checks if a user is in the room
func (r *Room) HasUser(userID string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for client := range r.clients {
		if client.userID == userID {
			return true
		}
	}
	return false
}

// GetStats returns room statistics
func (r *Room) GetStats() RoomStats {
	r.stats.mutex.RLock()
	defer r.stats.mutex.RUnlock()
	return r.stats
}

// GetLastActivity returns the last activity time
func (r *Room) GetLastActivity() time.Time {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.lastActivity
}

// GetMessageCount returns total messages sent in this room
func (r *Room) GetMessageCount() int64 {
	r.stats.mutex.RLock()
	defer r.stats.mutex.RUnlock()
	return r.stats.MessagesSent
}

// Close shuts down the room and cleans up resources
func (r *Room) Close() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Stop cleanup ticker
	if r.cleanupTicker != nil {
		r.cleanupTicker.Stop()
	}

	// Signal cleanup goroutine to stop
	select {
	case r.done <- true:
	default:
	}

	// Disconnect all clients
	for client := range r.clients {
		delete(r.clients, client)
	}

	logrus.Infof("Room %s closed", r.ID)
}

// Private helper methods

func (r *Room) shouldSendToClient(client *Client, filter MessageFilter) bool {
	if client == nil || !client.isActive {
		return false
	}

	// Check exclude users
	for _, excludeUserID := range filter.ExcludeUsers {
		if client.userID == excludeUserID {
			return false
		}
	}

	// Check include users (if specified, only send to these users)
	if len(filter.IncludeUsers) > 0 {
		found := false
		for _, includeUserID := range filter.IncludeUsers {
			if client.userID == includeUserID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (r *Room) sendRoomEvent(client *Client, eventType string, data interface{}) {
	message := models.WSMessage{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	client.SendMessage(message)
}

func (r *Room) broadcastUserJoined(joinedClient *Client) {
	if joinedClient.user == nil {
		return
	}

	message := models.WSMessage{
		Type: models.WSTypeUserStatus,
		Data: models.WSUserStatus{
			UserID:    joinedClient.userID,
			IsOnline:  true,
			LastSeen:  time.Now(),
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Broadcast to all other clients in room
	filter := MessageFilter{
		ExcludeUsers: []string{joinedClient.userID},
	}

	r.Broadcast(message, filter)
}

func (r *Room) broadcastUserLeft(leftClient *Client) {
	if leftClient.user == nil {
		return
	}

	message := models.WSMessage{
		Type: models.WSTypeUserStatus,
		Data: models.WSUserStatus{
			UserID:    leftClient.userID,
			IsOnline:  false,
			LastSeen:  time.Now(),
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Broadcast to all remaining clients in room
	filter := MessageFilter{
		ExcludeUsers: []string{leftClient.userID},
	}

	r.Broadcast(message, filter)
}

func (r *Room) runCleanup() {
	for {
		select {
		case <-r.cleanupTicker.C:
			r.performCleanup()
		case <-r.done:
			return
		}
	}
}

func (r *Room) performCleanup() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Remove inactive clients
	for client := range r.clients {
		if !client.isActive || time.Since(client.lastActivity) > 10*time.Minute {
			logrus.Warnf("Removing inactive client %s from room %s", client.userID, r.ID)
			delete(r.clients, client)
			r.stats.ActiveClients--
		}
	}
}

func (r *Room) incrementMessagesSent(count int64) {
	r.stats.mutex.Lock()
	r.stats.MessagesSent += count
	r.stats.mutex.Unlock()
}

func (r *Room) incrementDroppedMessages() {
	r.stats.mutex.Lock()
	r.stats.MessagesDropped++
	r.stats.mutex.Unlock()
}

func (r *Room) updateLastActivity() {
	r.mutex.Lock()
	r.lastActivity = time.Now()
	r.mutex.Unlock()

	r.stats.mutex.Lock()
	r.stats.LastActivity = time.Now()
	r.stats.mutex.Unlock()
}
