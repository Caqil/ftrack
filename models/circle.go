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
