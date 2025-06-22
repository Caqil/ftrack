package controllers

import (
	"fmt"
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

// =================== BASIC PLACE OPERATIONS ===================

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

	req := models.GetPlacesRequest{
		CircleID: c.Query("circleId"),
		Category: c.Query("category"),
		Shared:   c.Query("shared") == "true",
		Page:     page,
		PageSize: pageSize,
	}

	places, total, err := pc.placeService.GetPlaces(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get places failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get places")
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Places retrieved successfully", places, meta)
}

// GetPlace gets a single place by ID
// @Summary Get place
// @Description Get place details by ID
// @Tags Places
// @Security BearerAuth
// @Produce json
// @Param placeId path string true "Place ID"
// @Success 200 {object} models.APIResponse{data=models.Place}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/{placeId} [get]
func (pc *PlaceController) GetPlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
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
// @Param placeId path string true "Place ID"
// @Param request body models.UpdatePlaceRequest true "Updated place data"
// @Success 200 {object} models.APIResponse{data=models.Place}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/{placeId} [put]
func (pc *PlaceController) UpdatePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
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
// @Param placeId path string true "Place ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /places/{placeId} [delete]
func (pc *PlaceController) DeletePlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
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

// =================== PLACE CATEGORIES ===================

// GetPlaceCategories gets all place categories
func (pc *PlaceController) GetPlaceCategories(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	categories, err := pc.placeService.GetPlaceCategories(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get place categories failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get place categories")
		return
	}

	utils.SuccessResponse(c, "Place categories retrieved successfully", categories)
}

// CreatePlaceCategory creates a new place category
func (pc *PlaceController) CreatePlaceCategory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreatePlaceCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	category, err := pc.placeService.CreatePlaceCategory(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create place category failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid category data")
		case "category already exists":
			utils.ConflictResponse(c, "Category already exists")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create place category")
		}
		return
	}

	utils.CreatedResponse(c, "Place category created successfully", category)
}

// UpdatePlaceCategory updates a place category
func (pc *PlaceController) UpdatePlaceCategory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	categoryID := c.Param("categoryId")
	if categoryID == "" {
		utils.BadRequestResponse(c, "Category ID is required")
		return
	}

	var req models.UpdatePlaceCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	category, err := pc.placeService.UpdatePlaceCategory(c.Request.Context(), userID, categoryID, req)
	if err != nil {
		logrus.Errorf("Update place category failed: %v", err)
		switch err.Error() {
		case "category not found":
			utils.NotFoundResponse(c, "Category")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own categories")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid category data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place category")
		}
		return
	}

	utils.SuccessResponse(c, "Place category updated successfully", category)
}

// DeletePlaceCategory deletes a place category
func (pc *PlaceController) DeletePlaceCategory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	categoryID := c.Param("categoryId")
	if categoryID == "" {
		utils.BadRequestResponse(c, "Category ID is required")
		return
	}

	err := pc.placeService.DeletePlaceCategory(c.Request.Context(), userID, categoryID)
	if err != nil {
		logrus.Errorf("Delete place category failed: %v", err)
		switch err.Error() {
		case "category not found":
			utils.NotFoundResponse(c, "Category")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own categories")
		case "category in use":
			utils.ConflictResponse(c, "Cannot delete category that is in use")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete place category")
		}
		return
	}

	utils.SuccessResponse(c, "Place category deleted successfully", nil)
}

// GetPlacesByCategory gets places by category
func (pc *PlaceController) GetPlacesByCategory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	categoryID := c.Param("categoryId")
	if categoryID == "" {
		utils.BadRequestResponse(c, "Category ID is required")
		return
	}

	places, err := pc.placeService.GetPlacesByCategory(c.Request.Context(), userID, categoryID)
	if err != nil {
		logrus.Errorf("Get places by category failed: %v", err)
		switch err.Error() {
		case "category not found":
			utils.NotFoundResponse(c, "Category")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get places by category")
		}
		return
	}

	utils.SuccessResponse(c, "Places retrieved successfully", places)
}

// =================== PLACE SEARCH AND DISCOVERY ===================

// SearchPlaces searches for places
func (pc *PlaceController) SearchPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	query := c.Query("q")
	if query == "" {
		utils.BadRequestResponse(c, "Search query is required")
		return
	}

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

	req := models.SearchPlacesRequest{
		Query:    query,
		Category: c.Query("category"),
		Page:     page,
		PageSize: pageSize,
	}

	places, total, err := pc.placeService.SearchPlaces(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Search places failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to search places")
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Places search completed", places, meta)
}

// SearchNearbyPlaces searches for nearby places
func (pc *PlaceController) SearchNearbyPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	latStr := c.Query("lat")
	lonStr := c.Query("lon")
	radiusStr := c.Query("radius")

	if latStr == "" || lonStr == "" {
		utils.BadRequestResponse(c, "Latitude and longitude are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid latitude")
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid longitude")
		return
	}

	radius := 5000.0 // Default 5km
	if radiusStr != "" {
		if r, err := strconv.ParseFloat(radiusStr, 64); err == nil && r > 0 {
			radius = r
		}
	}

	req := models.SearchNearbyPlacesRequest{
		Latitude:  lat,
		Longitude: lon,
		Radius:    radius,
		Category:  c.Query("category"),
	}

	places, err := pc.placeService.SearchNearbyPlaces(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Search nearby places failed: %v", err)
		switch err.Error() {
		case "invalid coordinates":
			utils.BadRequestResponse(c, "Invalid coordinates")
		default:
			utils.InternalServerErrorResponse(c, "Failed to search nearby places")
		}
		return
	}

	utils.SuccessResponse(c, "Nearby places retrieved successfully", places)
}

// GetPopularPlaces gets popular places
func (pc *PlaceController) GetPopularPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	places, err := pc.placeService.GetPopularPlaces(c.Request.Context(), userID, limit)
	if err != nil {
		logrus.Errorf("Get popular places failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get popular places")
		return
	}

	utils.SuccessResponse(c, "Popular places retrieved successfully", places)
}

// GetRecentPlaces gets recently visited places
func (pc *PlaceController) GetRecentPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	places, err := pc.placeService.GetRecentPlaces(c.Request.Context(), userID, limit)
	if err != nil {
		logrus.Errorf("Get recent places failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get recent places")
		return
	}

	utils.SuccessResponse(c, "Recent places retrieved successfully", places)
}

// GetFavoritePlaces gets favorite places
func (pc *PlaceController) GetFavoritePlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	places, err := pc.placeService.GetFavoritePlaces(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get favorite places failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get favorite places")
		return
	}

	utils.SuccessResponse(c, "Favorite places retrieved successfully", places)
}

// AdvancedPlaceSearch performs advanced place search
func (pc *PlaceController) AdvancedPlaceSearch(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.AdvancedSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	places, total, err := pc.placeService.AdvancedPlaceSearch(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Advanced place search failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to perform advanced search")
		return
	}

	meta := utils.CreatePaginationMeta(req.Page, req.PageSize, total)
	utils.SuccessResponseWithMeta(c, "Advanced search completed", places, meta)
}

// =================== GEOFENCING AND AUTOMATION ===================

// GetGeofenceSettings gets geofence settings for a place
func (pc *PlaceController) GetGeofenceSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	settings, err := pc.placeService.GetGeofenceSettings(c.Request.Context(), userID, placeID)
	if err != nil {
		logrus.Errorf("Get geofence settings failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get geofence settings")
		}
		return
	}

	utils.SuccessResponse(c, "Geofence settings retrieved successfully", settings)
}

// UpdateGeofenceSettings updates geofence settings for a place
func (pc *PlaceController) UpdateGeofenceSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.GeofenceSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := pc.placeService.UpdateGeofenceSettings(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Update geofence settings failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update settings for your own places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid settings data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update geofence settings")
		}
		return
	}

	utils.SuccessResponse(c, "Geofence settings updated successfully", settings)
}

// TestGeofence tests geofence functionality
func (pc *PlaceController) TestGeofence(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.TestGeofenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := pc.placeService.TestGeofence(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Test geofence failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to test geofence")
		}
		return
	}

	utils.SuccessResponse(c, "Geofence test completed", result)
}

// GetGeofenceEvents gets geofence events for a place
func (pc *PlaceController) GetGeofenceEvents(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

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

	events, total, err := pc.placeService.GetGeofenceEvents(c.Request.Context(), userID, placeID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get geofence events failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get geofence events")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Geofence events retrieved successfully", events, meta)
}

// GetGeofenceActivity gets geofence activity for a place
func (pc *PlaceController) GetGeofenceActivity(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	period := c.Query("period")
	if period == "" {
		period = "week"
	}

	activity, err := pc.placeService.GetGeofenceActivity(c.Request.Context(), userID, placeID, period)
	if err != nil {
		logrus.Errorf("Get geofence activity failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get geofence activity")
		}
		return
	}

	utils.SuccessResponse(c, "Geofence activity retrieved successfully", activity)
}

// =================== PLACE NOTIFICATIONS ===================

// GetPlaceNotifications gets notification settings for a place
func (pc *PlaceController) GetPlaceNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	notifications, err := pc.placeService.GetPlaceNotifications(c.Request.Context(), userID, placeID)
	if err != nil {
		logrus.Errorf("Get place notifications failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place notifications")
		}
		return
	}

	utils.SuccessResponse(c, "Place notifications retrieved successfully", notifications)
}

// UpdatePlaceNotifications updates notification settings for a place
func (pc *PlaceController) UpdatePlaceNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.PlaceNotifications
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	notifications, err := pc.placeService.UpdatePlaceNotifications(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Update place notifications failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update notifications for your own places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid notification settings")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place notifications")
		}
		return
	}

	utils.SuccessResponse(c, "Place notifications updated successfully", notifications)
}

// TestPlaceNotification tests place notification
func (pc *PlaceController) TestPlaceNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.TestNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := pc.placeService.TestPlaceNotification(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Test place notification failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to test place notification")
		}
		return
	}

	utils.SuccessResponse(c, "Test notification sent successfully", nil)
}

// GetNotificationHistory gets notification history for a place
func (pc *PlaceController) GetNotificationHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

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

	history, total, err := pc.placeService.GetNotificationHistory(c.Request.Context(), userID, placeID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get notification history failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get notification history")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Notification history retrieved successfully", history, meta)
}

// =================== PLACE SHARING AND PERMISSIONS ===================

// GetPlaceSharing gets sharing settings for a place
func (pc *PlaceController) GetPlaceSharing(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	sharing, err := pc.placeService.GetPlaceSharing(c.Request.Context(), userID, placeID)
	if err != nil {
		logrus.Errorf("Get place sharing failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place sharing")
		}
		return
	}

	utils.SuccessResponse(c, "Place sharing retrieved successfully", sharing)
}

// UpdatePlaceSharing updates sharing settings for a place
func (pc *PlaceController) UpdatePlaceSharing(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.UpdatePlaceSharingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	sharing, err := pc.placeService.UpdatePlaceSharing(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Update place sharing failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update sharing for your own places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid sharing settings")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place sharing")
		}
		return
	}

	utils.SuccessResponse(c, "Place sharing updated successfully", sharing)
}

// InviteToPlace invites users to a place
func (pc *PlaceController) InviteToPlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.InviteToPlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	invitation, err := pc.placeService.InviteToPlace(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Invite to place failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only invite to your own places")
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "already invited":
			utils.ConflictResponse(c, "User already invited")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid invitation data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to send invitation")
		}
		return
	}

	utils.SuccessResponse(c, "Invitation sent successfully", invitation)
}

// GetPlaceMembers gets members of a place
func (pc *PlaceController) GetPlaceMembers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	members, err := pc.placeService.GetPlaceMembers(c.Request.Context(), userID, placeID)
	if err != nil {
		logrus.Errorf("Get place members failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place members")
		}
		return
	}

	utils.SuccessResponse(c, "Place members retrieved successfully", members)
}

// UpdatePlaceMember updates a place member
func (pc *PlaceController) UpdatePlaceMember(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	memberUserID := c.Param("userId")
	if placeID == "" || memberUserID == "" {
		utils.BadRequestResponse(c, "Place ID and User ID are required")
		return
	}

	var req models.UpdatePlaceMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	member, err := pc.placeService.UpdatePlaceMember(c.Request.Context(), userID, placeID, memberUserID, req)
	if err != nil {
		logrus.Errorf("Update place member failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "member not found":
			utils.NotFoundResponse(c, "Member")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to update this member")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid member data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place member")
		}
		return
	}

	utils.SuccessResponse(c, "Place member updated successfully", member)
}

// RemovePlaceMember removes a member from a place
func (pc *PlaceController) RemovePlaceMember(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	memberUserID := c.Param("userId")
	if placeID == "" || memberUserID == "" {
		utils.BadRequestResponse(c, "Place ID and User ID are required")
		return
	}

	err := pc.placeService.RemovePlaceMember(c.Request.Context(), userID, placeID, memberUserID)
	if err != nil {
		logrus.Errorf("Remove place member failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "member not found":
			utils.NotFoundResponse(c, "Member")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to remove this member")
		case "cannot remove owner":
			utils.ConflictResponse(c, "Cannot remove place owner")
		default:
			utils.InternalServerErrorResponse(c, "Failed to remove place member")
		}
		return
	}

	utils.SuccessResponse(c, "Place member removed successfully", nil)
}

// =================== PLACE VISIT TRACKING ===================

// GetPlaceVisits gets visits for a place
func (pc *PlaceController) GetPlaceVisits(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

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

	visits, total, err := pc.placeService.GetPlaceVisits(c.Request.Context(), userID, placeID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get place visits failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place visits")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Place visits retrieved successfully", visits, meta)
}

// RecordPlaceVisit records a visit to a place
func (pc *PlaceController) RecordPlaceVisit(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.RecordPlaceVisitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	visit, err := pc.placeService.RecordPlaceVisit(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Record place visit failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		case "visit already in progress":
			utils.ConflictResponse(c, "Visit already in progress")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid visit data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to record place visit")
		}
		return
	}

	utils.CreatedResponse(c, "Place visit recorded successfully", visit)
}

// GetPlaceVisit gets a specific place visit
func (pc *PlaceController) GetPlaceVisit(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	visitID := c.Param("visitId")
	if placeID == "" || visitID == "" {
		utils.BadRequestResponse(c, "Place ID and Visit ID are required")
		return
	}

	visit, err := pc.placeService.GetPlaceVisit(c.Request.Context(), userID, placeID, visitID)
	if err != nil {
		logrus.Errorf("Get place visit failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "visit not found":
			utils.NotFoundResponse(c, "Visit")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this visit")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place visit")
		}
		return
	}

	utils.SuccessResponse(c, "Place visit retrieved successfully", visit)
}

// UpdatePlaceVisit updates a place visit
func (pc *PlaceController) UpdatePlaceVisit(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	visitID := c.Param("visitId")
	if placeID == "" || visitID == "" {
		utils.BadRequestResponse(c, "Place ID and Visit ID are required")
		return
	}

	var req models.UpdatePlaceVisitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	visit, err := pc.placeService.UpdatePlaceVisit(c.Request.Context(), userID, placeID, visitID, req)
	if err != nil {
		logrus.Errorf("Update place visit failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "visit not found":
			utils.NotFoundResponse(c, "Visit")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own visits")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid visit data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place visit")
		}
		return
	}

	utils.SuccessResponse(c, "Place visit updated successfully", visit)
}

// DeletePlaceVisit deletes a place visit
func (pc *PlaceController) DeletePlaceVisit(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	visitID := c.Param("visitId")
	if placeID == "" || visitID == "" {
		utils.BadRequestResponse(c, "Place ID and Visit ID are required")
		return
	}

	err := pc.placeService.DeletePlaceVisit(c.Request.Context(), userID, placeID, visitID)
	if err != nil {
		logrus.Errorf("Delete place visit failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "visit not found":
			utils.NotFoundResponse(c, "Visit")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own visits")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete place visit")
		}
		return
	}

	utils.SuccessResponse(c, "Place visit deleted successfully", nil)
}

// GetVisitStats gets visit statistics for a place
func (pc *PlaceController) GetVisitStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	period := c.Query("period")
	if period == "" {
		period = "month"
	}

	stats, err := pc.placeService.GetVisitStats(c.Request.Context(), userID, placeID, period)
	if err != nil {
		logrus.Errorf("Get visit stats failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get visit statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Visit statistics retrieved successfully", stats)
}

// =================== PLACE HOURS AND AVAILABILITY ===================

// GetPlaceHours gets operating hours for a place
func (pc *PlaceController) GetPlaceHours(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	hours, err := pc.placeService.GetPlaceHours(c.Request.Context(), userID, placeID)
	if err != nil {
		logrus.Errorf("Get place hours failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place hours")
		}
		return
	}

	utils.SuccessResponse(c, "Place hours retrieved successfully", hours)
}

// UpdatePlaceHours updates operating hours for a place
func (pc *PlaceController) UpdatePlaceHours(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.PlaceHours
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	hours, err := pc.placeService.UpdatePlaceHours(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Update place hours failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update hours for your own places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid hours data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place hours")
		}
		return
	}

	utils.SuccessResponse(c, "Place hours updated successfully", hours)
}

// GetCurrentStatus gets current status of a place
func (pc *PlaceController) GetCurrentStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	status, err := pc.placeService.GetCurrentStatus(c.Request.Context(), userID, placeID)
	if err != nil {
		logrus.Errorf("Get current status failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get current status")
		}
		return
	}

	utils.SuccessResponse(c, "Current status retrieved successfully", status)
}

// CreateHoursOverride creates an hours override
func (pc *PlaceController) CreateHoursOverride(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.CreateHoursOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	override, err := pc.placeService.CreateHoursOverride(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Create hours override failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only create overrides for your own places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid override data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create hours override")
		}
		return
	}

	utils.CreatedResponse(c, "Hours override created successfully", override)
}

// DeleteHoursOverride deletes an hours override
func (pc *PlaceController) DeleteHoursOverride(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	overrideID := c.Param("overrideId")
	if placeID == "" || overrideID == "" {
		utils.BadRequestResponse(c, "Place ID and Override ID are required")
		return
	}

	err := pc.placeService.DeleteHoursOverride(c.Request.Context(), userID, placeID, overrideID)
	if err != nil {
		logrus.Errorf("Delete hours override failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "override not found":
			utils.NotFoundResponse(c, "Override")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete overrides for your own places")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete hours override")
		}
		return
	}

	utils.SuccessResponse(c, "Hours override deleted successfully", nil)
}

// =================== PLACE AUTOMATION AND RULES ===================

// GetAutomationRules gets automation rules for a place
func (pc *PlaceController) GetAutomationRules(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	rules, err := pc.placeService.GetAutomationRules(c.Request.Context(), userID, placeID)
	if err != nil {
		logrus.Errorf("Get automation rules failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get automation rules")
		}
		return
	}

	utils.SuccessResponse(c, "Automation rules retrieved successfully", rules)
}

// CreateAutomationRule creates an automation rule
func (pc *PlaceController) CreateAutomationRule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.CreateAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	rule, err := pc.placeService.CreateAutomationRule(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Create automation rule failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only create rules for your own places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid rule data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create automation rule")
		}
		return
	}

	utils.CreatedResponse(c, "Automation rule created successfully", rule)
}

// UpdateAutomationRule updates an automation rule
func (pc *PlaceController) UpdateAutomationRule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	ruleID := c.Param("ruleId")
	if placeID == "" || ruleID == "" {
		utils.BadRequestResponse(c, "Place ID and Rule ID are required")
		return
	}

	var req models.UpdateAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	rule, err := pc.placeService.UpdateAutomationRule(c.Request.Context(), userID, placeID, ruleID, req)
	if err != nil {
		logrus.Errorf("Update automation rule failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "rule not found":
			utils.NotFoundResponse(c, "Rule")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update rules for your own places")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid rule data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update automation rule")
		}
		return
	}

	utils.SuccessResponse(c, "Automation rule updated successfully", rule)
}

// DeleteAutomationRule deletes an automation rule
func (pc *PlaceController) DeleteAutomationRule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	ruleID := c.Param("ruleId")
	if placeID == "" || ruleID == "" {
		utils.BadRequestResponse(c, "Place ID and Rule ID are required")
		return
	}

	err := pc.placeService.DeleteAutomationRule(c.Request.Context(), userID, placeID, ruleID)
	if err != nil {
		logrus.Errorf("Delete automation rule failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "rule not found":
			utils.NotFoundResponse(c, "Rule")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete rules for your own places")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete automation rule")
		}
		return
	}

	utils.SuccessResponse(c, "Automation rule deleted successfully", nil)
}

// TestAutomationRule tests an automation rule
func (pc *PlaceController) TestAutomationRule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	ruleID := c.Param("ruleId")
	if placeID == "" || ruleID == "" {
		utils.BadRequestResponse(c, "Place ID and Rule ID are required")
		return
	}

	var req models.TestAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := pc.placeService.TestAutomationRule(c.Request.Context(), userID, placeID, ruleID, req)
	if err != nil {
		logrus.Errorf("Test automation rule failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "rule not found":
			utils.NotFoundResponse(c, "Rule")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this rule")
		default:
			utils.InternalServerErrorResponse(c, "Failed to test automation rule")
		}
		return
	}

	utils.SuccessResponse(c, "Automation rule test completed", result)
}

// GetAvailableTriggers gets available automation triggers
func (pc *PlaceController) GetAvailableTriggers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	triggers, err := pc.placeService.GetAvailableTriggers(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get available triggers failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get available triggers")
		return
	}

	utils.SuccessResponse(c, "Available triggers retrieved successfully", triggers)
}

// GetAvailableActions gets available automation actions
func (pc *PlaceController) GetAvailableActions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	actions, err := pc.placeService.GetAvailableActions(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get available actions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get available actions")
		return
	}

	utils.SuccessResponse(c, "Available actions retrieved successfully", actions)
}

// =================== PLACE MEDIA ===================

// GetPlaceMedia gets media for a place
func (pc *PlaceController) GetPlaceMedia(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

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

	media, total, err := pc.placeService.GetPlaceMedia(c.Request.Context(), userID, placeID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get place media failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place media")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Place media retrieved successfully", media, meta)
}

// UploadPlaceMedia uploads media for a place
func (pc *PlaceController) UploadPlaceMedia(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.BadRequestResponse(c, "File is required")
		return
	}
	defer file.Close()

	description := c.PostForm("description")
	tags := c.PostFormArray("tags")

	req := models.UploadPlaceMediaRequest{
		File:        file,
		FileName:    header.Filename,
		FileSize:    header.Size,
		ContentType: header.Header.Get("Content-Type"),
		Description: description,
		Tags:        tags,
	}

	media, err := pc.placeService.UploadPlaceMedia(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Upload place media failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only upload media to your own places")
		case "file too large":
			utils.BadRequestResponse(c, "File size exceeds limit")
		case "invalid file type":
			utils.BadRequestResponse(c, "Invalid file type")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid media data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to upload place media")
		}
		return
	}

	utils.CreatedResponse(c, "Place media uploaded successfully", media)
}

// DeletePlaceMedia deletes place media
func (pc *PlaceController) DeletePlaceMedia(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	mediaID := c.Param("mediaId")
	if placeID == "" || mediaID == "" {
		utils.BadRequestResponse(c, "Place ID and Media ID are required")
		return
	}

	err := pc.placeService.DeletePlaceMedia(c.Request.Context(), userID, placeID, mediaID)
	if err != nil {
		logrus.Errorf("Delete place media failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "media not found":
			utils.NotFoundResponse(c, "Media")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own media")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete place media")
		}
		return
	}

	utils.SuccessResponse(c, "Place media deleted successfully", nil)
}

// UpdatePlaceMedia updates place media
func (pc *PlaceController) UpdatePlaceMedia(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	mediaID := c.Param("mediaId")
	if placeID == "" || mediaID == "" {
		utils.BadRequestResponse(c, "Place ID and Media ID are required")
		return
	}

	var req models.UpdatePlaceMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	media, err := pc.placeService.UpdatePlaceMedia(c.Request.Context(), userID, placeID, mediaID, req)
	if err != nil {
		logrus.Errorf("Update place media failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "media not found":
			utils.NotFoundResponse(c, "Media")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own media")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid media data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place media")
		}
		return
	}

	utils.SuccessResponse(c, "Place media updated successfully", media)
}

// GetMediaThumbnail gets media thumbnail
func (pc *PlaceController) GetMediaThumbnail(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	mediaID := c.Param("mediaId")
	if placeID == "" || mediaID == "" {
		utils.BadRequestResponse(c, "Place ID and Media ID are required")
		return
	}

	thumbnail, err := pc.placeService.GetMediaThumbnail(c.Request.Context(), userID, placeID, mediaID)
	if err != nil {
		logrus.Errorf("Get media thumbnail failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "media not found":
			utils.NotFoundResponse(c, "Media")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this media")
		case "thumbnail not available":
			utils.NotFoundResponse(c, "Thumbnail")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get media thumbnail")
		}
		return
	}

	c.Header("Content-Type", thumbnail.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", len(thumbnail.Data)))
	c.Data(200, thumbnail.ContentType, thumbnail.Data)
}

// =================== PLACE REVIEWS AND RATINGS ===================

// GetPlaceReviews gets reviews for a place
func (pc *PlaceController) GetPlaceReviews(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

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

	reviews, total, err := pc.placeService.GetPlaceReviews(c.Request.Context(), userID, placeID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get place reviews failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place reviews")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Place reviews retrieved successfully", reviews, meta)
}

// CreatePlaceReview creates a review for a place
func (pc *PlaceController) CreatePlaceReview(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.CreatePlaceReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	review, err := pc.placeService.CreatePlaceReview(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Create place review failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		case "review already exists":
			utils.ConflictResponse(c, "You have already reviewed this place")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid review data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create place review")
		}
		return
	}

	utils.CreatedResponse(c, "Place review created successfully", review)
}

// UpdatePlaceReview updates a place review
func (pc *PlaceController) UpdatePlaceReview(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	reviewID := c.Param("reviewId")
	if placeID == "" || reviewID == "" {
		utils.BadRequestResponse(c, "Place ID and Review ID are required")
		return
	}

	var req models.UpdatePlaceReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	review, err := pc.placeService.UpdatePlaceReview(c.Request.Context(), userID, placeID, reviewID, req)
	if err != nil {
		logrus.Errorf("Update place review failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "review not found":
			utils.NotFoundResponse(c, "Review")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own reviews")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid review data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place review")
		}
		return
	}

	utils.SuccessResponse(c, "Place review updated successfully", review)
}

// DeletePlaceReview deletes a place review
func (pc *PlaceController) DeletePlaceReview(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	reviewID := c.Param("reviewId")
	if placeID == "" || reviewID == "" {
		utils.BadRequestResponse(c, "Place ID and Review ID are required")
		return
	}

	err := pc.placeService.DeletePlaceReview(c.Request.Context(), userID, placeID, reviewID)
	if err != nil {
		logrus.Errorf("Delete place review failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "review not found":
			utils.NotFoundResponse(c, "Review")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own reviews")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete place review")
		}
		return
	}

	utils.SuccessResponse(c, "Place review deleted successfully", nil)
}

// GetReviewStats gets review statistics for a place
func (pc *PlaceController) GetReviewStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	stats, err := pc.placeService.GetReviewStats(c.Request.Context(), userID, placeID)
	if err != nil {
		logrus.Errorf("Get review stats failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get review statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Review statistics retrieved successfully", stats)
}

// MarkReviewHelpful marks a review as helpful
func (pc *PlaceController) MarkReviewHelpful(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	reviewID := c.Param("reviewId")
	if placeID == "" || reviewID == "" {
		utils.BadRequestResponse(c, "Place ID and Review ID are required")
		return
	}

	var req models.MarkReviewHelpfulRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := pc.placeService.MarkReviewHelpful(c.Request.Context(), userID, placeID, reviewID, req.Helpful)
	if err != nil {
		logrus.Errorf("Mark review helpful failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "review not found":
			utils.NotFoundResponse(c, "Review")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		case "already marked":
			utils.ConflictResponse(c, "You have already marked this review")
		default:
			utils.InternalServerErrorResponse(c, "Failed to mark review as helpful")
		}
		return
	}

	utils.SuccessResponse(c, "Review marked as helpful successfully", nil)
}

// =================== PLACE CHECK-INS ===================

// GetPlaceCheckins gets check-ins for a place
func (pc *PlaceController) GetPlaceCheckins(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

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

	checkins, total, err := pc.placeService.GetPlaceCheckins(c.Request.Context(), userID, placeID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get place check-ins failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place check-ins")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Place check-ins retrieved successfully", checkins, meta)
}

// CheckInToPlace creates a check-in to a place
func (pc *PlaceController) CheckInToPlace(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	var req models.CheckInToPlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	checkin, err := pc.placeService.CheckInToPlace(c.Request.Context(), userID, placeID, req)
	if err != nil {
		logrus.Errorf("Check in to place failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		case "already checked in":
			utils.ConflictResponse(c, "You are already checked in")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid check-in data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to check in to place")
		}
		return
	}

	utils.CreatedResponse(c, "Checked in to place successfully", checkin)
}

// GetCheckin gets a specific check-in
func (pc *PlaceController) GetCheckin(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	checkinID := c.Param("checkinId")
	if placeID == "" || checkinID == "" {
		utils.BadRequestResponse(c, "Place ID and Check-in ID are required")
		return
	}

	checkin, err := pc.placeService.GetCheckin(c.Request.Context(), userID, placeID, checkinID)
	if err != nil {
		logrus.Errorf("Get check-in failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "checkin not found":
			utils.NotFoundResponse(c, "Check-in")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this check-in")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get check-in")
		}
		return
	}

	utils.SuccessResponse(c, "Check-in retrieved successfully", checkin)
}

// UpdateCheckin updates a check-in
func (pc *PlaceController) UpdateCheckin(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	checkinID := c.Param("checkinId")
	if placeID == "" || checkinID == "" {
		utils.BadRequestResponse(c, "Place ID and Check-in ID are required")
		return
	}

	var req models.UpdateCheckinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	checkin, err := pc.placeService.UpdateCheckin(c.Request.Context(), userID, placeID, checkinID, req)
	if err != nil {
		logrus.Errorf("Update check-in failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "checkin not found":
			utils.NotFoundResponse(c, "Check-in")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own check-ins")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid check-in data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update check-in")
		}
		return
	}

	utils.SuccessResponse(c, "Check-in updated successfully", checkin)
}

// DeleteCheckin deletes a check-in
func (pc *PlaceController) DeleteCheckin(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	checkinID := c.Param("checkinId")
	if placeID == "" || checkinID == "" {
		utils.BadRequestResponse(c, "Place ID and Check-in ID are required")
		return
	}

	err := pc.placeService.DeleteCheckin(c.Request.Context(), userID, placeID, checkinID)
	if err != nil {
		logrus.Errorf("Delete check-in failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "checkin not found":
			utils.NotFoundResponse(c, "Check-in")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own check-ins")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete check-in")
		}
		return
	}

	utils.SuccessResponse(c, "Check-in deleted successfully", nil)
}

// GetCheckinLeaderboard gets check-in leaderboard for a place
func (pc *PlaceController) GetCheckinLeaderboard(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	period := c.Query("period")
	if period == "" {
		period = "month"
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	leaderboard, err := pc.placeService.GetCheckinLeaderboard(c.Request.Context(), userID, placeID, period, limit)
	if err != nil {
		logrus.Errorf("Get check-in leaderboard failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get check-in leaderboard")
		}
		return
	}

	utils.SuccessResponse(c, "Check-in leaderboard retrieved successfully", leaderboard)
}

// =================== PLACE RECOMMENDATIONS ===================

// GetPlaceRecommendations gets place recommendations
func (pc *PlaceController) GetPlaceRecommendations(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	recommendations, err := pc.placeService.GetPlaceRecommendations(c.Request.Context(), userID, limit)
	if err != nil {
		logrus.Errorf("Get place recommendations failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get place recommendations")
		return
	}

	utils.SuccessResponse(c, "Place recommendations retrieved successfully", recommendations)
}

// GetNearbyRecommendations gets nearby place recommendations
func (pc *PlaceController) GetNearbyRecommendations(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	latStr := c.Query("lat")
	lonStr := c.Query("lon")
	if latStr == "" || lonStr == "" {
		utils.BadRequestResponse(c, "Latitude and longitude are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid latitude")
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid longitude")
		return
	}

	radius := 5000.0 // Default 5km
	if radiusStr := c.Query("radius"); radiusStr != "" {
		if r, err := strconv.ParseFloat(radiusStr, 64); err == nil && r > 0 {
			radius = r
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	req := models.GetNearbyRecommendationsRequest{
		Latitude:  lat,
		Longitude: lon,
		Radius:    radius,
		Limit:     limit,
	}

	recommendations, err := pc.placeService.GetNearbyRecommendations(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get nearby recommendations failed: %v", err)
		switch err.Error() {
		case "invalid coordinates":
			utils.BadRequestResponse(c, "Invalid coordinates")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get nearby recommendations")
		}
		return
	}

	utils.SuccessResponse(c, "Nearby recommendations retrieved successfully", recommendations)
}

// GetTrendingPlaces gets trending places
func (pc *PlaceController) GetTrendingPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.Query("period")
	if period == "" {
		period = "week"
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	places, err := pc.placeService.GetTrendingPlaces(c.Request.Context(), userID, period, limit)
	if err != nil {
		logrus.Errorf("Get trending places failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get trending places")
		return
	}

	utils.SuccessResponse(c, "Trending places retrieved successfully", places)
}

// GetSimilarPlaces gets similar places
func (pc *PlaceController) GetSimilarPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	placeID := c.Param("placeId")
	if placeID == "" {
		utils.BadRequestResponse(c, "Place ID is required")
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	places, err := pc.placeService.GetSimilarPlaces(c.Request.Context(), userID, placeID, limit)
	if err != nil {
		logrus.Errorf("Get similar places failed: %v", err)
		switch err.Error() {
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this place")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get similar places")
		}
		return
	}

	utils.SuccessResponse(c, "Similar places retrieved successfully", places)
}

// ProvideFeedback provides feedback on recommendations
func (pc *PlaceController) ProvideFeedback(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.RecommendationFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := pc.placeService.ProvideFeedback(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Provide feedback failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid feedback data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to provide feedback")
		}
		return
	}

	utils.SuccessResponse(c, "Feedback provided successfully", nil)
}

// =================== PLACE COLLECTIONS ===================

// GetPlaceCollections gets user's place collections
func (pc *PlaceController) GetPlaceCollections(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	collections, err := pc.placeService.GetPlaceCollections(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get place collections failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get place collections")
		return
	}

	utils.SuccessResponse(c, "Place collections retrieved successfully", collections)
}

// CreatePlaceCollection creates a new place collection
func (pc *PlaceController) CreatePlaceCollection(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreatePlaceCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	collection, err := pc.placeService.CreatePlaceCollection(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create place collection failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid collection data")
		case "collection already exists":
			utils.ConflictResponse(c, "Collection already exists")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create place collection")
		}
		return
	}

	utils.CreatedResponse(c, "Place collection created successfully", collection)
}

// GetPlaceCollection gets a specific place collection
func (pc *PlaceController) GetPlaceCollection(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	collectionID := c.Param("collectionId")
	if collectionID == "" {
		utils.BadRequestResponse(c, "Collection ID is required")
		return
	}

	collection, err := pc.placeService.GetPlaceCollection(c.Request.Context(), userID, collectionID)
	if err != nil {
		logrus.Errorf("Get place collection failed: %v", err)
		switch err.Error() {
		case "collection not found":
			utils.NotFoundResponse(c, "Collection")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this collection")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place collection")
		}
		return
	}

	utils.SuccessResponse(c, "Place collection retrieved successfully", collection)
}

// UpdatePlaceCollection updates a place collection
func (pc *PlaceController) UpdatePlaceCollection(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	collectionID := c.Param("collectionId")
	if collectionID == "" {
		utils.BadRequestResponse(c, "Collection ID is required")
		return
	}

	var req models.UpdatePlaceCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	collection, err := pc.placeService.UpdatePlaceCollection(c.Request.Context(), userID, collectionID, req)
	if err != nil {
		logrus.Errorf("Update place collection failed: %v", err)
		switch err.Error() {
		case "collection not found":
			utils.NotFoundResponse(c, "Collection")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own collections")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid collection data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place collection")
		}
		return
	}

	utils.SuccessResponse(c, "Place collection updated successfully", collection)
}

// DeletePlaceCollection deletes a place collection
func (pc *PlaceController) DeletePlaceCollection(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	collectionID := c.Param("collectionId")
	if collectionID == "" {
		utils.BadRequestResponse(c, "Collection ID is required")
		return
	}

	err := pc.placeService.DeletePlaceCollection(c.Request.Context(), userID, collectionID)
	if err != nil {
		logrus.Errorf("Delete place collection failed: %v", err)
		switch err.Error() {
		case "collection not found":
			utils.NotFoundResponse(c, "Collection")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own collections")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete place collection")
		}
		return
	}

	utils.SuccessResponse(c, "Place collection deleted successfully", nil)
}

// AddPlaceToCollection adds a place to a collection
func (pc *PlaceController) AddPlaceToCollection(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	collectionID := c.Param("collectionId")
	placeID := c.Param("placeId")
	if collectionID == "" || placeID == "" {
		utils.BadRequestResponse(c, "Collection ID and Place ID are required")
		return
	}

	err := pc.placeService.AddPlaceToCollection(c.Request.Context(), userID, collectionID, placeID)
	if err != nil {
		logrus.Errorf("Add place to collection failed: %v", err)
		switch err.Error() {
		case "collection not found":
			utils.NotFoundResponse(c, "Collection")
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this collection or place")
		case "place already in collection":
			utils.ConflictResponse(c, "Place is already in collection")
		default:
			utils.InternalServerErrorResponse(c, "Failed to add place to collection")
		}
		return
	}

	utils.SuccessResponse(c, "Place added to collection successfully", nil)
}

// RemovePlaceFromCollection removes a place from a collection
func (pc *PlaceController) RemovePlaceFromCollection(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	collectionID := c.Param("collectionId")
	placeID := c.Param("placeId")
	if collectionID == "" || placeID == "" {
		utils.BadRequestResponse(c, "Collection ID and Place ID are required")
		return
	}

	err := pc.placeService.RemovePlaceFromCollection(c.Request.Context(), userID, collectionID, placeID)
	if err != nil {
		logrus.Errorf("Remove place from collection failed: %v", err)
		switch err.Error() {
		case "collection not found":
			utils.NotFoundResponse(c, "Collection")
		case "place not found":
			utils.NotFoundResponse(c, "Place")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this collection")
		case "place not in collection":
			utils.NotFoundResponse(c, "Place not in collection")
		default:
			utils.InternalServerErrorResponse(c, "Failed to remove place from collection")
		}
		return
	}

	utils.SuccessResponse(c, "Place removed from collection successfully", nil)
}

// =================== PLACE DATA IMPORT/EXPORT ===================

// ImportPlaces imports places from file
func (pc *PlaceController) ImportPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.BadRequestResponse(c, "File is required")
		return
	}
	defer file.Close()

	req := models.ImportPlacesRequest{
		File:     file,
		FileName: header.Filename,
		FileSize: header.Size,
		Format:   c.PostForm("format"),
	}

	result, err := pc.placeService.ImportPlaces(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Import places failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid import data")
		case "unsupported format":
			utils.BadRequestResponse(c, "Unsupported file format")
		case "file too large":
			utils.BadRequestResponse(c, "File size exceeds limit")
		default:
			utils.InternalServerErrorResponse(c, "Failed to import places")
		}
		return
	}

	utils.SuccessResponse(c, "Places imported successfully", result)
}

// ExportPlaces exports places
func (pc *PlaceController) ExportPlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.ExportPlacesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	exportJob, err := pc.placeService.ExportPlaces(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Export places failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid export parameters")
		default:
			utils.InternalServerErrorResponse(c, "Failed to export places")
		}
		return
	}

	utils.SuccessResponse(c, "Export job created successfully", exportJob)
}

// DownloadPlaceExport downloads exported places
func (pc *PlaceController) DownloadPlaceExport(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	exportID := c.Param("exportId")
	if exportID == "" {
		utils.BadRequestResponse(c, "Export ID is required")
		return
	}

	fileData, err := pc.placeService.DownloadPlaceExport(c.Request.Context(), userID, exportID)
	if err != nil {
		logrus.Errorf("Download place export failed: %v", err)
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

	c.Header("Content-Type", fileData.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileData.FileName))
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileData.Data)))
	c.Data(200, fileData.ContentType, fileData.Data)
}

// GetImportTemplates gets import templates
func (pc *PlaceController) GetImportTemplates(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templates, err := pc.placeService.GetImportTemplates(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get import templates failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get import templates")
		return
	}

	utils.SuccessResponse(c, "Import templates retrieved successfully", templates)
}

// BulkCreatePlaces creates multiple places
func (pc *PlaceController) BulkCreatePlaces(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.BulkCreatePlacesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := pc.placeService.BulkCreatePlaces(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Bulk create places failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid places data")
		case "too many places":
			utils.BadRequestResponse(c, "Too many places in request")
		default:
			utils.InternalServerErrorResponse(c, "Failed to bulk create places")
		}
		return
	}

	utils.SuccessResponse(c, "Places created successfully", result)
}

// =================== PLACE TEMPLATES ===================

// GetPlaceTemplates gets place templates
func (pc *PlaceController) GetPlaceTemplates(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templates, err := pc.placeService.GetPlaceTemplates(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get place templates failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get place templates")
		return
	}

	utils.SuccessResponse(c, "Place templates retrieved successfully", templates)
}

// CreatePlaceTemplate creates a place template
func (pc *PlaceController) CreatePlaceTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreatePlaceTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	template, err := pc.placeService.CreatePlaceTemplate(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create place template failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid template data")
		case "template already exists":
			utils.ConflictResponse(c, "Template already exists")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create place template")
		}
		return
	}

	utils.CreatedResponse(c, "Place template created successfully", template)
}

// GetPlaceTemplate gets a specific place template
func (pc *PlaceController) GetPlaceTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templateID := c.Param("templateId")
	if templateID == "" {
		utils.BadRequestResponse(c, "Template ID is required")
		return
	}

	template, err := pc.placeService.GetPlaceTemplate(c.Request.Context(), userID, templateID)
	if err != nil {
		logrus.Errorf("Get place template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this template")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place template")
		}
		return
	}

	utils.SuccessResponse(c, "Place template retrieved successfully", template)
}

// UpdatePlaceTemplate updates a place template
func (pc *PlaceController) UpdatePlaceTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templateID := c.Param("templateId")
	if templateID == "" {
		utils.BadRequestResponse(c, "Template ID is required")
		return
	}

	var req models.UpdatePlaceTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	template, err := pc.placeService.UpdatePlaceTemplate(c.Request.Context(), userID, templateID, req)
	if err != nil {
		logrus.Errorf("Update place template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own templates")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid template data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update place template")
		}
		return
	}

	utils.SuccessResponse(c, "Place template updated successfully", template)
}

// DeletePlaceTemplate deletes a place template
func (pc *PlaceController) DeletePlaceTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templateID := c.Param("templateId")
	if templateID == "" {
		utils.BadRequestResponse(c, "Template ID is required")
		return
	}

	err := pc.placeService.DeletePlaceTemplate(c.Request.Context(), userID, templateID)
	if err != nil {
		logrus.Errorf("Delete place template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own templates")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete place template")
		}
		return
	}

	utils.SuccessResponse(c, "Place template deleted successfully", nil)
}

// UsePlaceTemplate uses a place template to create a place
func (pc *PlaceController) UsePlaceTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templateID := c.Param("templateId")
	if templateID == "" {
		utils.BadRequestResponse(c, "Template ID is required")
		return
	}

	var req models.UsePlaceTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	place, err := pc.placeService.UsePlaceTemplate(c.Request.Context(), userID, templateID, req)
	if err != nil {
		logrus.Errorf("Use place template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this template")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid place data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to use place template")
		}
		return
	}

	utils.CreatedResponse(c, "Place created from template successfully", place)
}

// =================== PLACE ANALYTICS ===================

// GetPlaceStats gets place statistics
func (pc *PlaceController) GetPlaceStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.Query("period")
	if period == "" {
		period = "month"
	}

	stats, err := pc.placeService.GetPlaceStats(c.Request.Context(), userID, period)
	if err != nil {
		logrus.Errorf("Get place stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get place statistics")
		return
	}

	utils.SuccessResponse(c, "Place statistics retrieved successfully", stats)
}

// GetPlaceUsageStats gets place usage statistics
func (pc *PlaceController) GetPlaceUsageStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.Query("period")
	if period == "" {
		period = "month"
	}

	groupBy := c.Query("groupBy")
	if groupBy == "" {
		groupBy = "day"
	}

	stats, err := pc.placeService.GetPlaceUsageStats(c.Request.Context(), userID, period, groupBy)
	if err != nil {
		logrus.Errorf("Get place usage stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get place usage statistics")
		return
	}

	utils.SuccessResponse(c, "Place usage statistics retrieved successfully", stats)
}

// GetPlaceTrends gets place trends
func (pc *PlaceController) GetPlaceTrends(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.Query("period")
	if period == "" {
		period = "month"
	}

	metric := c.Query("metric")
	if metric == "" {
		metric = "visits"
	}

	trends, err := pc.placeService.GetPlaceTrends(c.Request.Context(), userID, period, metric)
	if err != nil {
		logrus.Errorf("Get place trends failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get place trends")
		return
	}

	utils.SuccessResponse(c, "Place trends retrieved successfully", trends)
}

// GetPlaceHeatmap gets place heatmap data
func (pc *PlaceController) GetPlaceHeatmap(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	latStr := c.Query("lat")
	lonStr := c.Query("lon")
	zoomStr := c.Query("zoom")

	var lat, lon float64
	var zoom int
	var err error

	if latStr != "" && lonStr != "" {
		lat, err = strconv.ParseFloat(latStr, 64)
		if err != nil {
			utils.BadRequestResponse(c, "Invalid latitude")
			return
		}

		lon, err = strconv.ParseFloat(lonStr, 64)
		if err != nil {
			utils.BadRequestResponse(c, "Invalid longitude")
			return
		}
	}

	if zoomStr != "" {
		zoom, err = strconv.Atoi(zoomStr)
		if err != nil || zoom < 1 || zoom > 20 {
			zoom = 10 // Default zoom
		}
	} else {
		zoom = 10
	}

	period := c.Query("period")
	if period == "" {
		period = "month"
	}

	req := models.GetPlaceHeatmapRequest{
		Latitude:  lat,
		Longitude: lon,
		Zoom:      zoom,
		Period:    period,
	}

	heatmap, err := pc.placeService.GetPlaceHeatmap(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get place heatmap failed: %v", err)
		switch err.Error() {
		case "invalid coordinates":
			utils.BadRequestResponse(c, "Invalid coordinates")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get place heatmap")
		}
		return
	}

	utils.SuccessResponse(c, "Place heatmap retrieved successfully", heatmap)
}

// GetPlaceInsights gets place insights
func (pc *PlaceController) GetPlaceInsights(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.Query("period")
	if period == "" {
		period = "month"
	}

	insightType := c.Query("type")
	if insightType == "" {
		insightType = "summary"
	}

	insights, err := pc.placeService.GetPlaceInsights(c.Request.Context(), userID, period, insightType)
	if err != nil {
		logrus.Errorf("Get place insights failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get place insights")
		return
	}

	utils.SuccessResponse(c, "Place insights retrieved successfully", insights)
}
