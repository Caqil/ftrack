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

// ==================== TRACKING ENDPOINTS ====================

// UpdateLocation updates user's current location
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

// BulkUpdateLocation updates multiple location points
func (lc *LocationController) BulkUpdateLocation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var locations []models.Location
	if err := c.ShouldBindJSON(&locations); err != nil {
		utils.BadRequestResponse(c, "Invalid location data")
		return
	}

	result, err := lc.locationService.BulkUpdateLocation(c.Request.Context(), userID, locations)
	if err != nil {
		logrus.Errorf("Bulk update location failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update locations")
		return
	}

	utils.SuccessResponse(c, "Locations updated successfully", result)
}

// GetCurrentLocation gets user's current location
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
		default:
			utils.InternalServerErrorResponse(c, "Failed to get current location")
		}
		return
	}

	utils.SuccessResponse(c, "Current location retrieved successfully", location)
}

// GetLocationHistory gets user's location history
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

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	// Parse time parameters
	var startTime, endTime *time.Time
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	history, err := lc.locationService.GetLocationHistory(c.Request.Context(), userID, targetUserID, startTime, endTime, page, pageSize)
	if err != nil {
		logrus.Errorf("Get location history failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location history")
		return
	}

	utils.SuccessResponse(c, "Location history retrieved successfully", history)
}

// ClearLocationHistory clears user's location history
func (lc *LocationController) ClearLocationHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	err := lc.locationService.ClearLocationHistory(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Clear location history failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to clear location history")
		return
	}

	utils.SuccessResponse(c, "Location history cleared successfully", nil)
}

// ==================== SHARING ENDPOINTS ====================

// GetLocationSettings gets user's location sharing settings
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

// UpdateLocationSettings updates user's location sharing settings
func (lc *LocationController) UpdateLocationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var settings models.LocationSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.BadRequestResponse(c, "Invalid settings data")
		return
	}

	updatedSettings, err := lc.locationService.UpdateLocationSettings(c.Request.Context(), userID, settings)
	if err != nil {
		logrus.Errorf("Update location settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update location settings")
		return
	}

	utils.SuccessResponse(c, "Location settings updated successfully", updatedSettings)
}

// GetSharingPermissions gets user's sharing permissions
func (lc *LocationController) GetSharingPermissions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	permissions, err := lc.locationService.GetSharingPermissions(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get sharing permissions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get sharing permissions")
		return
	}

	utils.SuccessResponse(c, "Sharing permissions retrieved successfully", permissions)
}

// UpdateSharingPermissions updates user's sharing permissions
func (lc *LocationController) UpdateSharingPermissions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var permissions models.SharingPermissions
	if err := c.ShouldBindJSON(&permissions); err != nil {
		utils.BadRequestResponse(c, "Invalid permissions data")
		return
	}

	updatedPermissions, err := lc.locationService.UpdateSharingPermissions(c.Request.Context(), userID, permissions)
	if err != nil {
		logrus.Errorf("Update sharing permissions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update sharing permissions")
		return
	}

	utils.SuccessResponse(c, "Sharing permissions updated successfully", updatedPermissions)
}

// CreateTemporaryShare creates a temporary location share
func (lc *LocationController) CreateTemporaryShare(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var shareRequest models.TemporaryShareRequest
	if err := c.ShouldBindJSON(&shareRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid share request data")
		return
	}

	share, err := lc.locationService.CreateTemporaryShare(c.Request.Context(), userID, shareRequest)
	if err != nil {
		logrus.Errorf("Create temporary share failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to create temporary share")
		return
	}

	utils.CreatedResponse(c, "Temporary share created successfully", share)
}

// GetTemporaryShares gets user's temporary shares
func (lc *LocationController) GetTemporaryShares(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	shares, err := lc.locationService.GetTemporaryShares(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get temporary shares failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get temporary shares")
		return
	}

	utils.SuccessResponse(c, "Temporary shares retrieved successfully", shares)
}

// DeleteTemporaryShare deletes a temporary share
func (lc *LocationController) DeleteTemporaryShare(c *gin.Context) {
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

	err := lc.locationService.DeleteTemporaryShare(c.Request.Context(), userID, shareID)
	if err != nil {
		logrus.Errorf("Delete temporary share failed: %v", err)
		switch err.Error() {
		case "share not found":
			utils.NotFoundResponse(c, "Temporary share")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to delete this share")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete temporary share")
		}
		return
	}

	utils.SuccessResponse(c, "Temporary share deleted successfully", nil)
}

// ==================== PROXIMITY ENDPOINTS ====================

// GetNearbyUsers gets nearby users
func (lc *LocationController) GetNearbyUsers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	radiusStr := c.DefaultQuery("radius", "1000")
	radius, _ := strconv.ParseFloat(radiusStr, 64)

	users, err := lc.locationService.GetNearbyUsers(c.Request.Context(), userID, radius)
	if err != nil {
		logrus.Errorf("Get nearby users failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get nearby users")
		return
	}

	utils.SuccessResponse(c, "Nearby users retrieved successfully", users)
}

// GetNearbyCircleMembers gets nearby circle members
func (lc *LocationController) GetNearbyCircleMembers(c *gin.Context) {
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

	radiusStr := c.DefaultQuery("radius", "1000")
	radius, _ := strconv.ParseFloat(radiusStr, 64)

	members, err := lc.locationService.GetNearbyCircleMembers(c.Request.Context(), userID, circleID, radius)
	if err != nil {
		logrus.Errorf("Get nearby circle members failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get nearby circle members")
		return
	}

	utils.SuccessResponse(c, "Nearby circle members retrieved successfully", members)
}

// CreateProximityAlert creates a proximity alert
func (lc *LocationController) CreateProximityAlert(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var alertRequest models.ProximityAlertRequest
	if err := c.ShouldBindJSON(&alertRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid alert request data")
		return
	}

	alert, err := lc.locationService.CreateProximityAlert(c.Request.Context(), userID, alertRequest)
	if err != nil {
		logrus.Errorf("Create proximity alert failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to create proximity alert")
		return
	}

	utils.CreatedResponse(c, "Proximity alert created successfully", alert)
}

// GetProximityAlerts gets user's proximity alerts
func (lc *LocationController) GetProximityAlerts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	alerts, err := lc.locationService.GetProximityAlerts(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get proximity alerts failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get proximity alerts")
		return
	}

	utils.SuccessResponse(c, "Proximity alerts retrieved successfully", alerts)
}

// UpdateProximityAlert updates a proximity alert
func (lc *LocationController) UpdateProximityAlert(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	alertID := c.Param("alertId")
	if alertID == "" {
		utils.BadRequestResponse(c, "Alert ID is required")
		return
	}

	var alertUpdate models.ProximityAlertUpdate
	if err := c.ShouldBindJSON(&alertUpdate); err != nil {
		utils.BadRequestResponse(c, "Invalid alert update data")
		return
	}

	alert, err := lc.locationService.UpdateProximityAlert(c.Request.Context(), userID, alertID, alertUpdate)
	if err != nil {
		logrus.Errorf("Update proximity alert failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update proximity alert")
		return
	}

	utils.SuccessResponse(c, "Proximity alert updated successfully", alert)
}

// DeleteProximityAlert deletes a proximity alert
func (lc *LocationController) DeleteProximityAlert(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	alertID := c.Param("alertId")
	if alertID == "" {
		utils.BadRequestResponse(c, "Alert ID is required")
		return
	}

	err := lc.locationService.DeleteProximityAlert(c.Request.Context(), userID, alertID)
	if err != nil {
		logrus.Errorf("Delete proximity alert failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to delete proximity alert")
		return
	}

	utils.SuccessResponse(c, "Proximity alert deleted successfully", nil)
}

// ==================== TRIP ENDPOINTS ====================

// GetTrips gets user's trips
func (lc *LocationController) GetTrips(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	trips, err := lc.locationService.GetTrips(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get trips failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get trips")
		return
	}

	utils.SuccessResponse(c, "Trips retrieved successfully", trips)
}

// StartTrip starts a new trip
func (lc *LocationController) StartTrip(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var tripRequest models.StartTripRequest
	if err := c.ShouldBindJSON(&tripRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid trip request data")
		return
	}

	trip, err := lc.locationService.StartTrip(c.Request.Context(), userID, tripRequest)
	if err != nil {
		logrus.Errorf("Start trip failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to start trip")
		return
	}

	utils.CreatedResponse(c, "Trip started successfully", trip)
}

// EndTrip ends a trip
func (lc *LocationController) EndTrip(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	tripID := c.Param("tripId")
	if tripID == "" {
		utils.BadRequestResponse(c, "Trip ID is required")
		return
	}

	trip, err := lc.locationService.EndTrip(c.Request.Context(), userID, tripID)
	if err != nil {
		logrus.Errorf("End trip failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to end trip")
		return
	}

	utils.SuccessResponse(c, "Trip ended successfully", trip)
}

// GetTrip gets a specific trip
func (lc *LocationController) GetTrip(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	tripID := c.Param("tripId")
	if tripID == "" {
		utils.BadRequestResponse(c, "Trip ID is required")
		return
	}

	trip, err := lc.locationService.GetTrip(c.Request.Context(), userID, tripID)
	if err != nil {
		logrus.Errorf("Get trip failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get trip")
		return
	}

	utils.SuccessResponse(c, "Trip retrieved successfully", trip)
}

// UpdateTrip updates a trip
func (lc *LocationController) UpdateTrip(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	tripID := c.Param("tripId")
	if tripID == "" {
		utils.BadRequestResponse(c, "Trip ID is required")
		return
	}

	var tripUpdate models.TripUpdate
	if err := c.ShouldBindJSON(&tripUpdate); err != nil {
		utils.BadRequestResponse(c, "Invalid trip update data")
		return
	}

	trip, err := lc.locationService.UpdateTrip(c.Request.Context(), userID, tripID, tripUpdate)
	if err != nil {
		logrus.Errorf("Update trip failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update trip")
		return
	}

	utils.SuccessResponse(c, "Trip updated successfully", trip)
}

// DeleteTrip deletes a trip
func (lc *LocationController) DeleteTrip(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	tripID := c.Param("tripId")
	if tripID == "" {
		utils.BadRequestResponse(c, "Trip ID is required")
		return
	}

	err := lc.locationService.DeleteTrip(c.Request.Context(), userID, tripID)
	if err != nil {
		logrus.Errorf("Delete trip failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to delete trip")
		return
	}

	utils.SuccessResponse(c, "Trip deleted successfully", nil)
}

// GetTripRoute gets trip route
func (lc *LocationController) GetTripRoute(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	tripID := c.Param("tripId")
	if tripID == "" {
		utils.BadRequestResponse(c, "Trip ID is required")
		return
	}

	route, err := lc.locationService.GetTripRoute(c.Request.Context(), userID, tripID)
	if err != nil {
		logrus.Errorf("Get trip route failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get trip route")
		return
	}

	utils.SuccessResponse(c, "Trip route retrieved successfully", route)
}

// GetTripStats gets trip statistics
func (lc *LocationController) GetTripStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	tripID := c.Param("tripId")
	if tripID == "" {
		utils.BadRequestResponse(c, "Trip ID is required")
		return
	}

	stats, err := lc.locationService.GetTripStats(c.Request.Context(), userID, tripID)
	if err != nil {
		logrus.Errorf("Get trip stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get trip stats")
		return
	}

	utils.SuccessResponse(c, "Trip stats retrieved successfully", stats)
}

// ShareTrip shares a trip
func (lc *LocationController) ShareTrip(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	tripID := c.Param("tripId")
	if tripID == "" {
		utils.BadRequestResponse(c, "Trip ID is required")
		return
	}

	var shareRequest models.ShareTripRequest
	if err := c.ShouldBindJSON(&shareRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid share request data")
		return
	}

	share, err := lc.locationService.ShareTrip(c.Request.Context(), userID, tripID, shareRequest)
	if err != nil {
		logrus.Errorf("Share trip failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to share trip")
		return
	}

	utils.CreatedResponse(c, "Trip shared successfully", share)
}

// ==================== DRIVING ENDPOINTS ====================

// GetDrivingStatus gets current driving status
func (lc *LocationController) GetDrivingStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	status, err := lc.locationService.GetDrivingStatus(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get driving status failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get driving status")
		return
	}

	utils.SuccessResponse(c, "Driving status retrieved successfully", status)
}

// StartDriving starts driving session
func (lc *LocationController) StartDriving(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var startRequest models.StartDrivingRequest
	if err := c.ShouldBindJSON(&startRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid start driving request data")
		return
	}

	session, err := lc.locationService.StartDriving(c.Request.Context(), userID, startRequest)
	if err != nil {
		logrus.Errorf("Start driving failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to start driving session")
		return
	}

	utils.CreatedResponse(c, "Driving session started successfully", session)
}

// StopDriving stops driving session
func (lc *LocationController) StopDriving(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	session, err := lc.locationService.StopDriving(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Stop driving failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to stop driving session")
		return
	}

	utils.SuccessResponse(c, "Driving session stopped successfully", session)
}

// GetDrivingSessions gets driving sessions
func (lc *LocationController) GetDrivingSessions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	sessions, err := lc.locationService.GetDrivingSessions(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get driving sessions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get driving sessions")
		return
	}

	utils.SuccessResponse(c, "Driving sessions retrieved successfully", sessions)
}

// GetDrivingSession gets a specific driving session
func (lc *LocationController) GetDrivingSession(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	sessionID := c.Param("sessionId")
	if sessionID == "" {
		utils.BadRequestResponse(c, "Session ID is required")
		return
	}

	session, err := lc.locationService.GetDrivingSession(c.Request.Context(), userID, sessionID)
	if err != nil {
		logrus.Errorf("Get driving session failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get driving session")
		return
	}

	utils.SuccessResponse(c, "Driving session retrieved successfully", session)
}

// GetDrivingReports gets driving reports
func (lc *LocationController) GetDrivingReports(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	reports, err := lc.locationService.GetDrivingReports(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get driving reports failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get driving reports")
		return
	}

	utils.SuccessResponse(c, "Driving reports retrieved successfully", reports)
}

// GetDrivingReport gets a specific driving report
func (lc *LocationController) GetDrivingReport(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	reportID := c.Param("reportId")
	if reportID == "" {
		utils.BadRequestResponse(c, "Report ID is required")
		return
	}

	report, err := lc.locationService.GetDrivingReport(c.Request.Context(), userID, reportID)
	if err != nil {
		logrus.Errorf("Get driving report failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get driving report")
		return
	}

	utils.SuccessResponse(c, "Driving report retrieved successfully", report)
}

// GetDrivingScore gets driving score
func (lc *LocationController) GetDrivingScore(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.DefaultQuery("period", "week")

	score, err := lc.locationService.GetDrivingScore(c.Request.Context(), userID, period)
	if err != nil {
		logrus.Errorf("Get driving score failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get driving score")
		return
	}

	utils.SuccessResponse(c, "Driving score retrieved successfully", score)
}

// GetDrivingEvents gets driving events
func (lc *LocationController) GetDrivingEvents(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	events, err := lc.locationService.GetDrivingEvents(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get driving events failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get driving events")
		return
	}

	utils.SuccessResponse(c, "Driving events retrieved successfully", events)
}

// ReportDrivingEvent reports a driving event
func (lc *LocationController) ReportDrivingEvent(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var eventRequest models.DrivingEventRequest
	if err := c.ShouldBindJSON(&eventRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid event request data")
		return
	}

	event, err := lc.locationService.ReportDrivingEvent(c.Request.Context(), userID, eventRequest)
	if err != nil {
		logrus.Errorf("Report driving event failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to report driving event")
		return
	}

	utils.CreatedResponse(c, "Driving event reported successfully", event)
}

// ==================== ANALYTICS ENDPOINTS ====================

// GetLocationStats gets location statistics
func (lc *LocationController) GetLocationStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.DefaultQuery("period", "week")

	stats, err := lc.locationService.GetLocationStats(c.Request.Context(), userID, period)
	if err != nil {
		logrus.Errorf("Get location stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location stats")
		return
	}

	utils.SuccessResponse(c, "Location stats retrieved successfully", stats)
}

// GetLocationHeatmap gets location heatmap data
func (lc *LocationController) GetLocationHeatmap(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.DefaultQuery("period", "month")

	heatmap, err := lc.locationService.GetLocationHeatmap(c.Request.Context(), userID, period)
	if err != nil {
		logrus.Errorf("Get location heatmap failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location heatmap")
		return
	}

	utils.SuccessResponse(c, "Location heatmap retrieved successfully", heatmap)
}

// GetLocationPatterns gets location patterns
func (lc *LocationController) GetLocationPatterns(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	patterns, err := lc.locationService.GetLocationPatterns(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get location patterns failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location patterns")
		return
	}

	utils.SuccessResponse(c, "Location patterns retrieved successfully", patterns)
}

// GetLocationInsights gets location insights
func (lc *LocationController) GetLocationInsights(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	insights, err := lc.locationService.GetLocationInsights(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get location insights failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location insights")
		return
	}

	utils.SuccessResponse(c, "Location insights retrieved successfully", insights)
}

// GetLocationTimeline gets location timeline
func (lc *LocationController) GetLocationTimeline(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Parse time parameters
	var startTime, endTime *time.Time
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	timeline, err := lc.locationService.GetLocationTimeline(c.Request.Context(), userID, startTime, endTime)
	if err != nil {
		logrus.Errorf("Get location timeline failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location timeline")
		return
	}

	utils.SuccessResponse(c, "Location timeline retrieved successfully", timeline)
}

// GetLocationSummary gets location summary for a period
func (lc *LocationController) GetLocationSummary(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.Param("period")
	if period == "" {
		utils.BadRequestResponse(c, "Period is required")
		return
	}

	summary, err := lc.locationService.GetLocationSummary(c.Request.Context(), userID, period)
	if err != nil {
		logrus.Errorf("Get location summary failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location summary")
		return
	}

	utils.SuccessResponse(c, "Location summary retrieved successfully", summary)
}

// ==================== GEOFENCING ENDPOINTS ====================

// GetGeofenceEvents gets geofence events
func (lc *LocationController) GetGeofenceEvents(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	events, err := lc.locationService.GetGeofenceEvents(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get geofence events failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get geofence events")
		return
	}

	utils.SuccessResponse(c, "Geofence events retrieved successfully", events)
}

// GetGeofenceEvent gets a specific geofence event
func (lc *LocationController) GetGeofenceEvent(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	eventID := c.Param("eventId")
	if eventID == "" {
		utils.BadRequestResponse(c, "Event ID is required")
		return
	}

	event, err := lc.locationService.GetGeofenceEvent(c.Request.Context(), userID, eventID)
	if err != nil {
		logrus.Errorf("Get geofence event failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get geofence event")
		return
	}

	utils.SuccessResponse(c, "Geofence event retrieved successfully", event)
}

// TestGeofence tests geofence with current location
func (lc *LocationController) TestGeofence(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var testRequest models.GeofenceTestRequest
	if err := c.ShouldBindJSON(&testRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid test request data")
		return
	}

	result, err := lc.locationService.TestGeofence(c.Request.Context(), userID, testRequest)
	if err != nil {
		logrus.Errorf("Test geofence failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to test geofence")
		return
	}

	utils.SuccessResponse(c, "Geofence test completed successfully", result)
}

// GetGeofenceStatus gets geofence status
func (lc *LocationController) GetGeofenceStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	status, err := lc.locationService.GetGeofenceStatus(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get geofence status failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get geofence status")
		return
	}

	utils.SuccessResponse(c, "Geofence status retrieved successfully", status)
}

// ==================== DATA MANAGEMENT ENDPOINTS ====================

// ExportLocationData exports location data
func (lc *LocationController) ExportLocationData(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var exportRequest models.LocationExportRequest
	if err := c.ShouldBindJSON(&exportRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid export request data")
		return
	}

	export, err := lc.locationService.ExportLocationData(c.Request.Context(), userID, exportRequest)
	if err != nil {
		logrus.Errorf("Export location data failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to export location data")
		return
	}

	utils.SuccessResponse(c, "Location data export started successfully", export)
}

// GetExportStatus gets export status
func (lc *LocationController) GetExportStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	exportID := c.Query("exportId")
	if exportID == "" {
		utils.BadRequestResponse(c, "Export ID is required")
		return
	}

	status, err := lc.locationService.GetExportStatus(c.Request.Context(), userID, exportID)
	if err != nil {
		logrus.Errorf("Get export status failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get export status")
		return
	}

	utils.SuccessResponse(c, "Export status retrieved successfully", status)
}

// DownloadLocationExport downloads location export
func (lc *LocationController) DownloadLocationExport(c *gin.Context) {
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

	fileData, fileName, err := lc.locationService.DownloadLocationExport(c.Request.Context(), userID, exportID)
	if err != nil {
		logrus.Errorf("Download location export failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to download location export")
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")
	c.Data(200, "application/octet-stream", fileData)
}

// PurgeLocationData purges old location data
func (lc *LocationController) PurgeLocationData(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var purgeRequest models.LocationPurgeRequest
	if err := c.ShouldBindJSON(&purgeRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid purge request data")
		return
	}

	result, err := lc.locationService.PurgeLocationData(c.Request.Context(), userID, purgeRequest)
	if err != nil {
		logrus.Errorf("Purge location data failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to purge location data")
		return
	}

	utils.SuccessResponse(c, "Location data purged successfully", result)
}

// GetDataUsage gets data usage statistics
func (lc *LocationController) GetDataUsage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	usage, err := lc.locationService.GetDataUsage(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get data usage failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get data usage")
		return
	}

	utils.SuccessResponse(c, "Data usage retrieved successfully", usage)
}

// ==================== EMERGENCY ENDPOINTS ====================

// ShareEmergencyLocation shares location for emergency
func (lc *LocationController) ShareEmergencyLocation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var emergencyRequest models.EmergencyLocationRequest
	if err := c.ShouldBindJSON(&emergencyRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid emergency request data")
		return
	}

	share, err := lc.locationService.ShareEmergencyLocation(c.Request.Context(), userID, emergencyRequest)
	if err != nil {
		logrus.Errorf("Share emergency location failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to share emergency location")
		return
	}

	utils.CreatedResponse(c, "Emergency location shared successfully", share)
}

// GetLastKnownLocation gets last known location
func (lc *LocationController) GetLastKnownLocation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	targetUserID := c.Query("userId")
	if targetUserID == "" {
		targetUserID = userID
	}

	location, err := lc.locationService.GetLastKnownLocation(c.Request.Context(), userID, targetUserID)
	if err != nil {
		logrus.Errorf("Get last known location failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get last known location")
		return
	}

	utils.SuccessResponse(c, "Last known location retrieved successfully", location)
}

// SendLocationPing sends a location ping
func (lc *LocationController) SendLocationPing(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var pingRequest models.LocationPingRequest
	if err := c.ShouldBindJSON(&pingRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid ping request data")
		return
	}

	ping, err := lc.locationService.SendLocationPing(c.Request.Context(), userID, pingRequest)
	if err != nil {
		logrus.Errorf("Send location ping failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to send location ping")
		return
	}

	utils.CreatedResponse(c, "Location ping sent successfully", ping)
}

// GetLocationPing gets a location ping
func (lc *LocationController) GetLocationPing(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	pingID := c.Param("pingId")
	if pingID == "" {
		utils.BadRequestResponse(c, "Ping ID is required")
		return
	}

	ping, err := lc.locationService.GetLocationPing(c.Request.Context(), userID, pingID)
	if err != nil {
		logrus.Errorf("Get location ping failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location ping")
		return
	}

	utils.SuccessResponse(c, "Location ping retrieved successfully", ping)
}

// ==================== CALIBRATION ENDPOINTS ====================

// GetLocationAccuracy gets location accuracy info
func (lc *LocationController) GetLocationAccuracy(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	accuracy, err := lc.locationService.GetLocationAccuracy(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get location accuracy failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location accuracy")
		return
	}

	utils.SuccessResponse(c, "Location accuracy retrieved successfully", accuracy)
}

// CalibrateLocation calibrates location settings
func (lc *LocationController) CalibrateLocation(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var calibrationRequest models.LocationCalibrationRequest
	if err := c.ShouldBindJSON(&calibrationRequest); err != nil {
		utils.BadRequestResponse(c, "Invalid calibration request data")
		return
	}

	result, err := lc.locationService.CalibrateLocation(c.Request.Context(), userID, calibrationRequest)
	if err != nil {
		logrus.Errorf("Calibrate location failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to calibrate location")
		return
	}

	utils.SuccessResponse(c, "Location calibrated successfully", result)
}

// GetLocationProviders gets available location providers
func (lc *LocationController) GetLocationProviders(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	providers, err := lc.locationService.GetLocationProviders(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get location providers failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location providers")
		return
	}

	utils.SuccessResponse(c, "Location providers retrieved successfully", providers)
}

// UpdateLocationProviders updates location provider settings
func (lc *LocationController) UpdateLocationProviders(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var providersUpdate models.LocationProvidersUpdate
	if err := c.ShouldBindJSON(&providersUpdate); err != nil {
		utils.BadRequestResponse(c, "Invalid providers update data")
		return
	}

	providers, err := lc.locationService.UpdateLocationProviders(c.Request.Context(), userID, providersUpdate)
	if err != nil {
		logrus.Errorf("Update location providers failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update location providers")
		return
	}

	utils.SuccessResponse(c, "Location providers updated successfully", providers)
}

// ==================== BATTERY OPTIMIZATION ENDPOINTS ====================

// GetBatteryOptimization gets battery optimization settings
func (lc *LocationController) GetBatteryOptimization(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	optimization, err := lc.locationService.GetBatteryOptimization(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get battery optimization failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get battery optimization")
		return
	}

	utils.SuccessResponse(c, "Battery optimization retrieved successfully", optimization)
}

// UpdateBatteryOptimization updates battery optimization settings
func (lc *LocationController) UpdateBatteryOptimization(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var optimizationUpdate models.BatteryOptimizationUpdate
	if err := c.ShouldBindJSON(&optimizationUpdate); err != nil {
		utils.BadRequestResponse(c, "Invalid optimization update data")
		return
	}

	optimization, err := lc.locationService.UpdateBatteryOptimization(c.Request.Context(), userID, optimizationUpdate)
	if err != nil {
		logrus.Errorf("Update battery optimization failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to update battery optimization")
		return
	}

	utils.SuccessResponse(c, "Battery optimization updated successfully", optimization)
}

// GetBatteryUsage gets battery usage for location tracking
func (lc *LocationController) GetBatteryUsage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	usage, err := lc.locationService.GetBatteryUsage(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get battery usage failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get battery usage")
		return
	}

	utils.SuccessResponse(c, "Battery usage retrieved successfully", usage)
}

// SetPowerMode sets power mode for location tracking
func (lc *LocationController) SetPowerMode(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	mode := c.Param("mode")
	if mode == "" {
		utils.BadRequestResponse(c, "Power mode is required")
		return
	}

	result, err := lc.locationService.SetPowerMode(c.Request.Context(), userID, mode)
	if err != nil {
		logrus.Errorf("Set power mode failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to set power mode")
		return
	}

	utils.SuccessResponse(c, "Power mode set successfully", result)
}
