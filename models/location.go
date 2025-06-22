// models/location.go
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Location struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID primitive.ObjectID `json:"userId" bson:"userId"`

	// GPS Coordinates
	Latitude  float64 `json:"latitude" bson:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude float64 `json:"longitude" bson:"longitude" validate:"required,gte=-180,lte=180"`
	Accuracy  float64 `json:"accuracy" bson:"accuracy"` // GPS accuracy in meters
	Altitude  float64 `json:"altitude" bson:"altitude"` // Altitude in meters

	// Movement Data
	Speed        float64 `json:"speed" bson:"speed"`               // Speed in m/s
	Bearing      float64 `json:"bearing" bson:"bearing"`           // Direction in degrees (0-360)
	IsDriving    bool    `json:"isDriving" bson:"isDriving"`       // Auto-detected or manual
	IsMoving     bool    `json:"isMoving" bson:"isMoving"`         // Movement detection
	MovementType string  `json:"movementType" bson:"movementType"` // walking, driving, cycling, stationary

	// Device Information
	BatteryLevel   int    `json:"batteryLevel" bson:"batteryLevel"`     // 0-100
	IsCharging     bool   `json:"isCharging" bson:"isCharging"`         // Charging status
	DeviceType     string `json:"deviceType" bson:"deviceType"`         // ios, android
	NetworkType    string `json:"networkType" bson:"networkType"`       // wifi, cellular, gps
	SignalStrength int    `json:"signalStrength" bson:"signalStrength"` // Signal strength 0-100

	// Location Context & Geocoding
	Address    string `json:"address" bson:"address"`       // Reverse geocoded address
	Country    string `json:"country" bson:"country"`       // Country name
	State      string `json:"state" bson:"state"`           // State/Province
	City       string `json:"city" bson:"city"`             // City name
	PostalCode string `json:"postalCode" bson:"postalCode"` // ZIP/Postal code

	// Place Association
	PlaceID       primitive.ObjectID `json:"placeId,omitempty" bson:"placeId,omitempty"`
	PlaceName     string             `json:"placeName,omitempty" bson:"placeName,omitempty"`
	PlaceCategory string             `json:"placeCategory,omitempty" bson:"placeCategory,omitempty"`
	IsAtPlace     bool               `json:"isAtPlace" bson:"isAtPlace"`

	// Weather Information (optional)
	Weather WeatherInfo `json:"weather,omitempty" bson:"weather,omitempty"`

	// Privacy & Sharing
	Visibility string   `json:"visibility" bson:"visibility"` // public, circles, private
	SharedWith []string `json:"sharedWith" bson:"sharedWith"` // Circle IDs
	IsPrivate  bool     `json:"isPrivate" bson:"isPrivate"`   // Hide from all

	// Timing
	DeviceTime time.Time `json:"deviceTime" bson:"deviceTime"` // Time from device
	ServerTime time.Time `json:"serverTime" bson:"serverTime"` // Server received time
	Timezone   string    `json:"timezone" bson:"timezone"`     // Device timezone

	// Quality & Reliability
	Source       string  `json:"source" bson:"source"`         // gps, network, passive
	Confidence   float64 `json:"confidence" bson:"confidence"` // 0.0 to 1.0
	IsFiltered   bool    `json:"isFiltered" bson:"isFiltered"` // Passed quality filters
	FilterReason string  `json:"filterReason,omitempty" bson:"filterReason,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
	ExpiresAt time.Time `json:"expiresAt,omitempty" bson:"expiresAt,omitempty"` // Auto-cleanup
}

type WeatherInfo struct {
	Temperature   float64 `json:"temperature" bson:"temperature"`     // Celsius
	Humidity      int     `json:"humidity" bson:"humidity"`           // Percentage
	Condition     string  `json:"condition" bson:"condition"`         // sunny, cloudy, rainy, etc.
	WindSpeed     float64 `json:"windSpeed" bson:"windSpeed"`         // km/h
	WindDirection int     `json:"windDirection" bson:"windDirection"` // Degrees
	Visibility    float64 `json:"visibility" bson:"visibility"`       // km
	UVIndex       int     `json:"uvIndex" bson:"uvIndex"`             // 0-11
}

// Location tracking session
type LocationSession struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID `json:"userId" bson:"userId"`
	StartTime     time.Time          `json:"startTime" bson:"startTime"`
	EndTime       time.Time          `json:"endTime,omitempty" bson:"endTime,omitempty"`
	IsActive      bool               `json:"isActive" bson:"isActive"`
	LocationCount int                `json:"locationCount" bson:"locationCount"`

	// Session Statistics
	Stats LocationSessionStats `json:"stats" bson:"stats"`

	// Session Type
	Type    string `json:"type" bson:"type"`       // manual, auto, emergency, tracking
	Purpose string `json:"purpose" bson:"purpose"` // family_tracking, emergency, trip

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type LocationSessionStats struct {
	TotalDistance   float64 `json:"totalDistance" bson:"totalDistance"`     // meters
	MaxSpeed        float64 `json:"maxSpeed" bson:"maxSpeed"`               // m/s
	AverageSpeed    float64 `json:"averageSpeed" bson:"averageSpeed"`       // m/s
	TimeMoving      int64   `json:"timeMoving" bson:"timeMoving"`           // seconds
	TimeStationary  int64   `json:"timeStationary" bson:"timeStationary"`   // seconds
	BatteryConsumed int     `json:"batteryConsumed" bson:"batteryConsumed"` // percentage
	PlacesVisited   int     `json:"placesVisited" bson:"placesVisited"`     // count
	AccuracyAverage float64 `json:"accuracyAverage" bson:"accuracyAverage"` // meters
}

// Safety score tracking
type SafetyScore struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID primitive.ObjectID `json:"userId" bson:"userId"`

	// Scores
	OverallScore  int `json:"overallScore" bson:"overallScore"`   // 0-100
	SpeedingScore int `json:"speedingScore" bson:"speedingScore"` // 0-100
	BrakingScore  int `json:"brakingScore" bson:"brakingScore"`   // 0-100
	AccelScore    int `json:"accelScore" bson:"accelScore"`       // 0-100
	TurningScore  int `json:"turningScore" bson:"turningScore"`   // 0-100
	PhoneScore    int `json:"phoneScore" bson:"phoneScore"`       // 0-100

	// Statistics
	TotalTrips    int     `json:"totalTrips" bson:"totalTrips"`
	TotalDistance float64 `json:"totalDistance" bson:"totalDistance"` // km
	TotalTime     int64   `json:"totalTime" bson:"totalTime"`         // seconds

	// Recent Events
	RecentEvents []DrivingEvent `json:"recentEvents" bson:"recentEvents"`

	// Period
	Period    string    `json:"period" bson:"period"` // daily, weekly, monthly, overall
	StartDate time.Time `json:"startDate" bson:"startDate"`
	EndDate   time.Time `json:"endDate" bson:"endDate"`

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// Location sharing preferences
type LocationSharing struct {
	Enabled         bool     `json:"enabled" bson:"enabled"`
	Precision       string   `json:"precision" bson:"precision"`   // exact, approximate, city
	ShareWith       []string `json:"shareWith" bson:"shareWith"`   // Circle IDs
	UpdateFrequency int      `json:"updateFreq" bson:"updateFreq"` // seconds

	// Advanced Settings
	ShareDriving bool `json:"shareDriving" bson:"shareDriving"`
	SharePlaces  bool `json:"sharePlaces" bson:"sharePlaces"`
	ShareBattery bool `json:"shareBattery" bson:"shareBattery"`

	// Privacy Modes
	StealthMode bool `json:"stealthMode" bson:"stealthMode"` // Hide from specific people
	WorkMode    bool `json:"workMode" bson:"workMode"`       // Limited sharing during work
	SleepMode   bool `json:"sleepMode" bson:"sleepMode"`     // Reduced sharing at night

	// Schedule-based sharing
	Schedule LocationSchedule `json:"schedule" bson:"schedule"`
}

type LocationSchedule struct {
	Enabled        bool                   `json:"enabled" bson:"enabled"`
	WeeklySchedule map[string]DaySchedule `json:"weeklySchedule" bson:"weeklySchedule"` // monday, tuesday, etc.
	Timezone       string                 `json:"timezone" bson:"timezone"`
}

type DaySchedule struct {
	Enabled   bool   `json:"enabled" bson:"enabled"`
	StartTime string `json:"startTime" bson:"startTime"` // HH:MM
	EndTime   string `json:"endTime" bson:"endTime"`     // HH:MM
	Precision string `json:"precision" bson:"precision"` // exact, approximate, city
}

// Request/Response DTOs
type LocationUpdateRequest struct {
	Latitude     float64 `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude    float64 `json:"longitude" validate:"required,gte=-180,lte=180"`
	Accuracy     float64 `json:"accuracy"`
	Altitude     float64 `json:"altitude"`
	Speed        float64 `json:"speed"`
	Bearing      float64 `json:"bearing"`
	BatteryLevel int     `json:"batteryLevel" validate:"gte=0,lte=100"`
	IsCharging   bool    `json:"isCharging"`
	IsDriving    bool    `json:"isDriving"`
	IsMoving     bool    `json:"isMoving"`
	MovementType string  `json:"movementType"`
	NetworkType  string  `json:"networkType"`
	Source       string  `json:"source"`
	DeviceTime   string  `json:"deviceTime"` // RFC3339 format
	Timezone     string  `json:"timezone"`
}

type LocationHistoryRequest struct {
	UserID      string    `json:"userId" validate:"required"`
	StartDate   time.Time `json:"startDate" validate:"required"`
	EndDate     time.Time `json:"endDate" validate:"required"`
	Limit       int       `json:"limit" validate:"min=1,max=1000"`
	Source      string    `json:"source,omitempty"`
	MinAccuracy float64   `json:"minAccuracy,omitempty"`
}

type LocationStatsRequest struct {
	UserID    string    `json:"userId" validate:"required"`
	StartDate time.Time `json:"startDate" validate:"required"`
	EndDate   time.Time `json:"endDate" validate:"required"`
	Period    string    `json:"period"` // day, week, month
}

type BulkLocationUpdateRequest struct {
	Locations []LocationUpdateRequest `json:"locations" validate:"required,min=1,max=100"`
}

type NearbyUsersRequest struct {
	Latitude  float64 `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude float64 `json:"longitude" validate:"required,gte=-180,lte=180"`
	Radius    float64 `json:"radius" validate:"required,min=100,max=50000"` // meters
	CircleID  string  `json:"circleId,omitempty"`
}

type LocationSharingUpdateRequest struct {
	Settings LocationSharing `json:"settings" validate:"required"`
}

type TripStartRequest struct {
	Name           string `json:"name" validate:"required,min=1,max=100"`
	Description    string `json:"description,omitempty"`
	Type           string `json:"type,omitempty"`
	Transportation string `json:"transportation,omitempty"`
	Purpose        string `json:"purpose,omitempty"`
}

type TripEndRequest struct {
	TripID string `json:"tripId" validate:"required"`
}

// Response DTOs
type LocationResponse struct {
	Location  Location `json:"location"`
	Distance  float64  `json:"distance,omitempty"`  // Distance from requester
	Direction string   `json:"direction,omitempty"` // Relative direction
	LastSeen  string   `json:"lastSeen"`            // Relative time
}

type LocationStatsResponse struct {
	TotalDistance   float64 `json:"totalDistance"` // meters
	TotalTime       int64   `json:"totalTime"`     // seconds
	AverageSpeed    float64 `json:"averageSpeed"`  // m/s
	MaxSpeed        float64 `json:"maxSpeed"`      // m/s
	PlacesVisited   int     `json:"placesVisited"`
	TripsCount      int     `json:"tripsCount"`
	DrivingTime     int64   `json:"drivingTime"`     // seconds
	WalkingTime     int64   `json:"walkingTime"`     // seconds
	StationaryTime  int64   `json:"stationaryTime"`  // seconds
	BatteryConsumed int     `json:"batteryConsumed"` // percentage
	CO2Footprint    float64 `json:"co2Footprint"`    // kg
	SafetyScore     int     `json:"safetyScore"`     // 0-100
}

type CircleLocationsResponse struct {
	Locations map[string]LocationResponse `json:"locations"` // userID -> location
	Center    Coordinate                  `json:"center"`    // Center of all locations
	Radius    float64                     `json:"radius"`    // Radius containing all members
	UpdatedAt time.Time                   `json:"updatedAt"`
}

type NearbyUsersResponse struct {
	Users  []NearbyUser `json:"users"`
	Count  int          `json:"count"`
	Radius float64      `json:"radius"`
	Center Coordinate   `json:"center"`
}

// LocationSettings represents user's location sharing settings
type LocationSettings struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          string             `json:"userId" bson:"userId"`
	Enabled         bool               `json:"enabled" bson:"enabled"`
	UpdateFrequency int                `json:"updateFrequency" bson:"updateFrequency"` // seconds
	Precision       string             `json:"precision" bson:"precision"`             // exact, approximate, city
	ShareWith       []string           `json:"shareWith" bson:"shareWith"`             // Circle IDs
	ShareDriving    bool               `json:"shareDriving" bson:"shareDriving"`
	SharePlaces     bool               `json:"sharePlaces" bson:"sharePlaces"`
	ShareBattery    bool               `json:"shareBattery" bson:"shareBattery"`
	StealthMode     bool               `json:"stealthMode" bson:"stealthMode"`
	WorkMode        bool               `json:"workMode" bson:"workMode"`
	SleepMode       bool               `json:"sleepMode" bson:"sleepMode"`
	CreatedAt       time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// SharingPermissions represents who can see user's location
type SharingPermissions struct {
	ID        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserID    string               `json:"userId" bson:"userId"`
	Circles   []CirclePermission   `json:"circles" bson:"circles"`
	Users     []UserPermission     `json:"users" bson:"users"`
	Emergency EmergencyPermissions `json:"emergency" bson:"emergency"`
	CreatedAt time.Time            `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt" bson:"updatedAt"`
}

type CirclePermission struct {
	CircleID        string          `json:"circleId" bson:"circleId"`
	CanView         bool            `json:"canView" bson:"canView"`
	CanTrack        bool            `json:"canTrack" bson:"canTrack"`
	Precision       string          `json:"precision" bson:"precision"` // exact, approximate, city
	TimeRestriction TimeRestriction `json:"timeRestriction" bson:"timeRestriction"`
}

type UserPermission struct {
	UserID    string `json:"userId" bson:"userId"`
	CanView   bool   `json:"canView" bson:"canView"`
	CanTrack  bool   `json:"canTrack" bson:"canTrack"`
	Precision string `json:"precision" bson:"precision"`
	Duration  int    `json:"duration" bson:"duration"` // seconds, 0 = permanent
}

type EmergencyPermissions struct {
	AutoShare         bool     `json:"autoShare" bson:"autoShare"`
	ShareWithAll      bool     `json:"shareWithAll" bson:"shareWithAll"`
	EmergencyContacts []string `json:"emergencyContacts" bson:"emergencyContacts"`
	Duration          int      `json:"duration" bson:"duration"` // seconds
}

type TimeRestriction struct {
	Enabled  bool                  `json:"enabled" bson:"enabled"`
	Schedule map[string][]TimeSlot `json:"schedule" bson:"schedule"` // monday, tuesday, etc.
}

type TimeSlot struct {
	Start string `json:"start" bson:"start"` // HH:MM format
	End   string `json:"end" bson:"end"`     // HH:MM format
}

// ==================== TEMPORARY SHARING ====================

type TemporaryShare struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID     string             `json:"userId" bson:"userId"`
	ShareCode  string             `json:"shareCode" bson:"shareCode"`
	Duration   int                `json:"duration" bson:"duration"` // seconds
	Recipients []string           `json:"recipients" bson:"recipients"`
	Message    string             `json:"message" bson:"message"`
	ExpiresAt  time.Time          `json:"expiresAt" bson:"expiresAt"`
	IsActive   bool               `json:"isActive" bson:"isActive"`
	UsageCount int                `json:"usageCount" bson:"usageCount"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
}

type TemporaryShareRequest struct {
	Duration   int      `json:"duration" validate:"required,min=60,max=86400"` // 1 minute to 24 hours
	Recipients []string `json:"recipients,omitempty"`
	Message    string   `json:"message,omitempty"`
}

// ==================== PROXIMITY ALERTS ====================

type ProximityAlert struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        string             `json:"userId" bson:"userId"`
	TargetUserID  string             `json:"targetUserId" bson:"targetUserId"`
	Radius        float64            `json:"radius" bson:"radius"`       // meters
	AlertType     string             `json:"alertType" bson:"alertType"` // entering, leaving, both
	IsActive      bool               `json:"isActive" bson:"isActive"`
	LastTriggered *time.Time         `json:"lastTriggered,omitempty" bson:"lastTriggered,omitempty"`
	TriggerCount  int                `json:"triggerCount" bson:"triggerCount"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
}

type ProximityAlertRequest struct {
	TargetUserID string  `json:"targetUserId" validate:"required"`
	Radius       float64 `json:"radius" validate:"required,min=10,max=10000"`
	AlertType    string  `json:"alertType" validate:"required,oneof=entering leaving both"`
}

type ProximityAlertUpdate struct {
	Radius    *float64 `json:"radius,omitempty"`
	AlertType *string  `json:"alertType,omitempty"`
	IsActive  *bool    `json:"isActive,omitempty"`
}

type NearbyUser struct {
	UserID         string    `json:"userId"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	ProfilePicture string    `json:"profilePicture"`
	Location       Location  `json:"location"`
	Distance       float64   `json:"distance"`  // meters
	Direction      string    `json:"direction"` // N, NE, E, SE, S, SW, W, NW
	LastSeen       time.Time `json:"lastSeen"`
	IsOnline       bool      `json:"isOnline"`
}

// ==================== TRIP MANAGEMENT ====================

type Trip struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID         string             `json:"userId" bson:"userId"`
	Name           string             `json:"name" bson:"name"`
	Description    string             `json:"description" bson:"description"`
	Type           string             `json:"type" bson:"type"`                     // business, personal, vacation, commute
	Transportation string             `json:"transportation" bson:"transportation"` // car, walk, bike, transit
	Purpose        string             `json:"purpose" bson:"purpose"`
	StartTime      time.Time          `json:"startTime" bson:"startTime"`
	EndTime        *time.Time         `json:"endTime,omitempty" bson:"endTime,omitempty"`
	IsActive       bool               `json:"isActive" bson:"isActive"`
	Stats          *TripStats         `json:"stats,omitempty" bson:"stats,omitempty"`
	Route          []Location         `json:"route,omitempty" bson:"route,omitempty"`
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type TripStats struct {
	TotalDistance float64   `json:"totalDistance"` // meters
	TotalTime     int64     `json:"totalTime"`     // seconds
	MaxSpeed      float64   `json:"maxSpeed"`      // m/s
	AvgSpeed      float64   `json:"avgSpeed"`      // m/s
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	PlacesVisited int       `json:"placesVisited"`
	Countries     []string  `json:"countries"`
	Cities        []string  `json:"cities"`
}

type TripRoute struct {
	TripID    string     `json:"tripId"`
	Points    []Location `json:"points"`
	Waypoints []Location `json:"waypoints"`
	Distance  float64    `json:"distance"`
	Duration  int64      `json:"duration"`
}

type TripUpdate struct {
	Name           *string    `json:"name,omitempty"`
	Description    *string    `json:"description,omitempty"`
	Type           *string    `json:"type,omitempty"`
	Transportation *string    `json:"transportation,omitempty"`
	Purpose        *string    `json:"purpose,omitempty"`
	EndTime        *time.Time `json:"endTime,omitempty"`
	IsActive       *bool      `json:"isActive,omitempty"`
	Stats          *TripStats `json:"stats,omitempty"`
}

type StartTripRequest struct {
	Name           string `json:"name" validate:"required,min=1,max=100"`
	Description    string `json:"description,omitempty"`
	Type           string `json:"type,omitempty"`
	Transportation string `json:"transportation,omitempty"`
	Purpose        string `json:"purpose,omitempty"`
}

type TripShare struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TripID     string             `json:"tripId" bson:"tripId"`
	UserID     string             `json:"userId" bson:"userId"`
	SharedWith []string           `json:"sharedWith" bson:"sharedWith"`
	ShareCode  string             `json:"shareCode" bson:"shareCode"`
	ExpiresAt  time.Time          `json:"expiresAt" bson:"expiresAt"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
}

type ShareTripRequest struct {
	SharedWith []string `json:"sharedWith" validate:"required"`
	Duration   int      `json:"duration" validate:"required,min=3600,max=2592000"` // 1 hour to 30 days
}

// ==================== DRIVING DETECTION ====================

type DrivingStatus struct {
	UserID      string     `json:"userId"`
	IsDriving   bool       `json:"isDriving"`
	StartTime   *time.Time `json:"startTime,omitempty"`
	CurrentTrip *Trip      `json:"currentTrip,omitempty"`
	Score       int        `json:"score"` // 0-100
	LastUpdate  time.Time  `json:"lastUpdate"`
}

type DrivingSession struct {
	ID          primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserID      string               `json:"userId" bson:"userId"`
	StartTime   time.Time            `json:"startTime" bson:"startTime"`
	EndTime     *time.Time           `json:"endTime,omitempty" bson:"endTime,omitempty"`
	IsActive    bool                 `json:"isActive" bson:"isActive"`
	VehicleType string               `json:"vehicleType" bson:"vehicleType"`
	Stats       *DrivingSessionStats `json:"stats,omitempty" bson:"stats,omitempty"`
	Events      []DrivingEvent       `json:"events,omitempty" bson:"events,omitempty"`
	CreatedAt   time.Time            `json:"createdAt" bson:"createdAt"`
}

type DrivingSessionStats struct {
	TotalDistance   float64 `json:"totalDistance"`   // meters
	MaxSpeed        float64 `json:"maxSpeed"`        // m/s
	AverageSpeed    float64 `json:"averageSpeed"`    // m/s
	TimeMoving      int64   `json:"timeMoving"`      // seconds
	TimeStationary  int64   `json:"timeStationary"`  // seconds
	BatteryConsumed int     `json:"batteryConsumed"` // percentage
	HardBrakes      int     `json:"hardBrakes"`
	RapidAccel      int     `json:"rapidAccel"`
	SharpTurns      int     `json:"sharpTurns"`
	PhoneUsage      int64   `json:"phoneUsage"` // seconds
}

type DrivingEvent struct {
	ID        primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	UserID    string                 `json:"userId" bson:"userId"`
	SessionID string                 `json:"sessionId,omitempty" bson:"sessionId,omitempty"`
	EventType string                 `json:"eventType" bson:"eventType"` // hard_brake, rapid_accel, sharp_turn, speeding, phone_use
	Severity  string                 `json:"severity" bson:"severity"`   // low, medium, high
	Location  Location               `json:"location" bson:"location"`
	Details   map[string]interface{} `json:"details" bson:"details"`
	Timestamp time.Time              `json:"timestamp" bson:"timestamp"`
	CreatedAt time.Time              `json:"createdAt" bson:"createdAt"`
}

type DrivingReport struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       string             `json:"userId" bson:"userId"`
	Period       string             `json:"period" bson:"period"` // daily, weekly, monthly
	StartDate    time.Time          `json:"startDate" bson:"startDate"`
	EndDate      time.Time          `json:"endDate" bson:"endDate"`
	OverallScore int                `json:"overallScore" bson:"overallScore"` // 0-100
	Scores       DrivingScores      `json:"scores" bson:"scores"`
	Stats        DrivingReportStats `json:"stats" bson:"stats"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
}

// DrivingScore - was referenced in repository but not defined
type DrivingScore struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       string             `json:"userId" bson:"userId"`
	OverallScore int                `json:"overallScore" bson:"overallScore"` // 0-100

	// Detailed Scores
	SpeedingScore int `json:"speedingScore" bson:"speedingScore"` // 0-100
	BrakingScore  int `json:"brakingScore" bson:"brakingScore"`   // 0-100
	AccelScore    int `json:"accelScore" bson:"accelScore"`       // 0-100
	TurningScore  int `json:"turningScore" bson:"turningScore"`   // 0-100
	PhoneScore    int `json:"phoneScore" bson:"phoneScore"`       // 0-100

	// Trip Statistics
	TotalTrips    int     `json:"totalTrips" bson:"totalTrips"`
	TotalDistance float64 `json:"totalDistance" bson:"totalDistance"` // km
	TotalTime     int64   `json:"totalTime" bson:"totalTime"`         // seconds

	// Event Counts
	HardBrakes     int `json:"hardBrakes" bson:"hardBrakes"`
	RapidAccels    int `json:"rapidAccels" bson:"rapidAccels"`
	SharpTurns     int `json:"sharpTurns" bson:"sharpTurns"`
	SpeedingEvents int `json:"speedingEvents" bson:"speedingEvents"`
	PhoneEvents    int `json:"phoneEvents" bson:"phoneEvents"`

	// Period Information
	Period    string    `json:"period" bson:"period"` // daily, weekly, monthly, overall
	StartDate time.Time `json:"startDate" bson:"startDate"`
	EndDate   time.Time `json:"endDate" bson:"endDate"`

	// Improvements and Trends
	ScoreChange     int      `json:"scoreChange" bson:"scoreChange"`         // +/- from previous period
	Improvements    []string `json:"improvements" bson:"improvements"`       // Areas of improvement
	AreasOfConcern  []string `json:"areasOfConcern" bson:"areasOfConcern"`   // Areas needing attention
	Recommendations []string `json:"recommendations" bson:"recommendations"` // Driving tips

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}
type DrivingScores struct {
	SpeedingScore int `json:"speedingScore"` // 0-100
	BrakingScore  int `json:"brakingScore"`  // 0-100
	AccelScore    int `json:"accelScore"`    // 0-100
	TurningScore  int `json:"turningScore"`  // 0-100
	PhoneScore    int `json:"phoneScore"`    // 0-100
}

type DrivingReportStats struct {
	TotalTrips    int            `json:"totalTrips"`
	TotalDistance float64        `json:"totalDistance"` // km
	TotalTime     int64          `json:"totalTime"`     // seconds
	EventCounts   map[string]int `json:"eventCounts"`
}

type StartDrivingRequest struct {
	VehicleType string `json:"vehicleType,omitempty"`
}

type DrivingEventRequest struct {
	EventType string                 `json:"eventType" validate:"required"`
	Severity  string                 `json:"severity" validate:"required,oneof=low medium high"`
	Location  Location               `json:"location" validate:"required"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// ==================== ANALYTICS ====================

type LocationStats struct {
	UserID           string                 `json:"userId"`
	Period           string                 `json:"period"`
	TotalDistance    float64                `json:"totalDistance"` // meters
	TotalTime        int64                  `json:"totalTime"`     // seconds
	PlacesVisited    int                    `json:"placesVisited"`
	CountriesVisited int                    `json:"countriesVisited"`
	CitiesVisited    int                    `json:"citiesVisited"`
	MostVisitedPlace string                 `json:"mostVisitedPlace"`
	DailyStats       []DailyLocationStats   `json:"dailyStats"`
	ActivityPattern  map[string]interface{} `json:"activityPattern"`
}

type DailyLocationStats struct {
	Date          string  `json:"date"`
	Distance      float64 `json:"distance"`
	Time          int64   `json:"time"`
	PlacesVisited int     `json:"placesVisited"`
}

type LocationHeatmap struct {
	UserID    string         `json:"userId"`
	Period    string         `json:"period"`
	Points    []HeatmapPoint `json:"points"`
	Bounds    GeoBounds      `json:"bounds"`
	Generated time.Time      `json:"generated"`
}

type HeatmapPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Weight    float64 `json:"weight"` // 0.0 to 1.0
	Count     int     `json:"count"`
}

type GeoBounds struct {
	Northeast Coordinate `json:"northeast"`
	Southwest Coordinate `json:"southwest"`
}

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type LocationPatterns struct {
	UserID         string                `json:"userId"`
	HomeLocation   *Location             `json:"homeLocation,omitempty"`
	WorkLocation   *Location             `json:"workLocation,omitempty"`
	CommutePattern *CommutePattern       `json:"commutePattern,omitempty"`
	WeeklyPattern  map[string][]Location `json:"weeklyPattern"`
	FrequentPlaces []FrequentPlace       `json:"frequentPlaces"`
	TravelPatterns []TravelPattern       `json:"travelPatterns"`
	Generated      time.Time             `json:"generated"`
}

type CommutePattern struct {
	HomeToWork      CommuteStats `json:"homeToWork"`
	WorkToHome      CommuteStats `json:"workToHome"`
	AverageTime     int64        `json:"averageTime"`     // seconds
	AverageDistance float64      `json:"averageDistance"` // meters
}

type CommuteStats struct {
	AverageTime     int64      `json:"averageTime"`     // seconds
	AverageDistance float64    `json:"averageDistance"` // meters
	UsualStartTime  string     `json:"usualStartTime"`  // HH:MM
	Route           []Location `json:"route"`
}

type FrequentPlace struct {
	Location   Location  `json:"location"`
	Name       string    `json:"name"`
	Category   string    `json:"category"`
	VisitCount int       `json:"visitCount"`
	TotalTime  int64     `json:"totalTime"` // seconds
	LastVisit  time.Time `json:"lastVisit"`
}

type TravelPattern struct {
	FromLocation Location  `json:"fromLocation"`
	ToLocation   Location  `json:"toLocation"`
	Frequency    int       `json:"frequency"`
	AverageTime  int64     `json:"averageTime"` // seconds
	LastTravel   time.Time `json:"lastTravel"`
}

type LocationInsights struct {
	UserID          string           `json:"userId"`
	Summary         InsightSummary   `json:"summary"`
	Insights        []Insight        `json:"insights"`
	Recommendations []Recommendation `json:"recommendations"`
	Generated       time.Time        `json:"generated"`
}

type InsightSummary struct {
	TotalDistance    float64 `json:"totalDistance"`
	MostVisitedPlace string  `json:"mostVisitedPlace"`
	LongestTrip      string  `json:"longestTrip"`
	FavoriteTime     string  `json:"favoriteTime"`
	ActivityLevel    string  `json:"activityLevel"` // low, medium, high
}

type Insight struct {
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Priority    string                 `json:"priority"` // low, medium, high
}

type Recommendation struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

type LocationTimeline struct {
	UserID    string          `json:"userId"`
	StartDate time.Time       `json:"startDate"`
	EndDate   time.Time       `json:"endDate"`
	Events    []TimelineEvent `json:"events"`
}

type TimelineEvent struct {
	Type      string                 `json:"type"` // arrival, departure, travel, place_visit
	Timestamp time.Time              `json:"timestamp"`
	Location  Location               `json:"location"`
	Place     *string                `json:"place,omitempty"`
	Duration  int64                  `json:"duration,omitempty"` // seconds
	Details   map[string]interface{} `json:"details,omitempty"`
}

type LocationSummary struct {
	UserID    string                 `json:"userId"`
	Period    string                 `json:"period"`
	Stats     LocationStats          `json:"stats"`
	TopPlaces []FrequentPlace        `json:"topPlaces"`
	Activity  map[string]interface{} `json:"activity"`
}

// ==================== GEOFENCING ====================

type GeofenceEvent struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    string             `json:"userId" bson:"userId"`
	PlaceID   string             `json:"placeId" bson:"placeId"`
	PlaceName string             `json:"placeName" bson:"placeName"`
	EventType string             `json:"eventType" bson:"eventType"` // enter, exit
	Location  Location           `json:"location" bson:"location"`
	Timestamp time.Time          `json:"timestamp" bson:"timestamp"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}

type GeofenceTestRequest struct {
	Latitude  float64 `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude float64 `json:"longitude" validate:"required,gte=-180,lte=180"`
	Radius    float64 `json:"radius" validate:"required,min=10,max=5000"`
}

type GeofenceTestResult struct {
	IsInside bool     `json:"isInside"`
	Distance float64  `json:"distance"` // meters from center
	Location Location `json:"location"`
}

type GeofenceStatus struct {
	UserID       string    `json:"userId"`
	ActiveFences int       `json:"activeFences"`
	LastUpdate   time.Time `json:"lastUpdate"`
	Status       string    `json:"status"` // active, inactive, error
}

// ==================== DATA MANAGEMENT ====================

type LocationExport struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    string             `json:"userId" bson:"userId"`
	DataTypes []string           `json:"dataTypes" bson:"dataTypes"` // locations, trips, places, events
	StartDate *time.Time         `json:"startDate,omitempty" bson:"startDate,omitempty"`
	EndDate   *time.Time         `json:"endDate,omitempty" bson:"endDate,omitempty"`
	Format    string             `json:"format" bson:"format"` // json, csv, kml, gpx
	Status    string             `json:"status" bson:"status"` // pending, processing, completed, failed
	FileURL   string             `json:"fileUrl,omitempty" bson:"fileUrl,omitempty"`
	FileSize  int64              `json:"fileSize,omitempty" bson:"fileSize,omitempty"`
	Progress  int                `json:"progress" bson:"progress"` // 0-100
	Error     string             `json:"error,omitempty" bson:"error,omitempty"`
	ExpiresAt time.Time          `json:"expiresAt" bson:"expiresAt"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type LocationExportRequest struct {
	DataTypes []string   `json:"dataTypes" validate:"required"` // locations, trips, places, events
	StartDate *time.Time `json:"startDate,omitempty"`
	EndDate   *time.Time `json:"endDate,omitempty"`
	Format    string     `json:"format" validate:"required,oneof=json csv kml gpx"`
}

type LocationPurgeRequest struct {
	DataTypes []string   `json:"dataTypes" validate:"required"` // locations, trips, places, events
	StartDate *time.Time `json:"startDate,omitempty"`
	EndDate   *time.Time `json:"endDate,omitempty"`
	Confirm   bool       `json:"confirm" validate:"required"`
}

type LocationPurgeResult struct {
	LocationsDeleted int `json:"locationsDeleted"`
	TripsDeleted     int `json:"tripsDeleted"`
	PlacesDeleted    int `json:"placesDeleted"`
	EventsDeleted    int `json:"eventsDeleted"`
}

type DataUsage struct {
	UserID        string    `json:"userId"`
	LocationCount int64     `json:"locationCount"`
	TripCount     int64     `json:"tripCount"`
	PlaceCount    int64     `json:"placeCount"`
	EventCount    int64     `json:"eventCount"`
	StorageUsed   int64     `json:"storageUsed"` // bytes
	LastUpdate    time.Time `json:"lastUpdate"`
	RetentionDays int       `json:"retentionDays"`
}

// ==================== EMERGENCY ====================

type EmergencyLocationShare struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        string             `json:"userId" bson:"userId"`
	ShareCode     string             `json:"shareCode" bson:"shareCode"`
	EmergencyType string             `json:"emergencyType" bson:"emergencyType"`
	Duration      int                `json:"duration" bson:"duration"` // seconds
	SharedWith    []string           `json:"sharedWith" bson:"sharedWith"`
	ExpiresAt     time.Time          `json:"expiresAt" bson:"expiresAt"`
	IsActive      bool               `json:"isActive" bson:"isActive"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
}

type EmergencyLocationRequest struct {
	EmergencyType string   `json:"emergencyType" validate:"required"`
	Duration      int      `json:"duration" validate:"required,min=300,max=86400"` // 5 minutes to 24 hours
	SharedWith    []string `json:"sharedWith,omitempty"`
}

type LocationPing struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       string             `json:"userId" bson:"userId"`
	TargetUserID string             `json:"targetUserId" bson:"targetUserId"`
	Location     Location           `json:"location" bson:"location"`
	Message      string             `json:"message" bson:"message"`
	ExpiresAt    time.Time          `json:"expiresAt" bson:"expiresAt"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
}

type LocationPingRequest struct {
	TargetUserID string `json:"targetUserId" validate:"required"`
	Message      string `json:"message,omitempty"`
	Duration     int    `json:"duration" validate:"required,min=60,max=3600"` // 1 minute to 1 hour
}

// ==================== CALIBRATION ====================

type LocationAccuracy struct {
	UserID          string    `json:"userId"`
	GPSAccuracy     float64   `json:"gpsAccuracy"`     // meters
	NetworkAccuracy float64   `json:"networkAccuracy"` // meters
	LastCalibration time.Time `json:"lastCalibration"`
	Provider        string    `json:"provider"` // gps, network, passive
	Score           int       `json:"score"`    // 0-100
}

type LocationCalibrationRequest struct {
	AccuracyTarget float64 `json:"accuracyTarget" validate:"required,min=1,max=100"`
	Provider       string  `json:"provider" validate:"required,oneof=gps network passive"`
}

type LocationCalibrationResult struct {
	UserID       string    `json:"userId"`
	Accuracy     float64   `json:"accuracy"`
	Provider     string    `json:"provider"`
	Success      bool      `json:"success"`
	Improvements []string  `json:"improvements"`
	CreatedAt    time.Time `json:"createdAt"`
}

type LocationProviders struct {
	UserID    string         `json:"userId"`
	GPS       ProviderConfig `json:"gps"`
	Network   ProviderConfig `json:"network"`
	Passive   ProviderConfig `json:"passive"`
	Priority  []string       `json:"priority"` // Order of preference
	UpdatedAt time.Time      `json:"updatedAt"`
}

type ProviderConfig struct {
	Enabled     bool    `json:"enabled"`
	MinAccuracy float64 `json:"minAccuracy"` // meters
	MaxAge      int     `json:"maxAge"`      // seconds
}

type LocationProvidersUpdate struct {
	GPS      *ProviderConfig `json:"gps,omitempty"`
	Network  *ProviderConfig `json:"network,omitempty"`
	Passive  *ProviderConfig `json:"passive,omitempty"`
	Priority []string        `json:"priority,omitempty"`
}

// ==================== BATTERY OPTIMIZATION ====================

type BatteryOptimization struct {
	UserID          string             `json:"userId"`
	PowerMode       string             `json:"powerMode"`       // high, balanced, low, custom
	UpdateFrequency int                `json:"updateFrequency"` // seconds
	GPSSettings     GPSSettings        `json:"gpsSettings"`
	NetworkSettings NetworkSettings    `json:"networkSettings"`
	BackgroundMode  BackgroundSettings `json:"backgroundMode"`
	EstimatedUsage  int                `json:"estimatedUsage"` // percentage per hour
	UpdatedAt       time.Time          `json:"updatedAt"`
}

type GPSSettings struct {
	Enabled     bool    `json:"enabled"`
	Accuracy    string  `json:"accuracy"`    // high, medium, low
	Interval    int     `json:"interval"`    // seconds
	Timeout     int     `json:"timeout"`     // seconds
	MinDistance float64 `json:"minDistance"` // meters
}

type NetworkSettings struct {
	Enabled  bool `json:"enabled"`
	WiFi     bool `json:"wifi"`
	Cellular bool `json:"cellular"`
	Interval int  `json:"interval"` // seconds
}

type BackgroundSettings struct {
	Enabled         bool `json:"enabled"`
	ReducedAccuracy bool `json:"reducedAccuracy"`
	LimitedUpdates  bool `json:"limitedUpdates"`
	Interval        int  `json:"interval"` // seconds
}

type BatteryOptimizationUpdate struct {
	PowerMode       *string             `json:"powerMode,omitempty"`
	UpdateFrequency *int                `json:"updateFrequency,omitempty"`
	GPSSettings     *GPSSettings        `json:"gpsSettings,omitempty"`
	NetworkSettings *NetworkSettings    `json:"networkSettings,omitempty"`
	BackgroundMode  *BackgroundSettings `json:"backgroundMode,omitempty"`
}

type BatteryUsage struct {
	UserID           string               `json:"userId"`
	LastHour         float64              `json:"lastHour"`    // percentage
	Last24Hours      float64              `json:"last24Hours"` // percentage
	LastWeek         float64              `json:"lastWeek"`    // percentage
	HourlyUsage      []HourlyBatteryUsage `json:"hourlyUsage"`
	OptimizationTips []OptimizationTip    `json:"optimizationTips"`
}

type HourlyBatteryUsage struct {
	Hour  int     `json:"hour"`  // 0-23
	Usage float64 `json:"usage"` // percentage
}

type OptimizationTip struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // low, medium, high
}

type PowerModeResult struct {
	UserID    string    `json:"userId"`
	Mode      string    `json:"mode"`
	Applied   bool      `json:"applied"`
	Changes   []string  `json:"changes"`
	CreatedAt time.Time `json:"createdAt"`
}

// ==================== RESPONSE MODELS ====================

type BulkUpdateResult struct {
	Successful int      `json:"successful"`
	Failed     int      `json:"failed"`
	Errors     []string `json:"errors,omitempty"`
}

type LocationHistoryResponse struct {
	Locations []Location     `json:"locations"`
	Meta      PaginationMeta `json:"meta"`
}

type TripsResponse struct {
	Trips []Trip         `json:"trips"`
	Meta  PaginationMeta `json:"meta"`
}

type DrivingSessionsResponse struct {
	Sessions []DrivingSession `json:"sessions"`
	Meta     PaginationMeta   `json:"meta"`
}

type DrivingReportsResponse struct {
	Reports []DrivingReport `json:"reports"`
	Meta    PaginationMeta  `json:"meta"`
}

type DrivingEventsResponse struct {
	Events []DrivingEvent `json:"events"`
	Meta   PaginationMeta `json:"meta"`
}

type GeofenceEventsResponse struct {
	Events []GeofenceEvent `json:"events"`
	Meta   PaginationMeta  `json:"meta"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// Constants
const (
	// Movement types
	MovementTypeStationary = "stationary"
	MovementTypeWalking    = "walking"
	MovementTypeDriving    = "driving"
	MovementTypeCycling    = "cycling"
	MovementTypeRunning    = "running"

	// Location sources
	LocationSourceGPS     = "gps"
	LocationSourceNetwork = "network"
	LocationSourcePassive = "passive"
	LocationSourceManual  = "manual"

	// Visibility levels
	VisibilityPublic  = "public"
	VisibilityCircles = "circles"
	VisibilityPrivate = "private"

	// Precision levels
	PrecisionExact       = "exact"
	PrecisionApproximate = "approximate"
	PrecisionCity        = "city"

	// Trip types
	TripTypeCommute   = "commute"
	TripTypeLeisure   = "leisure"
	TripTypeBusiness  = "business"
	TripTypeEmergency = "emergency"

	// Transportation modes
	TransportDriving   = "driving"
	TransportWalking   = "walking"
	TransportCycling   = "cycling"
	TransportPublic    = "public_transport"
	TransportRideshare = "rideshare"
	TransportFlight    = "flight"

	// Driving event types
	DrivingEventSpeeding   = "speeding"
	DrivingEventHardBrake  = "hard_brake"
	DrivingEventHardAccel  = "hard_acceleration"
	DrivingEventSharpTurn  = "sharp_turn"
	DrivingEventPhoneUsage = "phone_usage"
	DrivingEventRapidAccel = "rapid_acceleration"

	// Event severity levels
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)
