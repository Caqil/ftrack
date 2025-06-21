package services

import (
	"context"
	"errors"
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

func (cs *CircleService) GetCircleByID(ctx context.Context, userID, circleID string) (*models.Circle, error) {
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

func (cs *CircleService) JoinCircle(ctx context.Context, userID string, req models.JoinCircleRequest) (*models.Circle, error) {
	// Validate request
	if validationErrors := cs.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Find circle by invite code
	circle, err := cs.circleRepo.GetByInviteCode(ctx, req.InviteCode)
	if err != nil {
		return nil, errors.New("invalid invite code")
	}

	// Check if user is already a member
	isMember, err := cs.circleRepo.IsMember(ctx, circle.ID.Hex(), userID)
	if err != nil {
		return nil, err
	}

	if isMember {
		return nil, errors.New("already a member of this circle")
	}

	// Check member limit
	if len(circle.Members) >= circle.Settings.MaxMembers {
		return nil, errors.New("circle has reached maximum member limit")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Add member
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
	}

	err = cs.circleRepo.AddMember(ctx, circle.ID.Hex(), newMember)
	if err != nil {
		return nil, err
	}

	// Update stats
	circle.Stats.TotalMembers++
	circle.Stats.ActiveMembers++
	err = cs.circleRepo.Update(ctx, circle.ID.Hex(), bson.M{
		"stats": circle.Stats,
	})
	if err != nil {
		// Log error but don't fail the operation
		utils.GetLogger().Warn("Failed to update circle stats: ", err)
	}

	return cs.circleRepo.GetByID(ctx, circle.ID.Hex())
}

func (cs *CircleService) LeaveCircle(ctx context.Context, userID, circleID string) error {
	// Check if user is a member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("not a member of this circle")
	}

	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role == "admin" {
		// Get circle to check member count
		circle, err := cs.circleRepo.GetByID(ctx, circleID)
		if err != nil {
			return err
		}

		if len(circle.Members) > 1 {
			return errors.New("admin cannot leave circle with other members. Transfer admin role first")
		}
	}

	return cs.circleRepo.RemoveMember(ctx, circleID, userID)
}

func (cs *CircleService) InviteMember(ctx context.Context, userID, circleID string, req models.InviteMemberRequest) error {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("only admins can invite members")
	}

	// Validate request
	if validationErrors := cs.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return errors.New("validation failed")
	}

	// Check if email or phone is provided
	if req.Email == "" && req.Phone == "" {
		return errors.New("email or phone number is required")
	}

	// Find user by email or phone
	var invitedUser *models.User
	if req.Email != "" {
		invitedUser, err = cs.userRepo.GetByEmail(ctx, req.Email)
		if err != nil {
			return errors.New("user not found")
		}
	} else {
		invitedUser, err = cs.userRepo.GetByPhone(ctx, req.Phone)
		if err != nil {
			return errors.New("user not found")
		}
	}

	// Check if user is already a member
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, invitedUser.ID.Hex())
	if err != nil {
		return err
	}

	if isMember {
		return errors.New("user is already a member")
	}

	// Add member with invited status
	newMember := models.CircleMember{
		UserID:      invitedUser.ID,
		Role:        "member",
		Status:      "invited",
		Permissions: req.Permissions,
		InvitedBy:   primitive.ObjectID{}, // Convert userID to ObjectID
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	newMember.InvitedBy = userObjectID

	return cs.circleRepo.AddMember(ctx, circleID, newMember)
}

func (cs *CircleService) UpdateMemberPermissions(ctx context.Context, userID, circleID string, req models.UpdateMemberPermissionsRequest) error {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("only admins can update member permissions")
	}

	// Validate request
	if validationErrors := cs.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return errors.New("validation failed")
	}

	return cs.circleRepo.UpdateMemberPermissions(ctx, circleID, req.UserID, req.Permissions)
}

func (cs *CircleService) UpdateCircle(ctx context.Context, userID, circleID string, req models.UpdateCircleRequest) (*models.Circle, error) {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, errors.New("only admins can update circle")
	}

	// Validate request
	if validationErrors := cs.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Build update document
	update := bson.M{}

	if req.Name != nil {
		update["name"] = *req.Name
	}
	if req.Settings != nil {
		update["settings"] = *req.Settings
	}

	if len(update) == 0 {
		return nil, errors.New("no fields to update")
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
		return errors.New("only admins can delete circle")
	}

	return cs.circleRepo.Delete(ctx, circleID)
}

func (cs *CircleService) UpdateLastActivity(ctx context.Context, circleID, userID string) error {
	return cs.circleRepo.UpdateLastActivity(ctx, circleID, userID)
}

// GetCircle - wrapper for GetCircleByID to match controller expectations
func (cs *CircleService) GetCircle(ctx context.Context, userID, circleID string) (*models.Circle, error) {
	return cs.GetCircleByID(ctx, userID, circleID)
}

// AcceptInvitation accepts a circle invitation
func (cs *CircleService) AcceptInvitation(ctx context.Context, userID, invitationID string) (*models.Circle, error) {
	// Get invitation details
	invitation, err := cs.circleRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return nil, errors.New("invitation not found")
	}

	// Check if invitation is for this user
	if invitation.InviteeID.Hex() != userID {
		return nil, errors.New("access denied")
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

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Add member to circle
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

	err = cs.circleRepo.AddMember(ctx, invitation.CircleID.Hex(), newMember)
	if err != nil {
		return nil, err
	}

	// Update invitation status
	err = cs.circleRepo.UpdateInvitationStatus(ctx, invitationID, "accepted")
	if err != nil {
		// Log error but don't fail the operation
		utils.GetLogger().Warn("Failed to update invitation status: ", err)
	}

	// Update circle stats
	circle.Stats.TotalMembers++
	circle.Stats.ActiveMembers++
	err = cs.circleRepo.Update(ctx, invitation.CircleID.Hex(), bson.M{
		"stats": circle.Stats,
	})
	if err != nil {
		// Log error but don't fail the operation
		utils.GetLogger().Warn("Failed to update circle stats: ", err)
	}

	return cs.circleRepo.GetByID(ctx, invitation.CircleID.Hex())
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

// GetMembers gets all members of a circle
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

// UpdateMemberRole updates a member's role in the circle
func (cs *CircleService) UpdateMemberRole(ctx context.Context, userID, circleID, memberID string, req models.UpdateMemberRoleRequest) error {
	// Check if user is admin
	role, err := cs.circleRepo.GetMemberRole(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if role != "admin" {
		return errors.New("access denied")
	}

	// Validate request
	if validationErrors := cs.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return errors.New("validation failed")
	}

	// Validate role
	validRoles := []string{"admin", "member", "moderator"}
	roleValid := false
	for _, validRole := range validRoles {
		if req.Role == validRole {
			roleValid = true
			break
		}
	}
	if !roleValid {
		return errors.New("validation failed")
	}

	// Check if member exists in circle
	isMember, err := cs.circleRepo.IsMember(ctx, circleID, memberID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("member not found")
	}

	// Don't allow changing own role
	if userID == memberID {
		return errors.New("cannot change your own role")
	}

	// Update member role
	return cs.circleRepo.UpdateMemberRole(ctx, circleID, memberID, req.Role)
}

// RemoveMember removes a member from the circle
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
		return errors.New("cannot remove another admin")
	}

	// Remove member
	err = cs.circleRepo.RemoveMember(ctx, circleID, memberID)
	if err != nil {
		return err
	}

	// Update circle stats
	circle, err := cs.circleRepo.GetByID(ctx, circleID)
	if err == nil {
		circle.Stats.TotalMembers--
		circle.Stats.ActiveMembers--
		cs.circleRepo.Update(ctx, circleID, bson.M{
			"stats": circle.Stats,
		})
	}

	return nil
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
