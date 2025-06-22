package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Place struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID   primitive.ObjectID `json:"userId" bson:"userId"`
	CircleID primitive.ObjectID `json:"circleId,omitempty" bson:"circleId,omitempty"`

	// Basic Info
	Name      string  `json:"name" bson:"name" validate:"required,min=1,max=100"`
	Address   string  `json:"address" bson:"address"`
	Latitude  float64 `json:"latitude" bson:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude float64 `json:"longitude" bson:"longitude" validate:"required,gte=-180,lte=180"`
	Radius    int     `json:"radius" bson:"radius" validate:"min=10,max=5000"` // meters

	// Categorization
	Category string `json:"category" bson:"category"` // home, work, school, gym, restaurant, other
	Icon     string `json:"icon" bson:"icon"`
	Color    string `json:"color" bson:"color"`

	// Notifications
	Notifications PlaceNotifications `json:"notifications" bson:"notifications"`

	// Auto-detection
	Detection PlaceDetection `json:"detection" bson:"detection"`

	// Statistics
	Stats PlaceStats `json:"stats" bson:"stats"`

	// Sharing
	IsShared bool               `json:"isShared" bson:"isShared"`
	SharedBy primitive.ObjectID `json:"sharedBy,omitempty" bson:"sharedBy,omitempty"`

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type PlaceNotifications struct {
	OnArrival       bool       `json:"onArrival" bson:"onArrival"`
	OnDeparture     bool       `json:"onDeparture" bson:"onDeparture"`
	OnExtendedStay  bool       `json:"onExtendedStay" bson:"onExtendedStay"`
	ExtendedStayMin int        `json:"extendedStayMin" bson:"extendedStayMin"` // minutes
	NotifyMembers   []string   `json:"notifyMembers" bson:"notifyMembers"`     // user IDs
	QuietHours      QuietHours `json:"quietHours" bson:"quietHours"`
}



type PlaceDetection struct {
	IsAutoDetected bool      `json:"isAutoDetected" bson:"isAutoDetected"`
	Confidence     float64   `json:"confidence" bson:"confidence"` // 0.0 to 1.0
	FirstDetected  time.Time `json:"firstDetected,omitempty" bson:"firstDetected,omitempty"`
	LastVisit      time.Time `json:"lastVisit,omitempty" bson:"lastVisit,omitempty"`
	VisitPattern   string    `json:"visitPattern" bson:"visitPattern"` // daily, weekly, irregular
}

type PlaceStats struct {
	VisitCount       int       `json:"visitCount" bson:"visitCount"`
	TotalTimeSpent   int64     `json:"totalTimeSpent" bson:"totalTimeSpent"`   // seconds
	AverageStayTime  int64     `json:"averageStayTime" bson:"averageStayTime"` // seconds
	LastVisit        time.Time `json:"lastVisit,omitempty" bson:"lastVisit,omitempty"`
	MostVisitedDay   string    `json:"mostVisitedDay" bson:"mostVisitedDay"`
	UsualArrivalTime string    `json:"usualArrivalTime" bson:"usualArrivalTime"`
	FavoriteRating   float64   `json:"favoriteRating" bson:"favoriteRating"` // 0.0 to 5.0
}

type PlaceVisit struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	PlaceID       primitive.ObjectID `json:"placeId" bson:"placeId"`
	UserID        primitive.ObjectID `json:"userId" bson:"userId"`
	ArrivalTime   time.Time          `json:"arrivalTime" bson:"arrivalTime"`
	DepartureTime *time.Time         `json:"departureTime,omitempty" bson:"departureTime,omitempty"`
	Duration      int64              `json:"duration" bson:"duration"` // seconds
	IsOngoing     bool               `json:"isOngoing" bson:"isOngoing"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
}

// Request DTOs
type CreatePlaceRequest struct {
	Name          string             `json:"name" validate:"required,min=1,max=100"`
	Address       string             `json:"address"`
	Latitude      float64            `json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude     float64            `json:"longitude" validate:"required,gte=-180,lte=180"`
	Radius        int                `json:"radius" validate:"min=10,max=5000"`
	Category      string             `json:"category"`
	Icon          string             `json:"icon"`
	Color         string             `json:"color"`
	Notifications PlaceNotifications `json:"notifications"`
	CircleID      string             `json:"circleId,omitempty"`
}

type UpdatePlaceRequest struct {
	Name          *string             `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Address       *string             `json:"address,omitempty"`
	Radius        *int                `json:"radius,omitempty" validate:"omitempty,min=10,max=5000"`
	Category      *string             `json:"category,omitempty"`
	Icon          *string             `json:"icon,omitempty"`
	Color         *string             `json:"color,omitempty"`
	Notifications *PlaceNotifications `json:"notifications,omitempty"`
}
