// models/user.go - Updated User model with authentication fields
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email    string             `json:"email" bson:"email"`
	Phone    string             `json:"phone" bson:"phone"`
	Password string             `json:"-" bson:"password"` // Never include in JSON responses

	// Basic Info
	FirstName      string `json:"firstName" bson:"firstName"`
	LastName       string `json:"lastName" bson:"lastName"`
	ProfilePicture string `json:"profilePicture,omitempty" bson:"profilePicture,omitempty"`
	DateOfBirth    string `json:"dateOfBirth,omitempty" bson:"dateOfBirth,omitempty"`
	Gender         string `json:"gender,omitempty" bson:"gender,omitempty"`
	DeviceToken    string `json:"-" bson:"deviceToken"`
	DeviceType     string `json:"deviceType" bson:"deviceType"` // ios, android
	AppVersion     string `json:"appVersion" bson:"appVersion"`
	// Account Status
	IsActive   bool      `json:"isActive" bson:"isActive"`
	IsOnline   bool      `json:"isOnline" bson:"isOnline"`
	IsVerified bool      `json:"isVerified" bson:"isVerified"`
	LastSeen   time.Time `json:"lastSeen" bson:"lastSeen"`

	// Authentication & Security
	VerificationToken string    `json:"-" bson:"verificationToken,omitempty"`
	ResetToken        string    `json:"-" bson:"resetToken,omitempty"`
	TokenExpiresAt    time.Time `json:"-" bson:"tokenExpiresAt,omitempty"`

	// Two-Factor Authentication
	TwoFactorEnabled bool     `json:"twoFactorEnabled" bson:"twoFactorEnabled"`
	TwoFactorSecret  string   `json:"-" bson:"twoFactorSecret,omitempty"`
	BackupCodes      []string `json:"-" bson:"backupCodes,omitempty"`

	// OAuth Authentication
	AuthProvider   string `json:"authProvider,omitempty" bson:"authProvider,omitempty"`
	AuthProviderID string `json:"authProviderId,omitempty" bson:"authProviderId,omitempty"`

	// Account Security
	LoginAttempts int       `json:"-" bson:"loginAttempts,omitempty"`
	LockedUntil   time.Time `json:"-" bson:"lockedUntil,omitempty"`

	// Contact Information
	EmergencyContact EmergencyContact `json:"emergencyContact" bson:"emergencyContact"`

	// Location & Privacy
	LocationSharing LocationSharing `json:"locationSharing" bson:"locationSharing"`
	Preferences     UserPreferences `json:"preferences" bson:"preferences"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`

	Role          string     `json:"role" bson:"role"`                                   // user, moderator, admin, superadmin
	Permissions   []string   `json:"permissions,omitempty" bson:"permissions,omitempty"` // specific permissions
	IsAdmin       bool       `json:"isAdmin" bson:"isAdmin"`                             // quick admin check
	DeactivatedAt *time.Time `json:"deactivatedAt,omitempty" bson:"deactivatedAt,omitempty"`
}

type UserPreferences struct {
	Notifications NotificationPrefs `json:"notifications" bson:"notifications"`
	Privacy       PrivacySettings   `json:"privacy" bson:"privacy"`
	Driving       DrivingPrefs      `json:"driving" bson:"driving"`
	Language      string            `json:"language" bson:"language"`
	Timezone      string            `json:"timezone" bson:"timezone"`
	Theme         string            `json:"theme" bson:"theme"` // light, dark, auto
}

type NotificationPrefs struct {
	PushEnabled     bool `json:"pushEnabled" bson:"pushEnabled"`
	SMSEnabled      bool `json:"smsEnabled" bson:"smsEnabled"`
	EmailEnabled    bool `json:"emailEnabled" bson:"emailEnabled"`
	LocationAlerts  bool `json:"locationAlerts" bson:"locationAlerts"`
	DrivingAlerts   bool `json:"drivingAlerts" bson:"drivingAlerts"`
	EmergencyAlerts bool `json:"emergencyAlerts" bson:"emergencyAlerts"`

	// Quiet Hours
	QuietHours QuietHours `json:"quietHours" bson:"quietHours"`
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

type UpdateUserRequest struct {
	FirstName        *string           `json:"firstName,omitempty"`
	LastName         *string           `json:"lastName,omitempty"`
	ProfilePicture   *string           `json:"profilePicture,omitempty"`
	DateOfBirth      *string           `json:"dateOfBirth,omitempty"`
	Gender           *string           `json:"gender,omitempty"`
	LocationSharing  *LocationSharing  `json:"locationSharing,omitempty"`
	EmergencyContact *EmergencyContact `json:"emergencyContact,omitempty"`
	Preferences      *UserPreferences  `json:"preferences,omitempty"`
}

type UserStatsResponse struct {
	TotalCircles   int     `json:"totalCircles"`
	TotalMessages  int     `json:"totalMessages"`
	TotalTrips     int     `json:"totalTrips"`
	TotalDistance  float64 `json:"totalDistance"` // km
	TotalPlaces    int     `json:"totalPlaces"`
	DrivingScore   int     `json:"drivingScore"`
	SafetyScore    int     `json:"safetyScore"`
	MemberSince    string  `json:"memberSince"`
	LastActiveDate string  `json:"lastActiveDate"`
}

type UserActivityStats struct {
	DailyActivity   []ActivityPoint `json:"dailyActivity"`
	WeeklyActivity  []ActivityPoint `json:"weeklyActivity"`
	MonthlyActivity []ActivityPoint `json:"monthlyActivity"`
	TopLocations    []LocationStat  `json:"topLocations"`
	TopCircles      []CircleStat    `json:"topCircles"`
}

type ActivityPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
	Type  string `json:"type"` // messages, locations, trips
}

type LocationStat struct {
	Name      string `json:"name"`
	Count     int    `json:"count"`
	Duration  int64  `json:"duration"` // seconds
	LastVisit string `json:"lastVisit"`
}

type CircleStat struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MessageCount int    `json:"messageCount"`
	LastActivity string `json:"lastActivity"`
}

// Export and Data Management
type UserDataExport struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"userId" bson:"userId"`
	Status      string             `json:"status" bson:"status"` // pending, processing, completed, failed
	DataTypes   []string           `json:"dataTypes" bson:"dataTypes"`
	FileURL     string             `json:"fileUrl,omitempty" bson:"fileUrl,omitempty"`
	FileSize    int64              `json:"fileSize,omitempty" bson:"fileSize,omitempty"`
	ExpiresAt   time.Time          `json:"expiresAt" bson:"expiresAt"`
	CompletedAt time.Time          `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
}

// Privacy and Security
type DataPurgeRequest struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"userId" bson:"userId"`
	Reason      string             `json:"reason" bson:"reason"`
	DataTypes   []string           `json:"dataTypes" bson:"dataTypes"`
	Status      string             `json:"status" bson:"status"` // pending, processing, completed
	ScheduledAt time.Time          `json:"scheduledAt" bson:"scheduledAt"`
	CompletedAt time.Time          `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
}

// Friend System (if implemented)
type FriendRequest struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	FromUserID  primitive.ObjectID `json:"fromUserId" bson:"fromUserId"`
	ToUserID    primitive.ObjectID `json:"toUserId" bson:"toUserId"`
	Status      string             `json:"status" bson:"status"` // pending, accepted, declined
	Message     string             `json:"message,omitempty" bson:"message,omitempty"`
	ResponsedAt time.Time          `json:"responsedAt,omitempty" bson:"responsedAt,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
}

type BlockedUser struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID `json:"userId" bson:"userId"`
	BlockedUserID primitive.ObjectID `json:"blockedUserId" bson:"blockedUserId"`
	Reason        string             `json:"reason,omitempty" bson:"reason,omitempty"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
}

type UserReport struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ReporterID     primitive.ObjectID `json:"reporterId" bson:"reporterId"`
	ReportedUserID primitive.ObjectID `json:"reportedUserId" bson:"reportedUserId"`
	Reason         string             `json:"reason" bson:"reason"`
	Description    string             `json:"description,omitempty" bson:"description,omitempty"`
	Status         string             `json:"status" bson:"status"` // pending, investigating, resolved, dismissed
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Device and Push Notifications
type UserDevice struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       primitive.ObjectID `json:"userId" bson:"userId"`
	DeviceType   string             `json:"deviceType" bson:"deviceType"` // ios, android, web
	DeviceName   string             `json:"deviceName" bson:"deviceName"`
	DeviceID     string             `json:"deviceId" bson:"deviceId"`
	PushToken    string             `json:"pushToken,omitempty" bson:"pushToken,omitempty"`
	IsActive     bool               `json:"isActive" bson:"isActive"`
	LastUsed     time.Time          `json:"lastUsed" bson:"lastUsed"`
	IPAddress    string             `json:"ipAddress" bson:"ipAddress"`
	UserAgent    string             `json:"userAgent" bson:"userAgent"`
	Location     string             `json:"location,omitempty" bson:"location,omitempty"`
	Capabilities []string           `json:"capabilities,omitempty" bson:"capabilities,omitempty"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Profile Picture Upload
type UploadProfilePictureRequest struct {
	File   interface{} `json:"-"` // multipart.File
	Header interface{} `json:"-"` // *multipart.FileHeader
}

// User Search
type SearchUsersRequest struct {
	Query    string `json:"query" validate:"required,min=2"`
	Limit    int    `json:"limit" validate:"min=1,max=50"`
	CircleID string `json:"circleId,omitempty"`
}

type SearchUsersResponse struct {
	Users []UserSearchResult `json:"users"`
	Total int                `json:"total"`
}

type UserSearchResult struct {
	ID             string `json:"id"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Email          string `json:"email,omitempty"` // Only if privacy allows
	ProfilePicture string `json:"profilePicture,omitempty"`
	IsOnline       bool   `json:"isOnline"`
	MutualCircles  int    `json:"mutualCircles"`
}

// Constants for user-related features
const (
	// User status
	UserStatusActive    = "active"
	UserStatusInactive  = "inactive"
	UserStatusSuspended = "suspended"
	UserStatusDeleted   = "deleted"

	// Gender options
	GenderMale         = "male"
	GenderFemale       = "female"
	GenderOther        = "other"
	GenderNotSpecified = "not_specified"

	// Privacy settings
	PrivacyPublic  = "public"
	PrivacyFriends = "friends"
	PrivacyPrivate = "private"

	// Relationship types
	RelationshipSpouse  = "spouse"
	RelationshipPartner = "partner"
	RelationshipParent  = "parent"
	RelationshipChild   = "child"
	RelationshipSibling = "sibling"
	RelationshipFriend  = "friend"
	RelationshipOther   = "other"

	// Themes
	ThemeLight = "light"
	ThemeDark  = "dark"
	ThemeAuto  = "auto"
)

type UserStatistics struct {
	TotalUsers    int64            `json:"totalUsers" bson:"totalUsers"`
	ActiveUsers   int64            `json:"activeUsers" bson:"activeUsers"`
	VerifiedUsers int64            `json:"verifiedUsers" bson:"verifiedUsers"`
	OnlineUsers   int64            `json:"onlineUsers" bson:"onlineUsers"`
	UsersByRole   map[string]int64 `json:"usersByRole" bson:"usersByRole"`
}
