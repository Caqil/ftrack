package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Core Emergency struct
type Emergency struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"userId" bson:"userId"`
	CircleID    primitive.ObjectID `json:"circleId,omitempty" bson:"circleId,omitempty"`
	Type        string             `json:"type" bson:"type"`
	Priority    string             `json:"priority" bson:"priority"`
	Status      string             `json:"status" bson:"status"`
	Title       string             `json:"title" bson:"title"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Location    EmergencyLocation  `json:"location" bson:"location"`
	Detection   EmergencyDetection `json:"detection" bson:"detection"`
	Response    EmergencyResponse  `json:"response" bson:"response"`
	Contacts    []EmergencyContact `json:"contacts,omitempty" bson:"contacts,omitempty"`
	Media       []EmergencyMedia   `json:"media,omitempty" bson:"media,omitempty"`
	Events      []EmergencyEvent   `json:"events,omitempty" bson:"events,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
	ResolvedAt  time.Time          `json:"resolvedAt,omitempty" bson:"resolvedAt,omitempty"`

	// Dismissal/Cancellation info
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
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt"`
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
	EmergencyStatusDismissed  = "dismissed"
)

// =================== REQUEST/RESPONSE MODELS ===================

// Basic Emergency Requests
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

// Alert Management
type CreateEmergencyAlertRequest struct {
	Type        string            `json:"type" validate:"required"`
	Title       string            `json:"title" validate:"required"`
	Description string            `json:"description,omitempty"`
	Priority    string            `json:"priority" validate:"required,oneof=low medium high critical"`
	Location    EmergencyLocation `json:"location" validate:"required"`
	CircleID    string            `json:"circleId,omitempty"`
	AutoResolve bool              `json:"autoResolve"`
	ExpiresAt   *time.Time        `json:"expiresAt,omitempty"`
}

type UpdateEmergencyAlertRequest struct {
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	Priority    string     `json:"priority,omitempty" validate:"omitempty,oneof=low medium high critical"`
	Status      string     `json:"status,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

type DismissAlertRequest struct {
	Reason string `json:"reason,omitempty"`
}

type ResolveAlertRequest struct {
	Resolution string `json:"resolution,omitempty"`
}

// SOS Functionality
type TriggerSOSRequest struct {
	Type         string            `json:"type" validate:"required,oneof=manual panic medical"`
	Message      string            `json:"message,omitempty"`
	Location     EmergencyLocation `json:"location" validate:"required"`
	Silent       bool              `json:"silent"`
	Contacts     []string          `json:"contacts,omitempty"`
	AutoCall     bool              `json:"autoCall"`
	CountdownSec int               `json:"countdownSec"`
}

type CancelSOSRequest struct {
	Reason    string `json:"reason,omitempty"`
	Confirmed bool   `json:"confirmed"`
}

type SOSSettingsRequest struct {
	AutoCall          bool     `json:"autoCall"`
	CountdownDuration int      `json:"countdownDuration"`
	DefaultMessage    string   `json:"defaultMessage,omitempty"`
	SilentMode        bool     `json:"silentMode"`
	RequireConfirm    bool     `json:"requireConfirm"`
	EmergencyContacts []string `json:"emergencyContacts,omitempty"`
	EmergencyNumbers  []string `json:"emergencyNumbers,omitempty"`
}

// Crash Detection
type CrashDetectionRequest struct {
	SensorData     map[string]interface{} `json:"sensorData" validate:"required"`
	Location       EmergencyLocation      `json:"location" validate:"required"`
	ImpactSeverity float64                `json:"impactSeverity"`
	Confidence     float64                `json:"confidence"`
	DeviceInfo     map[string]interface{} `json:"deviceInfo,omitempty"`
}

type ConfirmCrashRequest struct {
	Confirmed   bool   `json:"confirmed"`
	Injuries    bool   `json:"injuries"`
	Description string `json:"description,omitempty"`
	NeedHelp    bool   `json:"needHelp"`
}

type FalseAlarmRequest struct {
	Reason string `json:"reason,omitempty"`
}

type CrashDetectionSettingsRequest struct {
	Enabled           bool    `json:"enabled"`
	Sensitivity       float64 `json:"sensitivity" validate:"min=0,max=1"`
	CountdownDuration int     `json:"countdownDuration"`
	AutoCall          bool    `json:"autoCall"`
	RequireConfirm    bool    `json:"requireConfirm"`
}

type CalibrateCrashRequest struct {
	DeviceType      string                 `json:"deviceType"`
	CalibrationData map[string]interface{} `json:"calibrationData"`
}

// Emergency Contacts
type AddEmergencyContactRequest struct {
	Name          string   `json:"name" validate:"required"`
	Phone         string   `json:"phone" validate:"required"`
	Email         string   `json:"email,omitempty" validate:"omitempty,email"`
	Relationship  string   `json:"relationship" validate:"required"`
	Priority      int      `json:"priority"`
	NotifyMethods []string `json:"notifyMethods,omitempty"`
}

type UpdateEmergencyContactRequest struct {
	Name          string   `json:"name,omitempty"`
	Phone         string   `json:"phone,omitempty"`
	Email         string   `json:"email,omitempty" validate:"omitempty,email"`
	Relationship  string   `json:"relationship,omitempty"`
	Priority      int      `json:"priority,omitempty"`
	NotifyMethods []string `json:"notifyMethods,omitempty"`
}

type VerifyContactRequest struct {
	Method string `json:"method" validate:"required,oneof=sms call email"`
}

type NotifyContactRequest struct {
	Message string `json:"message" validate:"required"`
	Method  string `json:"method" validate:"required,oneof=sms call email push"`
	Urgent  bool   `json:"urgent"`
}

// Emergency Services
type EmergencyCallRequest struct {
	Location    EmergencyLocation `json:"location" validate:"required"`
	Description string            `json:"description,omitempty"`
	Urgent      bool              `json:"urgent"`
	Language    string            `json:"language,omitempty"`
}

type UpdateEmergencyNumbersRequest struct {
	Police   string `json:"police,omitempty"`
	Medical  string `json:"medical,omitempty"`
	Fire     string `json:"fire,omitempty"`
	Roadside string `json:"roadside,omitempty"`
	Country  string `json:"country,omitempty"`
}

// Location Sharing
type ShareLocationRequest struct {
	Recipients      []string   `json:"recipients" validate:"required"`
	Duration        int        `json:"duration"` // minutes, 0 for indefinite
	Message         string     `json:"message,omitempty"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
	ShareLevel      string     `json:"shareLevel" validate:"required,oneof=precise approximate general"`
	NotifyOnArrival bool       `json:"notifyOnArrival"`
}

type UpdateLocationShareRequest struct {
	Duration        int        `json:"duration,omitempty"`
	Message         string     `json:"message,omitempty"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
	ShareLevel      string     `json:"shareLevel,omitempty"`
	NotifyOnArrival bool       `json:"notifyOnArrival,omitempty"`
}

// Emergency Response
type EmergencyResponseRequest struct {
	Type     string            `json:"type" validate:"required,oneof=acknowledge offer_help eta status_update"`
	Message  string            `json:"message,omitempty"`
	ETA      int               `json:"eta,omitempty"` // minutes
	Location EmergencyLocation `json:"location,omitempty"`
	Skills   []string          `json:"skills,omitempty"`
	Capacity int               `json:"capacity,omitempty"`
}

type UpdateEmergencyResponseRequest struct {
	Message  string            `json:"message,omitempty"`
	ETA      int               `json:"eta,omitempty"`
	Location EmergencyLocation `json:"location,omitempty"`
	Status   string            `json:"status,omitempty"`
}

type RequestHelpRequest struct {
	Type        string   `json:"type" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Skills      []string `json:"skills,omitempty"`
	Urgency     string   `json:"urgency" validate:"required,oneof=low medium high critical"`
	Radius      float64  `json:"radius,omitempty"` // km
}

type OfferHelpRequest struct {
	Message  string            `json:"message" validate:"required"`
	Skills   []string          `json:"skills,omitempty"`
	ETA      int               `json:"eta,omitempty"`
	Location EmergencyLocation `json:"location,omitempty"`
	Capacity int               `json:"capacity,omitempty"`
}

// Check-in Safety
type SafeCheckInRequest struct {
	Message  string            `json:"message,omitempty"`
	Location EmergencyLocation `json:"location,omitempty"`
}

type NotSafeCheckInRequest struct {
	Issue       string            `json:"issue" validate:"required"`
	Severity    string            `json:"severity" validate:"required,oneof=low medium high critical"`
	Description string            `json:"description,omitempty"`
	Location    EmergencyLocation `json:"location" validate:"required"`
	NeedHelp    bool              `json:"needHelp"`
}

type CheckInSettingsRequest struct {
	Enabled           bool     `json:"enabled"`
	Frequency         int      `json:"frequency"` // hours
	AutoReminder      bool     `json:"autoReminder"`
	QuietHours        []string `json:"quietHours,omitempty"`
	EmergencyContacts []string `json:"emergencyContacts,omitempty"`
	GeoFencing        bool     `json:"geoFencing"`
}

type RequestCheckInRequest struct {
	Message string `json:"message,omitempty"`
	Urgent  bool   `json:"urgent"`
}

// History and Analytics
type ExportHistoryRequest struct {
	Format    string     `json:"format" validate:"required,oneof=json csv pdf"`
	StartDate *time.Time `json:"startDate,omitempty"`
	EndDate   *time.Time `json:"endDate,omitempty"`
	Types     []string   `json:"types,omitempty"`
	Include   []string   `json:"include,omitempty"` // events, media, contacts, etc.
}

// Settings
type EmergencySettingsRequest struct {
	CrashDetection       bool `json:"crashDetection"`
	FallDetection        bool `json:"fallDetection"`
	HeartRateAlert       bool `json:"heartRateAlert"`
	AutoCallEmergency    bool `json:"autoCallEmergency"`
	AutoNotifyContacts   bool `json:"autoNotifyContacts"`
	CountdownDuration    int  `json:"countdownDuration"`
	ShareLocationAlways  bool `json:"shareLocationAlways"`
	ShareWithAuthorities bool `json:"shareWithAuthorities"`
}

type EmergencyNotificationSettingsRequest struct {
	PushNotifications  bool     `json:"pushNotifications"`
	SMSNotifications   bool     `json:"smsNotifications"`
	EmailNotifications bool     `json:"emailNotifications"`
	CallNotifications  bool     `json:"callNotifications"`
	QuietHours         []string `json:"quietHours,omitempty"`
	NotificationSound  string   `json:"notificationSound,omitempty"`
	Vibration          bool     `json:"vibration"`
}

type EmergencyAutomationSettingsRequest struct {
	AutoTriggerRules        []AutoTriggerRule `json:"autoTriggerRules,omitempty"`
	AutoResponseEnabled     bool              `json:"autoResponseEnabled"`
	AutoLocationSharing     bool              `json:"autoLocationSharing"`
	AutoContactNotification bool              `json:"autoContactNotification"`
	SmartDetection          bool              `json:"smartDetection"`
}

type AutoTriggerRule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Trigger    string                 `json:"trigger"`
	Conditions map[string]interface{} `json:"conditions"`
	Actions    []string               `json:"actions"`
	Enabled    bool                   `json:"enabled"`
}

// Emergency Drills
type CreateEmergencyDrillRequest struct {
	Name        string    `json:"name" validate:"required"`
	Type        string    `json:"type" validate:"required,oneof=fire earthquake medical evacuation"`
	Description string    `json:"description,omitempty"`
	Scenario    string    `json:"scenario" validate:"required"`
	Duration    int       `json:"duration"` // minutes
	ScheduledAt time.Time `json:"scheduledAt,omitempty"`
	CircleID    string    `json:"circleId,omitempty"`
}

type CompleteEmergencyDrillRequest struct {
	CompletionTime int                    `json:"completionTime"` // seconds
	Success        bool                   `json:"success"`
	Issues         []string               `json:"issues,omitempty"`
	Feedback       string                 `json:"feedback,omitempty"`
	Participants   []string               `json:"participants,omitempty"`
	Results        map[string]interface{} `json:"results,omitempty"`
}

// Medical Information
type MedicalInformationRequest struct {
	BloodType        string   `json:"bloodType,omitempty"`
	Allergies        []string `json:"allergies,omitempty"`
	Medications      []string `json:"medications,omitempty"`
	Conditions       []string `json:"conditions,omitempty"`
	EmergencyContact string   `json:"emergencyContact,omitempty"`
	InsuranceInfo    string   `json:"insuranceInfo,omitempty"`
	DoctorContact    string   `json:"doctorContact,omitempty"`
	SpecialNeeds     string   `json:"specialNeeds,omitempty"`
}

type AllergiesRequest struct {
	Allergies []MedicalAllergy `json:"allergies"`
}

type MedicalAllergy struct {
	Name      string `json:"name" validate:"required"`
	Severity  string `json:"severity" validate:"required,oneof=mild moderate severe"`
	Reaction  string `json:"reaction,omitempty"`
	Treatment string `json:"treatment,omitempty"`
}

type MedicationsRequest struct {
	Medications []Medication `json:"medications"`
}

type Medication struct {
	Name       string `json:"name" validate:"required"`
	Dosage     string `json:"dosage" validate:"required"`
	Frequency  string `json:"frequency" validate:"required"`
	Purpose    string `json:"purpose,omitempty"`
	Prescriber string `json:"prescriber,omitempty"`
}

type MedicalConditionsRequest struct {
	Conditions []MedicalCondition `json:"conditions"`
}

type MedicalCondition struct {
	Name      string `json:"name" validate:"required"`
	Severity  string `json:"severity" validate:"required,oneof=mild moderate severe"`
	Treatment string `json:"treatment,omitempty"`
	Notes     string `json:"notes,omitempty"`
	Diagnosed string `json:"diagnosed,omitempty"`
}

// Emergency Broadcast
type BroadcastEmergencyRequest struct {
	Type       string     `json:"type" validate:"required,oneof=alert warning info update"`
	Title      string     `json:"title" validate:"required"`
	Message    string     `json:"message" validate:"required"`
	Priority   string     `json:"priority" validate:"required,oneof=low medium high critical"`
	Recipients []string   `json:"recipients,omitempty"` // circles or user IDs
	Channels   []string   `json:"channels" validate:"required"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	RequireAck bool       `json:"requireAck"`
}

type UpdateBroadcastRequest struct {
	Title      string     `json:"title,omitempty"`
	Message    string     `json:"message,omitempty"`
	Priority   string     `json:"priority,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	RequireAck bool       `json:"requireAck,omitempty"`
}

type AcknowledgeBroadcastRequest struct {
	Message  string            `json:"message,omitempty"`
	Location EmergencyLocation `json:"location,omitempty"`
	Status   string            `json:"status,omitempty"`
}

// =================== RESPONSE MODELS ===================

type EmergencyStats struct {
	TotalEmergencies    int64                   `json:"totalEmergencies"`
	ActiveEmergencies   int64                   `json:"activeEmergencies"`
	ResolvedEmergencies int64                   `json:"resolvedEmergencies"`
	FalseAlarms         int64                   `json:"falseAlarms"`
	ResponseTime        map[string]float64      `json:"responseTime"` // avg, min, max
	TypeBreakdown       map[string]int64        `json:"typeBreakdown"`
	MonthlyTrend        []MonthlyEmergencyStats `json:"monthlyTrend"`
	MostCommonTypes     []EmergencyTypeStats    `json:"mostCommonTypes"`
}

type MonthlyEmergencyStats struct {
	Month       string `json:"month"`
	Count       int64  `json:"count"`
	Resolved    int64  `json:"resolved"`
	FalseAlarms int64  `json:"falseAlarms"`
}

type EmergencyTypeStats struct {
	Type       string  `json:"type"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type SOSStatus struct {
	Active        bool      `json:"active"`
	TriggeredAt   time.Time `json:"triggeredAt,omitempty"`
	Type          string    `json:"type,omitempty"`
	CountdownLeft int       `json:"countdownLeft,omitempty"`
	AutoCall      bool      `json:"autoCall"`
}

type CheckInStatus struct {
	Status      string            `json:"status"` // safe, not_safe, overdue, unknown
	LastCheckIn time.Time         `json:"lastCheckIn,omitempty"`
	NextDue     time.Time         `json:"nextDue,omitempty"`
	Location    EmergencyLocation `json:"location,omitempty"`
}

type EmergencyFileExport struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Progress    float64   `json:"progress"`
	FileURL     string    `json:"fileUrl,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type EmergencyExportFile struct {
	Data        []byte
	Filename    string
	ContentType string
}

// =================== NOTIFICATION CONSTANTS ===================

// Emergency notification types
const (
	NotificationEmergencyMedical     = "emergency_medical"
	NotificationEmergencyAlert       = "emergency_alert"
	NotificationEmergencyResolved    = "emergency_resolved"
	NotificationEmergencyResponse    = "emergency_response"
	NotificationCheckInRequest       = "checkin_request"
	NotificationCheckInOverdue       = "checkin_overdue"
	NotificationEmergencyBroadcast   = "emergency_broadcast"
	NotificationLocationShareRequest = "location_share_request"
	NotificationHelpRequest          = "help_request"
	NotificationHelpOffer            = "help_offer"
	NotificationContactVerification  = "contact_verification"
	NotificationDrillReminder        = "drill_reminder"
)

// =================== ADDITIONAL UTILITY MODELS ===================

type EmergencyNotificationChannel struct {
	Type     string `json:"type"`    // sms, email, push, call
	Address  string `json:"address"` // phone, email, etc.
	Verified bool   `json:"verified"`
	Primary  bool   `json:"primary"`
}

type EmergencyResponseTeam struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name         string             `json:"name" bson:"name"`
	Type         string             `json:"type" bson:"type"` // medical, fire, police, rescue
	Members      []string           `json:"members" bson:"members"`
	Skills       []string           `json:"skills" bson:"skills"`
	Location     EmergencyLocation  `json:"location" bson:"location"`
	Available    bool               `json:"available" bson:"available"`
	ResponseTime int                `json:"responseTime" bson:"responseTime"` // minutes
}

type EmergencyZone struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Type        string             `json:"type" bson:"type"`               // danger, safe, evacuation
	Coordinates [][]float64        `json:"coordinates" bson:"coordinates"` // polygon coordinates
	Description string             `json:"description" bson:"description"`
	Active      bool               `json:"active" bson:"active"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	ExpiresAt   *time.Time         `json:"expiresAt,omitempty" bson:"expiresAt,omitempty"`
}

type EmergencyProtocol struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name          string             `json:"name" bson:"name"`
	Type          string             `json:"type" bson:"type"`
	Description   string             `json:"description" bson:"description"`
	Steps         []ProtocolStep     `json:"steps" bson:"steps"`
	Triggers      []string           `json:"triggers" bson:"triggers"`
	Automated     bool               `json:"automated" bson:"automated"`
	RequiredRoles []string           `json:"requiredRoles" bson:"requiredRoles"`
	Priority      string             `json:"priority" bson:"priority"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type ProtocolStep struct {
	ID           string                 `json:"id"`
	Order        int                    `json:"order"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Actions      []string               `json:"actions"`
	Required     bool                   `json:"required"`
	Automated    bool                   `json:"automated"`
	TimeLimit    int                    `json:"timeLimit,omitempty"` // seconds
	Dependencies []string               `json:"dependencies,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
}
