package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type EmergencyController struct {
	emergencyService *services.EmergencyService
}

func NewEmergencyController(emergencyService *services.EmergencyService) *EmergencyController {
	return &EmergencyController{
		emergencyService: emergencyService,
	}
}

// CreateEmergency creates a new emergency alert
// @Summary Create emergency alert
// @Description Create a new emergency alert (SOS, crash detection, etc.)
// @Tags Emergency
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.CreateEmergencyRequest true "Emergency data"
// @Success 201 {object} models.APIResponse{data=models.Emergency}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /emergency [post]
func (ec *EmergencyController) CreateEmergency(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreateEmergencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	emergency, err := ec.emergencyService.CreateEmergency(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create emergency failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid emergency data")
		case "invalid coordinates":
			utils.BadRequestResponse(c, "Invalid location coordinates")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create emergency alert")
		}
		return
	}

	utils.CreatedResponse(c, "Emergency alert created successfully", emergency)
}

// GetEmergency gets a specific emergency by ID
// @Summary Get emergency by ID
// @Description Get emergency details by ID
// @Tags Emergency
// @Security BearerAuth
// @Produce json
// @Param id path string true "Emergency ID"
// @Success 200 {object} models.APIResponse{data=models.Emergency}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /emergency/{id} [get]
func (ec *EmergencyController) GetEmergency(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	emergencyID := c.Param("id")
	if emergencyID == "" {
		utils.BadRequestResponse(c, "Emergency ID is required")
		return
	}

	emergency, err := ec.emergencyService.GetEmergency(c.Request.Context(), userID, emergencyID)
	if err != nil {
		logrus.Errorf("Get emergency failed: %v", err)

		switch err.Error() {
		case "emergency not found":
			utils.NotFoundResponse(c, "Emergency")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this emergency")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get emergency")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency retrieved successfully", emergency)
}

// GetUserEmergencies gets user's emergency history
// @Summary Get user emergencies
// @Description Get all emergencies for the authenticated user
// @Tags Emergency
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Param status query string false "Emergency status filter"
// @Success 200 {object} models.APIResponse{data=[]models.Emergency}
// @Failure 401 {object} models.APIResponse
// @Router /emergency/user [get]
func (ec *EmergencyController) GetUserEmergencies(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	emergencies, err := ec.emergencyService.GetUserEmergencies(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get user emergencies failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get user emergencies")
		return
	}

	utils.SuccessResponse(c, "User emergencies retrieved successfully", emergencies)
}

// GetCircleEmergencies gets emergencies for a circle
// @Summary Get circle emergencies
// @Description Get all emergencies for a specific circle
// @Tags Emergency
// @Security BearerAuth
// @Produce json
// @Param circleId path string true "Circle ID"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Param status query string false "Emergency status filter"
// @Success 200 {object} models.APIResponse{data=[]models.Emergency}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /emergency/circle/{circleId} [get]
func (ec *EmergencyController) GetCircleEmergencies(c *gin.Context) {
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

	emergencies, err := ec.emergencyService.GetCircleEmergencies(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get circle emergencies failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get circle emergencies")
		}
		return
	}

	utils.SuccessResponse(c, "Circle emergencies retrieved successfully", emergencies)
}

// UpdateEmergency updates an emergency alert
// @Summary Update emergency
// @Description Update emergency status or details
// @Tags Emergency
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Emergency ID"
// @Param request body models.UpdateEmergencyRequest true "Emergency update data"
// @Success 200 {object} models.APIResponse{data=models.Emergency}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /emergency/{id} [put]
func (ec *EmergencyController) UpdateEmergency(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	emergencyID := c.Param("id")
	if emergencyID == "" {
		utils.BadRequestResponse(c, "Emergency ID is required")
		return
	}

	var req models.UpdateEmergencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	emergency, err := ec.emergencyService.UpdateEmergency(c.Request.Context(), userID, emergencyID, req)
	if err != nil {
		logrus.Errorf("Update emergency failed: %v", err)

		switch err.Error() {
		case "emergency not found":
			utils.NotFoundResponse(c, "Emergency")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to update this emergency")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid emergency data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update emergency")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency updated successfully", emergency)
}

// ResolveEmergency marks an emergency as resolved
// @Summary Resolve emergency
// @Description Mark an emergency as resolved
// @Tags Emergency
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Emergency ID"
// @Param request body object{resolution=string} false "Resolution details"
// @Success 200 {object} models.APIResponse{data=models.Emergency}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /emergency/{id}/resolve [post]
func (ec *EmergencyController) ResolveEmergency(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	emergencyID := c.Param("id")
	if emergencyID == "" {
		utils.BadRequestResponse(c, "Emergency ID is required")
		return
	}

	var req struct {
		Resolution string `json:"resolution,omitempty"`
	}
	c.ShouldBindJSON(&req)

	err := ec.emergencyService.ResolveEmergency(c.Request.Context(), userID, emergencyID, req.Resolution)
	if err != nil {
		logrus.Errorf("Resolve emergency failed: %v", err)

		switch err.Error() {
		case "emergency not found":
			utils.NotFoundResponse(c, "Emergency")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to resolve this emergency")
		case "emergency already resolved":
			utils.BadRequestResponse(c, "Emergency is already resolved")
		default:
			utils.InternalServerErrorResponse(c, "Failed to resolve emergency")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency resolved successfully", nil)
}

// DismissEmergency dismisses a false alarm
// @Summary Dismiss emergency
// @Description Dismiss an emergency as false alarm
// @Tags Emergency
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Emergency ID"
// @Param request body object{reason=string} false "Dismissal reason"
// @Success 200 {object} models.APIResponse{data=models.Emergency}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /emergency/{id}/dismiss [post]
func (ec *EmergencyController) DismissEmergency(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	emergencyID := c.Param("id")
	if emergencyID == "" {
		utils.BadRequestResponse(c, "Emergency ID is required")
		return
	}

	var req struct {
		Reason string `json:"reason,omitempty"`
	}
	c.ShouldBindJSON(&req)

	emergency, err := ec.emergencyService.DismissEmergency(c.Request.Context(), userID, emergencyID, req.Reason)
	if err != nil {
		logrus.Errorf("Dismiss emergency failed: %v", err)

		switch err.Error() {
		case "emergency not found":
			utils.NotFoundResponse(c, "Emergency")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to dismiss this emergency")
		case "emergency already dismissed":
			utils.BadRequestResponse(c, "Emergency is already dismissed")
		default:
			utils.InternalServerErrorResponse(c, "Failed to dismiss emergency")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency dismissed successfully", emergency)
}

// CancelEmergency cancels an active emergency
// @Summary Cancel emergency
// @Description Cancel an active emergency alert
// @Tags Emergency
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Emergency ID"
// @Param request body object{reason=string} false "Cancellation reason"
// @Success 200 {object} models.APIResponse{data=models.Emergency}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /emergency/{id}/cancel [post]
func (ec *EmergencyController) CancelEmergency(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	emergencyID := c.Param("id")
	if emergencyID == "" {
		utils.BadRequestResponse(c, "Emergency ID is required")
		return
	}

	var req struct {
		Reason string `json:"reason,omitempty"`
	}
	c.ShouldBindJSON(&req)

	emergency, err := ec.emergencyService.CancelEmergency(c.Request.Context(), userID, emergencyID, req.Reason)
	if err != nil {
		logrus.Errorf("Cancel emergency failed: %v", err)

		switch err.Error() {
		case "emergency not found":
			utils.NotFoundResponse(c, "Emergency")
		case "access denied":
			utils.ForbiddenResponse(c, "Only the emergency creator can cancel the alert")
		case "emergency already cancelled":
			utils.BadRequestResponse(c, "Emergency is already cancelled")
		default:
			utils.InternalServerErrorResponse(c, "Failed to cancel emergency")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency cancelled successfully", emergency)
}

// GetActiveEmergencies gets all active emergencies (admin only)
// @Summary Get active emergencies
// @Description Get all active emergencies (admin endpoint)
// @Tags Emergency
// @Security BearerAuth
// @Produce json
// @Param priority query string false "Priority filter"
// @Success 200 {object} models.APIResponse{data=[]models.Emergency}
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /emergency/active [get]
func (ec *EmergencyController) GetActiveEmergencies(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Check if user has admin privileges
	role := c.GetString("role")
	if role != "admin" {
		utils.ForbiddenResponse(c, "Admin access required")
		return
	}

	priority := c.Query("priority")

	emergencies, err := ec.emergencyService.GetActiveEmergencies(c.Request.Context(), priority)
	if err != nil {
		logrus.Errorf("Get active emergencies failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get active emergencies")
		return
	}

	utils.SuccessResponse(c, "Active emergencies retrieved successfully", emergencies)
}

// GetEmergencyStats gets emergency statistics
// @Summary Get emergency statistics
// @Description Get emergency statistics for user or circle
// @Tags Emergency
// @Security BearerAuth
// @Produce json
// @Param circleId query string false "Circle ID for circle stats"
// @Param period query string false "Time period (day, week, month, year)" default(month)
// @Success 200 {object} models.APIResponse{data=models.EmergencyStats}
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /emergency/stats [get]
func (ec *EmergencyController) GetEmergencyStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	c.Query("circleId")
	period := c.Query("period")
	if period == "" {
		period = "month"
	}

	stats, err := ec.emergencyService.GetEmergencyStats(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency stats failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get emergency statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency statistics retrieved successfully", stats)
}

// TestEmergency creates a test emergency for development/testing
// @Summary Test emergency alert
// @Description Create a test emergency alert (development only)
// @Tags Emergency
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.CreateEmergencyRequest true "Test emergency data"
// @Success 201 {object} models.APIResponse{data=models.Emergency}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /emergency/test [post]
func (ec *EmergencyController) TestEmergency(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreateEmergencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	// Mark as test emergency
	req.Type = "test"

	emergency, err := ec.emergencyService.CreateEmergency(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create test emergency failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid test emergency data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create test emergency")
		}
		return
	}

	utils.CreatedResponse(c, "Test emergency created successfully", emergency)
}
