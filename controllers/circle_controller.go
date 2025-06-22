package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type CircleController struct {
	circleService *services.CircleService
}

func NewCircleController(circleService *services.CircleService) *CircleController {
	return &CircleController{
		circleService: circleService,
	}
}

// ========================
// Basic CRUD Operations
// ========================

// CreateCircle creates a new circle
func (cc *CircleController) CreateCircle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreateCircleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	circle, err := cc.circleService.CreateCircle(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create circle failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid circle data")
		case "circle limit reached":
			utils.BadRequestResponse(c, "Circle limit reached")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create circle")
		}
		return
	}

	utils.CreatedResponse(c, "Circle created successfully", circle)
}

// GetUserCircles gets user's circles
func (cc *CircleController) GetUserCircles(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circles, err := cc.circleService.GetUserCircles(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get circles failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get circles")
		return
	}

	utils.SuccessResponse(c, "Circles retrieved successfully", circles)
}

// GetCircle gets a specific circle
func (cc *CircleController) GetCircle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	circle, err := cc.circleService.GetCircle(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get circle failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get circle")
		}
		return
	}

	utils.SuccessResponse(c, "Circle retrieved successfully", circle)
}

// UpdateCircle updates a circle
func (cc *CircleController) UpdateCircle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req models.UpdateCircleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	circle, err := cc.circleService.UpdateCircle(c.Request.Context(), userID, circleID, req)
	if err != nil {
		logrus.Errorf("Update circle failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "Only circle admins can update circle settings")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid circle data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update circle")
		}
		return
	}

	utils.SuccessResponse(c, "Circle updated successfully", circle)
}

// DeleteCircle deletes a circle
func (cc *CircleController) DeleteCircle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	err := cc.circleService.DeleteCircle(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Delete circle failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "Only circle admins can delete the circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete circle")
		}
		return
	}

	utils.SuccessResponse(c, "Circle deleted successfully", nil)
}

// ========================
// Invitation Management
// ========================

// GetCircleInvitations gets all invitations for a circle
func (cc *CircleController) GetCircleInvitations(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	invitations, err := cc.circleService.GetCircleInvitations(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get circle invitations failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to view invitations")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get invitations")
		}
		return
	}

	utils.SuccessResponse(c, "Invitations retrieved successfully", invitations)
}

// CreateInvitation creates a new invitation
func (cc *CircleController) CreateInvitation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req models.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	invitation, err := cc.circleService.CreateInvitation(c.Request.Context(), userID, circleID, req)
	if err != nil {
		logrus.Errorf("Create invitation failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to invite members")
		case "user already invited":
			utils.BadRequestResponse(c, "User already has a pending invitation")
		case "user already member":
			utils.BadRequestResponse(c, "User is already a member of this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create invitation")
		}
		return
	}

	utils.CreatedResponse(c, "Invitation created successfully", invitation)
}

// GetInvitation gets a specific invitation
func (cc *CircleController) GetInvitation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	invitationID := c.Param("invitationId")
	if invitationID == "" {
		utils.BadRequestResponse(c, "Invitation ID is required")
		return
	}

	invitation, err := cc.circleService.GetInvitation(c.Request.Context(), userID, invitationID)
	if err != nil {
		logrus.Errorf("Get invitation failed: %v", err)
		switch err.Error() {
		case "invitation not found":
			utils.NotFoundResponse(c, "Invitation")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this invitation")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get invitation")
		}
		return
	}

	utils.SuccessResponse(c, "Invitation retrieved successfully", invitation)
}

// UpdateInvitation updates an invitation
func (cc *CircleController) UpdateInvitation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	invitationID := c.Param("invitationId")
	if invitationID == "" {
		utils.BadRequestResponse(c, "Invitation ID is required")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	invitation, err := cc.circleService.UpdateInvitation(c.Request.Context(), userID, invitationID, req)
	if err != nil {
		logrus.Errorf("Update invitation failed: %v", err)
		switch err.Error() {
		case "invitation not found":
			utils.NotFoundResponse(c, "Invitation")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to update this invitation")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update invitation")
		}
		return
	}

	utils.SuccessResponse(c, "Invitation updated successfully", invitation)
}

// DeleteInvitation deletes an invitation
func (cc *CircleController) DeleteInvitation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	invitationID := c.Param("invitationId")
	if invitationID == "" {
		utils.BadRequestResponse(c, "Invitation ID is required")
		return
	}

	err := cc.circleService.DeleteInvitation(c.Request.Context(), userID, invitationID)
	if err != nil {
		logrus.Errorf("Delete invitation failed: %v", err)
		switch err.Error() {
		case "invitation not found":
			utils.NotFoundResponse(c, "Invitation")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to delete this invitation")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete invitation")
		}
		return
	}

	utils.SuccessResponse(c, "Invitation deleted successfully", nil)
}

// ResendInvitation resends an invitation
func (cc *CircleController) ResendInvitation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	invitationID := c.Param("invitationId")
	if invitationID == "" {
		utils.BadRequestResponse(c, "Invitation ID is required")
		return
	}

	err := cc.circleService.ResendInvitation(c.Request.Context(), userID, invitationID)
	if err != nil {
		logrus.Errorf("Resend invitation failed: %v", err)
		switch err.Error() {
		case "invitation not found":
			utils.NotFoundResponse(c, "Invitation")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to resend this invitation")
		case "invitation not pending":
			utils.BadRequestResponse(c, "Only pending invitations can be resent")
		default:
			utils.InternalServerErrorResponse(c, "Failed to resend invitation")
		}
		return
	}

	utils.SuccessResponse(c, "Invitation resent successfully", nil)
}

// ========================
// Join Operations
// ========================

// JoinByInviteCode joins a circle using invite code
func (cc *CircleController) JoinByInviteCode(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.JoinCircleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	circle, err := cc.circleService.JoinByInviteCode(c.Request.Context(), userID, req.InviteCode)
	if err != nil {
		logrus.Errorf("Join by invite code failed: %v", err)
		switch err.Error() {
		case "invalid invite code":
			utils.BadRequestResponse(c, "Invalid or expired invite code")
		case "already member":
			utils.BadRequestResponse(c, "You are already a member of this circle")
		case "circle full":
			utils.BadRequestResponse(c, "Circle has reached maximum capacity")
		default:
			utils.InternalServerErrorResponse(c, "Failed to join circle")
		}
		return
	}

	utils.SuccessResponse(c, "Successfully joined circle", circle)
}

// JoinByInvitation joins a circle using invitation ID
func (cc *CircleController) JoinByInvitation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	invitationID := c.Param("invitationId")
	if invitationID == "" {
		utils.BadRequestResponse(c, "Invitation ID is required")
		return
	}

	circle, err := cc.circleService.AcceptInvitation(c.Request.Context(), userID, invitationID)
	if err != nil {
		logrus.Errorf("Join by invitation failed: %v", err)
		switch err.Error() {
		case "invitation not found":
			utils.NotFoundResponse(c, "Invitation")
		case "access denied":
			utils.ForbiddenResponse(c, "This invitation is not for you")
		case "invitation expired":
			utils.BadRequestResponse(c, "Invitation has expired")
		case "invitation not pending":
			utils.BadRequestResponse(c, "Invitation is no longer pending")
		default:
			utils.InternalServerErrorResponse(c, "Failed to accept invitation")
		}
		return
	}

	utils.SuccessResponse(c, "Successfully joined circle", circle)
}

// RequestToJoin requests to join a circle
func (cc *CircleController) RequestToJoin(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req struct {
		Message string `json:"message,omitempty"`
	}
	c.ShouldBindJSON(&req)

	joinRequest, err := cc.circleService.RequestToJoin(c.Request.Context(), userID, circleID, req.Message)
	if err != nil {
		logrus.Errorf("Request to join failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "already member":
			utils.BadRequestResponse(c, "You are already a member of this circle")
		case "already requested":
			utils.BadRequestResponse(c, "You have already requested to join this circle")
		case "circle not accepting requests":
			utils.BadRequestResponse(c, "This circle is not accepting join requests")
		default:
			utils.InternalServerErrorResponse(c, "Failed to request to join")
		}
		return
	}

	utils.CreatedResponse(c, "Join request sent successfully", joinRequest)
}

// ========================
// Member Management
// ========================

// GetCircleMembers gets all members of a circle
func (cc *CircleController) GetCircleMembers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	members, err := cc.circleService.GetMembers(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get members failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get members")
		}
		return
	}

	utils.SuccessResponse(c, "Members retrieved successfully", members)
}

// GetCircleMember gets a specific member
func (cc *CircleController) GetCircleMember(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	memberID := c.Param("userId")
	if circleID == "" || memberID == "" {
		utils.BadRequestResponse(c, "Circle ID and User ID are required")
		return
	}

	member, err := cc.circleService.GetMember(c.Request.Context(), userID, circleID, memberID)
	if err != nil {
		logrus.Errorf("Get member failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "member not found":
			utils.NotFoundResponse(c, "Member")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this member")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get member")
		}
		return
	}

	utils.SuccessResponse(c, "Member retrieved successfully", member)
}

// UpdateCircleMember updates a circle member
func (cc *CircleController) UpdateCircleMember(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	memberID := c.Param("userId")
	if circleID == "" || memberID == "" {
		utils.BadRequestResponse(c, "Circle ID and User ID are required")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	member, err := cc.circleService.UpdateMember(c.Request.Context(), userID, circleID, memberID, req)
	if err != nil {
		logrus.Errorf("Update member failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "member not found":
			utils.NotFoundResponse(c, "Member")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to update this member")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update member")
		}
		return
	}

	utils.SuccessResponse(c, "Member updated successfully", member)
}

// RemoveCircleMember removes a member from circle
func (cc *CircleController) RemoveCircleMember(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	memberID := c.Param("userId")
	if circleID == "" || memberID == "" {
		utils.BadRequestResponse(c, "Circle ID and User ID are required")
		return
	}

	err := cc.circleService.RemoveMember(c.Request.Context(), userID, circleID, memberID)
	if err != nil {
		logrus.Errorf("Remove member failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "member not found":
			utils.NotFoundResponse(c, "Member")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to remove members")
		case "cannot remove admin":
			utils.BadRequestResponse(c, "Cannot remove circle admin")
		default:
			utils.InternalServerErrorResponse(c, "Failed to remove member")
		}
		return
	}

	utils.SuccessResponse(c, "Member removed successfully", nil)
}

// PromoteMember promotes a member to admin
func (cc *CircleController) PromoteMember(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	memberID := c.Param("userId")
	if circleID == "" || memberID == "" {
		utils.BadRequestResponse(c, "Circle ID and User ID are required")
		return
	}

	err := cc.circleService.PromoteMember(c.Request.Context(), userID, circleID, memberID)
	if err != nil {
		logrus.Errorf("Promote member failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "member not found":
			utils.NotFoundResponse(c, "Member")
		case "access denied":
			utils.ForbiddenResponse(c, "Only admins can promote members")
		case "already admin":
			utils.BadRequestResponse(c, "Member is already an admin")
		default:
			utils.InternalServerErrorResponse(c, "Failed to promote member")
		}
		return
	}

	utils.SuccessResponse(c, "Member promoted successfully", nil)
}

// DemoteMember demotes an admin to member
func (cc *CircleController) DemoteMember(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	memberID := c.Param("userId")
	if circleID == "" || memberID == "" {
		utils.BadRequestResponse(c, "Circle ID and User ID are required")
		return
	}

	err := cc.circleService.DemoteMember(c.Request.Context(), userID, circleID, memberID)
	if err != nil {
		logrus.Errorf("Demote member failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "member not found":
			utils.NotFoundResponse(c, "Member")
		case "access denied":
			utils.ForbiddenResponse(c, "Only admins can demote members")
		case "not admin":
			utils.BadRequestResponse(c, "Member is not an admin")
		case "cannot demote self":
			utils.BadRequestResponse(c, "Cannot demote yourself")
		default:
			utils.InternalServerErrorResponse(c, "Failed to demote member")
		}
		return
	}

	utils.SuccessResponse(c, "Member demoted successfully", nil)
}

// UpdateMemberPermissions updates member permissions
func (cc *CircleController) UpdateMemberPermissions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	memberID := c.Param("userId")
	if circleID == "" || memberID == "" {
		utils.BadRequestResponse(c, "Circle ID and User ID are required")
		return
	}

	var req models.UpdateMemberPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := cc.circleService.UpdateMemberPermissions(c.Request.Context(), userID, circleID, memberID, req.Permissions)
	if err != nil {
		logrus.Errorf("Update member permissions failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "member not found":
			utils.NotFoundResponse(c, "Member")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to update member permissions")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update member permissions")
		}
		return
	}

	utils.SuccessResponse(c, "Member permissions updated successfully", nil)
}

// GetMemberActivity gets member activity
func (cc *CircleController) GetMemberActivity(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	memberID := c.Param("userId")
	if circleID == "" || memberID == "" {
		utils.BadRequestResponse(c, "Circle ID and User ID are required")
		return
	}

	activity, err := cc.circleService.GetMemberActivity(c.Request.Context(), userID, circleID, memberID)
	if err != nil {
		logrus.Errorf("Get member activity failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "member not found":
			utils.NotFoundResponse(c, "Member")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this member's activity")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get member activity")
		}
		return
	}

	utils.SuccessResponse(c, "Member activity retrieved successfully", activity)
}

// ========================
// Join Requests Management
// ========================

// GetJoinRequests gets all join requests for a circle
func (cc *CircleController) GetJoinRequests(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	requests, err := cc.circleService.GetJoinRequests(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get join requests failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to view join requests")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get join requests")
		}
		return
	}

	utils.SuccessResponse(c, "Join requests retrieved successfully", requests)
}

// ApproveJoinRequest approves a join request
func (cc *CircleController) ApproveJoinRequest(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	requestID := c.Param("requestId")
	if circleID == "" || requestID == "" {
		utils.BadRequestResponse(c, "Circle ID and Request ID are required")
		return
	}

	err := cc.circleService.ApproveJoinRequest(c.Request.Context(), userID, circleID, requestID)
	if err != nil {
		logrus.Errorf("Approve join request failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "request not found":
			utils.NotFoundResponse(c, "Join request")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to approve join requests")
		case "circle full":
			utils.BadRequestResponse(c, "Circle has reached maximum capacity")
		default:
			utils.InternalServerErrorResponse(c, "Failed to approve join request")
		}
		return
	}

	utils.SuccessResponse(c, "Join request approved successfully", nil)
}

// DeclineJoinRequest declines a join request
func (cc *CircleController) DeclineJoinRequest(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	requestID := c.Param("requestId")
	if circleID == "" || requestID == "" {
		utils.BadRequestResponse(c, "Circle ID and Request ID are required")
		return
	}

	err := cc.circleService.DeclineJoinRequest(c.Request.Context(), userID, circleID, requestID)
	if err != nil {
		logrus.Errorf("Decline join request failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "request not found":
			utils.NotFoundResponse(c, "Join request")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to decline join requests")
		default:
			utils.InternalServerErrorResponse(c, "Failed to decline join request")
		}
		return
	}

	utils.SuccessResponse(c, "Join request declined successfully", nil)
}

// DeleteJoinRequest deletes a join request
func (cc *CircleController) DeleteJoinRequest(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	requestID := c.Param("requestId")
	if circleID == "" || requestID == "" {
		utils.BadRequestResponse(c, "Circle ID and Request ID are required")
		return
	}

	err := cc.circleService.DeleteJoinRequest(c.Request.Context(), userID, circleID, requestID)
	if err != nil {
		logrus.Errorf("Delete join request failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "request not found":
			utils.NotFoundResponse(c, "Join request")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to delete this join request")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete join request")
		}
		return
	}

	utils.SuccessResponse(c, "Join request deleted successfully", nil)
}

// ========================
// Settings and Configuration
// ========================

// GetCircleSettings gets circle settings
func (cc *CircleController) GetCircleSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	settings, err := cc.circleService.GetCircleSettings(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get circle settings failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle's settings")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get circle settings")
		}
		return
	}

	utils.SuccessResponse(c, "Circle settings retrieved successfully", settings)
}

// UpdateCircleSettings updates circle settings
func (cc *CircleController) UpdateCircleSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req models.CircleSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := cc.circleService.UpdateCircleSettings(c.Request.Context(), userID, circleID, req)
	if err != nil {
		logrus.Errorf("Update circle settings failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "Only admins can update circle settings")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update circle settings")
		}
		return
	}

	utils.SuccessResponse(c, "Circle settings updated successfully", settings)
}

// GetPrivacySettings gets privacy settings
func (cc *CircleController) GetPrivacySettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	settings, err := cc.circleService.GetPrivacySettings(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get privacy settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get privacy settings")
		return
	}

	utils.SuccessResponse(c, "Privacy settings retrieved successfully", settings)
}

// UpdatePrivacySettings updates privacy settings
func (cc *CircleController) UpdatePrivacySettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := cc.circleService.UpdatePrivacySettings(c.Request.Context(), userID, circleID, req)
	if err != nil {
		logrus.Errorf("Update privacy settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update privacy settings")
		return
	}

	utils.SuccessResponse(c, "Privacy settings updated successfully", settings)
}

// GetPermissionSettings gets permission settings
func (cc *CircleController) GetPermissionSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	settings, err := cc.circleService.GetPermissionSettings(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get permission settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get permission settings")
		return
	}

	utils.SuccessResponse(c, "Permission settings retrieved successfully", settings)
}

// UpdatePermissionSettings updates permission settings
func (cc *CircleController) UpdatePermissionSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := cc.circleService.UpdatePermissionSettings(c.Request.Context(), userID, circleID, req)
	if err != nil {
		logrus.Errorf("Update permission settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update permission settings")
		return
	}

	utils.SuccessResponse(c, "Permission settings updated successfully", settings)
}

// GetNotificationSettings gets notification settings
func (cc *CircleController) GetNotificationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	settings, err := cc.circleService.GetNotificationSettings(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get notification settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification settings")
		return
	}

	utils.SuccessResponse(c, "Notification settings retrieved successfully", settings)
}

// UpdateNotificationSettings updates notification settings
func (cc *CircleController) UpdateNotificationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := cc.circleService.UpdateNotificationSettings(c.Request.Context(), userID, circleID, req)
	if err != nil {
		logrus.Errorf("Update notification settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update notification settings")
		return
	}

	utils.SuccessResponse(c, "Notification settings updated successfully", settings)
}

// ========================
// Activity and Monitoring
// ========================

// GetCircleActivity gets circle activity
func (cc *CircleController) GetCircleActivity(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	// Parse query parameters for pagination and filtering
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	activityType := c.Query("type")

	activity, err := cc.circleService.GetCircleActivity(c.Request.Context(), userID, circleID, page, pageSize, activityType)
	if err != nil {
		logrus.Errorf("Get circle activity failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle's activity")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get circle activity")
		}
		return
	}

	utils.SuccessResponse(c, "Circle activity retrieved successfully", activity)
}

// GetActivityFeed gets activity feed
func (cc *CircleController) GetActivityFeed(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	feed, err := cc.circleService.GetActivityFeed(c.Request.Context(), userID, circleID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get activity feed failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get activity feed")
		return
	}

	utils.SuccessResponse(c, "Activity feed retrieved successfully", feed)
}

// GetMemberLocations gets current member locations
func (cc *CircleController) GetMemberLocations(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	locations, err := cc.circleService.GetMemberLocations(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get member locations failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to view member locations")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get member locations")
		}
		return
	}

	utils.SuccessResponse(c, "Member locations retrieved successfully", locations)
}

// GetActivityTimeline gets activity timeline
func (cc *CircleController) GetActivityTimeline(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	timeline, err := cc.circleService.GetActivityTimeline(c.Request.Context(), userID, circleID, startDate, endDate)
	if err != nil {
		logrus.Errorf("Get activity timeline failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get activity timeline")
		return
	}

	utils.SuccessResponse(c, "Activity timeline retrieved successfully", timeline)
}

// GetCircleEvents gets circle events
func (cc *CircleController) GetCircleEvents(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	eventType := c.Query("type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	events, err := cc.circleService.GetCircleEvents(c.Request.Context(), userID, circleID, eventType, page, pageSize)
	if err != nil {
		logrus.Errorf("Get circle events failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get circle events")
		return
	}

	utils.SuccessResponse(c, "Circle events retrieved successfully", events)
}

// ========================
// Statistics and Analytics
// ========================

// GetCircleStats gets circle statistics
func (cc *CircleController) GetCircleStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	stats, err := cc.circleService.GetCircleStats(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get circle stats failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle's statistics")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get circle statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Circle statistics retrieved successfully", stats)
}

// GetStatsOverview gets stats overview
func (cc *CircleController) GetStatsOverview(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	overview, err := cc.circleService.GetStatsOverview(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get stats overview failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get stats overview")
		return
	}

	utils.SuccessResponse(c, "Stats overview retrieved successfully", overview)
}

// GetLocationStats gets location statistics
func (cc *CircleController) GetLocationStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	period := c.DefaultQuery("period", "week")

	stats, err := cc.circleService.GetLocationStats(c.Request.Context(), userID, circleID, period)
	if err != nil {
		logrus.Errorf("Get location stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location statistics")
		return
	}

	utils.SuccessResponse(c, "Location statistics retrieved successfully", stats)
}

// GetDrivingStats gets driving statistics
func (cc *CircleController) GetDrivingStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	period := c.DefaultQuery("period", "week")

	stats, err := cc.circleService.GetDrivingStats(c.Request.Context(), userID, circleID, period)
	if err != nil {
		logrus.Errorf("Get driving stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get driving statistics")
		return
	}

	utils.SuccessResponse(c, "Driving statistics retrieved successfully", stats)
}

// GetPlaceStats gets place statistics
func (cc *CircleController) GetPlaceStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	stats, err := cc.circleService.GetPlaceStats(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get place stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get place statistics")
		return
	}

	utils.SuccessResponse(c, "Place statistics retrieved successfully", stats)
}

// GetSafetyStats gets safety statistics
func (cc *CircleController) GetSafetyStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	period := c.DefaultQuery("period", "month")

	stats, err := cc.circleService.GetSafetyStats(c.Request.Context(), userID, circleID, period)
	if err != nil {
		logrus.Errorf("Get safety stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get safety statistics")
		return
	}

	utils.SuccessResponse(c, "Safety statistics retrieved successfully", stats)
}

// ========================
// Places and Geofences
// ========================

// GetCirclePlaces gets all places for a circle
func (cc *CircleController) GetCirclePlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	places, err := cc.circleService.GetCirclePlaces(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get circle places failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle's places")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get circle places")
		}
		return
	}

	utils.SuccessResponse(c, "Circle places retrieved successfully", places)
}

// CreateCirclePlace creates a new place for the circle
func (cc *CircleController) CreateCirclePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	place, err := cc.circleService.CreateCirclePlace(c.Request.Context(), userID, circleID, req)
	if err != nil {
		logrus.Errorf("Create circle place failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to create places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid place data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create place")
		}
		return
	}

	utils.CreatedResponse(c, "Place created successfully", place)
}

// GetCirclePlace gets a specific place
func (cc *CircleController) GetCirclePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	placeID := c.Param("placeId")
	if circleID == "" || placeID == "" {
		utils.BadRequestResponse(c, "Circle ID and Place ID are required")
		return
	}

	place, err := cc.circleService.GetCirclePlace(c.Request.Context(), userID, circleID, placeID)
	if err != nil {
		logrus.Errorf("Get circle place failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place")
		}
		return
	}

	utils.SuccessResponse(c, "Place retrieved successfully", place)
}

// UpdateCirclePlace updates a place
func (cc *CircleController) UpdateCirclePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	placeID := c.Param("placeId")
	if circleID == "" || placeID == "" {
		utils.BadRequestResponse(c, "Circle ID and Place ID are required")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	place, err := cc.circleService.UpdateCirclePlace(c.Request.Context(), userID, circleID, placeID, req)
	if err != nil {
		logrus.Errorf("Update circle place failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to update this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place")
		}
		return
	}

	utils.SuccessResponse(c, "Place updated successfully", place)
}

// DeleteCirclePlace deletes a place
func (cc *CircleController) DeleteCirclePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	placeID := c.Param("placeId")
	if circleID == "" || placeID == "" {
		utils.BadRequestResponse(c, "Circle ID and Place ID are required")
		return
	}

	err := cc.circleService.DeleteCirclePlace(c.Request.Context(), userID, circleID, placeID)
	if err != nil {
		logrus.Errorf("Delete circle place failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to delete this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete place")
		}
		return
	}

	utils.SuccessResponse(c, "Place deleted successfully", nil)
}

// GetPlaceActivity gets place activity
func (cc *CircleController) GetPlaceActivity(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	placeID := c.Param("placeId")
	if circleID == "" || placeID == "" {
		utils.BadRequestResponse(c, "Circle ID and Place ID are required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	activity, err := cc.circleService.GetPlaceActivity(c.Request.Context(), userID, circleID, placeID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get place activity failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get place activity")
		return
	}

	utils.SuccessResponse(c, "Place activity retrieved successfully", activity)
}

// ========================
// Communication
// ========================

// GetAnnouncements gets circle announcements
func (cc *CircleController) GetAnnouncements(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	announcements, err := cc.circleService.GetAnnouncements(c.Request.Context(), userID, circleID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get announcements failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle's announcements")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get announcements")
		}
		return
	}

	utils.SuccessResponse(c, "Announcements retrieved successfully", announcements)
}

// CreateAnnouncement creates a new announcement
func (cc *CircleController) CreateAnnouncement(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	announcement, err := cc.circleService.CreateAnnouncement(c.Request.Context(), userID, circleID, req)
	if err != nil {
		logrus.Errorf("Create announcement failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to create announcements")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create announcement")
		}
		return
	}

	utils.CreatedResponse(c, "Announcement created successfully", announcement)
}

// UpdateAnnouncement updates an announcement
func (cc *CircleController) UpdateAnnouncement(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	announcementID := c.Param("announcementId")
	if circleID == "" || announcementID == "" {
		utils.BadRequestResponse(c, "Circle ID and Announcement ID are required")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	announcement, err := cc.circleService.UpdateAnnouncement(c.Request.Context(), userID, circleID, announcementID, req)
	if err != nil {
		logrus.Errorf("Update announcement failed: %v", err)
		switch err.Error() {
		case "announcement not found":
			utils.NotFoundResponse(c, "Announcement")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to update this announcement")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update announcement")
		}
		return
	}

	utils.SuccessResponse(c, "Announcement updated successfully", announcement)
}

// DeleteAnnouncement deletes an announcement
func (cc *CircleController) DeleteAnnouncement(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	announcementID := c.Param("announcementId")
	if circleID == "" || announcementID == "" {
		utils.BadRequestResponse(c, "Circle ID and Announcement ID are required")
		return
	}

	err := cc.circleService.DeleteAnnouncement(c.Request.Context(), userID, circleID, announcementID)
	if err != nil {
		logrus.Errorf("Delete announcement failed: %v", err)
		switch err.Error() {
		case "announcement not found":
			utils.NotFoundResponse(c, "Announcement")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to delete this announcement")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete announcement")
		}
		return
	}

	utils.SuccessResponse(c, "Announcement deleted successfully", nil)
}

// BroadcastMessage broadcasts a message to all circle members
func (cc *CircleController) BroadcastMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req struct {
		Message string `json:"message" validate:"required"`
		Type    string `json:"type,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := cc.circleService.BroadcastMessage(c.Request.Context(), userID, circleID, req.Message, req.Type)
	if err != nil {
		logrus.Errorf("Broadcast message failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to broadcast messages")
		default:
			utils.InternalServerErrorResponse(c, "Failed to broadcast message")
		}
		return
	}

	utils.SuccessResponse(c, "Message broadcasted successfully", nil)
}

// ========================
// Backup and Export
// ========================

// ExportCircleData initiates circle data export
func (cc *CircleController) ExportCircleData(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req struct {
		Format   string   `json:"format,omitempty"`
		Includes []string `json:"includes,omitempty"`
	}
	c.ShouldBindJSON(&req)

	exportJob, err := cc.circleService.ExportCircleData(c.Request.Context(), userID, circleID, req.Format, req.Includes)
	if err != nil {
		logrus.Errorf("Export circle data failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to export circle data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to initiate export")
		}
		return
	}

	utils.CreatedResponse(c, "Export initiated successfully", exportJob)
}

// GetExportStatus gets export job status
func (cc *CircleController) GetExportStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	jobID := c.Query("jobId")
	if jobID == "" {
		utils.BadRequestResponse(c, "Job ID is required")
		return
	}

	status, err := cc.circleService.GetExportStatus(c.Request.Context(), userID, circleID, jobID)
	if err != nil {
		logrus.Errorf("Get export status failed: %v", err)
		switch err.Error() {
		case "job not found":
			utils.NotFoundResponse(c, "Export job")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this export job")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get export status")
		}
		return
	}

	utils.SuccessResponse(c, "Export status retrieved successfully", status)
}

// DownloadExport downloads exported data
func (cc *CircleController) DownloadExport(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	exportID := c.Param("exportId")
	if circleID == "" || exportID == "" {
		utils.BadRequestResponse(c, "Circle ID and Export ID are required")
		return
	}

	downloadURL, err := cc.circleService.DownloadExport(c.Request.Context(), userID, circleID, exportID)
	if err != nil {
		logrus.Errorf("Download export failed: %v", err)
		switch err.Error() {
		case "export not found":
			utils.NotFoundResponse(c, "Export")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this export")
		case "export not ready":
			utils.BadRequestResponse(c, "Export is not ready for download")
		default:
			utils.InternalServerErrorResponse(c, "Failed to download export")
		}
		return
	}

	utils.SuccessResponse(c, "Download URL generated successfully", map[string]string{"downloadURL": downloadURL})
}

// ========================
// Leave Circle
// ========================

// LeaveCircle allows a user to leave a circle
func (cc *CircleController) LeaveCircle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	err := cc.circleService.LeaveCircle(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Leave circle failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You are not a member of this circle")
		case "cannot leave as admin":
			utils.BadRequestResponse(c, "Admins cannot leave the circle. Transfer ownership first")
		default:
			utils.InternalServerErrorResponse(c, "Failed to leave circle")
		}
		return
	}

	utils.SuccessResponse(c, "Left circle successfully", nil)
}

// ========================
// Discovery
// ========================

// GetPublicCircles gets public circles
func (cc *CircleController) GetPublicCircles(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	category := c.Query("category")

	circles, err := cc.circleService.GetPublicCircles(c.Request.Context(), userID, page, pageSize, category)
	if err != nil {
		logrus.Errorf("Get public circles failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get public circles")
		return
	}

	utils.SuccessResponse(c, "Public circles retrieved successfully", circles)
}

// GetRecommendedCircles gets recommended circles for the user
func (cc *CircleController) GetRecommendedCircles(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	circles, err := cc.circleService.GetRecommendedCircles(c.Request.Context(), userID, limit)
	if err != nil {
		logrus.Errorf("Get recommended circles failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get recommended circles")
		return
	}

	utils.SuccessResponse(c, "Recommended circles retrieved successfully", circles)
}

// SearchPublicCircles searches public circles
func (cc *CircleController) SearchPublicCircles(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Query    string   `json:"query" validate:"required"`
		Filters  []string `json:"filters,omitempty"`
		Location string   `json:"location,omitempty"`
		Radius   int      `json:"radius,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	circles, err := cc.circleService.SearchPublicCircles(c.Request.Context(), userID, req.Query, req.Filters, req.Location, req.Radius, page, pageSize)
	if err != nil {
		logrus.Errorf("Search public circles failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to search public circles")
		return
	}

	utils.SuccessResponse(c, "Search results retrieved successfully", circles)
}
