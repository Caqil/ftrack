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

// Trip tracking
type Trip struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId"`
	SessionID primitive.ObjectID `json:"sessionId" bson:"sessionId"`

	// Trip Details
	Name        string    `json:"name" bson:"name"`
	Description string    `json:"description,omitempty" bson:"description,omitempty"`
	StartTime   time.Time `json:"startTime" bson:"startTime"`
	EndTime     time.Time `json:"endTime,omitempty" bson:"endTime,omitempty"`
	IsActive    bool      `json:"isActive" bson:"isActive"`

	// Geographic Data
	StartLocation Location   `json:"startLocation" bson:"startLocation"`
	EndLocation   Location   `json:"endLocation,omitempty" bson:"endLocation,omitempty"`
	Route         []Location `json:"route" bson:"route"` // Simplified route points

	// Trip Statistics
	Stats TripStats `json:"stats" bson:"stats"`

	// Trip Classification
	Type           string `json:"type" bson:"type"`                     // commute, leisure, business, emergency
	Transportation string `json:"transportation" bson:"transportation"` // driving, walking, cycling, public_transport
	Purpose        string `json:"purpose,omitempty" bson:"purpose,omitempty"`

	// Sharing
	IsShared   bool     `json:"isShared" bson:"isShared"`
	SharedWith []string `json:"sharedWith,omitempty" bson:"sharedWith,omitempty"` // Circle IDs

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type TripStats struct {
	Distance      float64  `json:"distance" bson:"distance"`                             // Total distance in meters
	Duration      int64    `json:"duration" bson:"duration"`                             // Total duration in seconds
	MovingTime    int64    `json:"movingTime" bson:"movingTime"`                         // Time actually moving
	MaxSpeed      float64  `json:"maxSpeed" bson:"maxSpeed"`                             // Maximum speed m/s
	AverageSpeed  float64  `json:"averageSpeed" bson:"averageSpeed"`                     // Average speed m/s
	StopCount     int      `json:"stopCount" bson:"stopCount"`                           // Number of stops
	PlacesVisited []string `json:"placesVisited" bson:"placesVisited"`                   // Place IDs visited
	FuelConsumed  float64  `json:"fuelConsumed,omitempty" bson:"fuelConsumed,omitempty"` // Estimated liters
	CO2Emissions  float64  `json:"co2Emissions,omitempty" bson:"co2Emissions,omitempty"` // Estimated kg
	Cost          float64  `json:"cost,omitempty" bson:"cost,omitempty"`                 // Estimated cost
}

// Driving behavior analysis
type DrivingEvent struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID     primitive.ObjectID `json:"userId" bson:"userId"`
	TripID     primitive.ObjectID `json:"tripId,omitempty" bson:"tripId,omitempty"`
	LocationID primitive.ObjectID `json:"locationId" bson:"locationId"`

	// Event Details
	Type        string `json:"type" bson:"type"`         // hard_brake, hard_acceleration, speeding, phone_usage, sharp_turn
	Severity    string `json:"severity" bson:"severity"` // low, medium, high, critical
	Description string `json:"description" bson:"description"`
	Score       int    `json:"score" bson:"score"` // Safety score impact (-100 to 0)

	// Event Data
	Speed        float64  `json:"speed" bson:"speed"`                                   // Speed at event
	SpeedLimit   int      `json:"speedLimit,omitempty" bson:"speedLimit,omitempty"`     // Posted speed limit
	Acceleration float64  `json:"acceleration,omitempty" bson:"acceleration,omitempty"` // G-force
	Location     Location `json:"location" bson:"location"`                             // Where it happened

	// Context
	TimeOfDay string `json:"timeOfDay" bson:"timeOfDay"` // morning, afternoon, evening, night
	Weather   string `json:"weather,omitempty" bson:"weather,omitempty"`
	RoadType  string `json:"roadType,omitempty" bson:"roadType,omitempty"` // highway, city, residential

	// Detection
	DetectionMethod string  `json:"detectionMethod" bson:"detectionMethod"` // accelerometer, gps, manual
	Confidence      float64 `json:"confidence" bson:"confidence"`           // 0.0 to 1.0

	CreatedAt   time.Time `json:"createdAt" bson:"createdAt"`
	ProcessedAt time.Time `json:"processedAt,omitempty" bson:"processedAt,omitempty"`
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

type NearbyUser struct {
	UserID    string   `json:"userId"`
	Name      string   `json:"name"`
	Location  Location `json:"location"`
	Distance  float64  `json:"distance"`  // meters
	Direction string   `json:"direction"` // N, NE, E, etc.
	LastSeen  string   `json:"lastSeen"`  // relative time
}

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
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
