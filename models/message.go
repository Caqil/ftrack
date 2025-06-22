package models

import (
	"errors"
	"mime/multipart"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CircleID primitive.ObjectID `json:"circleId" bson:"circleId"`
	SenderID primitive.ObjectID `json:"senderId" bson:"senderId"`

	// Message Content
	Type     string          `json:"type" bson:"type"` // text, photo, location, voice, sticker
	Content  string          `json:"content,omitempty" bson:"content,omitempty"`
	Media    MessageMedia    `json:"media,omitempty" bson:"media,omitempty"`
	Location MessageLocation `json:"location,omitempty" bson:"location,omitempty"`

	// Message State
	Status    string              `json:"status" bson:"status"` // sent, delivered, read
	ReadBy    []MessageReadStatus `json:"readBy" bson:"readBy"`
	Reactions []MessageReaction   `json:"reactions" bson:"reactions"`

	// References
	ReplyTo  primitive.ObjectID `json:"replyTo,omitempty" bson:"replyTo,omitempty"`
	ThreadID primitive.ObjectID `json:"threadId,omitempty" bson:"threadId,omitempty"`

	// Metadata
	IsEdited  bool      `json:"isEdited" bson:"isEdited"`
	EditedAt  time.Time `json:"editedAt,omitempty" bson:"editedAt,omitempty"`
	IsDeleted bool      `json:"isDeleted" bson:"isDeleted"`
	DeletedAt time.Time `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type MessageLocation struct {
	Latitude  float64 `json:"latitude" bson:"latitude"`
	Longitude float64 `json:"longitude" bson:"longitude"`
	Address   string  `json:"address,omitempty" bson:"address,omitempty"`
	PlaceName string  `json:"placeName,omitempty" bson:"placeName,omitempty"`
}

type MessageReadStatus struct {
	UserID primitive.ObjectID `json:"userId" bson:"userId"`
	ReadAt time.Time          `json:"readAt" bson:"readAt"`
}

type MessageReaction struct {
	UserID  primitive.ObjectID `json:"userId" bson:"userId"`
	Emoji   string             `json:"emoji" bson:"emoji"`
	AddedAt time.Time          `json:"addedAt" bson:"addedAt"`
}

type ChatRoom struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CircleID primitive.ObjectID `json:"circleId" bson:"circleId"`
	Name     string             `json:"name" bson:"name"`
	Type     string             `json:"type" bson:"type"` // general, emergency, announcements

	// Settings
	IsActive     bool                 `json:"isActive" bson:"isActive"`
	Participants []primitive.ObjectID `json:"participants" bson:"participants"`
	Admins       []primitive.ObjectID `json:"admins" bson:"admins"`
	Settings     ChatRoomSettings     `json:"settings" bson:"settings"`

	// Statistics
	MessageCount int       `json:"messageCount" bson:"messageCount"`
	LastMessage  time.Time `json:"lastMessage" bson:"lastMessage"`

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type ChatRoomSettings struct {
	AllowMedia        bool `json:"allowMedia" bson:"allowMedia"`
	AllowLocation     bool `json:"allowLocation" bson:"allowLocation"`
	AllowVoice        bool `json:"allowVoice" bson:"allowVoice"`
	AdminOnly         bool `json:"adminOnly" bson:"adminOnly"`
	MuteNotifications bool `json:"muteNotifications" bson:"muteNotifications"`
}

// Request DTOs
type SendMessageRequest struct {
	CircleID string           `json:"circleId" validate:"required"`
	Type     string           `json:"type" validate:"required,oneof=text photo location voice sticker"`
	Content  string           `json:"content"`
	Media    *MessageMedia    `json:"media,omitempty"`
	Location *MessageLocation `json:"location,omitempty"`
	ReplyTo  string           `json:"replyTo,omitempty"`
}

type EditMessageRequest struct {
	Content string `json:"content" validate:"required"`
}

type AddReactionRequest struct {
	Emoji string `json:"emoji" validate:"required"`
}

type MarkAsReadRequest struct {
	MessageIDs []string `json:"messageIds" validate:"required"`
}

// Message Templates
type MessageTemplate struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID     primitive.ObjectID `json:"userId" bson:"userId"`
	Name       string             `json:"name" bson:"name"`
	Category   string             `json:"category" bson:"category"`
	Content    string             `json:"content" bson:"content"`
	Type       string             `json:"type" bson:"type"` // text, media, location
	Media      *MessageMedia      `json:"media,omitempty" bson:"media,omitempty"`
	Variables  []string           `json:"variables" bson:"variables"` // Template variables like {name}, {date}
	UsageCount int                `json:"usageCount" bson:"usageCount"`
	IsPublic   bool               `json:"isPublic" bson:"isPublic"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt  time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Message Drafts
type MessageDraft struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId"`
	CircleID  primitive.ObjectID `json:"circleId" bson:"circleId"`
	Type      string             `json:"type" bson:"type"`
	Content   string             `json:"content" bson:"content"`
	Media     *MessageMedia      `json:"media,omitempty" bson:"media,omitempty"`
	Location  *MessageLocation   `json:"location,omitempty" bson:"location,omitempty"`
	ReplyTo   primitive.ObjectID `json:"replyTo,omitempty" bson:"replyTo,omitempty"`
	AutoSave  bool               `json:"autoSave" bson:"autoSave"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Scheduled Messages
type ScheduledMessage struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID  `json:"userId" bson:"userId"`
	CircleID    primitive.ObjectID  `json:"circleId" bson:"circleId"`
	Type        string              `json:"type" bson:"type"`
	Content     string              `json:"content" bson:"content"`
	Media       *MessageMedia       `json:"media,omitempty" bson:"media,omitempty"`
	Location    *MessageLocation    `json:"location,omitempty" bson:"location,omitempty"`
	ScheduledAt time.Time           `json:"scheduledAt" bson:"scheduledAt"`
	Status      string              `json:"status" bson:"status"` // pending, sent, cancelled, failed
	SentAt      *time.Time          `json:"sentAt,omitempty" bson:"sentAt,omitempty"`
	MessageID   *primitive.ObjectID `json:"messageId,omitempty" bson:"messageId,omitempty"`
	ErrorMsg    string              `json:"errorMsg,omitempty" bson:"errorMsg,omitempty"`
	CreatedAt   time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// Message Reports
type MessageReport struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	MessageID   primitive.ObjectID  `json:"messageId" bson:"messageId"`
	ReportedBy  primitive.ObjectID  `json:"reportedBy" bson:"reportedBy"`
	Reason      string              `json:"reason" bson:"reason"`
	Description string              `json:"description,omitempty" bson:"description,omitempty"`
	Status      string              `json:"status" bson:"status"`     // pending, reviewed, resolved, dismissed
	Severity    string              `json:"severity" bson:"severity"` // low, medium, high
	ReviewedBy  *primitive.ObjectID `json:"reviewedBy,omitempty" bson:"reviewedBy,omitempty"`
	ReviewedAt  *time.Time          `json:"reviewedAt,omitempty" bson:"reviewedAt,omitempty"`
	Resolution  string              `json:"resolution,omitempty" bson:"resolution,omitempty"`
	CreatedAt   time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// Automation Rules
type AutomationRule struct {
	ID            primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID  `json:"userId" bson:"userId"`
	CircleID      *primitive.ObjectID `json:"circleId,omitempty" bson:"circleId,omitempty"`
	Name          string              `json:"name" bson:"name"`
	Type          string              `json:"type" bson:"type"` // auto_reply, keyword_trigger, schedule
	IsActive      bool                `json:"isActive" bson:"isActive"`
	Conditions    []RuleCondition     `json:"conditions" bson:"conditions"`
	Actions       []RuleAction        `json:"actions" bson:"actions"`
	TriggerCount  int                 `json:"triggerCount" bson:"triggerCount"`
	LastTriggered *time.Time          `json:"lastTriggered,omitempty" bson:"lastTriggered,omitempty"`
	CreatedAt     time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt" bson:"updatedAt"`
}


// Message Exports
type MessageExport struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       primitive.ObjectID `json:"userId" bson:"userId"`
	CircleID     primitive.ObjectID `json:"circleId" bson:"circleId"`
	Format       string             `json:"format" bson:"format"`     // json, csv, txt
	Status       string             `json:"status" bson:"status"`     // processing, completed, failed
	Progress     int                `json:"progress" bson:"progress"` // 0-100
	FileURL      string             `json:"fileUrl,omitempty" bson:"fileUrl,omitempty"`
	FileSize     int64              `json:"fileSize,omitempty" bson:"fileSize,omitempty"`
	MessageCount int                `json:"messageCount" bson:"messageCount"`
	DateRange    ExportDateRange    `json:"dateRange" bson:"dateRange"`
	IncludeMedia bool               `json:"includeMedia" bson:"includeMedia"`
	ErrorMsg     string             `json:"errorMsg,omitempty" bson:"errorMsg,omitempty"`
	ExpiresAt    time.Time          `json:"expiresAt" bson:"expiresAt"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type ExportDateRange struct {
	From *time.Time `json:"from,omitempty" bson:"from,omitempty"`
	To   *time.Time `json:"to,omitempty" bson:"to,omitempty"`
}

// Message Forwards
type MessageForward struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	OriginalMessageID  string             `json:"originalMessageId" bson:"originalMessageId"`
	ForwardedMessageID string             `json:"forwardedMessageId" bson:"forwardedMessageId"`
	ForwardedBy        string             `json:"forwardedBy" bson:"forwardedBy"`
	ForwardedTo        string             `json:"forwardedTo" bson:"forwardedTo"` // Circle ID or User ID
	ForwardType        string             `json:"forwardType" bson:"forwardType"` // circle, user
	Comment            string             `json:"comment,omitempty" bson:"comment,omitempty"`
	ForwardedAt        time.Time          `json:"forwardedAt" bson:"forwardedAt"`
}

// Extended Message Media
type MessageMediaExtended struct {
	MessageMedia     `bson:",inline"`
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Compressed       bool               `json:"compressed" bson:"compressed"`
	OriginalSize     int64              `json:"originalSize,omitempty" bson:"originalSize,omitempty"`
	CompressionRatio float64            `json:"compressionRatio,omitempty" bson:"compressionRatio,omitempty"`
	Metadata         MediaMetadata      `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt        time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type MediaMetadata struct {
	Width      int               `json:"width,omitempty" bson:"width,omitempty"`
	Height     int               `json:"height,omitempty" bson:"height,omitempty"`
	Duration   int               `json:"duration,omitempty" bson:"duration,omitempty"`
	Bitrate    int               `json:"bitrate,omitempty" bson:"bitrate,omitempty"`
	Format     string            `json:"format,omitempty" bson:"format,omitempty"`
	ColorSpace string            `json:"colorSpace,omitempty" bson:"colorSpace,omitempty"`
	EXIF       map[string]string `json:"exif,omitempty" bson:"exif,omitempty"`
}

// Request Models

// Basic Message Requests
type GetMessagesRequest struct {
	CircleID string `json:"circleId" validate:"required"`
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
	Before   string `json:"before,omitempty"`
	After    string `json:"after,omitempty"`
}

type ReplyMessageRequest struct {
	Type     string           `json:"type" validate:"required,oneof=text photo location voice sticker"`
	Content  string           `json:"content"`
	Media    *MessageMedia    `json:"media,omitempty"`
	Location *MessageLocation `json:"location,omitempty"`
}

// Media Requests
type UploadMediaRequest struct {
	File      multipart.File        `json:"-"`
	Header    *multipart.FileHeader `json:"-"`
	MediaType string                `json:"mediaType" validate:"required,oneof=image video audio document"`
}

type CompressMediaRequest struct {
	Quality int   `json:"quality" validate:"min=1,max=100"`
	MaxSize int64 `json:"maxSize,omitempty"`
}

// Search Requests
type SearchMessagesRequest struct {
	Query       string `json:"query" validate:"required,min=1"`
	Page        int    `json:"page" validate:"min=1"`
	PageSize    int    `json:"pageSize" validate:"min=1,max=100"`
	MessageType string `json:"messageType,omitempty"`
	DateFrom    string `json:"dateFrom,omitempty"`
	DateTo      string `json:"dateTo,omitempty"`
}

type SearchInCircleRequest struct {
	CircleID string `json:"circleId" validate:"required"`
	Query    string `json:"query" validate:"required,min=1"`
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
}

type SearchMediaRequest struct {
	MediaType string `json:"mediaType,omitempty"`
	Page      int    `json:"page" validate:"min=1"`
	PageSize  int    `json:"pageSize" validate:"min=1,max=100"`
	CircleID  string `json:"circleId,omitempty"`
}

type SearchMentionsRequest struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
	CircleID string `json:"circleId,omitempty"`
}

type SearchLinksRequest struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
	CircleID string `json:"circleId,omitempty"`
	Domain   string `json:"domain,omitempty"`
}

type SearchFilesRequest struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
	FileType string `json:"fileType,omitempty"`
	CircleID string `json:"circleId,omitempty"`
}

// Status Requests
type BulkMarkAsReadRequest struct {
	MessageIDs []string `json:"messageIds" validate:"required,min=1"`
}

// Forwarding Requests
type ForwardMessageRequest struct {
	Comment string `json:"comment,omitempty"`
}

// Scheduling Requests
type ScheduleMessageRequest struct {
	CircleID    string           `json:"circleId" validate:"required"`
	Type        string           `json:"type" validate:"required,oneof=text photo location voice sticker"`
	Content     string           `json:"content"`
	Media       *MessageMedia    `json:"media,omitempty"`
	Location    *MessageLocation `json:"location,omitempty"`
	ScheduledAt time.Time        `json:"scheduledAt" validate:"required"`
}

type GetScheduledMessagesRequest struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
	Status   string `json:"status,omitempty"`
}

type UpdateScheduledMessageRequest struct {
	Content     string           `json:"content,omitempty"`
	Media       *MessageMedia    `json:"media,omitempty"`
	Location    *MessageLocation `json:"location,omitempty"`
	ScheduledAt *time.Time       `json:"scheduledAt,omitempty"`
}

// Template Requests
type GetTemplatesRequest struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
	Category string `json:"category,omitempty"`
}

type CreateTemplateRequest struct {
	Name      string        `json:"name" validate:"required,min=1,max=100"`
	Category  string        `json:"category" validate:"required"`
	Content   string        `json:"content" validate:"required"`
	Type      string        `json:"type" validate:"required,oneof=text media location"`
	Media     *MessageMedia `json:"media,omitempty"`
	Variables []string      `json:"variables,omitempty"`
	IsPublic  bool          `json:"isPublic"`
}

type UpdateTemplateRequest struct {
	Name      string        `json:"name,omitempty"`
	Category  string        `json:"category,omitempty"`
	Content   string        `json:"content,omitempty"`
	Media     *MessageMedia `json:"media,omitempty"`
	Variables []string      `json:"variables,omitempty"`
	IsPublic  *bool         `json:"isPublic,omitempty"`
}

type UseTemplateRequest struct {
	CircleID  string            `json:"circleId" validate:"required"`
	Variables map[string]string `json:"variables,omitempty"`
}

// Export/Import Requests
type ExportMessagesRequest struct {
	Format       string          `json:"format" validate:"required,oneof=json csv txt"`
	DateRange    ExportDateRange `json:"dateRange,omitempty"`
	IncludeMedia bool            `json:"includeMedia"`
}

type ImportMessagesRequest struct {
	File     multipart.File        `json:"-"`
	Header   *multipart.FileHeader `json:"-"`
	CircleID string                `json:"circleId" validate:"required"`
}

// Moderation Requests
type ReportMessageRequest struct {
	Reason      string `json:"reason" validate:"required,oneof=spam harassment inappropriate violence false_information other"`
	Description string `json:"description,omitempty"`
}

type FlagMessageRequest struct {
	Reason   string `json:"reason" validate:"required"`
	Severity string `json:"severity" validate:"required,oneof=low medium high"`
	Action   string `json:"action,omitempty"` // hide, delete, warn
}

type AdminDeleteMessageRequest struct {
	Reason string `json:"reason,omitempty"`
	Notify bool   `json:"notify"`
}

type GetReportsRequest struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
	Status   string `json:"status,omitempty"`
	Severity string `json:"severity,omitempty"`
}

type HandleReportRequest struct {
	Action     string `json:"action" validate:"required,oneof=dismiss warn hide delete"`
	Resolution string `json:"resolution" validate:"required"`
	Notify     bool   `json:"notify"`
}

// Analytics Requests
type GetStatsRequest struct {
	Period   string `json:"period" validate:"required,oneof=1d 7d 30d 90d"`
	CircleID string `json:"circleId,omitempty"`
}

type GetCircleStatsRequest struct {
	CircleID string `json:"circleId" validate:"required"`
	Period   string `json:"period" validate:"required,oneof=1d 7d 30d 90d"`
}

type GetActivityRequest struct {
	Period      string `json:"period" validate:"required,oneof=1d 7d 30d 90d"`
	Granularity string `json:"granularity" validate:"required,oneof=hour day"`
	CircleID    string `json:"circleId,omitempty"`
}

type GetTrendsRequest struct {
	Period    string `json:"period" validate:"required,oneof=7d 30d 90d"`
	TrendType string `json:"trendType" validate:"required,oneof=volume engagement types"`
}

type GetPopularMessagesRequest struct {
	Period   string `json:"period" validate:"required,oneof=1d 7d 30d"`
	Metric   string `json:"metric" validate:"required,oneof=reactions replies views"`
	Limit    int    `json:"limit" validate:"min=1,max=50"`
	CircleID string `json:"circleId,omitempty"`
}

// Automation Requests
type GetAutomationRulesRequest struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
	RuleType string `json:"ruleType,omitempty"`
	Status   string `json:"status,omitempty"`
}

type CreateAutomationRuleRequest struct {
	Name       string          `json:"name" validate:"required,min=1,max=100"`
	Type       string          `json:"type" validate:"required,oneof=auto_reply keyword_trigger schedule"`
	CircleID   string          `json:"circleId,omitempty"`
	Conditions []RuleCondition `json:"conditions" validate:"required,min=1"`
	Actions    []RuleAction    `json:"actions" validate:"required,min=1"`
	IsActive   bool            `json:"isActive"`
}

type UpdateAutomationRuleRequest struct {
	Name       string          `json:"name,omitempty"`
	Conditions []RuleCondition `json:"conditions,omitempty"`
	Actions    []RuleAction    `json:"actions,omitempty"`
	IsActive   *bool           `json:"isActive,omitempty"`
}

type TestAutomationRuleRequest struct {
	TestMessage string            `json:"testMessage" validate:"required"`
	Context     map[string]string `json:"context,omitempty"`
}

// Draft Requests
type GetDraftsRequest struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"pageSize" validate:"min=1,max=100"`
	CircleID string `json:"circleId,omitempty"`
}

type SaveDraftRequest struct {
	CircleID string           `json:"circleId" validate:"required"`
	Type     string           `json:"type" validate:"required,oneof=text photo location voice sticker"`
	Content  string           `json:"content"`
	Media    *MessageMedia    `json:"media,omitempty"`
	Location *MessageLocation `json:"location,omitempty"`
	ReplyTo  string           `json:"replyTo,omitempty"`
	AutoSave bool             `json:"autoSave"`
}

type UpdateDraftRequest struct {
	Content  string           `json:"content,omitempty"`
	Media    *MessageMedia    `json:"media,omitempty"`
	Location *MessageLocation `json:"location,omitempty"`
}

// Response Models

type MessagesResponse struct {
	Messages    []Message `json:"messages"`
	Total       int64     `json:"total"`
	Page        int       `json:"page"`
	PageSize    int       `json:"pageSize"`
	HasNext     bool      `json:"hasNext"`
	HasPrevious bool      `json:"hasPrevious"`
}

type RepliesResponse struct {
	Replies     []Message `json:"replies"`
	Total       int64     `json:"total"`
	Page        int       `json:"page"`
	PageSize    int       `json:"pageSize"`
	HasNext     bool      `json:"hasNext"`
	HasPrevious bool      `json:"hasPrevious"`
}

type ReactionsResponse struct {
	MessageID string                     `json:"messageId"`
	Reactions map[string]ReactionSummary `json:"reactions"`
	Total     int                        `json:"total"`
}

type ReactionSummary struct {
	Emoji string   `json:"emoji"`
	Count int      `json:"count"`
	Users []string `json:"users"`
}

type ReactionUsersResponse struct {
	MessageID string     `json:"messageId"`
	Emoji     string     `json:"emoji"`
	Users     []UserInfo `json:"users"`
	Count     int        `json:"count"`
}

type UserInfo struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Avatar    string `json:"avatar,omitempty"`
}

type MediaThumbnail struct {
	MediaID      string `json:"mediaId"`
	ThumbnailURL string `json:"thumbnailUrl"`
	Type         string `json:"type"`
}

type SearchResponse struct {
	Messages    []Message `json:"messages"`
	Total       int64     `json:"total"`
	Page        int       `json:"page"`
	PageSize    int       `json:"pageSize"`
	Query       string    `json:"query"`
	HasNext     bool      `json:"hasNext"`
	HasPrevious bool      `json:"hasPrevious"`
}

type MediaSearchResponse struct {
	Media       []MessageMediaExtended `json:"media"`
	Total       int64                  `json:"total"`
	Page        int                    `json:"page"`
	PageSize    int                    `json:"pageSize"`
	HasNext     bool                   `json:"hasNext"`
	HasPrevious bool                   `json:"hasPrevious"`
}

type LinksSearchResponse struct {
	Links       []LinkInfo `json:"links"`
	Total       int64      `json:"total"`
	Page        int        `json:"page"`
	PageSize    int        `json:"pageSize"`
	HasNext     bool       `json:"hasNext"`
	HasPrevious bool       `json:"hasPrevious"`
}

type LinkInfo struct {
	URL         string    `json:"url"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Image       string    `json:"image,omitempty"`
	Domain      string    `json:"domain"`
	MessageID   string    `json:"messageId"`
	SenderID    string    `json:"senderId"`
	CircleID    string    `json:"circleId"`
	CreatedAt   time.Time `json:"createdAt"`
}

type FilesSearchResponse struct {
	Files       []FileInfo `json:"files"`
	Total       int64      `json:"total"`
	Page        int        `json:"page"`
	PageSize    int        `json:"pageSize"`
	HasNext     bool       `json:"hasNext"`
	HasPrevious bool       `json:"hasPrevious"`
}

type FileInfo struct {
	FileID    string    `json:"fileId"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	Type      string    `json:"type"`
	URL       string    `json:"url"`
	MessageID string    `json:"messageId"`
	SenderID  string    `json:"senderId"`
	CircleID  string    `json:"circleId"`
	CreatedAt time.Time `json:"createdAt"`
}

type DeliveryStatusResponse struct {
	MessageID string                 `json:"messageId"`
	Status    string                 `json:"status"`
	Delivered int                    `json:"delivered"`
	Total     int                    `json:"total"`
	Details   []DeliveryStatusDetail `json:"details"`
}

type DeliveryStatusDetail struct {
	UserID      string    `json:"userId"`
	Status      string    `json:"status"`
	DeliveredAt time.Time `json:"deliveredAt,omitempty"`
}

type ReadReceiptsResponse struct {
	MessageID string            `json:"messageId"`
	ReadBy    []ReadReceiptInfo `json:"readBy"`
	ReadCount int               `json:"readCount"`
	Total     int               `json:"total"`
}

type ReadReceiptInfo struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Avatar    string    `json:"avatar,omitempty"`
	ReadAt    time.Time `json:"readAt"`
}

type ForwardHistoryResponse struct {
	MessageID string        `json:"messageId"`
	Forwards  []ForwardInfo `json:"forwards"`
	Count     int           `json:"count"`
}

type ForwardInfo struct {
	ForwardedTo string    `json:"forwardedTo"`
	ForwardedBy string    `json:"forwardedBy"`
	ForwardType string    `json:"forwardType"`
	Comment     string    `json:"comment,omitempty"`
	ForwardedAt time.Time `json:"forwardedAt"`
}

type ExportStatusResponse struct {
	ExportID     string    `json:"exportId"`
	Status       string    `json:"status"`
	Progress     int       `json:"progress"`
	FileURL      string    `json:"fileUrl,omitempty"`
	FileSize     int64     `json:"fileSize,omitempty"`
	MessageCount int       `json:"messageCount"`
	CreatedAt    time.Time `json:"createdAt"`
	CompletedAt  time.Time `json:"completedAt,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type ExportDownload struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Data        []byte `json:"-"`
}

type MessageStatsResponse struct {
	Period         string           `json:"period"`
	TotalMessages  int64            `json:"totalMessages"`
	MessagesByType map[string]int64 `json:"messagesByType"`
	ActiveUsers    int64            `json:"activeUsers"`
	BusiestHour    int              `json:"busiestHour"`
	BusiestDay     string           `json:"busiestDay"`
	Engagement     EngagementStats  `json:"engagement"`
	Growth         GrowthStats      `json:"growth"`
}

type EngagementStats struct {
	TotalReactions int64   `json:"totalReactions"`
	TotalReplies   int64   `json:"totalReplies"`
	AvgReactions   float64 `json:"avgReactions"`
	AvgReplies     float64 `json:"avgReplies"`
}

type GrowthStats struct {
	PreviousPeriod int64   `json:"previousPeriod"`
	GrowthRate     float64 `json:"growthRate"`
	Trend          string  `json:"trend"` // up, down, stable
}

type ActivityDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Messages  int64     `json:"messages"`
	Users     int64     `json:"users"`
}

type TrendDataPoint struct {
	Date   time.Time `json:"date"`
	Value  float64   `json:"value"`
	Change float64   `json:"change"`
}

type PopularMessageInfo struct {
	Message       Message `json:"message"`
	Score         float64 `json:"score"`
	ReactionCount int     `json:"reactionCount"`
	ReplyCount    int     `json:"replyCount"`
	ViewCount     int     `json:"viewCount,omitempty"`
}

// WebSocket Message Types for new features
const (
	WSTypeMessageEdit      = "message_edit"
	WSTypeMessageDelete    = "message_delete"
	WSTypeReaction         = "reaction"
	WSTypeReadReceipt      = "read_receipt"
	WSTypeBulkReadReceipt  = "bulk_read_receipt"
	WSTypeTypingStart      = "typing_start"
	WSTypeTypingStop       = "typing_stop"
	WSTypeScheduledMessage = "scheduled_message"
	WSTypeMessageForward   = "message_forward"
)

// WebSocket Message Data Types
type WSMessageEditData struct {
	MessageID  string    `json:"messageId"`
	CircleID   string    `json:"circleId"`
	SenderID   string    `json:"senderId"`
	NewContent string    `json:"newContent"`
	Timestamp  time.Time `json:"timestamp"`
}

type WSMessageDeleteData struct {
	MessageID string    `json:"messageId"`
	CircleID  string    `json:"circleId"`
	SenderID  string    `json:"senderId"`
	Timestamp time.Time `json:"timestamp"`
}

type WSReactionData struct {
	MessageID string    `json:"messageId"`
	CircleID  string    `json:"circleId"`
	UserID    string    `json:"userId"`
	Emoji     string    `json:"emoji"`
	Action    string    `json:"action"` // add, remove
	Timestamp time.Time `json:"timestamp"`
}

type WSReadReceiptData struct {
	MessageID string    `json:"messageId"`
	CircleID  string    `json:"circleId"`
	UserID    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
}

type WSBulkReadReceiptData struct {
	MessageIDs []string  `json:"messageIds"`
	CircleID   string    `json:"circleId"`
	UserID     string    `json:"userId"`
	Timestamp  time.Time `json:"timestamp"`
}

type WSTypingData struct {
	CircleID  string    `json:"circleId"`
	UserID    string    `json:"userId"`
	IsTyping  bool      `json:"isTyping"`
	Timestamp time.Time `json:"timestamp"`
}

// Media Dimensions
type MediaDimensions struct {
	Width  int `json:"width" bson:"width"`
	Height int `json:"height" bson:"height"`
}

// Extended Message Media with additional fields
type MessageMedia struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	URL              string             `json:"url" bson:"url"`
	Type             string             `json:"type" bson:"type"`
	Size             int64              `json:"size" bson:"size"`
	Filename         string             `json:"filename" bson:"filename"`
	MimeType         string             `json:"mimeType" bson:"mimeType"`
	ThumbnailURL     string             `json:"thumbnailUrl,omitempty" bson:"thumbnailUrl,omitempty"`
	Duration         int                `json:"duration,omitempty" bson:"duration,omitempty"`
	Dimensions       *MediaDimensions   `json:"dimensions,omitempty" bson:"dimensions,omitempty"`
	UploadedBy       string             `json:"uploadedBy" bson:"uploadedBy"`
	UploadedAt       time.Time          `json:"uploadedAt" bson:"uploadedAt"`
	Compressed       bool               `json:"compressed" bson:"compressed"`
	OriginalSize     int64              `json:"originalSize,omitempty" bson:"originalSize,omitempty"`
	CompressionRatio float64            `json:"compressionRatio,omitempty" bson:"compressionRatio,omitempty"`
	IsDeleted        bool               `json:"isDeleted,omitempty" bson:"isDeleted,omitempty"`
	DeletedAt        *time.Time         `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
	CreatedAt        time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Storage Stats
type StorageStats struct {
	TotalSize int64 `json:"totalSize"`
	FileCount int64 `json:"fileCount"`
	UsedSpace int64 `json:"usedSpace"`
}

// Report Stats
type ReportStats struct {
	TotalReports   int64            `json:"totalReports"`
	PendingCount   int64            `json:"pendingCount"`
	ReviewedCount  int64            `json:"reviewedCount"`
	ResolvedCount  int64            `json:"resolvedCount"`
	HighSeverity   int64            `json:"highSeverity"`
	MediumSeverity int64            `json:"mediumSeverity"`
	LowSeverity    int64            `json:"lowSeverity"`
	ReasonCounts   map[string]int64 `json:"reasonCounts"`
}

// Automation Stats
type AutomationStats struct {
	TotalRules    int64            `json:"totalRules"`
	ActiveRules   int64            `json:"activeRules"`
	InactiveRules int64            `json:"inactiveRules"`
	TotalTriggers int64            `json:"totalTriggers"`
	TypeCounts    map[string]int64 `json:"typeCounts"`
}

// Export Stats
type ExportStats struct {
	TotalExports    int64            `json:"totalExports"`
	CompletedCount  int64            `json:"completedCount"`
	ProcessingCount int64            `json:"processingCount"`
	FailedCount     int64            `json:"failedCount"`
	TotalSize       int64            `json:"totalSize"`
	TotalMessages   int64            `json:"totalMessages"`
	FormatCounts    map[string]int64 `json:"formatCounts"`
}

// Response Models that were missing

type ScheduledMessagesResponse struct {
	Messages    []ScheduledMessage `json:"messages"`
	Total       int64              `json:"total"`
	Page        int                `json:"page"`
	PageSize    int                `json:"pageSize"`
	HasNext     bool               `json:"hasNext"`
	HasPrevious bool               `json:"hasPrevious"`
}

type TemplatesResponse struct {
	Templates   []MessageTemplate `json:"templates"`
	Total       int64             `json:"total"`
	Page        int               `json:"page"`
	PageSize    int               `json:"pageSize"`
	HasNext     bool              `json:"hasNext"`
	HasPrevious bool              `json:"hasPrevious"`
}

type DraftsResponse struct {
	Drafts      []MessageDraft `json:"drafts"`
	Total       int64          `json:"total"`
	Page        int            `json:"page"`
	PageSize    int            `json:"pageSize"`
	HasNext     bool           `json:"hasNext"`
	HasPrevious bool           `json:"hasPrevious"`
}

type ReportsResponse struct {
	Reports     []MessageReport `json:"reports"`
	Total       int64           `json:"total"`
	Page        int             `json:"page"`
	PageSize    int             `json:"pageSize"`
	HasNext     bool            `json:"hasNext"`
	HasPrevious bool            `json:"hasPrevious"`
}

type AutomationRulesResponse struct {
	Rules       []AutomationRule `json:"rules"`
	Total       int64            `json:"total"`
	Page        int              `json:"page"`
	PageSize    int              `json:"pageSize"`
	HasNext     bool             `json:"hasNext"`
	HasPrevious bool             `json:"hasPrevious"`
}

type ReportHandleResult struct {
	ReportID    string    `json:"reportId"`
	Action      string    `json:"action"`
	Resolution  string    `json:"resolution"`
	ActionTaken string    `json:"actionTaken"`
	HandledBy   string    `json:"handledBy"`
	HandledAt   time.Time `json:"handledAt"`
}

type AutomationTestResult struct {
	RuleID              string    `json:"ruleId"`
	TestMessage         string    `json:"testMessage"`
	Triggered           bool      `json:"triggered"`
	MatchedConditions   []string  `json:"matchedConditions"`
	WouldExecuteActions bool      `json:"wouldExecuteActions"`
	SimulatedActions    []string  `json:"simulatedActions,omitempty"`
	TestedAt            time.Time `json:"testedAt"`
}

type ImportJob struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	CircleID  string    `json:"circleId"`
	Status    string    `json:"status"`
	Progress  int       `json:"progress"`
	CreatedAt time.Time `json:"createdAt"`
}

// This is the internal model used between repository and service layers
type RawMessageStats struct {
	TotalMessages  int64            `json:"totalMessages" bson:"totalMessages"`
	MessagesByType map[string]int64 `json:"messagesByType" bson:"messagesByType"`
	TotalReactions int64            `json:"totalReactions" bson:"totalReactions"`
	TotalReplies   int64            `json:"totalReplies" bson:"totalReplies"`
	ActiveUsers    int64            `json:"activeUsers" bson:"activeUsers"`
	BusiestHour    int              `json:"busiestHour" bson:"busiestHour"` // 0-23
	BusiestDay     string           `json:"busiestDay" bson:"busiestDay"`   // Day name
}

// MessageActivityRequest represents the request for getting message activity data
type MessageActivityRequest struct {
	CircleID    string `json:"circleId,omitempty"`
	Period      string `json:"period" validate:"required,oneof=1d 7d 30d 90d"`
	Granularity string `json:"granularity" validate:"required,oneof=hour day"`
}

// MessageTrendsRequest represents the request for getting message trends
type MessageTrendsRequest struct {
	CircleID   string `json:"circleId,omitempty"`
	Period     string `json:"period" validate:"required,oneof=7d 30d 90d"`
	MetricType string `json:"metricType" validate:"required,oneof=messages reactions replies users"`
}

// TopUsersRequest represents the request for getting top active users
type TopUsersRequest struct {
	CircleID string `json:"circleId,omitempty"`
	Period   string `json:"period" validate:"required,oneof=1d 7d 30d 90d"`
	Limit    int    `json:"limit" validate:"min=1,max=50"`
}

// PopularMessagesRequest represents the request for getting popular messages
type PopularMessagesRequest struct {
	CircleID string `json:"circleId,omitempty"`
	Period   string `json:"period" validate:"required,oneof=1d 7d 30d 90d"`
	SortBy   string `json:"sortBy" validate:"required,oneof=reactions replies views score"`
	Limit    int    `json:"limit" validate:"min=1,max=50"`
}

// =============================================================================
// EXTENDED STATISTICS MODELS
// =============================================================================

// DetailedMessageStats represents comprehensive message statistics with additional metrics
type DetailedMessageStats struct {
	Basic              RawMessageStats         `json:"basic"`
	HourlyDistribution map[int]int64           `json:"hourlyDistribution"` // Hour (0-23) -> message count
	DailyDistribution  map[string]int64        `json:"dailyDistribution"`  // Day name -> message count
	TypeDistribution   map[string]TypeStats    `json:"typeDistribution"`   // Message type -> detailed stats
	UserActivity       []UserActivityStats     `json:"userActivity"`       // Top active users
	EngagementMetrics  DetailedEngagementStats `json:"engagementMetrics"`
	TimePatterns       TimePatternStats        `json:"timePatterns"`
}

// TypeStats represents statistics for a specific message type
type TypeStats struct {
	Count          int64   `json:"count"`
	Percentage     float64 `json:"percentage"`
	AvgReactions   float64 `json:"avgReactions"`
	AvgReplies     float64 `json:"avgReplies"`
	TotalReactions int64   `json:"totalReactions"`
	TotalReplies   int64   `json:"totalReplies"`
}


// DetailedEngagementStats represents detailed engagement metrics
type DetailedEngagementStats struct {
	EngagementStats                              // Embedded basic engagement stats
	ReactionBreakdown  map[string]int64          `json:"reactionBreakdown"` // Emoji -> count
	TopReactedMessages []PopularMessageInfo      `json:"topReactedMessages"`
	TopRepliedMessages []PopularMessageInfo      `json:"topRepliedMessages"`
	EngagementRate     float64                   `json:"engagementRate"` // (reactions + replies) / total messages
	UserEngagement     map[string]UserEngagement `json:"userEngagement"` // UserID -> engagement metrics
}

// UserEngagement represents engagement metrics for a specific user
type UserEngagement struct {
	ReactionsGiven    int64   `json:"reactionsGiven"`
	ReactionsReceived int64   `json:"reactionsReceived"`
	RepliesSent       int64   `json:"repliesSent"`
	RepliesReceived   int64   `json:"repliesReceived"`
	EngagementScore   float64 `json:"engagementScore"`
}

// TimePatternStats represents time-based pattern analysis
type TimePatternStats struct {
	PeakHours        []int               `json:"peakHours"`  // Hours with highest activity
	QuietHours       []int               `json:"quietHours"` // Hours with lowest activity
	PeakDays         []string            `json:"peakDays"`   // Days with highest activity
	WeekdayVsWeekend WeekdayWeekendStats `json:"weekdayVsWeekend"`
	SeasonalPatterns map[string]int64    `json:"seasonalPatterns"` // Month -> message count
}

// WeekdayWeekendStats represents weekday vs weekend activity comparison
type WeekdayWeekendStats struct {
	WeekdayMessages int64   `json:"weekdayMessages"`
	WeekendMessages int64   `json:"weekendMessages"`
	WeekdayAvg      float64 `json:"weekdayAvg"`
	WeekendAvg      float64 `json:"weekendAvg"`
	Ratio           float64 `json:"ratio"` // weekday/weekend ratio
}

// =============================================================================
// COMPARISON AND HISTORICAL MODELS
// =============================================================================

// HistoricalMessageStats represents message statistics with historical comparison
type HistoricalMessageStats struct {
	Current    RawMessageStats `json:"current"`
	Previous   RawMessageStats `json:"previous"`
	Comparison StatsComparison `json:"comparison"`
	Trend      StatsTrend      `json:"trend"`
	Forecast   *StatsForecast  `json:"forecast,omitempty"`
}

// StatsComparison represents comparison between current and previous periods
type StatsComparison struct {
	MessagesChange   int64   `json:"messagesChange"`
	MessagesPercent  float64 `json:"messagesPercent"`
	ReactionsChange  int64   `json:"reactionsChange"`
	ReactionsPercent float64 `json:"reactionsPercent"`
	RepliesChange    int64   `json:"repliesChange"`
	RepliesPercent   float64 `json:"repliesPercent"`
	UsersChange      int64   `json:"usersChange"`
	UsersPercent     float64 `json:"usersPercent"`
}

// StatsTrend represents trend analysis
type StatsTrend struct {
	Direction  string  `json:"direction"`  // up, down, stable
	Strength   string  `json:"strength"`   // strong, moderate, weak
	Confidence float64 `json:"confidence"` // 0-1
	TrendScore float64 `json:"trendScore"` // Overall trend metric
}

// StatsForecast represents forecasted statistics
type StatsForecast struct {
	NextPeriod        RawMessageStats `json:"nextPeriod"`
	Confidence        float64         `json:"confidence"`
	ForecastMethod    string          `json:"forecastMethod"`
	FactorsConsidered []string        `json:"factorsConsidered"`
}

// =============================================================================
// CIRCLE-SPECIFIC STATISTICS
// =============================================================================

// CircleMessageStats represents message statistics for a specific circle
type CircleMessageStats struct {
	CircleID       string                `json:"circleId"`
	CircleName     string                `json:"circleName"`
	Stats          RawMessageStats       `json:"stats"`
	MemberStats    []MemberActivityStats `json:"memberStats"`
	TopMessages    []PopularMessageInfo  `json:"topMessages"`
	RecentActivity []ActivityDataPoint   `json:"recentActivity"`
	Comparison     *CircleComparison     `json:"comparison,omitempty"`
}

// MemberActivityStats represents activity statistics for a circle member
type MemberActivityStats struct {
	UserID            string    `json:"userId"`
	Username          string    `json:"username,omitempty"`
	FirstName         string    `json:"firstName,omitempty"`
	LastName          string    `json:"lastName,omitempty"`
	Avatar            string    `json:"avatar,omitempty"`
	MessageCount      int64     `json:"messageCount"`
	ReactionCount     int64     `json:"reactionCount"`
	ReplyCount        int64     `json:"replyCount"`
	LastMessageAt     time.Time `json:"lastMessageAt"`
	JoinedAt          time.Time `json:"joinedAt"`
	ActivityLevel     string    `json:"activityLevel"` // high, medium, low
	ContributionScore float64   `json:"contributionScore"`
}

// CircleComparison represents comparison with other accessible circles
type CircleComparison struct {
	AverageMessages  float64 `json:"averageMessages"`
	AverageReactions float64 `json:"averageReactions"`
	AverageReplies   float64 `json:"averageReplies"`
	AverageUsers     float64 `json:"averageUsers"`
	RankByMessages   int     `json:"rankByMessages"`
	RankByEngagement int     `json:"rankByEngagement"`
	TotalCircles     int     `json:"totalCircles"`
}

// =============================================================================
// ANALYTICS RESPONSE MODELS
// =============================================================================

// MessageAnalyticsResponse represents comprehensive message analytics
type MessageAnalyticsResponse struct {
	Period          string                 `json:"period"`
	CircleID        string                 `json:"circleId,omitempty"`
	Basic           MessageStatsResponse   `json:"basic"`
	Detailed        DetailedMessageStats   `json:"detailed"`
	Historical      HistoricalMessageStats `json:"historical"`
	Activity        []ActivityDataPoint    `json:"activity"`
	Trends          []TrendDataPoint       `json:"trends"`
	TopUsers        []UserActivityStats    `json:"topUsers"`
	PopularMessages []PopularMessageInfo   `json:"popularMessages"`
	GeneratedAt     time.Time              `json:"generatedAt"`
}

// BulkStatsResponse represents statistics for multiple circles
type BulkStatsResponse struct {
	Period      string               `json:"period"`
	Summary     RawMessageStats      `json:"summary"`
	CircleStats []CircleMessageStats `json:"circleStats"`
	Comparisons []CircleComparison   `json:"comparisons"`
	GeneratedAt time.Time            `json:"generatedAt"`
}

// =============================================================================
// EXPORT AND REPORTING MODELS
// =============================================================================

// StatsExportRequest represents a request to export statistics
type StatsExportRequest struct {
	CircleIDs     []string `json:"circleIds,omitempty"`
	Period        string   `json:"period" validate:"required,oneof=1d 7d 30d 90d"`
	Format        string   `json:"format" validate:"required,oneof=json csv excel pdf"`
	IncludeRaw    bool     `json:"includeRaw"`
	IncludeCharts bool     `json:"includeCharts"`
}

// StatsReport represents a formatted statistics report
type StatsReport struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Period      string          `json:"period"`
	CircleIDs   []string        `json:"circleIds"`
	Summary     RawMessageStats `json:"summary"`
	Sections    []ReportSection `json:"sections"`
	Charts      []ChartData     `json:"charts,omitempty"`
	GeneratedAt time.Time       `json:"generatedAt"`
	GeneratedBy string          `json:"generatedBy"`
}

// ReportSection represents a section in a statistics report
type ReportSection struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // stats, chart, table, text
	Data        map[string]interface{} `json:"data"`
}

// ChartData represents chart configuration and data
type ChartData struct {
	ID      string                   `json:"id"`
	Title   string                   `json:"title"`
	Type    string                   `json:"type"` // line, bar, pie, area
	Data    []map[string]interface{} `json:"data"`
	Options map[string]interface{}   `json:"options"`
}

type ActivityResponse struct {
	Period      string              `json:"period"`
	Granularity string              `json:"granularity"`
	DataPoints  []ActivityDataPoint `json:"dataPoints"`
}

type TrendsResponse struct {
	Period     string           `json:"period"`
	TrendType  string           `json:"trendType"`
	DataPoints []TrendDataPoint `json:"dataPoints"`
}

type PopularMessagesResponse struct {
	Period   string               `json:"period"`
	Metric   string               `json:"metric"`
	Messages []PopularMessageInfo `json:"messages"`
	Count    int                  `json:"count"`
}

// Validation helpers
func (req *SearchMessagesRequest) Validate() error {
	if req.Query == "" {
		return errors.New("search query is required")
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	return nil
}

func (req *GetTemplatesRequest) Validate() error {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	return nil
}

func (req *GetDraftsRequest) Validate() error {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	return nil
}

func (req *GetScheduledMessagesRequest) Validate() error {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	return nil
}

func (req *GetReportsRequest) Validate() error {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	return nil
}

func (req *GetAutomationRulesRequest) Validate() error {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	return nil
}
