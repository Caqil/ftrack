package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ==================== PLACE MODELS ====================

type Place struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID `json:"userId" bson:"userId"`
	CircleID      primitive.ObjectID `json:"circleId,omitempty" bson:"circleId,omitempty"`
	Name          string             `json:"name" bson:"name"`
	Description   string             `json:"description,omitempty" bson:"description,omitempty"`
	Address       string             `json:"address,omitempty" bson:"address,omitempty"`
	Latitude      float64            `json:"latitude" bson:"latitude"`
	Longitude     float64            `json:"longitude" bson:"longitude"`
	Radius        int                `json:"radius" bson:"radius"` // meters
	Category      string             `json:"category" bson:"category"`
	Color         string             `json:"color" bson:"color"`
	Icon          string             `json:"icon" bson:"icon"`
	IsPublic      bool               `json:"isPublic" bson:"isPublic"`
	IsShared      bool               `json:"isShared" bson:"isShared"`
	IsActive      bool               `json:"isActive" bson:"isActive"`
	IsFavorite    bool               `json:"isFavorite" bson:"isFavorite"`
	Tags          []string           `json:"tags" bson:"tags"`
	Priority      int                `json:"priority" bson:"priority"`
	Notifications PlaceNotifications `json:"notifications" bson:"notifications"`
	Hours         PlaceHours         `json:"hours,omitempty" bson:"hours,omitempty"`
	Geofence      GeofenceSettings   `json:"geofence" bson:"geofence"`
	Metadata      PlaceMetadata      `json:"metadata" bson:"metadata"`
	Stats         PlaceStats         `json:"stats" bson:"stats"`
	Sharing       PlaceSharing       `json:"sharing" bson:"sharing"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type PlaceNotifications struct {
	OnArrival        bool `json:"onArrival" bson:"onArrival"`
	OnDeparture      bool `json:"onDeparture" bson:"onDeparture"`
	OnLongStay       bool `json:"onLongStay" bson:"onLongStay"`
	OnFirstTime      bool `json:"onFirstTime" bson:"onFirstTime"`
	LongStayDuration int  `json:"longStayDuration" bson:"longStayDuration"` // minutes
}

type PlaceHours struct {
	IsAlwaysOpen bool                   `json:"isAlwaysOpen" bson:"isAlwaysOpen"`
	Schedule     map[string]DaySchedule `json:"schedule" bson:"schedule"` // day -> schedule
	Timezone     string                 `json:"timezone" bson:"timezone"`
	Overrides    []HoursOverride        `json:"overrides" bson:"overrides"`
}

type HoursOverride struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Date      time.Time          `json:"date" bson:"date"`
	IsOpen    bool               `json:"isOpen" bson:"isOpen"`
	StartTime string             `json:"startTime,omitempty" bson:"startTime,omitempty"`
	EndTime   string             `json:"endTime,omitempty" bson:"endTime,omitempty"`
	Note      string             `json:"note,omitempty" bson:"note,omitempty"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}

type GeofenceSettings struct {
	IsEnabled    bool   `json:"isEnabled" bson:"isEnabled"`
	Shape        string `json:"shape" bson:"shape"`             // circle, polygon
	Sensitivity  string `json:"sensitivity" bson:"sensitivity"` // low, medium, high
	DwellTime    int    `json:"dwellTime" bson:"dwellTime"`     // seconds
	ExitDelay    int    `json:"exitDelay" bson:"exitDelay"`     // seconds
	CustomRadius int    `json:"customRadius,omitempty" bson:"customRadius,omitempty"`
}

type PlaceMetadata struct {
	Phone      string            `json:"phone,omitempty" bson:"phone,omitempty"`
	Website    string            `json:"website,omitempty" bson:"website,omitempty"`
	PlaceID    string            `json:"placeId,omitempty" bson:"placeId,omitempty"` // External place ID
	Provider   string            `json:"provider,omitempty" bson:"provider,omitempty"`
	Rating     float64           `json:"rating,omitempty" bson:"rating,omitempty"`
	PriceLevel int               `json:"priceLevel,omitempty" bson:"priceLevel,omitempty"`
	Custom     map[string]string `json:"custom,omitempty" bson:"custom,omitempty"`
}

type PlaceStats struct {
	VisitCount      int64     `json:"visitCount" bson:"visitCount"`
	TotalDuration   int64     `json:"totalDuration" bson:"totalDuration"` // seconds
	LastVisit       time.Time `json:"lastVisit,omitempty" bson:"lastVisit,omitempty"`
	AverageDuration int64     `json:"averageDuration" bson:"averageDuration"`
	PopularTimes    []int     `json:"popularTimes" bson:"popularTimes"` // 24 hours
	ReviewCount     int       `json:"reviewCount" bson:"reviewCount"`
	AverageRating   float64   `json:"averageRating" bson:"averageRating"`
	CheckinCount    int64     `json:"checkinCount" bson:"checkinCount"`
}

type PlaceSharing struct {
	IsPublic     bool                    `json:"isPublic" bson:"isPublic"`
	SharedWith   []PlaceMember           `json:"sharedWith" bson:"sharedWith"`
	Permissions  PlaceSharingPermissions `json:"permissions" bson:"permissions"`
	InviteCode   string                  `json:"inviteCode,omitempty" bson:"inviteCode,omitempty"`
	InviteExpiry time.Time               `json:"inviteExpiry,omitempty" bson:"inviteExpiry,omitempty"`
}

type PlaceMember struct {
	UserID     primitive.ObjectID `json:"userId" bson:"userId"`
	Role       string             `json:"role" bson:"role"` // viewer, editor, admin
	AddedBy    primitive.ObjectID `json:"addedBy" bson:"addedBy"`
	AddedAt    time.Time          `json:"addedAt" bson:"addedAt"`
	LastAccess time.Time          `json:"lastAccess,omitempty" bson:"lastAccess,omitempty"`
}

type PlaceSharingPermissions struct {
	CanView   bool `json:"canView" bson:"canView"`
	CanEdit   bool `json:"canEdit" bson:"canEdit"`
	CanShare  bool `json:"canShare" bson:"canShare"`
	CanDelete bool `json:"canDelete" bson:"canDelete"`
}

// ==================== PLACE CATEGORIES ====================

type PlaceCategory struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Icon        string             `json:"icon" bson:"icon"`
	Color       string             `json:"color" bson:"color"`
	IsDefault   bool               `json:"isDefault" bson:"isDefault"`
	UserID      primitive.ObjectID `json:"userId,omitempty" bson:"userId,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ==================== PLACE VISITS ====================

type PlaceVisit struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	PlaceID       primitive.ObjectID `json:"placeId" bson:"placeId"`
	UserID        primitive.ObjectID `json:"userId" bson:"userId"`
	ArrivalTime   time.Time          `json:"arrivalTime" bson:"arrivalTime"`
	DepartureTime *time.Time         `json:"departureTime,omitempty" bson:"departureTime,omitempty"`
	Duration      int64              `json:"duration" bson:"duration"` // seconds
	IsOngoing     bool               `json:"isOngoing" bson:"isOngoing"`
	Notes         string             `json:"notes,omitempty" bson:"notes,omitempty"`
	Photos        []string           `json:"photos,omitempty" bson:"photos,omitempty"`
	Rating        int                `json:"rating,omitempty" bson:"rating,omitempty"` // 1-5
	Mood          string             `json:"mood,omitempty" bson:"mood,omitempty"`
	Weather       string             `json:"weather,omitempty" bson:"weather,omitempty"`
	Companions    []string           `json:"companions,omitempty" bson:"companions,omitempty"`
	Activities    []string           `json:"activities,omitempty" bson:"activities,omitempty"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ==================== PLACE REVIEWS ====================

type PlaceReview struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	PlaceID      primitive.ObjectID `json:"placeId" bson:"placeId"`
	UserID       primitive.ObjectID `json:"userId" bson:"userId"`
	Rating       int                `json:"rating" bson:"rating"` // 1-5
	Title        string             `json:"title,omitempty" bson:"title,omitempty"`
	Comment      string             `json:"comment,omitempty" bson:"comment,omitempty"`
	Photos       []string           `json:"photos,omitempty" bson:"photos,omitempty"`
	Tags         []string           `json:"tags,omitempty" bson:"tags,omitempty"`
	IsPublic     bool               `json:"isPublic" bson:"isPublic"`
	HelpfulCount int                `json:"helpfulCount" bson:"helpfulCount"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ==================== PLACE CHECKINS ====================

type PlaceCheckin struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	PlaceID    primitive.ObjectID `json:"placeId" bson:"placeId"`
	UserID     primitive.ObjectID `json:"userId" bson:"userId"`
	Message    string             `json:"message,omitempty" bson:"message,omitempty"`
	Photos     []string           `json:"photos,omitempty" bson:"photos,omitempty"`
	IsPublic   bool               `json:"isPublic" bson:"isPublic"`
	Location   Location           `json:"location" bson:"location"`
	Companions []string           `json:"companions,omitempty" bson:"companions,omitempty"`
	Mood       string             `json:"mood,omitempty" bson:"mood,omitempty"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt  time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ==================== PLACE COLLECTIONS ====================

type PlaceCollection struct {
	ID          primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID   `json:"userId" bson:"userId"`
	Name        string               `json:"name" bson:"name"`
	Description string               `json:"description,omitempty" bson:"description,omitempty"`
	Icon        string               `json:"icon,omitempty" bson:"icon,omitempty"`
	Color       string               `json:"color,omitempty" bson:"color,omitempty"`
	PlaceIDs    []primitive.ObjectID `json:"placeIds" bson:"placeIds"`
	IsPublic    bool                 `json:"isPublic" bson:"isPublic"`
	Tags        []string             `json:"tags,omitempty" bson:"tags,omitempty"`
	CreatedAt   time.Time            `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt" bson:"updatedAt"`
}

// ==================== AUTOMATION RULES ====================

type AutomationTrigger struct {
	Type   string                 `json:"type" bson:"type"` // arrival, departure, long_stay, time_based
	Config map[string]interface{} `json:"config,omitempty" bson:"config,omitempty"`
}

type AutomationAction struct {
	Type   string                 `json:"type" bson:"type"` // notification, webhook, share_location, etc.
	Config map[string]interface{} `json:"config" bson:"config"`
}

type AutomationCondition struct {
	Type     string      `json:"type" bson:"type"`         // time, weather, companions, etc.
	Operator string      `json:"operator" bson:"operator"` // equals, contains, greater_than, etc.
	Value    interface{} `json:"value" bson:"value"`
}

// ==================== PLACE TEMPLATES ====================

type PlaceTemplate struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"userId,omitempty" bson:"userId,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Category    string             `json:"category" bson:"category"`
	IsPublic    bool               `json:"isPublic" bson:"isPublic"`
	Template    PlaceTemplateData  `json:"template" bson:"template"`
	UsageCount  int                `json:"usageCount" bson:"usageCount"`
	Rating      float64            `json:"rating" bson:"rating"`
	Tags        []string           `json:"tags,omitempty" bson:"tags,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type PlaceTemplateData struct {
	Category        string             `json:"category"`
	Color           string             `json:"color"`
	Icon            string             `json:"icon"`
	Radius          int                `json:"radius"`
	Notifications   PlaceNotifications `json:"notifications"`
	Hours           PlaceHours         `json:"hours,omitempty"`
	Geofence        GeofenceSettings   `json:"geofence"`
	AutomationRules []AutomationRule   `json:"automationRules,omitempty"`
}

// ==================== REQUEST/RESPONSE MODELS ====================

type CreatePlaceRequest struct {
	Name          string             `json:"name" validate:"required,min=1,max=100"`
	Description   string             `json:"description,omitempty" validate:"max=500"`
	Address       string             `json:"address,omitempty" validate:"max=200"`
	Latitude      float64            `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude     float64            `json:"longitude" validate:"required,gte=-180,lte=180"`
	Radius        int                `json:"radius" validate:"required,min=10,max=5000"`
	Category      string             `json:"category" validate:"required"`
	Color         string             `json:"color,omitempty"`
	Icon          string             `json:"icon,omitempty"`
	IsPublic      bool               `json:"isPublic"`
	IsShared      bool               `json:"isShared"`
	Tags          []string           `json:"tags,omitempty"`
	Priority      int                `json:"priority" validate:"min=0,max=10"`
	Notifications PlaceNotifications `json:"notifications"`
	Hours         PlaceHours         `json:"hours,omitempty"`
	Geofence      GeofenceSettings   `json:"geofence"`
	Metadata      PlaceMetadata      `json:"metadata,omitempty"`
}

type SearchPlacesRequest struct {
	Query     string  `form:"q"`
	Category  string  `form:"category"`
	Latitude  float64 `form:"latitude"`
	Longitude float64 `form:"longitude"`
	Radius    float64 `form:"radius"`
	Tags      string  `form:"tags"`
	Page      int     `form:"page"`
	PageSize  int     `form:"pageSize"`
}

type PlaceSearchResponse struct {
	Places      []PlaceResponse `json:"places"`
	Meta        PaginationMeta  `json:"meta"`
	Suggestions []string        `json:"suggestions,omitempty"`
}
type GetPlacesRequest struct {
	Category   string  `form:"category"`
	IsPublic   *bool   `form:"isPublic"`
	IsShared   *bool   `form:"isShared"`
	IsActive   *bool   `form:"isActive"`
	IsFavorite *bool   `form:"isFavorite"`
	Tags       string  `form:"tags"` // comma-separated
	Latitude   float64 `form:"latitude"`
	Longitude  float64 `form:"longitude"`
	Radius     float64 `form:"radius"` // meters
	Page       int     `form:"page"`
	PageSize   int     `form:"pageSize"`
	SortBy     string  `form:"sortBy"`
	SortOrder  string  `form:"sortOrder"`
}

type UpdatePlaceRequest struct {
	Name          *string             `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description   *string             `json:"description,omitempty" validate:"omitempty,max=500"`
	Address       *string             `json:"address,omitempty" validate:"omitempty,max=200"`
	Latitude      *float64            `json:"latitude,omitempty" validate:"omitempty,gte=-90,lte=90"`
	Longitude     *float64            `json:"longitude,omitempty" validate:"omitempty,gte=-180,lte=180"`
	Radius        *int                `json:"radius,omitempty" validate:"omitempty,min=10,max=5000"`
	Category      *string             `json:"category,omitempty"`
	Color         *string             `json:"color,omitempty"`
	Icon          *string             `json:"icon,omitempty"`
	IsPublic      *bool               `json:"isPublic,omitempty"`
	IsShared      *bool               `json:"isShared,omitempty"`
	IsActive      *bool               `json:"isActive,omitempty"`
	IsFavorite    *bool               `json:"isFavorite,omitempty"`
	Tags          []string            `json:"tags,omitempty"`
	Priority      *int                `json:"priority,omitempty" validate:"omitempty,min=0,max=10"`
	Notifications *PlaceNotifications `json:"notifications,omitempty"`
	Hours         *PlaceHours         `json:"hours,omitempty"`
	Metadata      *PlaceMetadata      `json:"metadata,omitempty"`
}

type PlaceResponse struct {
	Place    Place   `json:"place"`
	Distance float64 `json:"distance,omitempty"` // meters from search point
}

type PlacesResponse struct {
	Places []PlaceResponse `json:"places"`
	Meta   PaginationMeta  `json:"meta"`
}
type RuleCondition struct {
	Type          string              `json:"type" bson:"type"`
	Field         string              `json:"field" bson:"field"`
	Operator      string              `json:"operator" bson:"operator"`
	Value         interface{}         `json:"value" bson:"value"`
	CaseSensitive bool                `json:"caseSensitive" bson:"caseSensitive"`
	PlaceID       *primitive.ObjectID `json:"placeId,omitempty" bson:"placeId,omitempty"` // ADD THIS
}

type RuleAction struct {
	Type    string                 `json:"type" bson:"type"`
	Config  map[string]interface{} `json:"config" bson:"config"`
	PlaceID *primitive.ObjectID    `json:"placeId,omitempty" bson:"placeId,omitempty"` // ADD THIS
}
