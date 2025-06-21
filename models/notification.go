package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID   primitive.ObjectID `json:"userId" bson:"userId"`
	CircleID primitive.ObjectID `json:"circleId,omitempty" bson:"circleId,omitempty"`

	// Notification Content
	Type  string                 `json:"type" bson:"type"`
	Title string                 `json:"title" bson:"title"`
	Body  string                 `json:"body" bson:"body"`
	Data  map[string]interface{} `json:"data,omitempty" bson:"data,omitempty"`

	// Notification State
	Status   string `json:"status" bson:"status"`     // pending, sent, delivered, failed
	Priority string `json:"priority" bson:"priority"` // low, normal, high, urgent
	Category string `json:"category" bson:"category"` // location, emergency, social, system

	// Delivery Methods
	Channels NotificationChannels `json:"channels" bson:"channels"`

	// Scheduling
	ScheduledFor time.Time `json:"scheduledFor,omitempty" bson:"scheduledFor,omitempty"`
	SentAt       time.Time `json:"sentAt,omitempty" bson:"sentAt,omitempty"`

	// References
	RelatedID   primitive.ObjectID `json:"relatedId,omitempty" bson:"relatedId,omitempty"`
	RelatedType string             `json:"relatedType,omitempty" bson:"relatedType,omitempty"`

	// User Interaction
	IsRead      bool      `json:"isRead" bson:"isRead"`
	ReadAt      time.Time `json:"readAt,omitempty" bson:"readAt,omitempty"`
	ActionTaken string    `json:"actionTaken,omitempty" bson:"actionTaken,omitempty"`

	// Retry Logic
	RetryCount int       `json:"retryCount" bson:"retryCount"`
	LastRetry  time.Time `json:"lastRetry,omitempty" bson:"lastRetry,omitempty"`

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
	ExpiresAt time.Time `json:"expiresAt,omitempty" bson:"expiresAt,omitempty"`
}

type NotificationChannels struct {
	Push  bool `json:"push" bson:"push"`
	SMS   bool `json:"sms" bson:"sms"`
	Email bool `json:"email" bson:"email"`
	InApp bool `json:"inApp" bson:"inApp"`
}

type NotificationTemplate struct {
	ID   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`
	Type string             `json:"type" bson:"type"`

	// Template Content
	Title string                 `json:"title" bson:"title"`
	Body  string                 `json:"body" bson:"body"`
	Data  map[string]interface{} `json:"data,omitempty" bson:"data,omitempty"`

	// Settings
	Priority string               `json:"priority" bson:"priority"`
	Category string               `json:"category" bson:"category"`
	Channels NotificationChannels `json:"channels" bson:"channels"`

	// Variables for templating
	Variables []string `json:"variables" bson:"variables"`

	IsActive  bool      `json:"isActive" bson:"isActive"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type NotificationPreference struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID primitive.ObjectID `json:"userId" bson:"userId"`

	// Global Settings
	GlobalEnabled bool                   `json:"globalEnabled" bson:"globalEnabled"`
	QuietHours    NotificationQuietHours `json:"quietHours" bson:"quietHours"`

	// Category Preferences
	Categories map[string]NotificationCategoryPref `json:"categories" bson:"categories"`

	// Circle Specific
	CirclePrefs map[string]NotificationChannels `json:"circlePrefs" bson:"circlePrefs"`

	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type NotificationQuietHours struct {
	Enabled   bool   `json:"enabled" bson:"enabled"`
	StartTime string `json:"startTime" bson:"startTime"` // HH:MM
	EndTime   string `json:"endTime" bson:"endTime"`     // HH:MM
	Timezone  string `json:"timezone" bson:"timezone"`
	Weekdays  []int  `json:"weekdays" bson:"weekdays"` // 0=Sunday, 1=Monday, etc.
}

type NotificationCategoryPref struct {
	Enabled  bool                 `json:"enabled" bson:"enabled"`
	Channels NotificationChannels `json:"channels" bson:"channels"`
	Priority string               `json:"priority" bson:"priority"`
}

// Notification Type Constants
const (
	// Location Notifications
	NotificationLocationArrival   = "location_arrival"
	NotificationLocationDeparture = "location_departure"
	NotificationLocationSharing   = "location_sharing"

	// Emergency Notifications
	NotificationEmergencySOS   = "emergency_sos"
	NotificationEmergencyCrash = "emergency_crash"
	NotificationEmergencyHelp  = "emergency_help"

	// Social Notifications
	NotificationCircleInvite = "circle_invite"
	NotificationNewMessage   = "new_message"
	NotificationMemberJoined = "member_joined"
	NotificationMemberLeft   = "member_left"

	// Driving Notifications
	NotificationDrivingSpeed  = "driving_speed"
	NotificationDrivingHard   = "driving_hard_brake"
	NotificationDrivingPhone  = "driving_phone_usage"
	NotificationDrivingReport = "driving_report"

	// System Notifications
	NotificationSystemUpdate      = "system_update"
	NotificationSystemMaintenance = "system_maintenance"
	NotificationAccountSecurity   = "account_security"
)

// Request DTOs
type SendNotificationRequest struct {
	UserIDs     []string               `json:"userIds" validate:"required"`
	Type        string                 `json:"type" validate:"required"`
	Title       string                 `json:"title" validate:"required"`
	Body        string                 `json:"body" validate:"required"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Priority    string                 `json:"priority" validate:"oneof=low normal high urgent"`
	Channels    NotificationChannels   `json:"channels"`
	ScheduleFor *time.Time             `json:"scheduleFor,omitempty"`
}

type UpdateNotificationPrefsRequest struct {
	GlobalEnabled bool                                `json:"globalEnabled"`
	QuietHours    *NotificationQuietHours             `json:"quietHours,omitempty"`
	Categories    map[string]NotificationCategoryPref `json:"categories,omitempty"`
	CirclePrefs   map[string]NotificationChannels     `json:"circlePrefs,omitempty"`
}
