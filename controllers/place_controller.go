package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PlaceController struct {
	placeService *services.PlaceService
}

func NewPlaceController(placeService *services.PlaceService) *PlaceController {
	return &PlaceController{
		placeService: placeService,
	}
}

// CreatePlace creates a new place
// @Summary Create place
// @Description Create a new place/geofence
// @Tags Places
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.CreatePlaceRequest true "Place data"
// @Success 201 {object} models.APIResponse{data=models.Place}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /places [post]
func (pc *PlaceController) CreatePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreatePlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	place, err := pc.placeService.CreatePlace(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create place failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid place data")
		case "invalid coordinates":
			utils.BadRequestResponse(c, "Invalid latitude or longitude coordinates")
		case "place limit reached":
			utils.BadRequestResponse(c, "Place limit reached for this user")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create place")
		}
		return
	}

	utils.CreatedResponse(c, "Place created successfully", place)
}

// GetPlaces gets user's places
// @Summary Get places
// @Description Get all places for the authenticated user
// @Tags Places
// @Security BearerAuth
// @Produce json
// @Param circleId query string false "Filter by circle ID"
// @Param category query string false "Filter by category"
// @Param shared query bool false "Filter by shared status"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} models.APIResponse{data=[]models.Place}
// @Failure 401 {object} models.APIResponse
// @Router /places [get]
func (pc *PlaceController) GetPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Parse shared filter
	var shared *bool
	if sharedStr := c.Query("shared"); sharedStr != "" {
		if s, err := strconv.ParseBool(sharedStr); err == nil {
			shared = &s
		}
	}

	req := models.GetPlacesRequest{
		CircleID: c.Query("circleId"),
		Category: c.Query("category"),
		Shared:   shared,
		Page:     page,
		PageSize: pageSize,
	}

	places, total, err := pc.placeService.GetPlaces(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get places failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get places")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Places retrieved successfully", places, meta)
}

// GetPlace gets a specific place by ID
// @Summary Get place by ID
// @Description Get place details by ID
// @Tags Places
// @Security BearerAuth
// @Produce json
// @Param id path string true "Place ID"
// @Success 200 {object} models.APIResponse{data=models.Place}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/{id} [get]
func (pc *PlaceController) GetPlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("id")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	place, err := pc.placeService.GetPlace(c.Request.Context(), userID, placeID)
	if err != nil {
		logrus.Errorf("Get place failed: %v", err)

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

// UpdatePlace updates a place
// @Summary Update place
// @Description Update place information
// @Tags Places
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Place ID"
// @Param request body models.UpdatePlaceRequest true "Updated place data"
// @Success 200 {object} models.APIResponse{data=models.Place}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/{id} [put]
func (pc *PlaceController) UpdatePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("id")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.UpdatePlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	place, err := pc.placeService.UpdatePlace(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Update place failed: %v", err)

		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid place data")
		case "invalid coordinates":
			utils.BadRequestResponse(c, "Invalid latitude or longitude coordinates")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place")
		}
		return
	}

	utils.SuccessResponse(c, "Place updated successfully", place)
}

// DeletePlace deletes a place
// @Summary Delete place
// @Description Delete a place
// @Tags Places
// @Security BearerAuth
// @Produce json
// @Param id path string true "Place ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/{id} [delete]
func (pc *PlaceController) DeletePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("id")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	err := pc.placeService.DeletePlace(c.Request.Context(), userID, placeID)
	if err != nil {
		logrus.Errorf("Delete place failed: %v", err)

		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own places")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete place")
		}
		return
	}

	utils.SuccessResponse(c, "Place deleted successfully", nil)
}

// GetCirclePlaces gets places for a specific circle
// @Summary Get circle places
// @Description Get all places for a specific circle
// @Tags Places
// @Security BearerAuth
// @Produce json
// @Param circleId path string true "Circle ID"
// @Param category query string false "Filter by category"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} models.APIResponse{data=[]models.Place}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/circle/{circleId} [get]
func (pc *PlaceController) GetCirclePlaces(c *gin.Context) {
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

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	req := models.GetPlacesRequest{
		CircleID: circleID,
		Category: c.Query("category"),
		Page:     page,
		PageSize: pageSize,
	}

	places, total, err := pc.placeService.GetCirclePlaces(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get circle places failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get circle places")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Circle places retrieved successfully", places, meta)
}

// SharePlace shares a place with a circle
// @Summary Share place
// @Description Share a place with a circle
// @Tags Places
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Place ID"
// @Param request body models.SharePlaceRequest true "Share data"
// @Success 200 {object} models.APIResponse{data=models.Place}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/{id}/share [post]
func (pc *PlaceController) SharePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("id")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.SharePlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	place, err := pc.placeService.SharePlace(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Share place failed: %v", err)

		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only share your own places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid share data")
		case "already shared":
			utils.ConflictResponse(c, "Place is already shared with this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to share place")
		}
		return
	}

	utils.SuccessResponse(c, "Place shared successfully", place)
}

// UnsharePlace unshares a place from a circle
// @Summary Unshare place
// @Description Remove place sharing from a circle
// @Tags Places
// @Security BearerAuth
// @Produce json
// @Param id path string true "Place ID"
// @Param circleId path string true "Circle ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/{id}/unshare/{circleId} [post]
func (pc *PlaceController) UnsharePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("id")
	circleID := c.Param("circleId")
	if placeID == "" || circleID == "" {
		utils.BadRequestResponse(c, "Place ID and Circle ID are required")
		return
	}

	err := pc.placeService.UnsharePlace(c.Request.Context(), userID, placeID, circleID)
	if err != nil {
		logrus.Errorf("Unshare place failed: %v", err)

		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only unshare your own places")
		case "not shared":
			utils.BadRequestResponse(c, "Place is not shared with this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to unshare place")
		}
		return
	}

	utils.SuccessResponse(c, "Place unshared successfully", nil)
}

// GetPlaceVisits gets visit history for a place
// @Summary Get place visits
// @Description Get visit history for a specific place
// @Tags Places
// @Security BearerAuth
// @Produce json
// @Param id path string true "Place ID"
// @Param userId query string false "Filter by user ID"
// @Param startTime query string false "Start time (RFC3339 format)"
// @Param endTime query string false "End time (RFC3339 format)"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} models.APIResponse{data=[]models.PlaceVisit}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/{id}/visits [get]
func (pc *PlaceController) GetPlaceVisits(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("id")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	req := models.GetPlaceVisitsRequest{
		PlaceID:   placeID,
		UserID:    c.Query("userId"),
		StartTime: c.Query("startTime"),
		EndTime:   c.Query("endTime"),
		Page:      page,
		PageSize:  pageSize,
	}

	visits, total, err := pc.placeService.GetPlaceVisits(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get place visits failed: %v", err)

		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		case "invalid time format":
			utils.BadRequestResponse(c, "Invalid time format, use RFC3339")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place visits")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Place visits retrieved successfully", visits, meta)
}

// DetectPlaces automatically detects places from location history
// @Summary Detect places
// @Description Automatically detect frequently visited places from location history
// @Tags Places
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.DetectPlacesRequest true "Detection parameters"
// @Success 200 {object} models.APIResponse{data=[]models.Place}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /places/detect [post]
func (pc *PlaceController) DetectPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.DetectPlacesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use default values if request body is invalid
		req = models.DetectPlacesRequest{
			MinVisits:    5,
			MinDuration:  30, // 30 minutes
			RadiusMeters: 100,
			DaysBack:     30,
		}
	}

	places, err := pc.placeService.DetectPlaces(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Detect places failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid detection parameters")
		case "insufficient location data":
			utils.BadRequestResponse(c, "Insufficient location data for place detection")
		default:
			utils.InternalServerErrorResponse(c, "Failed to detect places")
		}
		return
	}

	utils.SuccessResponse(c, "Places detected successfully", places)
}

// GetNearbyPlaces gets nearby places
// @Summary Get nearby places
// @Description Get places near current location or specified coordinates
// @Tags Places
// @Security BearerAuth
// @Produce json
// @Param latitude query float64 false "Latitude (uses current location if not provided)"
// @Param longitude query float64 false "Longitude (uses current location if not provided)"
// @Param radius query float64 false "Search radius in kilometers" default(5.0)
// @Param category query string false "Filter by category"
// @Param ownOnly query bool false "Only user's own places" default(false)
// @Success 200 {object} models.APIResponse{data=[]models.Place}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /places/nearby [get]
func (pc *PlaceController) GetNearbyPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var latitude, longitude float64
	var err error

	// Parse coordinates if provided
	if latStr := c.Query("latitude"); latStr != "" {
		if latitude, err = strconv.ParseFloat(latStr, 64); err != nil {
			utils.BadRequestResponse(c, "Invalid latitude")
			return
		}
	}

	if lonStr := c.Query("longitude"); lonStr != "" {
		if longitude, err = strconv.ParseFloat(lonStr, 64); err != nil {
			utils.BadRequestResponse(c, "Invalid longitude")
			return
		}
	}

	// Parse radius
	radius := 5.0 // Default 5km
	if radiusStr := c.Query("radius"); radiusStr != "" {
		if r, err := strconv.ParseFloat(radiusStr, 64); err == nil && r > 0 && r <= 50 {
			radius = r
		}
	}

	// Parse ownOnly
	ownOnly := false
	if ownOnlyStr := c.Query("ownOnly"); ownOnlyStr != "" {
		if o, err := strconv.ParseBool(ownOnlyStr); err == nil {
			ownOnly = o
		}
	}

	req := models.GetNearbyPlacesRequest{
		Latitude:  latitude,
		Longitude: longitude,
		Radius:    radius,
		Category:  c.Query("category"),
		OwnOnly:   ownOnly,
	}

	places, err := pc.placeService.GetNearbyPlaces(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get nearby places failed: %v", err)

		switch err.Error() {
		case "invalid coordinates":
			utils.BadRequestResponse(c, "Invalid coordinates provided")
		case "location required":
			utils.BadRequestResponse(c, "Current location or coordinates are required")
		case "location not found":
			utils.BadRequestResponse(c, "Your current location is required to find nearby places")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get nearby places")
		}
		return
	}

	utils.SuccessResponse(c, "Nearby places retrieved successfully", places)
}

// GetPlaceStats gets place statistics
// @Summary Get place statistics
// @Description Get statistics for a specific place or all user places
// @Tags Places
// @Security BearerAuth
// @Produce json
// @Param id path string false "Place ID (if not provided, returns stats for all places)"
// @Param period query string false "Time period (day, week, month, year)" default(month)
// @Success 200 {object} models.APIResponse{data=models.PlaceStats}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/stats [get]
// @Router /places/{id}/stats [get]
func (pc *PlaceController) GetPlaceStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("id") // Optional - if not provided, returns stats for all places
	period := c.Query("period")
	if period == "" {
		period = "month"
	}

	stats, err := pc.placeService.GetPlaceStats(c.Request.Context(), userID, placeID, period)
	if err != nil {
		logrus.Errorf("Get place stats failed: %v", err)

		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Place statistics retrieved successfully", stats)
}

// UpdatePlaceSettings updates notification settings for a place
// @Summary Update place settings
// @Description Update notification and geofence settings for a place
// @Tags Places
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Place ID"
// @Param request body models.PlaceNotifications true "Place notification settings"
// @Success 200 {object} models.APIResponse{data=models.Place}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/{id}/settings [put]
func (pc *PlaceController) UpdatePlaceSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("id")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var settings models.PlaceNotifications
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.BadRequestResponse(c, "Invalid settings data")
		return
	}

	place, err := pc.placeService.UpdatePlaceSettings(c.Request.Context(), userID, placeID, settings)
	if err != nil {
		logrus.Errorf("Update place settings failed: %v", err)

		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update settings for your own places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid settings data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place settings")
		}
		return
	}

	utils.SuccessResponse(c, "Place settings updated successfully", place)
}
