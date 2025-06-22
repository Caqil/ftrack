package services

import (
	"context"
	"errors"
	"fmt"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CircleService struct {
	circleRepo *repositories.CircleRepository
	userRepo   *repositories.UserRepository
	validator  *utils.ValidationService
}

func NewCircleService(circleRepo *repositories.CircleRepository, userRepo *repositories.UserRepository) *CircleService {
	return &CircleService{
		circleRepo: circleRepo,
		userRepo:   userRepo,
		validator:  utils.NewValidationService(),
	}
}

// ========================
// Basic CRUD Operations
// ========================

func (cs *CircleService) CreateCircle(ctx context.Context, userID string, req models.CreateCircleRequest) (*models.Circle, error) {
	// Validate request
	if validationErrors := cs.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Create circle
	circle := models.Circle{
		Name:       req.Name,
		AdminID:    userObjectID,
		InviteCode: utils.GenerateInviteCode(),
		Settings: models.CircleSettings{
			AutoAcceptInvites:  false,
			RequireApproval:    true,
			MaxMembers:         20,
			LocationSharing:    true,
			DrivingReports:     true,
			EmergencyAlerts:    true,
			AutoCheckIn:        false,
			PlaceNotifications: true,
		},
		Stats: models.CircleStats{
			TotalMembers:  1,
			ActiveMembers: 1,
		},
	}

	// Add creator as admin member
	adminMember := models.CircleMember{
		UserID: userObjectID,
		Role:   "admin",
		Status: "active",
		Permissions: models.MemberPermissions{
			CanSeeLocation:   true,
			CanSeeDriving:    true,
			CanSendMessages:  true,
			CanManagePlaces:  true,
			CanReceiveAlerts: true,
			CanSendEmergency: true,
		},
		JoinedAt: time.Now(),
	}
	circle.Members = []models.CircleMember{adminMember}

	err = cs.circleRepo.Create(ctx, &circle)
	if err != nil {
		return nil, err
	}

	return &circle, nil
}

func (cs *CircleService) GetUserCircles(ctx context.Context, userID string) ([]models.Circle, error) {
	return cs.circleRepo.GetUserCircles(ctx, userID)
}

func (cs *CircleService) GetCircle(ctx context.Context, userID, circleID string) (*models.Circle, error) {
	// Check if user is a member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	return cs.circleRepo.GetByID(ctx, circleID)
}

func (cs *CircleService) UpdateCircle(ctx context.Context, userID, circleID string, req models.UpdateCircleRequest) (*models.Circle, error) {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// Build update document
	update := bson.M{}
	if req.Name != nil {
		update["name"] = *req.Name
	}
	if req.Settings != nil {
		update["settings"] = *req.Settings
	}

	err = cs.circleRepo.Update(ctx, circleID, update)
	if err != nil {
		return nil, err
	}

	return cs.circleRepo.GetByID(ctx, circleID)
}

func (cs *CircleService) DeleteCircle(ctx context.Context, userID, circleID string) error {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	return cs.circleRepo.Delete(ctx, circleID)
}

// ========================
// Invitation Management
// ========================

func (cs *CircleService) GetCircleInvitations(ctx context.Context, userID, circleID string) ([]models.CircleInvitation, error) {
	// Check if user is admin or member with appropriate permissions
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// Get all invitations for this circle
	invitationCollection := cs.circleRepo.GetInvitationCollection()
	circleObjectID, _ := primitive.ObjectIDFromHex(circleID)

	cursor, err := invitationCollection.Find(ctx, bson.M{"circleId": circleObjectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.CircleInvitation
	err = cursor.All(ctx, &invitations)
	return invitations, err
}

func (cs *CircleService) CreateInvitation(ctx context.Context, userID, circleID string, req models.InviteMemberRequest) (*models.CircleInvitation, error) {
	// Check if user has permission to invite
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// Validate request
	if validationErrors := cs.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check if circle exists
	if err != nil {
		return nil, errors.New("circle not found")
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	circleObjectID, _ := primitive.ObjectIDFromHex(circleID)

	// Check if user already exists and get their ID
	var inviteeID primitive.ObjectID
	if req.Email != "" {
		user, err := cs.userRepo.GetByEmail(ctx, req.Email)
		if err == nil {
			inviteeID = user.ID
			// Check if already a member
			isMember, _ := cs.circleRepo.IsMember(ctx, circleID, user.ID.Hex())
			if isMember {
				return nil, errors.New("user already member")
			}
		}
	}

	// Create invitation
	invitation := &models.CircleInvitation{
		CircleID:  circleObjectID,
		InviterID: userObjectID,
		InviteeID: inviteeID,
		Email:     req.Email,
		Role:      req.Role,
		Message:   req.Message,
		Status:    "pending",
		ExpiresAt: time.Now().AddDate(0, 0, 7), // 7 days from now
	}

	err = cs.circleRepo.CreateInvitation(ctx, invitation)
	if err != nil {
		return nil, err
	}

	return invitation, nil
}

func (cs *CircleService) GetInvitation(ctx context.Context, userID, invitationID string) (*models.CircleInvitation, error) {
	invitation, err := cs.circleRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to this invitation
	if invitation.InviterID.Hex() != userID && invitation.InviteeID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	return invitation, nil
}

func (cs *CircleService) UpdateInvitation(ctx context.Context, userID, invitationID string, req map[string]interface{}) (*models.CircleInvitation, error) {
	invitation, err := cs.circleRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}

	// Only inviter can update invitation
	if invitation.InviterID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	// Build update document
	update := bson.M{}
	for key, value := range req {
		switch key {
		case "role", "message":
			update[key] = value
		}
	}

	if len(update) > 0 {
		update["updatedAt"] = time.Now()
		invitationCollection := cs.circleRepo.GetInvitationCollection()
		objectID, _ := primitive.ObjectIDFromHex(invitationID)
		_, err = invitationCollection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": update})
		if err != nil {
			return nil, err
		}
	}

	return cs.circleRepo.GetInvitationByID(ctx, invitationID)
}

func (cs *CircleService) DeleteInvitation(ctx context.Context, userID, invitationID string) error {
	invitation, err := cs.circleRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return err
	}

	// Only inviter can delete invitation
	if invitation.InviterID.Hex() != userID {
		return errors.New("access denied")
	}

	invitationCollection := cs.circleRepo.GetInvitationCollection()
	objectID, _ := primitive.ObjectIDFromHex(invitationID)
	_, err = invitationCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (cs *CircleService) ResendInvitation(ctx context.Context, userID, invitationID string) error {
	invitation, err := cs.circleRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return err
	}

	// Only inviter can resend invitation
	if invitation.InviterID.Hex() != userID {
		return errors.New("access denied")
	}

	if invitation.Status != "pending" {
		return errors.New("invitation not pending")
	}

	// Update expiration time
	invitationCollection := cs.circleRepo.GetInvitationCollection()
	objectID, _ := primitive.ObjectIDFromHex(invitationID)
	_, err = invitationCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{
			"expiresAt": time.Now().AddDate(0, 0, 7),
			"updatedAt": time.Now(),
		}},
	)

	// TODO: Send invitation email/notification

	return err
}

// ========================
// Join Operations
// ========================

func (cs *CircleService) JoinByInviteCode(ctx context.Context, userID, inviteCode string) (*models.Circle, error) {
	// Find circle by invite code
	circle, err := cs.circleRepo.GetByInviteCode(ctx, inviteCode)
	if err != nil {
		return nil, errors.New("invalid invite code")
	}

	// Check if user is already a member
	isMember, err := cs.circleRepo.IsMember(ctx, circle.ID.Hex(), userID)
	if err != nil {
		return nil, err
	}

	if isMember {
		return nil, errors.New("already member")
	}

	// Check member limit
	if len(circle.Members) >= circle.Settings.MaxMembers {
		return nil, errors.New("circle full")
	}

	// Add user as member
	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	newMember := models.CircleMember{
		UserID: userObjectID,
		Role:   "member",
		Status: "active",
		Permissions: models.MemberPermissions{
			CanSeeLocation:   true,
			CanSeeDriving:    true,
			CanSendMessages:  true,
			CanManagePlaces:  false,
			CanReceiveAlerts: true,
			CanSendEmergency: true,
		},
		JoinedAt: time.Now(),
	}

	err = cs.circleRepo.AddMember(ctx, circle.ID.Hex(), newMember)
	if err != nil {
		return nil, err
	}

	// Update stats
	cs.circleRepo.Update(ctx, circle.ID.Hex(), bson.M{
		"stats.totalMembers":  len(circle.Members) + 1,
		"stats.activeMembers": len(circle.Members) + 1,
	})

	return cs.circleRepo.GetByID(ctx, circle.ID.Hex())
}

func (cs *CircleService) AcceptInvitation(ctx context.Context, userID, invitationID string) (*models.Circle, error) {
	// Get invitation details
	invitation, err := cs.circleRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return nil, errors.New("invitation not found")
	}

	// Check if invitation is for this user
	if invitation.InviteeID.Hex() != userID && invitation.Email != "" {
		// Check if user email matches invitation email
		user, err := cs.userRepo.GetByID(ctx, userID)
		if err != nil || user.Email != invitation.Email {
			return nil, errors.New("access denied")
		}
	}

	// Check if invitation is still valid
	if invitation.Status != "pending" {
		return nil, errors.New("invitation not pending")
	}

	// Check if invitation has expired
	if invitation.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("invitation expired")
	}

	// Get circle
	circle, err := cs.circleRepo.GetByID(ctx, invitation.CircleID.Hex())
	if err != nil {
		return nil, errors.New("circle not found")
	}

	// Check member limit
	if len(circle.Members) >= circle.Settings.MaxMembers {
		return nil, errors.New("circle has reached maximum member limit")
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)

	// Add member to circle
	newMember := models.CircleMember{
		UserID: userObjectID,
		Role:   invitation.Role,
		Status: "active",
		Permissions: models.MemberPermissions{
			CanSeeLocation:   true,
			CanSeeDriving:    true,
			CanSendMessages:  true,
			CanManagePlaces:  invitation.Role == "admin",
			CanReceiveAlerts: true,
			CanSendEmergency: true,
		},
		JoinedAt: time.Now(),
	}

	err = cs.circleRepo.AddMember(ctx, invitation.CircleID.Hex(), newMember)
	if err != nil {
		return nil, err
	}

	// Update invitation status
	cs.circleRepo.UpdateInvitationStatus(ctx, invitationID, "accepted")

	// Update circle stats
	cs.circleRepo.Update(ctx, invitation.CircleID.Hex(), bson.M{
		"stats.totalMembers":  len(circle.Members) + 1,
		"stats.activeMembers": len(circle.Members) + 1,
	})

	return cs.circleRepo.GetByID(ctx, invitation.CircleID.Hex())
}

func (cs *CircleService) RequestToJoin(ctx context.Context, userID, circleID, message string) (*models.JoinRequest, error) {
	// Check if circle exists
	circle, err := cs.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, errors.New("circle not found")
	}

	// Check if user is already a member
	isMember, _ := cs.circleRepo.IsMember(ctx, circleID, userID)
	if isMember {
		return nil, errors.New("already member")
	}

	// Check if circle accepts requests
	if !circle.Settings.RequireApproval {
		return nil, errors.New("circle not accepting requests")
	}

	// Create join request
	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	circleObjectID, _ := primitive.ObjectIDFromHex(circleID)

	joinRequest := &models.JoinRequest{
		ID:        primitive.NewObjectID(),
		CircleID:  circleObjectID,
		UserID:    userObjectID,
		Message:   message,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save join request (implement in repository)
	// For now, return the request
	return joinRequest, nil
}

// ========================
// Member Management
// ========================

func (cs *CircleService) GetMembers(ctx context.Context, userID, circleID string) ([]models.CircleMember, error) {
	// Check if user is a member of the circle
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// Get circle with members
	circle, err := cs.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, errors.New("circle not found")
	}

	return circle.Members, nil
}

func (cs *CircleService) GetMember(ctx context.Context, userID, circleID, memberID string) (*models.CircleMember, error) {
	// Check if user has access
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// Get circle members
	circle, err := cs.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, errors.New("circle not found")
	}

	memberObjectID, _ := primitive.ObjectIDFromHex(memberID)
	for _, member := range circle.Members {
		if member.UserID == memberObjectID {
			return &member, nil
		}
	}

	return nil, errors.New("member not found")
}

func (cs *CircleService) UpdateMember(ctx context.Context, userID, circleID, memberID string, req map[string]interface{}) (*models.CircleMember, error) {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// Update member (implement specific update logic based on req fields)
	// For now, return the member
	return cs.GetMember(ctx, userID, circleID, memberID)
}

func (cs *CircleService) RemoveMember(ctx context.Context, userID, circleID, memberID string) error {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	// Check if member exists in circle
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, memberID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("member not found")
	}

	// Don't allow removing self
	if userID == memberID {
		return errors.New("cannot remove yourself. Use leave circle instead")
	}

	// Get member role to prevent removing other admins
	memberRole, err := cs.circleRepo.GetMemberRole(ctx, circleID, memberID)
	if err != nil {
		return err
	}

	if memberRole == "admin" {
		return errors.New("cannot remove admin")
	}

	return cs.circleRepo.RemoveMember(ctx, circleID, memberID)
}

func (cs *CircleService) PromoteMember(ctx context.Context, userID, circleID, memberID string) error {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	// Check current member role
	memberRole, err := cs.circleRepo.GetMemberRole(ctx, circleID, memberID)
	if err != nil {
		return errors.New("member not found")
	}

	if memberRole == "admin" {
		return errors.New("already admin")
	}

	return cs.circleRepo.UpdateMemberRole(ctx, circleID, memberID, "admin")
}

func (cs *CircleService) DemoteMember(ctx context.Context, userID, circleID, memberID string) error {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	// Don't allow demoting self
	if userID == memberID {
		return errors.New("cannot demote self")
	}

	// Check current member role
	memberRole, err := cs.circleRepo.GetMemberRole(ctx, circleID, memberID)
	if err != nil {
		return errors.New("member not found")
	}

	if memberRole != "admin" {
		return errors.New("not admin")
	}

	return cs.circleRepo.UpdateMemberRole(ctx, circleID, memberID, "member")
}

func (cs *CircleService) UpdateMemberPermissions(ctx context.Context, userID, circleID, memberID string, permissions models.MemberPermissions) error {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	// Check if member exists
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, memberID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("member not found")
	}

	return cs.circleRepo.UpdateMemberPermissions(ctx, circleID, memberID, permissions)
}

func (cs *CircleService) GetMemberActivity(ctx context.Context, userID, circleID, memberID string) (interface{}, error) {
	// Check if user has access
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement member activity retrieval
	// For now, return placeholder data
	return map[string]interface{}{
		"lastSeen":        time.Now(),
		"messageCount":    0,
		"locationsShared": 0,
	}, nil
}

// ========================
// Join Requests Management
// ========================

func (cs *CircleService) GetJoinRequests(ctx context.Context, userID, circleID string) ([]models.JoinRequest, error) {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// TODO: Implement join requests retrieval from repository
	return []models.JoinRequest{}, nil
}

func (cs *CircleService) ApproveJoinRequest(ctx context.Context, userID, circleID, requestID string) error {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	// TODO: Implement join request approval logic
	return nil
}

func (cs *CircleService) DeclineJoinRequest(ctx context.Context, userID, circleID, requestID string) error {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	// TODO: Implement join request decline logic
	return nil
}

func (cs *CircleService) DeleteJoinRequest(ctx context.Context, userID, circleID, requestID string) error {
	// Check if user is admin or the requester
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		// TODO: Check if user is the original requester
		return errors.New("access denied")
	}

	// TODO: Implement join request deletion logic
	return nil
}

// ========================
// Settings and Configuration
// ========================

func (cs *CircleService) GetCircleSettings(ctx context.Context, userID, circleID string) (*models.CircleSettings, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	circle, err := cs.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, errors.New("circle not found")
	}

	return &circle.Settings, nil
}

func (cs *CircleService) UpdateCircleSettings(ctx context.Context, userID, circleID string, settings models.CircleSettings) (*models.CircleSettings, error) {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	err = cs.circleRepo.Update(ctx, circleID, bson.M{"settings": settings})
	if err != nil {
		return nil, err
	}

	circle, err := cs.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, err
	}

	return &circle.Settings, nil
}

func (cs *CircleService) GetPrivacySettings(ctx context.Context, userID, circleID string) (map[string]interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement privacy settings structure
	return map[string]interface{}{
		"locationSharing":   true,
		"profileVisibility": "members",
		"activityHistory":   true,
	}, nil
}

func (cs *CircleService) UpdatePrivacySettings(ctx context.Context, userID, circleID string, settings map[string]interface{}) (map[string]interface{}, error) {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// TODO: Implement privacy settings update
	return settings, nil
}

func (cs *CircleService) GetPermissionSettings(ctx context.Context, userID, circleID string) (map[string]interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement permission settings structure
	return map[string]interface{}{
		"memberCanInvite":       false,
		"memberCanCreatePlaces": true,
		"memberCanViewHistory":  true,
	}, nil
}

func (cs *CircleService) UpdatePermissionSettings(ctx context.Context, userID, circleID string, settings map[string]interface{}) (map[string]interface{}, error) {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// TODO: Implement permission settings update
	return settings, nil
}

func (cs *CircleService) GetNotificationSettings(ctx context.Context, userID, circleID string) (map[string]interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement notification settings structure
	return map[string]interface{}{
		"emergencyAlerts":      true,
		"arrivalNotifications": true,
		"drivingAlerts":        true,
		"placeNotifications":   true,
	}, nil
}

func (cs *CircleService) UpdateNotificationSettings(ctx context.Context, userID, circleID string, settings map[string]interface{}) (map[string]interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement notification settings update for user
	return settings, nil
}

// ========================
// Activity and Monitoring
// ========================

func (cs *CircleService) GetCircleActivity(ctx context.Context, userID, circleID string, page, pageSize int, activityType string) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement activity retrieval with pagination
	return map[string]interface{}{
		"activities": []interface{}{},
		"totalCount": 0,
		"page":       page,
		"pageSize":   pageSize,
	}, nil
}

func (cs *CircleService) GetActivityFeed(ctx context.Context, userID, circleID string, page, pageSize int) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement activity feed
	return map[string]interface{}{
		"feed":     []interface{}{},
		"page":     page,
		"pageSize": pageSize,
	}, nil
}

func (cs *CircleService) GetMemberLocations(ctx context.Context, userID, circleID string) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement member locations retrieval
	return map[string]interface{}{
		"locations": []interface{}{},
	}, nil
}

func (cs *CircleService) GetActivityTimeline(ctx context.Context, userID, circleID, startDate, endDate string) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement activity timeline
	return map[string]interface{}{
		"timeline":  []interface{}{},
		"startDate": startDate,
		"endDate":   endDate,
	}, nil
}

func (cs *CircleService) GetCircleEvents(ctx context.Context, userID, circleID, eventType string, page, pageSize int) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement events retrieval
	return map[string]interface{}{
		"events":    []interface{}{},
		"eventType": eventType,
		"page":      page,
		"pageSize":  pageSize,
	}, nil
}

// ========================
// Statistics and Analytics
// ========================

func (cs *CircleService) GetCircleStats(ctx context.Context, userID, circleID string) (*models.CircleStats, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	circle, err := cs.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, errors.New("circle not found")
	}

	return &circle.Stats, nil
}

func (cs *CircleService) GetStatsOverview(ctx context.Context, userID, circleID string) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	circle, err := cs.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, errors.New("circle not found")
	}

	// TODO: Implement comprehensive stats overview
	return map[string]interface{}{
		"basicStats": circle.Stats,
		"memberActivity": map[string]interface{}{
			"activeToday":    0,
			"activeThisWeek": 0,
		},
		"locationStats": map[string]interface{}{
			"totalCheckIns": 0,
			"uniquePlaces":  0,
		},
	}, nil
}

func (cs *CircleService) GetLocationStats(ctx context.Context, userID, circleID, period string) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement location statistics
	return map[string]interface{}{
		"period":          period,
		"totalLocations":  0,
		"uniquePlaces":    0,
		"averageDistance": 0,
	}, nil
}

func (cs *CircleService) GetDrivingStats(ctx context.Context, userID, circleID, period string) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement driving statistics
	return map[string]interface{}{
		"period":       period,
		"totalMiles":   0,
		"averageSpeed": 0,
		"tripCount":    0,
	}, nil
}

func (cs *CircleService) GetPlaceStats(ctx context.Context, userID, circleID string) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement place statistics
	return map[string]interface{}{
		"totalPlaces":      0,
		"mostVisited":      []interface{}{},
		"averageVisitTime": 0,
	}, nil
}

func (cs *CircleService) GetSafetyStats(ctx context.Context, userID, circleID, period string) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement safety statistics
	return map[string]interface{}{
		"period":              period,
		"emergencyCount":      0,
		"safeArrivalRate":     100,
		"averageResponseTime": 0,
	}, nil
}

// ========================
// Places and Geofences
// ========================

func (cs *CircleService) GetCirclePlaces(ctx context.Context, userID, circleID string) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement places retrieval
	return []interface{}{}, nil
}

func (cs *CircleService) CreateCirclePlace(ctx context.Context, userID, circleID string, req map[string]interface{}) (interface{}, error) {
	// Check if user has permission
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// TODO: Implement place creation
	return map[string]interface{}{
		"id":        primitive.NewObjectID().Hex(),
		"name":      req["name"],
		"createdAt": time.Now(),
	}, nil
}

func (cs *CircleService) GetCirclePlace(ctx context.Context, userID, circleID, placeID string) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement place retrieval
	return map[string]interface{}{
		"id":   placeID,
		"name": "Sample Place",
	}, nil
}

func (cs *CircleService) UpdateCirclePlace(ctx context.Context, userID, circleID, placeID string, req map[string]interface{}) (interface{}, error) {
	// Check if user has permission
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// TODO: Implement place update
	return map[string]interface{}{
		"id":        placeID,
		"updatedAt": time.Now(),
	}, nil
}

func (cs *CircleService) DeleteCirclePlace(ctx context.Context, userID, circleID, placeID string) error {
	// Check if user has permission
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	// TODO: Implement place deletion
	return nil
}

func (cs *CircleService) GetPlaceActivity(ctx context.Context, userID, circleID, placeID string, page, pageSize int) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement place activity retrieval
	return map[string]interface{}{
		"activity": []interface{}{},
		"page":     page,
		"pageSize": pageSize,
	}, nil
}

// ========================
// Communication
// ========================

func (cs *CircleService) GetAnnouncements(ctx context.Context, userID, circleID string, page, pageSize int) (interface{}, error) {
	// Check if user is member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// TODO: Implement announcements retrieval
	return map[string]interface{}{
		"announcements": []interface{}{},
		"page":          page,
		"pageSize":      pageSize,
	}, nil
}

func (cs *CircleService) CreateAnnouncement(ctx context.Context, userID, circleID string, req map[string]interface{}) (interface{}, error) {
	// Check if user has permission
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// TODO: Implement announcement creation
	return map[string]interface{}{
		"id":        primitive.NewObjectID().Hex(),
		"title":     req["title"],
		"message":   req["message"],
		"createdAt": time.Now(),
	}, nil
}

func (cs *CircleService) UpdateAnnouncement(ctx context.Context, userID, circleID, announcementID string, req map[string]interface{}) (interface{}, error) {
	// Check if user has permission
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// TODO: Implement announcement update
	return map[string]interface{}{
		"id":        announcementID,
		"updatedAt": time.Now(),
	}, nil
}

func (cs *CircleService) DeleteAnnouncement(ctx context.Context, userID, circleID, announcementID string) error {
	// Check if user has permission
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	// TODO: Implement announcement deletion
	return nil
}

func (cs *CircleService) BroadcastMessage(ctx context.Context, userID, circleID, message, messageType string) error {
	// Check if user has permission
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	// TODO: Implement message broadcasting
	return nil
}

// ========================
// Backup and Export
// ========================

func (cs *CircleService) ExportCircleData(ctx context.Context, userID, circleID, format string, includes []string) (interface{}, error) {
	// Check if user has permission
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// TODO: Implement data export
	return map[string]interface{}{
		"jobId":    primitive.NewObjectID().Hex(),
		"status":   "initiated",
		"format":   format,
		"includes": includes,
	}, nil
}

func (cs *CircleService) GetExportStatus(ctx context.Context, userID, circleID, jobID string) (interface{}, error) {
	// Check if user has permission
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("access denied")
	}

	// TODO: Implement export status retrieval
	return map[string]interface{}{
		"jobId":    jobID,
		"status":   "completed",
		"progress": 100,
	}, nil
}

func (cs *CircleService) DownloadExport(ctx context.Context, userID, circleID, exportID string) (string, error) {
	// Check if user has permission
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return "", err
	}

	if role != "admin" {
		return "", errors.New("access denied")
	}

	// TODO: Implement export download URL generation
	return fmt.Sprintf("https://example.com/exports/%s", exportID), nil
}

// ========================
// Leave Circle
// ========================

func (cs *CircleService) LeaveCircle(ctx context.Context, userID, circleID string) error {
	// Check if user is a member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("access denied")
	}

	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role == "admin" {
		// Check if there are other members
		circle, err := cs.circleRepo.GetByID(ctx, circleID)
		if err != nil {
			return errors.New("circle not found")
		}

		if len(circle.Members) > 1 {
			return errors.New("cannot leave as admin")
		}
	}

	return cs.circleRepo.RemoveMember(ctx, circleID, userID)
}

// ========================
// Discovery
// ========================

func (cs *CircleService) GetPublicCircles(ctx context.Context, userID string, page, pageSize int, category string) (interface{}, error) {
	// TODO: Implement public circles retrieval
	return map[string]interface{}{
		"circles":  []interface{}{},
		"page":     page,
		"pageSize": pageSize,
		"category": category,
	}, nil
}

func (cs *CircleService) GetRecommendedCircles(ctx context.Context, userID string, limit int) (interface{}, error) {
	// TODO: Implement recommended circles based on user preferences/location
	return map[string]interface{}{
		"circles": []interface{}{},
		"limit":   limit,
	}, nil
}

func (cs *CircleService) SearchPublicCircles(ctx context.Context, userID, query string, filters []string, location string, radius, page, pageSize int) (interface{}, error) {
	// TODO: Implement public circles search
	return map[string]interface{}{
		"results":  []interface{}{},
		"query":    query,
		"filters":  filters,
		"location": location,
		"radius":   radius,
		"page":     page,
		"pageSize": pageSize,
	}, nil
}

// ========================
// Helper Methods
// ========================

// GetCircleByID - wrapper for backward compatibility
func (cs *CircleService) GetCircleByID(ctx context.Context, userID, circleID string) (*models.Circle, error) {
	return cs.GetCircle(ctx, userID, circleID)
}

// JoinCircle - wrapper for JoinByInviteCode
func (cs *CircleService) JoinCircle(ctx context.Context, userID string, req models.JoinCircleRequest) (*models.Circle, error) {
	return cs.JoinByInviteCode(ctx, userID, req.InviteCode)
}

// GetUserInvitations gets all invitations for a user
func (cs *CircleService) GetUserInvitations(ctx context.Context, userID, status string) ([]models.CircleInvitation, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Get invitations for user
	invitations, err := cs.circleRepo.GetUserInvitations(ctx, userObjectID, status)
	if err != nil {
		return nil, err
	}

	return invitations, nil
}

// RejectInvitation rejects a circle invitation
func (cs *CircleService) RejectInvitation(ctx context.Context, userID, invitationID string) error {
	// Get invitation details
	invitation, err := cs.circleRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return errors.New("invitation not found")
	}

	// Check if invitation is for this user
	if invitation.InviteeID.Hex() != userID {
		return errors.New("access denied")
	}

	// Check if invitation is still pending
	if invitation.Status != "pending" {
		return errors.New("invitation not pending")
	}

	// Update invitation status
	return cs.circleRepo.UpdateInvitationStatus(ctx, invitationID, "rejected")
}
