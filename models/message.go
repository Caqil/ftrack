package models

import (
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

type MessageMedia struct {
	URL          string `json:"url" bson:"url"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty" bson:"thumbnailUrl,omitempty"`
	FileName     string `json:"fileName" bson:"fileName"`
	FileSize     int64  `json:"fileSize" bson:"fileSize"`
	MimeType     string `json:"mimeType" bson:"mimeType"`
	Duration     int    `json:"duration,omitempty" bson:"duration,omitempty"` // for audio/video in seconds
	Width        int    `json:"width,omitempty" bson:"width,omitempty"`
	Height       int    `json:"height,omitempty" bson:"height,omitempty"`
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
