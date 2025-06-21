package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"

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

// CreateCircle creates a new circle
// @Summary Create a new circle
// @Description Create a new circle with the authenticated user as admin
// @Tags Circles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.CreateCircleRequest true "Circle data"
// @Success 201 {object} models.APIResponse{data=models.Circle}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /circles [post]
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

// GetCircles gets user's circles
// @Summary Get user circles
// @Description Get all circles the user is a member of
// @Tags Circles
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} models.APIResponse{data=[]models.Circle}
// @Failure 401 {object} models.APIResponse
// @Router /circles [get]
func (cc *CircleController) GetCircles(c *gin.Context) {
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
// @Summary Get circle by ID
// @Description Get circle details by ID
// @Tags Circles
// @Security BearerAuth
// @Produce json
// @Param id path string true "Circle ID"
// @Success 200 {object} models.APIResponse{data=models.Circle}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /circles/{id} [get]
func (cc *CircleController) GetCircle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("id")
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
// @Summary Update circle
// @Description Update circle information (admin only)
// @Tags Circles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Circle ID"
// @Param request body models.UpdateCircleRequest true "Updated circle data"
// @Success 200 {object} models.APIResponse{data=models.Circle}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /circles/{id} [put]
func (cc *CircleController) UpdateCircle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("id")
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
// @Summary Delete circle
// @Description Delete a circle (admin only)
// @Tags Circles
// @Security BearerAuth
// @Produce json
// @Param id path string true "Circle ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /circles/{id} [delete]
func (cc *CircleController) DeleteCircle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("id")
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

// InviteMember invites a member to the circle
// @Summary Invite member
// @Description Invite a user to join the circle
// @Tags Circles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Circle ID"
// @Param request body models.InviteMemberRequest true "Invitation data"
// @Success 200 {object} models.APIResponse{data=models.CircleInvitation}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /circles/{id}/invite [post]
func (cc *CircleController) InviteMember(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("id")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	var req models.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := cc.circleService.InviteMember(c.Request.Context(), userID, circleID, req)
	if err != nil {
		logrus.Errorf("Invite member failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to invite members")
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "user already member":
			utils.ConflictResponse(c, "User is already a member of this circle")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid invitation data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to invite member")
		}
		return
	}

	utils.SuccessResponse(c, "Member invited successfully", nil)
}

// AcceptInvitation accepts a circle invitation
// @Summary Accept invitation
// @Description Accept a circle invitation
// @Tags Circles
// @Security BearerAuth
// @Produce json
// @Param invitationId path string true "Invitation ID"
// @Success 200 {object} models.APIResponse{data=models.Circle}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /circles/invitations/{invitationId}/accept [post]
func (cc *CircleController) AcceptInvitation(c *gin.Context) {
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
		logrus.Errorf("Accept invitation failed: %v", err)

		switch err.Error() {
		case "invitation not found":
			utils.NotFoundResponse(c, "Invitation")
		case "invitation expired":
			utils.BadRequestResponse(c, "Invitation has expired")
		case "access denied":
			utils.ForbiddenResponse(c, "You are not authorized to accept this invitation")
		default:
			utils.InternalServerErrorResponse(c, "Failed to accept invitation")
		}
		return
	}

	utils.SuccessResponse(c, "Invitation accepted successfully", circle)
}

// RejectInvitation rejects a circle invitation
// @Summary Reject invitation
// @Description Reject a circle invitation
// @Tags Circles
// @Security BearerAuth
// @Produce json
// @Param invitationId path string true "Invitation ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /circles/invitations/{invitationId}/reject [post]
func (cc *CircleController) RejectInvitation(c *gin.Context) {
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

	err := cc.circleService.RejectInvitation(c.Request.Context(), userID, invitationID)
	if err != nil {
		logrus.Errorf("Reject invitation failed: %v", err)

		switch err.Error() {
		case "invitation not found":
			utils.NotFoundResponse(c, "Invitation")
		case "access denied":
			utils.ForbiddenResponse(c, "You are not authorized to reject this invitation")
		default:
			utils.InternalServerErrorResponse(c, "Failed to reject invitation")
		}
		return
	}

	utils.SuccessResponse(c, "Invitation rejected successfully", nil)
}

// GetMembers gets circle members
// @Summary Get circle members
// @Description Get all members of a circle
// @Tags Circles
// @Security BearerAuth
// @Produce json
// @Param id path string true "Circle ID"
// @Success 200 {object} models.APIResponse{data=[]models.CircleMember}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /circles/{id}/members [get]
func (cc *CircleController) GetMembers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("id")
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

// UpdateMemberRole updates a member's role
// @Summary Update member role
// @Description Update a member's role in the circle (admin only)
// @Tags Circles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Circle ID"
// @Param memberId path string true "Member ID"
// @Param request body models.UpdateMemberRoleRequest true "Role update data"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /circles/{id}/members/{memberId}/role [put]
func (cc *CircleController) UpdateMemberRole(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("id")
	memberID := c.Param("memberId")
	if circleID == "" || memberID == "" {
		utils.BadRequestResponse(c, "Circle ID and Member ID are required")
		return
	}

	var req models.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := cc.circleService.UpdateMemberRole(c.Request.Context(), userID, circleID, memberID, req)
	if err != nil {
		logrus.Errorf("Update member role failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "member not found":
			utils.NotFoundResponse(c, "Member")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to update member roles")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid role")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update member role")
		}
		return
	}

	utils.SuccessResponse(c, "Member role updated successfully", nil)
}

// RemoveMember removes a member from the circle
// @Summary Remove member
// @Description Remove a member from the circle (admin only)
// @Tags Circles
// @Security BearerAuth
// @Produce json
// @Param id path string true "Circle ID"
// @Param memberId path string true "Member ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /circles/{id}/members/{memberId} [delete]
func (cc *CircleController) RemoveMember(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("id")
	memberID := c.Param("memberId")
	if circleID == "" || memberID == "" {
		utils.BadRequestResponse(c, "Circle ID and Member ID are required")
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
		default:
			utils.InternalServerErrorResponse(c, "Failed to remove member")
		}
		return
	}

	utils.SuccessResponse(c, "Member removed successfully", nil)
}

// LeaveCircle allows a user to leave a circle
// @Summary Leave circle
// @Description Leave a circle
// @Tags Circles
// @Security BearerAuth
// @Produce json
// @Param id path string true "Circle ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /circles/{id}/leave [post]
func (cc *CircleController) LeaveCircle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("id")
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

// GetInvitations gets user's circle invitations
// @Summary Get invitations
// @Description Get all pending circle invitations for the user
// @Tags Circles
// @Security BearerAuth
// @Produce json
// @Param status query string false "Invitation status filter"
// @Success 200 {object} models.APIResponse{data=[]models.CircleInvitation}
// @Failure 401 {object} models.APIResponse
// @Router /circles/invitations [get]
func (cc *CircleController) GetInvitations(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	status := c.Query("status")

	invitations, err := cc.circleService.GetUserInvitations(c.Request.Context(), userID, status)
	if err != nil {
		logrus.Errorf("Get invitations failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get invitations")
		return
	}

	utils.SuccessResponse(c, "Invitations retrieved successfully", invitations)
}
