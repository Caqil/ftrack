package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Emergency struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID   primitive.ObjectID `json:"userId" bson:"userId"`
	CircleID primitive.ObjectID `json:"circleId,omitempty" bson:"circleId,omitempty"`

	// Emergency Details
	Type        string `json:"type" bson:"type"`         // sos, crash, help, medical, fire, police
	Status      string `json:"status" bson:"status"`     // active, resolved, false_alarm, cancelled
	Priority    string `json:"priority" bson:"priority"` // low, medium, high, critical
	Title       string `json:"title" bson:"title"`
	Description string `json:"description,omitempty" bson:"description,omitempty"`

	// Location Information
	Location EmergencyLocation `json:"location" bson:"location"`

	// Detection Data
	Detection EmergencyDetection `json:"detection,omitempty" bson:"detection,omitempty"`

	// Response Information
	Response EmergencyResponse `json:"response" bson:"response"`

	// Contacts Notified
	ContactsNotified []EmergencyContact `json:"contactsNotified" bson:"contactsNotified"`

	// Media Evidence
	Media []EmergencyMedia `json:"media,omitempty" bson:"media,omitempty"`

	// Timeline
	Timeline []EmergencyEvent `json:"timeline" bson:"timeline"`


	CreatedAt          time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt" bson:"updatedAt"`
	Resolution         string    `bson:"resolution,omitempty" json:"resolution,omitempty"`
	ResolvedAt         time.Time `bson:"resolvedAt,omitempty" json:"resolvedAt,omitempty"`
	ResolvedBy         string    `bson:"resolvedBy,omitempty" json:"resolvedBy,omitempty"`
	DismissalReason    string    `bson:"dismissalReason,omitempty" json:"dismissalReason,omitempty"`
	DismissedAt        time.Time `bson:"dismissedAt,omitempty" json:"dismissedAt,omitempty"`
	DismissedBy        string    `bson:"dismissedBy,omitempty" json:"dismissedBy,omitempty"`
	CancellationReason string    `bson:"cancellationReason,omitempty" json:"cancellationReason,omitempty"`
	CancelledAt        time.Time `bson:"cancelledAt,omitempty" json:"cancelledAt,omitempty"`
}

type EmergencyLocation struct {
	Latitude  float64   `json:"latitude" bson:"latitude"`
	Longitude float64   `json:"longitude" bson:"longitude"`
	Accuracy  float64   `json:"accuracy" bson:"accuracy"`
	Address   string    `json:"address,omitempty" bson:"address,omitempty"`
	PlaceName string    `json:"placeName,omitempty" bson:"placeName,omitempty"`
	Indoor    bool      `json:"indoor" bson:"indoor"`
	Floor     string    `json:"floor,omitempty" bson:"floor,omitempty"`
	Altitude  float64   `json:"altitude" bson:"altitude"`
	Speed     float64   `json:"speed" bson:"speed"`
	Bearing   float64   `json:"bearing" bson:"bearing"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

type EmergencyDetection struct {
	Method       string                 `json:"method" bson:"method"`         // manual, auto_crash, auto_fall, device_sensor
	Confidence   float64                `json:"confidence" bson:"confidence"` // 0.0 to 1.0
	SensorData   map[string]interface{} `json:"sensorData,omitempty" bson:"sensorData,omitempty"`
	TriggerEvent string                 `json:"triggerEvent,omitempty" bson:"triggerEvent,omitempty"`
}

type EmergencyResponse struct {
	AutoSent          bool               `json:"autoSent" bson:"autoSent"`
	ResponseTime      int64              `json:"responseTime" bson:"responseTime"` // seconds
	FirstResponder    primitive.ObjectID `json:"firstResponder,omitempty" bson:"firstResponder,omitempty"`
	FirstResponseAt   time.Time          `json:"firstResponseAt,omitempty" bson:"firstResponseAt,omitempty"`
	EmergencyServices EmergencyServices  `json:"emergencyServices" bson:"emergencyServices"`
}

type EmergencyServices struct {
	Police   ServiceContact `json:"police,omitempty" bson:"police,omitempty"`
	Medical  ServiceContact `json:"medical,omitempty" bson:"medical,omitempty"`
	Fire     ServiceContact `json:"fire,omitempty" bson:"fire,omitempty"`
	Roadside ServiceContact `json:"roadside,omitempty" bson:"roadside,omitempty"`
}

type ServiceContact struct {
	Contacted   bool      `json:"contacted" bson:"contacted"`
	ContactedAt time.Time `json:"contactedAt,omitempty" bson:"contactedAt,omitempty"`
	Phone       string    `json:"phone,omitempty" bson:"phone,omitempty"`
	Reference   string    `json:"reference,omitempty" bson:"reference,omitempty"`
	Status      string    `json:"status,omitempty" bson:"status,omitempty"`
}

type EmergencyContact struct {
	ContactID    primitive.ObjectID `json:"contactId" bson:"contactId"`
	Name         string             `json:"name" bson:"name"`
	Phone        string             `json:"phone" bson:"phone"`
	Email        string             `json:"email,omitempty" bson:"email,omitempty"`
	Relationship string             `json:"relationship" bson:"relationship"`
	NotifiedAt   time.Time          `json:"notifiedAt" bson:"notifiedAt"`
	NotifyMethod string             `json:"notifyMethod" bson:"notifyMethod"` // sms, call, push, email
	Acknowledged bool               `json:"acknowledged" bson:"acknowledged"`
	AckedAt      time.Time          `json:"ackedAt,omitempty" bson:"ackedAt,omitempty"`
	Response     string             `json:"response,omitempty" bson:"response,omitempty"`
}

type EmergencyMedia struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Type         string             `json:"type" bson:"type"` // photo, video, audio, document
	URL          string             `json:"url" bson:"url"`
	ThumbnailURL string             `json:"thumbnailUrl,omitempty" bson:"thumbnailUrl,omitempty"`
	FileName     string             `json:"fileName" bson:"fileName"`
	FileSize     int64              `json:"fileSize" bson:"fileSize"`
	MimeType     string             `json:"mimeType" bson:"mimeType"`
	Duration     int                `json:"duration,omitempty" bson:"duration,omitempty"`
	Location     EmergencyLocation  `json:"location,omitempty" bson:"location,omitempty"`
	UploadedBy   primitive.ObjectID `json:"uploadedBy" bson:"uploadedBy"`
	UploadedAt   time.Time          `json:"uploadedAt" bson:"uploadedAt"`
}

type EmergencyEvent struct {
	ID          primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Type        string                 `json:"type" bson:"type"`
	Description string                 `json:"description" bson:"description"`
	Actor       primitive.ObjectID     `json:"actor,omitempty" bson:"actor,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty" bson:"data,omitempty"`
	Timestamp   time.Time              `json:"timestamp" bson:"timestamp"`
}

type EmergencySettings struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID primitive.ObjectID `json:"userId" bson:"userId"`

	// Auto Detection
	CrashDetection bool `json:"crashDetection" bson:"crashDetection"`
	FallDetection  bool `json:"fallDetection" bson:"fallDetection"`
	HeartRateAlert bool `json:"heartRateAlert" bson:"heartRateAlert"`

	// Response Settings
	AutoCallEmergency  bool `json:"autoCallEmergency" bson:"autoCallEmergency"`
	AutoNotifyContacts bool `json:"autoNotifyContacts" bson:"autoNotifyContacts"`
	CountdownDuration  int  `json:"countdownDuration" bson:"countdownDuration"` // seconds before auto-call

	// Privacy
	ShareLocationAlways  bool `json:"shareLocationAlways" bson:"shareLocationAlways"`
	ShareWithAuthorities bool `json:"shareWithAuthorities" bson:"shareWithAuthorities"`

	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// Emergency Type Constants
const (
	EmergencyTypeSOS      = "sos"
	EmergencyTypeCrash    = "crash"
	EmergencyTypeHelp     = "help"
	EmergencyTypeMedical  = "medical"
	EmergencyTypeFire     = "fire"
	EmergencyTypePolice   = "police"
	EmergencyTypeRoadside = "roadside"
	EmergencyTypeFall     = "fall"
)

// Emergency Status Constants
const (
	EmergencyStatusActive     = "active"
	EmergencyStatusResolved   = "resolved"
	EmergencyStatusFalseAlarm = "false_alarm"
	EmergencyStatusCancelled  = "cancelled"
)

// Request DTOs
type CreateEmergencyRequest struct {
	Type        string            `json:"type" validate:"required,oneof=sos crash help medical fire police roadside"`
	Description string            `json:"description,omitempty"`
	Location    EmergencyLocation `json:"location" validate:"required"`
	Media       []string          `json:"media,omitempty"`
	Contacts    []string          `json:"contacts,omitempty"`
}

type UpdateEmergencyRequest struct {
	Status      string `json:"status,omitempty" validate:"omitempty,oneof=active resolved false_alarm cancelled"`
	Description string `json:"description,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
}

type EmergencyContactRequest struct {
	Name         string `json:"name" validate:"required"`
	Phone        string `json:"phone" validate:"required"`
	Email        string `json:"email,omitempty" validate:"omitempty,email"`
	Relationship string `json:"relationship" validate:"required"`
}
