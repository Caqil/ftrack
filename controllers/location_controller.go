package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type LocationController struct {
	locationService *services.LocationService
}

func NewLocationController(locationService *services.LocationService) *LocationController {
	return &LocationController{
		locationService: locationService,
	}
}

// UpdateLocation updates user's current location
// @Summary Update location
// @Description Update user's current location
// @Tags Location
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.Location true "Location data"
// @Success 200 {object} models.APIResponse{data=models.Location}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /location [post]
func (lc *LocationController) UpdateLocation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var location models.Location
	if err := c.ShouldBindJSON(&location); err != nil {
		utils.BadRequestResponse(c, "Invalid location data")
		return
	}

	updatedLocation, err := lc.locationService.UpdateLocation(c.Request.Context(), userID, location)
	if err != nil {
		logrus.Errorf("Update location failed: %v", err)

		switch err.Error() {
		case "invalid coordinates":
			utils.BadRequestResponse(c, "Invalid latitude or longitude coordinates")
		case "location sharing disabled":
			utils.ForbiddenResponse(c, "Location sharing is disabled for this user")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid location data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update location")
		}
		return
	}

	utils.SuccessResponse(c, "Location updated successfully", updatedLocation)
}

// GetCurrentLocation gets user's current location
// @Summary Get current location
// @Description Get user's current location
// @Tags Location
// @Security BearerAuth
// @Produce json
// @Param userId query string false "User ID (if different from authenticated user)"
// @Success 200 {object} models.APIResponse{data=models.Location}
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /location/current [get]
func (lc *LocationController) GetCurrentLocation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	targetUserID := c.Query("userId")
	if targetUserID == "" {
		targetUserID = userID
	}

	location, err := lc.locationService.GetCurrentLocation(c.Request.Context(), userID, targetUserID)
	if err != nil {
		logrus.Errorf("Get current location failed: %v", err)

		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "location not found":
			utils.NotFoundResponse(c, "Location")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to view this user's location")
		case "location sharing disabled":
			utils.ForbiddenResponse(c, "Location sharing is disabled for this user")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get current location")
		}
		return
	}

	utils.SuccessResponse(c, "Current location retrieved successfully", location)
}

// GetLocationHistory gets user's location history
// @Summary Get location history
// @Description Get user's location history with pagination
// @Tags Location
// @Security BearerAuth
// @Produce json
// @Param userId query string false "User ID (if different from authenticated user)"
// @Param startTime query string false "Start time (RFC3339 format)"
// @Param endTime query string false "End time (RFC3339 format)"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(50)
// @Success 200 {object} models.APIResponse{data=[]models.Location}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /location/history [get]
func (lc *LocationController) GetLocationHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	targetUserID := c.Query("userId")
	if targetUserID == "" {
		targetUserID = userID
	}

	// Parse time parameters
	var startTime, endTime *time.Time
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		} else {
			utils.BadRequestResponse(c, "Invalid startTime format, use RFC3339")
			return
		}
	}

	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		} else {
			utils.BadRequestResponse(c, "Invalid endTime format, use RFC3339")
			return
		}
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 50
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	req := models.LocationHistoryRequest{
		TargetUserID: targetUserID,
		StartTime:    startTime,
		EndTime:      endTime,
		Page:         page,
		PageSize:     pageSize,
	}

	locations, total, err := lc.locationService.GetLocationHistory(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get location history failed: %v", err)

		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to view this user's location history")
		case "location sharing disabled":
			utils.ForbiddenResponse(c, "Location sharing is disabled for this user")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get location history")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Location history retrieved successfully", locations, meta)
}

// GetCircleLocations gets current locations of all circle members
// @Summary Get circle locations
// @Description Get current locations of all members in a circle
// @Tags Location
// @Security BearerAuth
// @Produce json
// @Param circleId path string true "Circle ID"
// @Success 200 {object} models.APIResponse{data=[]models.UserLocation}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /location/circle/{circleId} [get]
func (lc *LocationController) GetCircleLocations(c *gin.Context) {
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

	locations, err := lc.locationService.GetCircleLocations(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get circle locations failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get circle locations")
		}
		return
	}

	utils.SuccessResponse(c, "Circle locations retrieved successfully", locations)
}

// ShareLocation temporarily shares location with specific users
// @Summary Share location
// @Description Temporarily share location with specific users or circles
// @Tags Location
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.ShareLocationRequest true "Share location data"
// @Success 200 {object} models.APIResponse{data=models.LocationShare}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /location/share [post]
func (lc *LocationController) ShareLocation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.ShareLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	share, err := lc.locationService.ShareLocation(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Share location failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid share location data")
		case "invalid duration":
			utils.BadRequestResponse(c, "Invalid sharing duration")
		default:
			utils.InternalServerErrorResponse(c, "Failed to share location")
		}
		return
	}

	utils.SuccessResponse(c, "Location shared successfully", share)
}

// StopSharingLocation stops sharing location
// @Summary Stop sharing location
// @Description Stop sharing location with specific users or circles
// @Tags Location
// @Security BearerAuth
// @Produce json
// @Param shareId path string true "Share ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /location/share/{shareId}/stop [post]
func (lc *LocationController) StopSharingLocation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	shareID := c.Param("shareId")
	if shareID == "" {
		utils.BadRequestResponse(c, "Share ID is required")
		return
	}

	err := lc.locationService.StopSharingLocation(c.Request.Context(), userID, shareID)
	if err != nil {
		logrus.Errorf("Stop sharing location failed: %v", err)

		switch err.Error() {
		case "share not found":
			utils.NotFoundResponse(c, "Location share")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to stop this share")
		default:
			utils.InternalServerErrorResponse(c, "Failed to stop sharing location")
		}
		return
	}

	utils.SuccessResponse(c, "Location sharing stopped successfully", nil)
}

// GetLocationShares gets user's active location shares
// @Summary Get location shares
// @Description Get user's active location shares
// @Tags Location
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.LocationShare}
// @Failure 401 {object} models.APIResponse
// @Router /location/shares [get]
func (lc *LocationController) GetLocationShares(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	shares, err := lc.locationService.GetLocationShares(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get location shares failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location shares")
		return
	}

	utils.SuccessResponse(c, "Location shares retrieved successfully", shares)
}

// UpdateLocationSettings updates user's location sharing settings
// @Summary Update location settings
// @Description Update user's location sharing preferences
// @Tags Location
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.LocationSharing true "Location settings"
// @Success 200 {object} models.APIResponse{data=models.LocationSharing}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /location/settings [put]
func (lc *LocationController) UpdateLocationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var settings models.LocationSharing
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.BadRequestResponse(c, "Invalid location settings")
		return
	}

	updatedSettings, err := lc.locationService.UpdateLocationSettings(c.Request.Context(), userID, settings)
	if err != nil {
		logrus.Errorf("Update location settings failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid location settings")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update location settings")
		}
		return
	}

	utils.SuccessResponse(c, "Location settings updated successfully", updatedSettings)
}

// GetLocationSettings gets user's location sharing settings
// @Summary Get location settings
// @Description Get user's current location sharing settings
// @Tags Location
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.LocationSharing}
// @Failure 401 {object} models.APIResponse
// @Router /location/settings [get]
func (lc *LocationController) GetLocationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := lc.locationService.GetLocationSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get location settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location settings")
		return
	}

	utils.SuccessResponse(c, "Location settings retrieved successfully", settings)
}

// GetNearbyUsers gets nearby users based on current location
// @Summary Get nearby users
// @Description Get users near current location (within specified radius)
// @Tags Location
// @Security BearerAuth
// @Produce json
// @Param radius query float64 false "Search radius in kilometers" default(1.0)
// @Param circleId query string false "Filter by circle ID"
// @Success 200 {object} models.APIResponse{data=[]models.NearbyUser}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /location/nearby [get]
func (lc *LocationController) GetNearbyUsers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	radius := 1.0 // Default 1km
	if radiusStr := c.Query("radius"); radiusStr != "" {
		if r, err := strconv.ParseFloat(radiusStr, 64); err == nil && r > 0 && r <= 50 {
			radius = r
		}
	}

	circleID := c.Query("circleId")

	req := models.NearbyUsersRequest{
		Radius:   radius,
		CircleID: circleID,
	}

	nearbyUsers, err := lc.locationService.GetNearbyUsers(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get nearby users failed: %v", err)

		switch err.Error() {
		case "location not found":
			utils.BadRequestResponse(c, "Your current location is required to find nearby users")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get nearby users")
		}
		return
	}

	utils.SuccessResponse(c, "Nearby users retrieved successfully", nearbyUsers)
}

// GetLocationStats gets location statistics
// @Summary Get location statistics
// @Description Get location statistics for user or circle
// @Tags Location
// @Security BearerAuth
// @Produce json
// @Param circleId query string false "Circle ID for circle stats"
// @Param period query string false "Time period (day, week, month, year)" default(week)
// @Success 200 {object} models.APIResponse{data=models.LocationStats}
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /location/stats [get]
func (lc *LocationController) GetLocationStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Query("circleId")
	period := c.Query("period")
	if period == "" {
		period = "week"
	}

	stats, err := lc.locationService.GetLocationStats(c.Request.Context(), userID, circleID, period)
	if err != nil {
		logrus.Errorf("Get location stats failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get location statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Location statistics retrieved successfully", stats)
}
