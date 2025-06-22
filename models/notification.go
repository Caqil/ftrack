// models/notification.go
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ========================
// Core Notification Models
// ========================

type Notification struct {
	ID               primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID           string                 `bson:"user_id" json:"user_id"`
	Title            string                 `bson:"title" json:"title"`
	Message          string                 `bson:"message" json:"message"`
	Type             string                 `bson:"type" json:"type"`
	Priority         string                 `bson:"priority" json:"priority"`
	Category         string                 `bson:"category" json:"category"`
	Status           string                 `bson:"status" json:"status"` // read, unread, archived
	CircleID         string                 `bson:"circle_id,omitempty" json:"circle_id,omitempty"`
	Data             interface{}            `bson:"data,omitempty" json:"data,omitempty"`
	ActionButtons    []ActionButton         `bson:"action_buttons,omitempty" json:"action_buttons,omitempty"`
	ImageURL         string                 `bson:"image_url,omitempty" json:"image_url,omitempty"`
	DeepLink         string                 `bson:"deep_link,omitempty" json:"deep_link,omitempty"`
	ExpiresAt        *time.Time             `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	ScheduledAt      *time.Time             `bson:"scheduled_at,omitempty" json:"scheduled_at,omitempty"`
	SnoozedUntil     *time.Time             `bson:"snoozed_until,omitempty" json:"snoozed_until,omitempty"`
	IsPinned         bool                   `bson:"is_pinned" json:"is_pinned"`
	IsArchived       bool                   `bson:"is_archived" json:"is_archived"`
	ReadAt           *time.Time             `bson:"read_at,omitempty" json:"read_at,omitempty"`
	CreatedAt        time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time              `bson:"updated_at" json:"updated_at"`
	DeliveryChannels []string               `bson:"delivery_channels" json:"delivery_channels"`
	Metadata         map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

type ActionButton struct {
	ID     string      `bson:"id" json:"id"`
	Label  string      `bson:"label" json:"label"`
	Style  string      `bson:"style" json:"style"` // primary, secondary, destructive
	Action string      `bson:"action" json:"action"`
	Data   interface{} `bson:"data,omitempty" json:"data,omitempty"`
}

type PaginatedNotifications struct {
	Notifications []Notification `json:"notifications"`
	Page          int            `json:"page"`
	PageSize      int            `json:"page_size"`
	Total         int64          `json:"total"`
	TotalPages    int            `json:"total_pages"`
	HasNext       bool           `json:"has_next"`
	HasPrev       bool           `json:"has_prev"`
}

// ========================
// Request Models
// ========================

type GetNotificationsRequest struct {
	UserID   string `json:"user_id"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Type     string `json:"type"`
	Status   string `json:"status"`
}

type SendNotificationRequest struct {
	Recipients       []string               `json:"recipients" validate:"required"`
	Title            string                 `json:"title" validate:"required"`
	Message          string                 `json:"message" validate:"required"`
	Type             string                 `json:"type" validate:"required"`
	Priority         string                 `json:"priority"`
	Category         string                 `json:"category"`
	Data             interface{}            `json:"data,omitempty"`
	ActionButtons    []ActionButton         `json:"action_buttons,omitempty"`
	ImageURL         string                 `json:"image_url,omitempty"`
	DeepLink         string                 `json:"deep_link,omitempty"`
	ScheduledAt      *time.Time             `json:"scheduled_at,omitempty"`
	ExpiresAt        *time.Time             `json:"expires_at,omitempty"`
	DeliveryChannels []string               `json:"delivery_channels"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type BulkNotificationRequest struct {
	NotificationIDs []string `json:"notification_ids" validate:"required"`
}

type BulkOperationResult struct {
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	FailedIDs    []string `json:"failed_ids,omitempty"`
}

// ========================
// Push Notification Models
// ========================

type PushSettings struct {
	UserID       string                    `bson:"user_id" json:"user_id"`
	Enabled      bool                      `bson:"enabled" json:"enabled"`
	Sound        bool                      `bson:"sound" json:"sound"`
	Vibration    bool                      `bson:"vibration" json:"vibration"`
	Badge        bool                      `bson:"badge" json:"badge"`
	Preview      bool                      `bson:"preview" json:"preview"`
	TypeSettings map[string]TypePreference `bson:"type_settings" json:"type_settings"`
	QuietHours   QuietHours                `bson:"quiet_hours" json:"quiet_hours"`
	CreatedAt    time.Time                 `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time                 `bson:"updated_at" json:"updated_at"`
}

type TypePreference struct {
	Enabled   bool   `bson:"enabled" json:"enabled"`
	Sound     string `bson:"sound" json:"sound"`
	Vibration bool   `bson:"vibration" json:"vibration"`
}

type QuietHours struct {
	Enabled   bool   `bson:"enabled" json:"enabled"`
	StartTime string `bson:"start_time" json:"start_time"` // HH:MM format
	EndTime   string `bson:"end_time" json:"end_time"`     // HH:MM format
	Timezone  string `bson:"timezone" json:"timezone"`
	Days      []int  `bson:"days" json:"days"` // 0=Sunday, 1=Monday, etc.
}

type UpdatePushSettingsRequest struct {
	Enabled      *bool                     `json:"enabled,omitempty"`
	Sound        *bool                     `json:"sound,omitempty"`
	Vibration    *bool                     `json:"vibration,omitempty"`
	Badge        *bool                     `json:"badge,omitempty"`
	Preview      *bool                     `json:"preview,omitempty"`
	TypeSettings map[string]TypePreference `json:"type_settings,omitempty"`
	QuietHours   *QuietHours               `json:"quiet_hours,omitempty"`
}

type TestNotificationRequest struct {
	Title   string      `json:"title"`
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Data    interface{} `json:"data,omitempty"`
}

type PushDevice struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"user_id" json:"user_id"`
	DeviceToken string             `bson:"device_token" json:"device_token"`
	DeviceType  string             `bson:"device_type" json:"device_type"` // ios, android, web
	AppVersion  string             `bson:"app_version" json:"app_version"`
	DeviceModel string             `bson:"device_model" json:"device_model"`
	OS          string             `bson:"os" json:"os"`
	OSVersion   string             `bson:"os_version" json:"os_version"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	LastUsed    time.Time          `bson:"last_used" json:"last_used"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type RegisterDeviceRequest struct {
	DeviceToken string `json:"device_token" validate:"required"`
	DeviceType  string `json:"device_type" validate:"required"`
	AppVersion  string `json:"app_version"`
	DeviceModel string `json:"device_model"`
	OS          string `json:"os"`
	OSVersion   string `json:"os_version"`
}

type UpdateDeviceRequest struct {
	AppVersion  string `json:"app_version,omitempty"`
	DeviceModel string `json:"device_model,omitempty"`
	OS          string `json:"os,omitempty"`
	OSVersion   string `json:"os_version,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// ========================
// Preference Models
// ========================

type NotificationPreferences struct {
	UserID          string                    `bson:"user_id" json:"user_id"`
	GlobalEnabled   bool                      `bson:"global_enabled" json:"global_enabled"`
	EmailEnabled    bool                      `bson:"email_enabled" json:"email_enabled"`
	SMSEnabled      bool                      `bson:"sms_enabled" json:"sms_enabled"`
	PushEnabled     bool                      `bson:"push_enabled" json:"push_enabled"`
	InAppEnabled    bool                      `bson:"in_app_enabled" json:"in_app_enabled"`
	TypePreferences map[string]TypePreference `bson:"type_preferences" json:"type_preferences"`
	Schedule        NotificationSchedule      `bson:"schedule" json:"schedule"`
	Language        string                    `bson:"language" json:"language"`
	Timezone        string                    `bson:"timezone" json:"timezone"`
	CreatedAt       time.Time                 `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time                 `bson:"updated_at" json:"updated_at"`
}

type NotificationSchedule struct {
	Enabled           bool                 `bson:"enabled" json:"enabled"`
	AllowedTimeRanges []TimeRange          `bson:"allowed_time_ranges" json:"allowed_time_ranges"`
	Timezone          string               `bson:"timezone" json:"timezone"`
	ExceptionDates    []time.Time          `bson:"exception_dates" json:"exception_dates"`
	SpecialSchedules  map[string]TimeRange `bson:"special_schedules" json:"special_schedules"`
}

type TimeRange struct {
	StartTime string `bson:"start_time" json:"start_time"` // HH:MM format
	EndTime   string `bson:"end_time" json:"end_time"`     // HH:MM format
	Days      []int  `bson:"days" json:"days"`             // 0=Sunday, 1=Monday, etc.
}

type UpdateNotificationPreferencesRequest struct {
	GlobalEnabled   *bool                     `json:"global_enabled,omitempty"`
	EmailEnabled    *bool                     `json:"email_enabled,omitempty"`
	SMSEnabled      *bool                     `json:"sms_enabled,omitempty"`
	PushEnabled     *bool                     `json:"push_enabled,omitempty"`
	InAppEnabled    *bool                     `json:"in_app_enabled,omitempty"`
	TypePreferences map[string]TypePreference `json:"type_preferences,omitempty"`
	Schedule        *NotificationSchedule     `json:"schedule,omitempty"`
	Language        string                    `json:"language,omitempty"`
	Timezone        string                    `json:"timezone,omitempty"`
}

type NotificationType struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Channels    []string `json:"channels"`
	IsSystem    bool     `json:"is_system"`
}

type UpdateTypePreferencesRequest struct {
	Enabled   *bool  `json:"enabled,omitempty"`
	Sound     string `json:"sound,omitempty"`
	Vibration *bool  `json:"vibration,omitempty"`
}

type UpdateNotificationScheduleRequest struct {
	Enabled           *bool                `json:"enabled,omitempty"`
	AllowedTimeRanges []TimeRange          `json:"allowed_time_ranges,omitempty"`
	Timezone          string               `json:"timezone,omitempty"`
	ExceptionDates    []time.Time          `json:"exception_dates,omitempty"`
	SpecialSchedules  map[string]TimeRange `json:"special_schedules,omitempty"`
}

// ========================
// Email Models
// ========================

type EmailSettings struct {
	UserID          string            `bson:"user_id" json:"user_id"`
	EmailAddress    string            `bson:"email_address" json:"email_address"`
	IsVerified      bool              `bson:"is_verified" json:"is_verified"`
	Enabled         bool              `bson:"enabled" json:"enabled"`
	DigestEnabled   bool              `bson:"digest_enabled" json:"digest_enabled"`
	DigestFrequency string            `bson:"digest_frequency" json:"digest_frequency"` // daily, weekly, monthly
	DigestTime      string            `bson:"digest_time" json:"digest_time"`           // HH:MM format
	TypeSettings    map[string]bool   `bson:"type_settings" json:"type_settings"`
	Templates       map[string]string `bson:"templates" json:"templates"`
	CreatedAt       time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time         `bson:"updated_at" json:"updated_at"`
}

type UpdateEmailSettingsRequest struct {
	EmailAddress    string            `json:"email_address,omitempty"`
	Enabled         *bool             `json:"enabled,omitempty"`
	DigestEnabled   *bool             `json:"digest_enabled,omitempty"`
	DigestFrequency string            `json:"digest_frequency,omitempty"`
	DigestTime      string            `json:"digest_time,omitempty"`
	TypeSettings    map[string]bool   `json:"type_settings,omitempty"`
	Templates       map[string]string `json:"templates,omitempty"`
}

type EmailTemplate struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"user_id" json:"user_id"`
	Name        string             `bson:"name" json:"name"`
	Type        string             `bson:"type" json:"type"`
	Subject     string             `bson:"subject" json:"subject"`
	HTMLContent string             `bson:"html_content" json:"html_content"`
	TextContent string             `bson:"text_content" json:"text_content"`
	Variables   []string           `bson:"variables" json:"variables"`
	IsDefault   bool               `bson:"is_default" json:"is_default"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type UpdateEmailTemplateRequest struct {
	Name        string   `json:"name,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	HTMLContent string   `json:"html_content,omitempty"`
	TextContent string   `json:"text_content,omitempty"`
	Variables   []string `json:"variables,omitempty"`
}

type TestEmailRequest struct {
	TemplateID string                 `json:"template_id,omitempty"`
	Subject    string                 `json:"subject"`
	Content    string                 `json:"content"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
}

// ========================
// SMS Models
// ========================

type SMSSettings struct {
	UserID       string          `bson:"user_id" json:"user_id"`
	PhoneNumber  string          `bson:"phone_number" json:"phone_number"`
	IsVerified   bool            `bson:"is_verified" json:"is_verified"`
	Enabled      bool            `bson:"enabled" json:"enabled"`
	TypeSettings map[string]bool `bson:"type_settings" json:"type_settings"`
	DailyLimit   int             `bson:"daily_limit" json:"daily_limit"`
	MonthlyLimit int             `bson:"monthly_limit" json:"monthly_limit"`
	CreatedAt    time.Time       `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `bson:"updated_at" json:"updated_at"`
}

type UpdateSMSSettingsRequest struct {
	PhoneNumber  string          `json:"phone_number,omitempty"`
	Enabled      *bool           `json:"enabled,omitempty"`
	TypeSettings map[string]bool `json:"type_settings,omitempty"`
	DailyLimit   *int            `json:"daily_limit,omitempty"`
	MonthlyLimit *int            `json:"monthly_limit,omitempty"`
}

type VerifyPhoneRequest struct {
	PhoneNumber      string `json:"phone_number"`
	VerificationCode string `json:"verification_code"`
}

type TestSMSRequest struct {
	Message string `json:"message"`
}

type SMSUsage struct {
	UserID           string    `bson:"user_id" json:"user_id"`
	Date             time.Time `bson:"date" json:"date"`
	Count            int       `bson:"count" json:"count"`
	DailyTotal       int       `json:"daily_total"`
	MonthlyTotal     int       `json:"monthly_total"`
	DailyLimit       int       `json:"daily_limit"`
	MonthlyLimit     int       `json:"monthly_limit"`
	RemainingDaily   int       `json:"remaining_daily"`
	RemainingMonthly int       `json:"remaining_monthly"`
}

// ========================
// In-App Models
// ========================

type InAppSettings struct {
	UserID           string            `bson:"user_id" json:"user_id"`
	Enabled          bool              `bson:"enabled" json:"enabled"`
	ShowBadges       bool              `bson:"show_badges" json:"show_badges"`
	SoundEnabled     bool              `bson:"sound_enabled" json:"sound_enabled"`
	VibrationEnabled bool              `bson:"vibration_enabled" json:"vibration_enabled"`
	PopupEnabled     bool              `bson:"popup_enabled" json:"popup_enabled"`
	AutoDismiss      bool              `bson:"auto_dismiss" json:"auto_dismiss"`
	DismissTimeout   int               `bson:"dismiss_timeout" json:"dismiss_timeout"` // seconds
	TypeSettings     map[string]bool   `bson:"type_settings" json:"type_settings"`
	SoundSettings    map[string]string `bson:"sound_settings" json:"sound_settings"`
	CreatedAt        time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time         `bson:"updated_at" json:"updated_at"`
}

type UpdateInAppSettingsRequest struct {
	Enabled          *bool             `json:"enabled,omitempty"`
	ShowBadges       *bool             `json:"show_badges,omitempty"`
	SoundEnabled     *bool             `json:"sound_enabled,omitempty"`
	VibrationEnabled *bool             `json:"vibration_enabled,omitempty"`
	PopupEnabled     *bool             `json:"popup_enabled,omitempty"`
	AutoDismiss      *bool             `json:"auto_dismiss,omitempty"`
	DismissTimeout   *int              `json:"dismiss_timeout,omitempty"`
	TypeSettings     map[string]bool   `json:"type_settings,omitempty"`
	SoundSettings    map[string]string `json:"sound_settings,omitempty"`
}

type NotificationBadges struct {
	UserID      string         `bson:"user_id" json:"user_id"`
	Total       int            `json:"total"`
	Unread      int            `json:"unread"`
	ByType      map[string]int `json:"by_type"`
	ByCircle    map[string]int `json:"by_circle"`
	ByPriority  map[string]int `json:"by_priority"`
	LastUpdated time.Time      `json:"last_updated"`
}

type ClearBadgesRequest struct {
	BadgeTypes []string `json:"badge_types,omitempty"`
}

type NotificationSound struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Duration int    `json:"duration"` // seconds
	Category string `json:"category"`
}

type UpdateSoundPreferencesRequest struct {
	DefaultSound string            `json:"default_sound,omitempty"`
	TypeSounds   map[string]string `json:"type_sounds,omitempty"`
	VolumeLevel  *float64          `json:"volume_level,omitempty"`
}

// ========================
// Channel Models
// ========================

type NotificationChannel struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID      string                 `bson:"user_id" json:"user_id"`
	Name        string                 `bson:"name" json:"name"`
	Type        string                 `bson:"type" json:"type"` // webhook, slack, discord, etc.
	Description string                 `bson:"description" json:"description"`
	Config      map[string]interface{} `bson:"config" json:"config"`
	IsActive    bool                   `bson:"is_active" json:"is_active"`
	TestResult  *ChannelTestResult     `bson:"test_result,omitempty" json:"test_result,omitempty"`
	CreatedAt   time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time              `bson:"updated_at" json:"updated_at"`
}

type ChannelTestResult struct {
	Success      bool      `bson:"success" json:"success"`
	Message      string    `bson:"message" json:"message"`
	TestedAt     time.Time `bson:"tested_at" json:"tested_at"`
	ResponseTime int       `bson:"response_time" json:"response_time"` // milliseconds
}

type CreateChannelRequest struct {
	Name        string                 `json:"name" validate:"required"`
	Type        string                 `json:"type" validate:"required"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config" validate:"required"`
}

type UpdateChannelRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	IsActive    *bool                  `json:"is_active,omitempty"`
}

// ========================
// Rule Models
// ========================

type NotificationRule struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"user_id" json:"user_id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Conditions  []RuleCondition    `bson:"conditions" json:"conditions"`
	Actions     []RuleAction       `bson:"actions" json:"actions"`
	Priority    int                `bson:"priority" json:"priority"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type CreateRuleRequest struct {
	Name        string          `json:"name" validate:"required"`
	Description string          `json:"description"`
	Conditions  []RuleCondition `json:"conditions" validate:"required"`
	Actions     []RuleAction    `json:"actions" validate:"required"`
	Priority    int             `json:"priority"`
}

type UpdateRuleRequest struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Conditions  []RuleCondition `json:"conditions,omitempty"`
	Actions     []RuleAction    `json:"actions,omitempty"`
	Priority    *int            `json:"priority,omitempty"`
	IsActive    *bool           `json:"is_active,omitempty"`
}

type RuleTestResult struct {
	RuleID   string      `json:"rule_id"`
	Matched  bool        `json:"matched"`
	Actions  []string    `json:"actions"`
	TestData interface{} `json:"test_data"`
	Result   interface{} `json:"result"`
	TestedAt time.Time   `json:"tested_at"`
}

// ========================
// Do Not Disturb Models
// ========================

type DoNotDisturbStatus struct {
	UserID     string     `bson:"user_id" json:"user_id"`
	Enabled    bool       `bson:"enabled" json:"enabled"`
	EnabledAt  *time.Time `bson:"enabled_at,omitempty" json:"enabled_at,omitempty"`
	DisabledAt *time.Time `bson:"disabled_at,omitempty" json:"disabled_at,omitempty"`
	ExpiresAt  *time.Time `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	Reason     string     `bson:"reason" json:"reason"`
	QuietHours QuietHours `bson:"quiet_hours" json:"quiet_hours"`
	Exceptions []string   `bson:"exceptions" json:"exceptions"` // notification types that bypass DND
	UpdatedAt  time.Time  `bson:"updated_at" json:"updated_at"`
}

type EnableDNDRequest struct {
	Duration int    `json:"duration"` // minutes, 0 for indefinite
	Reason   string `json:"reason"`
}

type UpdateQuietHoursRequest struct {
	Enabled   *bool  `json:"enabled,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
	Days      []int  `json:"days,omitempty"`
}

type UpdateDNDExceptionsRequest struct {
	Exceptions []string `json:"exceptions"`
}

// ========================
// Template Models
// ========================

type NotificationTemplate struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID    string                 `bson:"user_id" json:"user_id"`
	Name      string                 `bson:"name" json:"name"`
	Type      string                 `bson:"type" json:"type"`
	Category  string                 `bson:"category" json:"category"`
	Title     string                 `bson:"title" json:"title"`
	Content   string                 `bson:"content" json:"content"`
	Variables []string               `bson:"variables" json:"variables"`
	Config    map[string]interface{} `bson:"config" json:"config"`
	IsDefault bool                   `bson:"is_default" json:"is_default"`
	IsSystem  bool                   `bson:"is_system" json:"is_system"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`
}

type PreviewTemplateRequest struct {
	Variables map[string]interface{} `json:"variables"`
}

type TemplatePreview struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ========================
// Analytics Models
// ========================

type NotificationStats struct {
	Period    string           `json:"period"`
	Total     int64            `json:"total"`
	Sent      int64            `json:"sent"`
	Delivered int64            `json:"delivered"`
	Read      int64            `json:"read"`
	Failed    int64            `json:"failed"`
	ByType    map[string]int64 `json:"by_type"`
	ByChannel map[string]int64 `json:"by_channel"`
	Trends    []StatsTrend     `json:"trends"`
}

type DeliveryStats struct {
	Channel         string         `json:"channel"`
	Total           int64          `json:"total"`
	Delivered       int64          `json:"delivered"`
	Failed          int64          `json:"failed"`
	DeliveryRate    float64        `json:"delivery_rate"`
	AvgDeliveryTime int            `json:"avg_delivery_time"` // milliseconds
	Failures        []FailureStats `json:"failures"`
}

type FailureStats struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

type NotificationTrends struct {
	Metric string       `json:"metric"`
	Period string       `json:"period"`
	Data   []TrendPoint `json:"data"`
}

type TrendPoint struct {
	Date  time.Time `json:"date"`
	Value float64   `json:"value"`
}

type NotificationPerformance struct {
	Current  PerformanceMetrics `json:"current"`
	Previous PerformanceMetrics `json:"previous"`
	Change   PerformanceChange  `json:"change"`
}

type PerformanceMetrics struct {
	Total           int64   `json:"total"`
	DeliveryRate    float64 `json:"delivery_rate"`
	OpenRate        float64 `json:"open_rate"`
	ClickRate       float64 `json:"click_rate"`
	AvgResponseTime int     `json:"avg_response_time"`
}

type PerformanceChange struct {
	Total           float64 `json:"total"`
	DeliveryRate    float64 `json:"delivery_rate"`
	OpenRate        float64 `json:"open_rate"`
	ClickRate       float64 `json:"click_rate"`
	AvgResponseTime float64 `json:"avg_response_time"`
}

// ========================
// History Models
// ========================

type GetHistoryRequest struct {
	UserID    string `json:"user_id"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Type      string `json:"type"`
}

type NotificationHistory struct {
	Notifications []Notification `json:"notifications"`
	Page          int            `json:"page"`
	PageSize      int            `json:"page_size"`
	Total         int64          `json:"total"`
	TotalPages    int            `json:"total_pages"`
}

type DeliveryHistory struct {
	NotificationID string            `json:"notification_id"`
	Attempts       []DeliveryAttempt `json:"attempts"`
	Summary        DeliverySummary   `json:"summary"`
}

type DeliveryAttempt struct {
	Channel      string     `json:"channel"`
	Status       string     `json:"status"`
	AttemptedAt  time.Time  `json:"attempted_at"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	ResponseTime int        `json:"response_time"` // milliseconds
}

type DeliverySummary struct {
	TotalAttempts   int     `json:"total_attempts"`
	SuccessfulCount int     `json:"successful_count"`
	FailedCount     int     `json:"failed_count"`
	DeliveryRate    float64 `json:"delivery_rate"`
	AvgResponseTime int     `json:"avg_response_time"`
}

type ExportResult struct {
	ExportID  string    `json:"export_id"`
	Status    string    `json:"status"`
	Progress  int       `json:"progress"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CleanupResult struct {
	DeletedCount int      `json:"deleted_count"`
	DryRun       bool     `json:"dry_run"`
	Categories   []string `json:"categories"`
}

// ========================
// Subscription Models
// ========================

type NotificationSubscription struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID    string               `bson:"user_id" json:"user_id"`
	TopicID   string               `bson:"topic_id" json:"topic_id"`
	Channels  []string             `bson:"channels" json:"channels"`
	Filters   []SubscriptionFilter `bson:"filters" json:"filters"`
	IsActive  bool                 `bson:"is_active" json:"is_active"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time            `bson:"updated_at" json:"updated_at"`
}

type SubscriptionFilter struct {
	Field    string      `bson:"field" json:"field"`
	Operator string      `bson:"operator" json:"operator"`
	Value    interface{} `bson:"value" json:"value"`
}

type CreateSubscriptionRequest struct {
	TopicID  string               `json:"topic_id" validate:"required"`
	Channels []string             `json:"channels" validate:"required"`
	Filters  []SubscriptionFilter `json:"filters"`
}

type UpdateSubscriptionRequest struct {
	Channels []string             `json:"channels,omitempty"`
	Filters  []SubscriptionFilter `json:"filters,omitempty"`
	IsActive *bool                `json:"is_active,omitempty"`
}

type NotificationTopic struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Channels    []string `json:"channels"`
	IsSystem    bool     `json:"is_system"`
}

// ========================
// Action Models
// ========================

type NotificationAction struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	IsAvailable bool                   `json:"is_available"`
}

type ExecuteActionRequest struct {
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type ActionResult struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type SnoozeRequest struct {
	Duration int    `json:"duration"` // minutes
	Reason   string `json:"reason"`
}

type SnoozeResult struct {
	NotificationID string    `json:"notification_id"`
	SnoozedUntil   time.Time `json:"snoozed_until"`
	Reason         string    `json:"reason"`
}
