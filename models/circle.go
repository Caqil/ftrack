package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Circle struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name       string             `json:"name" bson:"name" validate:"required,min=2,max=50"`
	AdminID    primitive.ObjectID `json:"adminId" bson:"adminId"`
	Members    []CircleMember     `json:"members" bson:"members"`
	InviteCode string             `json:"inviteCode" bson:"inviteCode"`

	// Settings
	Settings CircleSettings `json:"settings" bson:"settings"`

	// Statistics
	Stats CircleStats `json:"stats" bson:"stats"`

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type CircleMember struct {
	UserID       primitive.ObjectID `json:"userId" bson:"userId"`
	Role         string             `json:"role" bson:"role"`     // admin, member
	Status       string             `json:"status" bson:"status"` // active, pending, invited
	JoinedAt     time.Time          `json:"joinedAt" bson:"joinedAt"`
	InvitedAt    time.Time          `json:"invitedAt,omitempty" bson:"invitedAt,omitempty"`
	InvitedBy    primitive.ObjectID `json:"invitedBy,omitempty" bson:"invitedBy,omitempty"`
	Permissions  MemberPermissions  `json:"permissions" bson:"permissions"`
	LastActivity time.Time          `json:"lastActivity" bson:"lastActivity"`
}

type MemberPermissions struct {
	CanSeeLocation   bool `json:"canSeeLocation" bson:"canSeeLocation"`
	CanSeeDriving    bool `json:"canSeeDriving" bson:"canSeeDriving"`
	CanSendMessages  bool `json:"canSendMessages" bson:"canSendMessages"`
	CanManagePlaces  bool `json:"canManagePlaces" bson:"canManagePlaces"`
	CanReceiveAlerts bool `json:"canReceiveAlerts" bson:"canReceiveAlerts"`
	CanSendEmergency bool `json:"canSendEmergency" bson:"canSendEmergency"`
}

type CircleSettings struct {
	AutoAcceptInvites  bool `json:"autoAcceptInvites" bson:"autoAcceptInvites"`
	RequireApproval    bool `json:"requireApproval" bson:"requireApproval"`
	MaxMembers         int  `json:"maxMembers" bson:"maxMembers"`
	LocationSharing    bool `json:"locationSharing" bson:"locationSharing"`
	DrivingReports     bool `json:"drivingReports" bson:"drivingReports"`
	EmergencyAlerts    bool `json:"emergencyAlerts" bson:"emergencyAlerts"`
	AutoCheckIn        bool `json:"autoCheckIn" bson:"autoCheckIn"`
	PlaceNotifications bool `json:"placeNotifications" bson:"placeNotifications"`
}

type CircleStats struct {
	TotalMembers  int       `json:"totalMembers" bson:"totalMembers"`
	ActiveMembers int       `json:"activeMembers" bson:"activeMembers"`
	TotalMessages int       `json:"totalMessages" bson:"totalMessages"`
	TotalPlaces   int       `json:"totalPlaces" bson:"totalPlaces"`
	LastActivity  time.Time `json:"lastActivity" bson:"lastActivity"`
}

// Request DTOs
type CreateCircleRequest struct {
	Name string `json:"name" validate:"required,min=2,max=50"`
}

type JoinCircleRequest struct {
	InviteCode string `json:"inviteCode" validate:"required"`
}

type InviteMemberRequest struct {
	Email       string            `json:"email,omitempty" validate:"omitempty,email"`
	Phone       string            `json:"phone,omitempty" validate:"omitempty,min=10"`
	Permissions MemberPermissions `json:"permissions"`
	Role        string            `json:"role" validate:"required,oneof=admin member"`
	Message     string            `json:"message,omitempty"`
}

type UpdateCircleRequest struct {
	Name     *string         `json:"name,omitempty" validate:"omitempty,min=2,max=50"`
	Settings *CircleSettings `json:"settings,omitempty"`
}

type UpdateMemberPermissionsRequest struct {
	UserID      string            `json:"userId" validate:"required"`
	Permissions MemberPermissions `json:"permissions"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member"`
}

// Invitation model
type CircleInvitation struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CircleID  primitive.ObjectID `json:"circleId" bson:"circleId"`
	InviterID primitive.ObjectID `json:"inviterId" bson:"inviterId"`
	InviteeID primitive.ObjectID `json:"inviteeId" bson:"inviteeId"`
	Email     string             `json:"email" bson:"email"`
	Role      string             `json:"role" bson:"role"`
	Message   string             `json:"message,omitempty" bson:"message,omitempty"`
	Status    string             `json:"status" bson:"status"` // pending, accepted, rejected, expired
	ExpiresAt time.Time          `json:"expiresAt" bson:"expiresAt"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Join Request model
type JoinRequest struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CircleID  primitive.ObjectID `json:"circleId" bson:"circleId"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId"`
	Message   string             `json:"message,omitempty" bson:"message,omitempty"`
	Status    string             `json:"status" bson:"status"` // pending, approved, declined
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Announcement model
type CircleAnnouncement struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CircleID  primitive.ObjectID `json:"circleId" bson:"circleId"`
	AuthorID  primitive.ObjectID `json:"authorId" bson:"authorId"`
	Title     string             `json:"title" bson:"title"`
	Message   string             `json:"message" bson:"message"`
	Type      string             `json:"type" bson:"type"` // info, warning, urgent
	Priority  int                `json:"priority" bson:"priority"`
	IsRead    map[string]bool    `json:"isRead" bson:"isRead"` // userID -> read status
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// Activity model
type CircleActivity struct {
	ID        primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	CircleID  primitive.ObjectID     `json:"circleId" bson:"circleId"`
	UserID    primitive.ObjectID     `json:"userId" bson:"userId"`
	Type      string                 `json:"type" bson:"type"` // join, leave, location, place, emergency, message
	Action    string                 `json:"action" bson:"action"`
	Data      map[string]interface{} `json:"data" bson:"data"`
	CreatedAt time.Time              `json:"createdAt" bson:"createdAt"`
}

// Export Job model
type ExportJob struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CircleID  primitive.ObjectID `json:"circleId" bson:"circleId"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId"`
	Format    string             `json:"format" bson:"format"` // json, csv, xlsx
	Includes  []string           `json:"includes" bson:"includes"`
	Status    string             `json:"status" bson:"status"` // pending, processing, completed, failed
	Progress  int                `json:"progress" bson:"progress"`
	FileURL   string             `json:"fileUrl,omitempty" bson:"fileUrl,omitempty"`
	Error     string             `json:"error,omitempty" bson:"error,omitempty"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
	ExpiresAt time.Time          `json:"expiresAt" bson:"expiresAt"`
}
