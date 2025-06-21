package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email          string             `json:"email" bson:"email" validate:"required,email"`
	Phone          string             `json:"phone" bson:"phone" validate:"required"`
	Password       string             `json:"-" bson:"password"`
	FirstName      string             `json:"firstName" bson:"firstName" validate:"required"`
	LastName       string             `json:"lastName" bson:"lastName" validate:"required"`
	ProfilePicture string             `json:"profilePicture" bson:"profilePicture"`

	// Location Settings
	LocationSharing LocationSharing `json:"locationSharing" bson:"locationSharing"`

	// Emergency Contact
	EmergencyContact EmergencyContact `json:"emergencyContact" bson:"emergencyContact"`

	// Device Info
	DeviceToken string `json:"-" bson:"deviceToken"`
	DeviceType  string `json:"deviceType" bson:"deviceType"` // ios, android
	AppVersion  string `json:"appVersion" bson:"appVersion"`

	// Preferences
	Preferences UserPreferences `json:"preferences" bson:"preferences"`

	// Status
	IsActive bool      `json:"isActive" bson:"isActive"`
	LastSeen time.Time `json:"lastSeen" bson:"lastSeen"`
	IsOnline bool      `json:"isOnline" bson:"isOnline"`

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}


type UserPreferences struct {
	Notifications NotificationPrefs `json:"notifications" bson:"notifications"`
	Privacy       PrivacySettings   `json:"privacy" bson:"privacy"`
	Driving       DrivingPrefs      `json:"driving" bson:"driving"`
}

type NotificationPrefs struct {
	PushEnabled     bool `json:"pushEnabled" bson:"pushEnabled"`
	SMSEnabled      bool `json:"smsEnabled" bson:"smsEnabled"`
	EmailEnabled    bool `json:"emailEnabled" bson:"emailEnabled"`
	LocationAlerts  bool `json:"locationAlerts" bson:"locationAlerts"`
	DrivingAlerts   bool `json:"drivingAlerts" bson:"drivingAlerts"`
	EmergencyAlerts bool `json:"emergencyAlerts" bson:"emergencyAlerts"`
}

type PrivacySettings struct {
	ShowInDirectory bool `json:"showInDirectory" bson:"showInDirectory"`
	AllowInvites    bool `json:"allowInvites" bson:"allowInvites"`
	ShareDriving    bool `json:"shareDriving" bson:"shareDriving"`
}

type DrivingPrefs struct {
	AutoDetect  bool `json:"autoDetect" bson:"autoDetect"`
	SpeedLimit  int  `json:"speedLimit" bson:"speedLimit"`
	HardBraking bool `json:"hardBraking" bson:"hardBraking"`
	PhoneUsage  bool `json:"phoneUsage" bson:"phoneUsage"`
}

// Request/Response DTOs
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type RegisterRequest struct {
	Email            string           `json:"email" validate:"required,email"`
	Phone            string           `json:"phone" validate:"required"`
	Password         string           `json:"password" validate:"required,min=6"`
	FirstName        string           `json:"firstName" validate:"required"`
	LastName         string           `json:"lastName" validate:"required"`
	EmergencyContact EmergencyContact `json:"emergencyContact"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type UpdateUserRequest struct {
	FirstName        *string           `json:"firstName,omitempty"`
	LastName         *string           `json:"lastName,omitempty"`
	ProfilePicture   *string           `json:"profilePicture,omitempty"`
	LocationSharing  *LocationSharing  `json:"locationSharing,omitempty"`
	EmergencyContact *EmergencyContact `json:"emergencyContact,omitempty"`
	Preferences      *UserPreferences  `json:"preferences,omitempty"`
}
