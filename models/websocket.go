// models/websocket.go
package models

import (
	"time"
)

// WebSocket Message Types
type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	UserID    string      `json:"userId,omitempty"`
	CircleID  string      `json:"circleId,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"requestId,omitempty"`
}

type WSLocationUpdate struct {
	UserID    string    `json:"userId"`
	Location  Location  `json:"location"`
	Timestamp time.Time `json:"timestamp"`
}

type WSPlaceEvent struct {
	UserID    string    `json:"userId"`
	PlaceID   string    `json:"placeId"`
	PlaceName string    `json:"placeName"`
	EventType string    `json:"eventType"` // arrival, departure
	Location  Location  `json:"location,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type WSEmergencyAlert struct {
	UserID      string            `json:"userId"`
	EmergencyID string            `json:"emergencyId"`
	Type        string            `json:"type"` // sos, crash, help
	Title       string            `json:"title"`
	Message     string            `json:"message,omitempty"`
	Location    EmergencyLocation `json:"location"`
	Priority    string            `json:"priority"`
	Timestamp   time.Time         `json:"timestamp"`
}

type WSCircleUpdate struct {
	CircleID  string      `json:"circleId"`
	Type      string      `json:"type"` // member_joined, member_left, settings_updated
	UserID    string      `json:"userId,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type WSUserStatus struct {
	UserID       string    `json:"userId"`
	IsOnline     bool      `json:"isOnline"`
	LastSeen     time.Time `json:"lastSeen,omitempty"`
	BatteryLevel int       `json:"batteryLevel,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

type WSNotification struct {
	NotificationID string                 `json:"notificationId"`
	UserID         string                 `json:"userId"`
	Type           string                 `json:"type"`
	Title          string                 `json:"title"`
	Body           string                 `json:"body"`
	Data           map[string]interface{} `json:"data,omitempty"`
	Priority       string                 `json:"priority"`
	Timestamp      time.Time              `json:"timestamp"`
}

type WSMessageData struct {
	MessageID string        `json:"messageId"`
	CircleID  string        `json:"circleId"`
	SenderID  string        `json:"senderId"`
	Type      string        `json:"type"`
	Content   string        `json:"content,omitempty"`
	Media     *MessageMedia `json:"media,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

type WSDrivingEvent struct {
	UserID      string    `json:"userId"`
	EventID     string    `json:"eventId"`
	Type        string    `json:"type"` // speeding, hard_brake, phone_usage
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Location    Location  `json:"location"`
	Speed       float64   `json:"speed"`
	SpeedLimit  int       `json:"speedLimit,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

type WSTypingIndicator struct {
	CircleID  string    `json:"circleId"`
	UserID    string    `json:"userId"`
	IsTyping  bool      `json:"isTyping"`
	Timestamp time.Time `json:"timestamp"`
}

type WSConnectionStatus struct {
	UserID       string    `json:"userId"`
	ConnectionID string    `json:"connectionId"`
	Status       string    `json:"status"` // connected, disconnected, reconnecting
	Timestamp    time.Time `json:"timestamp"`
}

// WebSocket Response Types
type WSResponse struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Success   bool        `json:"success"`
	Error     string      `json:"error,omitempty"`
	RequestID string      `json:"requestId,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type WSError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// WebSocket Request Types
type WSRequest struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data,omitempty"`
	RequestID string                 `json:"requestId,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type WSLocationRequest struct {
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Accuracy     float64 `json:"accuracy,omitempty"`
	Speed        float64 `json:"speed,omitempty"`
	Bearing      float64 `json:"bearing,omitempty"`
	BatteryLevel int     `json:"batteryLevel,omitempty"`
	IsCharging   bool    `json:"isCharging,omitempty"`
	IsDriving    bool    `json:"isDriving,omitempty"`
	IsMoving     bool    `json:"isMoving,omitempty"`
	MovementType string  `json:"movementType,omitempty"`
	NetworkType  string  `json:"networkType,omitempty"`
	Source       string  `json:"source,omitempty"`
	DeviceTime   string  `json:"deviceTime,omitempty"` // RFC3339 format
	Timezone     string  `json:"timezone,omitempty"`
}

type WSMessageRequest struct {
	CircleID string        `json:"circleId"`
	Type     string        `json:"type"`
	Content  string        `json:"content,omitempty"`
	Media    *MessageMedia `json:"media,omitempty"`
	ReplyTo  string        `json:"replyTo,omitempty"`
}

type WSEmergencyRequest struct {
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Location    EmergencyLocation `json:"location"`
}

// WebSocket Event Constants
const (
	// WebSocket message types
	WSTypeLocationUpdate   = "location_update"
	WSTypePlaceEvent       = "place_event"
	WSTypeEmergencyAlert   = "emergency_alert"
	WSTypeCircleUpdate     = "circle_update"
	WSTypeUserStatus       = "user_status"
	WSTypeMessage          = "message"
	WSTypeNotification     = "notification"
	WSTypeDrivingEvent     = "driving_event"
	WSTypeTypingIndicator  = "typing_indicator"
	WSTypeConnectionStatus = "connection_status"
	WSTypePing             = "ping"
	WSTypePong             = "pong"
	WSTypeAuth             = "auth"
	WSTypeError            = "error"
	WSTypeSuccess          = "success"

	// WebSocket request types
	WSRequestLocationUpdate = "location_update_request"
	WSRequestSendMessage    = "send_message_request"
	WSRequestEmergencyAlert = "emergency_alert_request"
	WSRequestJoinCircle     = "join_circle_request"
	WSRequestLeaveCircle    = "leave_circle_request"
	WSRequestTypingStart    = "typing_start_request"
	WSRequestTypingStop     = "typing_stop_request"

	// Connection states
	WSStatusConnected     = "connected"
	WSStatusDisconnected  = "disconnected"
	WSStatusReconnecting  = "reconnecting"
	WSStatusAuthenticated = "authenticated"
	WSStatusError         = "error"

	// Error codes
	WSErrorInvalidMessage  = "INVALID_MESSAGE"
	WSErrorUnauthorized    = "UNAUTHORIZED"
	WSErrorRateLimit       = "RATE_LIMIT"
	WSErrorCircleNotFound  = "CIRCLE_NOT_FOUND"
	WSErrorUserNotFound    = "USER_NOT_FOUND"
	WSErrorInvalidLocation = "INVALID_LOCATION"
	WSErrorConnectionLost  = "CONNECTION_LOST"
)

// WebSocket Event Handlers
type WSEventHandler interface {
	HandleLocationUpdate(userID string, data WSLocationRequest) error
	HandleMessage(userID string, data WSMessageRequest) error
	HandleEmergencyAlert(userID string, data WSEmergencyRequest) error
	HandleTypingIndicator(userID string, circleID string, isTyping bool) error
	HandleJoinCircle(userID string, circleID string) error
	HandleLeaveCircle(userID string, circleID string) error
}

// WebSocket Connection Info
type WSConnection struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"userId"`
	CircleIDs   []string               `json:"circleIds"`
	ConnectedAt time.Time              `json:"connectedAt"`
	LastPing    time.Time              `json:"lastPing"`
	IsActive    bool                   `json:"isActive"`
	DeviceType  string                 `json:"deviceType"` // ios, android, web
	AppVersion  string                 `json:"appVersion"`
	IPAddress   string                 `json:"ipAddress"`
	UserAgent   string                 `json:"userAgent"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WebSocket Room (Circle)
type WSRoom struct {
	CircleID     string                   `json:"circleId"`
	Connections  map[string]*WSConnection `json:"connections"`
	LastActivity time.Time                `json:"lastActivity"`
	MessageCount int64                    `json:"messageCount"`
	ActiveUsers  int                      `json:"activeUsers"`
	CreatedAt    time.Time                `json:"createdAt"`
}

// WebSocket Hub Stats
type WSHubStats struct {
	TotalConnections  int                    `json:"totalConnections"`
	ActiveConnections int                    `json:"activeConnections"`
	TotalRooms        int                    `json:"totalRooms"`
	ActiveRooms       int                    `json:"activeRooms"`
	MessagesPerSecond float64                `json:"messagesPerSecond"`
	ConnectionsByType map[string]int         `json:"connectionsByType"`
	RoomStats         map[string]WSRoomStats `json:"roomStats"`
	Uptime            time.Duration          `json:"uptime"`
	LastUpdate        time.Time              `json:"lastUpdate"`
}

type WSRoomStats struct {
	CircleID       string    `json:"circleId"`
	ActiveUsers    int       `json:"activeUsers"`
	TotalMessages  int64     `json:"totalMessages"`
	LastActivity   time.Time `json:"lastActivity"`
	AverageLatency float64   `json:"averageLatency"` // ms
}

// WebSocket Authentication
type WSAuthRequest struct {
	Token string `json:"token"`
	Type  string `json:"type"` // bearer, api_key
}

type WSAuthResponse struct {
	Success   bool      `json:"success"`
	UserID    string    `json:"userId,omitempty"`
	CircleIDs []string  `json:"circleIds,omitempty"`
	Error     string    `json:"error,omitempty"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

// WebSocket Heartbeat
type WSHeartbeat struct {
	Type      string                 `json:"type"` // ping, pong
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// WebSocket Rate Limiting
type WSRateLimit struct {
	UserID       string    `json:"userId"`
	WindowStart  time.Time `json:"windowStart"`
	RequestCount int       `json:"requestCount"`
	WindowSize   int       `json:"windowSize"` // seconds
	MaxRequests  int       `json:"maxRequests"`
	ResetAt      time.Time `json:"resetAt"`
}

// WebSocket Subscription Management
type WSSubscription struct {
	UserID     string                 `json:"userId"`
	CircleIDs  []string               `json:"circleIds"`
	EventTypes []string               `json:"eventTypes"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

type WSSubscriptionRequest struct {
	Action     string                 `json:"action"` // subscribe, unsubscribe, update
	CircleIDs  []string               `json:"circleIds,omitempty"`
	EventTypes []string               `json:"eventTypes,omitempty"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
}

// WebSocket Broadcasting
type WSBroadcastRequest struct {
	Type     string                 `json:"type"`
	Target   WSBroadcastTarget      `json:"target"`
	Data     interface{}            `json:"data"`
	Priority string                 `json:"priority"`      // low, normal, high, urgent
	TTL      int                    `json:"ttl,omitempty"` // seconds
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type WSBroadcastTarget struct {
	Type      string                 `json:"type"` // user, circle, all, custom
	UserIDs   []string               `json:"userIds,omitempty"`
	CircleIDs []string               `json:"circleIds,omitempty"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
}

// WebSocket Metrics
type WSMetrics struct {
	ConnectionsOpened int64     `json:"connectionsOpened"`
	ConnectionsClosed int64     `json:"connectionsClosed"`
	MessagesReceived  int64     `json:"messagesReceived"`
	MessagesSent      int64     `json:"messagesSent"`
	ErrorsCount       int64     `json:"errorsCount"`
	AverageLatency    float64   `json:"averageLatency"` // ms
	PeakConnections   int       `json:"peakConnections"`
	PeakTimestamp     time.Time `json:"peakTimestamp"`
	DataTransferred   int64     `json:"dataTransferred"` // bytes
	LastReset         time.Time `json:"lastReset"`
}
