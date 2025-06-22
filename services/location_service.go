package services

import (
	"context"
	"errors"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"ftrack/websocket"
	"time"

	"github.com/sirupsen/logrus"
)

type LocationService struct {
	locationRepo    *repositories.LocationRepository
	circleRepo      *repositories.CircleRepository
	placeRepo       *repositories.PlaceRepository
	userRepo        *repositories.UserRepository
	geofenceService *GeofenceService
	websocketHub    *websocket.Hub
	validator       *utils.ValidationService
}

func NewLocationService(
	locationRepo *repositories.LocationRepository,
	circleRepo *repositories.CircleRepository,
	placeRepo *repositories.PlaceRepository,
	userRepo *repositories.UserRepository,
	geofenceService *GeofenceService,
	websocketHub *websocket.Hub,
) *LocationService {
	return &LocationService{
		locationRepo:    locationRepo,
		circleRepo:      circleRepo,
		placeRepo:       placeRepo,
		userRepo:        userRepo,
		geofenceService: geofenceService,
		websocketHub:    websocketHub,
		validator:       utils.NewValidationService(),
	}
}

// ==================== TRACKING METHODS ====================

func (ls *LocationService) UpdateLocation(ctx context.Context, userID string, location models.Location) (*models.Location, error) {
	// Validate location
	if !utils.IsValidCoordinate(location.Latitude, location.Longitude) {
		return nil, errors.New("invalid coordinates")
	}

	// Get user's circles
	circles, err := ls.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		logrus.Warn("Failed to get user circles: ", err)
	}

	// Get previous location for geofence checking
	prevLocation, err := ls.locationRepo.GetCurrentLocation(ctx, userID)
	if err != nil {
		logrus.Debug("No previous location found for user: ", userID)
	}

	// Reverse geocoding (get address from coordinates)
	location.Address = ls.getAddressFromCoordinates(location.Latitude, location.Longitude)

	// Save location
	err = ls.locationRepo.Create(ctx, &location)
	if err != nil {
		return nil, err
	}

	// Check geofences and handle place events
	if prevLocation != nil {
		go ls.handleGeofenceEvents(ctx, userID, *prevLocation, location, circles)
	}

	// Broadcast location update via WebSocket
	go ls.broadcastLocationUpdate(userID, location, circles)

	return &location, nil
}

func (ls *LocationService) BulkUpdateLocation(ctx context.Context, userID string, locations []models.Location) (*models.BulkUpdateResult, error) {
	if len(locations) == 0 {
		return nil, errors.New("no locations provided")
	}

	var successful, failed int
	var errors []string

	for _, location := range locations {
		_, err := ls.UpdateLocation(ctx, userID, location)
		if err != nil {
			failed++
			errors = append(errors, err.Error())
		} else {
			successful++
		}
	}

	result := &models.BulkUpdateResult{
		Successful: successful,
		Failed:     failed,
		Errors:     errors,
	}

	return result, nil
}

func (ls *LocationService) GetCurrentLocation(ctx context.Context, requesterID, targetUserID string) (*models.Location, error) {
	// Check permissions
	hasPermission, err := ls.hasLocationPermission(ctx, requesterID, targetUserID)
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, errors.New("access denied")
	}

	location, err := ls.locationRepo.GetCurrentLocation(ctx, targetUserID)
	if err != nil {
		return nil, errors.New("location not found")
	}

	return location, nil
}

func (ls *LocationService) GetLocationHistory(ctx context.Context, requesterID, targetUserID string, startTime, endTime *time.Time, page, pageSize int) (*models.LocationHistoryResponse, error) {
	// Check permissions
	hasPermission, err := ls.hasLocationPermission(ctx, requesterID, targetUserID)
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, errors.New("access denied")
	}

	history, total, err := ls.locationRepo.GetLocationHistory(ctx, targetUserID, startTime, endTime, page, pageSize)
	if err != nil {
		return nil, err
	}

	response := &models.LocationHistoryResponse{
		Locations: history,
		Meta: models.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
		},
	}

	return response, nil
}

func (ls *LocationService) ClearLocationHistory(ctx context.Context, userID string) error {
	return ls.locationRepo.ClearLocationHistory(ctx, userID)
}

// ==================== SHARING METHODS ====================

func (ls *LocationService) GetLocationSettings(ctx context.Context, userID string) (*models.LocationSettings, error) {
	settings, err := ls.locationRepo.GetLocationSettings(ctx, userID)
	if err != nil {
		// Return default settings if none found
		return &models.LocationSettings{
			UserID:          userID,
			Enabled:         true,
			UpdateFrequency: 30,
			Precision:       "exact",
		}, nil
	}
	return settings, nil
}

func (ls *LocationService) UpdateLocationSettings(ctx context.Context, userID string, settings models.LocationSettings) (*models.LocationSettings, error) {
	settings.UserID = userID
	settings.UpdatedAt = time.Now()

	err := ls.locationRepo.UpdateLocationSettings(ctx, userID, settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func (ls *LocationService) GetSharingPermissions(ctx context.Context, userID string) (*models.SharingPermissions, error) {
	permissions, err := ls.locationRepo.GetSharingPermissions(ctx, userID)
	if err != nil {
		// Return default permissions if none found
		return &models.SharingPermissions{
			UserID:  userID,
			Circles: []models.CirclePermission{},
		}, nil
	}
	return permissions, nil
}

func (ls *LocationService) UpdateSharingPermissions(ctx context.Context, userID string, permissions models.SharingPermissions) (*models.SharingPermissions, error) {
	permissions.UserID = userID
	permissions.UpdatedAt = time.Now()

	err := ls.locationRepo.UpdateSharingPermissions(ctx, userID, permissions)
	if err != nil {
		return nil, err
	}

	return &permissions, nil
}

func (ls *LocationService) CreateTemporaryShare(ctx context.Context, userID string, request models.TemporaryShareRequest) (*models.TemporaryShare, error) {
	share := &models.TemporaryShare{
		UserID:    userID,
		ShareCode: utils.GenerateShareCode(),
		Duration:  request.Duration,
		ExpiresAt: time.Now().Add(time.Duration(request.Duration) * time.Second),
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	err := ls.locationRepo.CreateTemporaryShare(ctx, share)
	if err != nil {
		return nil, err
	}

	return share, nil
}

func (ls *LocationService) GetTemporaryShares(ctx context.Context, userID string) ([]models.TemporaryShare, error) {
	return ls.locationRepo.GetTemporaryShares(ctx, userID)
}

func (ls *LocationService) DeleteTemporaryShare(ctx context.Context, userID, shareID string) error {
	// Verify ownership
	share, err := ls.locationRepo.GetTemporaryShare(ctx, shareID)
	if err != nil {
		return errors.New("share not found")
	}
	if share.UserID != userID {
		return errors.New("access denied")
	}

	return ls.locationRepo.DeleteTemporaryShare(ctx, shareID)
}

// ==================== PROXIMITY METHODS ====================

func (ls *LocationService) GetNearbyUsers(ctx context.Context, userID string, radius float64) ([]models.NearbyUser, error) {
	// Get current location
	location, err := ls.locationRepo.GetCurrentLocation(ctx, userID)
	if err != nil {
		return nil, errors.New("current location not found")
	}

	// Get user's circles
	circles, err := ls.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	var circleIDs []string
	for _, circle := range circles {
		circleIDs = append(circleIDs, circle.ID.Hex())
	}

	return ls.locationRepo.GetNearbyUsers(ctx, location.Latitude, location.Longitude, radius, circleIDs)
}

func (ls *LocationService) GetNearbyCircleMembers(ctx context.Context, userID, circleID string, radius float64) ([]models.NearbyUser, error) {
	// Verify circle membership
	isMember, err := ls.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	// Get current location
	location, err := ls.locationRepo.GetCurrentLocation(ctx, userID)
	if err != nil {
		return nil, errors.New("current location not found")
	}

	return ls.locationRepo.GetNearbyCircleMembers(ctx, location.Latitude, location.Longitude, radius, circleID)
}

func (ls *LocationService) CreateProximityAlert(ctx context.Context, userID string, request models.ProximityAlertRequest) (*models.ProximityAlert, error) {
	alert := &models.ProximityAlert{
		UserID:       userID,
		TargetUserID: request.TargetUserID,
		Radius:       request.Radius,
		AlertType:    request.AlertType,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	err := ls.locationRepo.CreateProximityAlert(ctx, alert)
	if err != nil {
		return nil, err
	}

	return alert, nil
}

func (ls *LocationService) GetProximityAlerts(ctx context.Context, userID string) ([]models.ProximityAlert, error) {
	return ls.locationRepo.GetProximityAlerts(ctx, userID)
}

func (ls *LocationService) UpdateProximityAlert(ctx context.Context, userID, alertID string, update models.ProximityAlertUpdate) (*models.ProximityAlert, error) {
	// Verify ownership
	alert, err := ls.locationRepo.GetProximityAlert(ctx, alertID)
	if err != nil {
		return nil, errors.New("alert not found")
	}
	if alert.UserID != userID {
		return nil, errors.New("access denied")
	}

	err = ls.locationRepo.UpdateProximityAlert(ctx, alertID, update)
	if err != nil {
		return nil, err
	}

	return ls.locationRepo.GetProximityAlert(ctx, alertID)
}

func (ls *LocationService) DeleteProximityAlert(ctx context.Context, userID, alertID string) error {
	// Verify ownership
	alert, err := ls.locationRepo.GetProximityAlert(ctx, alertID)
	if err != nil {
		return errors.New("alert not found")
	}
	if alert.UserID != userID {
		return errors.New("access denied")
	}

	return ls.locationRepo.DeleteProximityAlert(ctx, alertID)
}

// ==================== TRIP METHODS ====================

func (ls *LocationService) GetTrips(ctx context.Context, userID string, page, pageSize int) (*models.TripsResponse, error) {
	trips, total, err := ls.locationRepo.GetTrips(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	response := &models.TripsResponse{
		Trips: trips,
		Meta: models.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
		},
	}

	return response, nil
}

func (ls *LocationService) StartTrip(ctx context.Context, userID string, request models.StartTripRequest) (*models.Trip, error) {
	trip := &models.Trip{
		UserID:         userID,
		Name:           request.Name,
		Description:    request.Description,
		Type:           request.Type,
		Transportation: request.Transportation,
		Purpose:        request.Purpose,
		StartTime:      time.Now(),
		IsActive:       true,
		CreatedAt:      time.Now(),
	}

	err := ls.locationRepo.CreateTrip(ctx, trip)
	if err != nil {
		return nil, err
	}

	return trip, nil
}

func (ls *LocationService) EndTrip(ctx context.Context, userID, tripID string) (*models.Trip, error) {
	// Verify ownership
	trip, err := ls.locationRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, errors.New("trip not found")
	}
	if trip.UserID != userID {
		return nil, errors.New("access denied")
	}

	// Calculate trip statistics
	stats, err := ls.calculateTripStats(ctx, tripID)
	if err != nil {
		logrus.Warn("Failed to calculate trip stats: ", err)
	}

	// Update trip
	now := time.Now()
	update := models.TripUpdate{
		EndTime:  &now,
		IsActive: &[]bool{false}[0],
		Stats:    stats,
	}

	err = ls.locationRepo.UpdateTrip(ctx, tripID, update)
	if err != nil {
		return nil, err
	}

	return ls.locationRepo.GetTrip(ctx, tripID)
}

func (ls *LocationService) GetTrip(ctx context.Context, userID, tripID string) (*models.Trip, error) {
	trip, err := ls.locationRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, errors.New("trip not found")
	}
	if trip.UserID != userID {
		return nil, errors.New("access denied")
	}

	return trip, nil
}

func (ls *LocationService) UpdateTrip(ctx context.Context, userID, tripID string, update models.TripUpdate) (*models.Trip, error) {
	// Verify ownership
	trip, err := ls.locationRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, errors.New("trip not found")
	}
	if trip.UserID != userID {
		return nil, errors.New("access denied")
	}

	err = ls.locationRepo.UpdateTrip(ctx, tripID, update)
	if err != nil {
		return nil, err
	}

	return ls.locationRepo.GetTrip(ctx, tripID)
}

func (ls *LocationService) DeleteTrip(ctx context.Context, userID, tripID string) error {
	// Verify ownership
	trip, err := ls.locationRepo.GetTrip(ctx, tripID)
	if err != nil {
		return errors.New("trip not found")
	}
	if trip.UserID != userID {
		return errors.New("access denied")
	}

	return ls.locationRepo.DeleteTrip(ctx, tripID)
}

func (ls *LocationService) GetTripRoute(ctx context.Context, userID, tripID string) (*models.TripRoute, error) {
	// Verify ownership
	trip, err := ls.locationRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, errors.New("trip not found")
	}
	if trip.UserID != userID {
		return nil, errors.New("access denied")
	}

	return ls.locationRepo.GetTripRoute(ctx, tripID)
}

func (ls *LocationService) GetTripStats(ctx context.Context, userID, tripID string) (*models.TripStats, error) {
	// Verify ownership
	trip, err := ls.locationRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, errors.New("trip not found")
	}
	if trip.UserID != userID {
		return nil, errors.New("access denied")
	}

	return ls.calculateTripStats(ctx, tripID)
}

func (ls *LocationService) ShareTrip(ctx context.Context, userID, tripID string, request models.ShareTripRequest) (*models.TripShare, error) {
	// Verify ownership
	trip, err := ls.locationRepo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, errors.New("trip not found")
	}
	if trip.UserID != userID {
		return nil, errors.New("access denied")
	}

	share := &models.TripShare{
		TripID:     tripID,
		UserID:     userID,
		SharedWith: request.SharedWith,
		ShareCode:  utils.GenerateShareCode(),
		ExpiresAt:  time.Now().Add(time.Duration(request.Duration) * time.Second),
		CreatedAt:  time.Now(),
	}

	err = ls.locationRepo.CreateTripShare(ctx, share)
	if err != nil {
		return nil, err
	}

	return share, nil
}

// ==================== DRIVING METHODS ====================

func (ls *LocationService) GetDrivingStatus(ctx context.Context, userID string) (*models.DrivingStatus, error) {
	return ls.locationRepo.GetDrivingStatus(ctx, userID)
}

func (ls *LocationService) StartDriving(ctx context.Context, userID string, request models.StartDrivingRequest) (*models.DrivingSession, error) {
	session := &models.DrivingSession{
		UserID:      userID,
		StartTime:   time.Now(),
		IsActive:    true,
		VehicleType: request.VehicleType,
		CreatedAt:   time.Now(),
	}

	err := ls.locationRepo.CreateDrivingSession(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (ls *LocationService) StopDriving(ctx context.Context, userID string) (*models.DrivingSession, error) {
	session, err := ls.locationRepo.GetActiveDrivingSession(ctx, userID)
	if err != nil {
		return nil, errors.New("no active driving session")
	}

	// Calculate session stats
	stats := ls.calculateDrivingStats(ctx, session.ID.Hex())

	// Update session
	now := time.Now()
	session.EndTime = &now
	session.IsActive = false
	session.Stats = stats

	err = ls.locationRepo.UpdateDrivingSession(ctx, session.ID.Hex(), *session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (ls *LocationService) GetDrivingSessions(ctx context.Context, userID string, page, pageSize int) (*models.DrivingSessionsResponse, error) {
	sessions, total, err := ls.locationRepo.GetDrivingSessions(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	response := &models.DrivingSessionsResponse{
		Sessions: sessions,
		Meta: models.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
		},
	}

	return response, nil
}

func (ls *LocationService) GetDrivingSession(ctx context.Context, userID, sessionID string) (*models.DrivingSession, error) {
	session, err := ls.locationRepo.GetDrivingSession(ctx, sessionID)
	if err != nil {
		return nil, errors.New("session not found")
	}
	if session.UserID != userID {
		return nil, errors.New("access denied")
	}

	return session, nil
}

func (ls *LocationService) GetDrivingReports(ctx context.Context, userID string, page, pageSize int) (*models.DrivingReportsResponse, error) {
	reports, total, err := ls.locationRepo.GetDrivingReports(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	response := &models.DrivingReportsResponse{
		Reports: reports,
		Meta: models.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
		},
	}

	return response, nil
}

func (ls *LocationService) GetDrivingReport(ctx context.Context, userID, reportID string) (*models.DrivingReport, error) {
	report, err := ls.locationRepo.GetDrivingReport(ctx, reportID)
	if err != nil {
		return nil, errors.New("report not found")
	}
	if report.UserID != userID {
		return nil, errors.New("access denied")
	}

	return report, nil
}

func (ls *LocationService) GetDrivingScore(ctx context.Context, userID, period string) (*models.DrivingScore, error) {
	return ls.locationRepo.GetDrivingScore(ctx, userID, period)
}

func (ls *LocationService) GetDrivingEvents(ctx context.Context, userID string, page, pageSize int) (*models.DrivingEventsResponse, error) {
	events, total, err := ls.locationRepo.GetDrivingEvents(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	response := &models.DrivingEventsResponse{
		Events: events,
		Meta: models.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
		},
	}

	return response, nil
}

func (ls *LocationService) ReportDrivingEvent(ctx context.Context, userID string, request models.DrivingEventRequest) (*models.DrivingEvent, error) {
	event := &models.DrivingEvent{
		UserID:    userID,
		EventType: request.EventType,
		Severity:  request.Severity,
		Location:  request.Location,
		Details:   request.Details,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	err := ls.locationRepo.CreateDrivingEvent(ctx, event)
	if err != nil {
		return nil, err
	}

	return event, nil
}

// ==================== ANALYTICS METHODS ====================

func (ls *LocationService) GetLocationStats(ctx context.Context, userID, period string) (*models.LocationStats, error) {
	return ls.locationRepo.GetLocationStats(ctx, userID, period)
}

func (ls *LocationService) GetLocationHeatmap(ctx context.Context, userID, period string) (*models.LocationHeatmap, error) {
	return ls.locationRepo.GetLocationHeatmap(ctx, userID, period)
}

func (ls *LocationService) GetLocationPatterns(ctx context.Context, userID string) (*models.LocationPatterns, error) {
	return ls.locationRepo.GetLocationPatterns(ctx, userID)
}

func (ls *LocationService) GetLocationInsights(ctx context.Context, userID string) (*models.LocationInsights, error) {
	return ls.locationRepo.GetLocationInsights(ctx, userID)
}

func (ls *LocationService) GetLocationTimeline(ctx context.Context, userID string, startTime, endTime *time.Time) (*models.LocationTimeline, error) {
	return ls.locationRepo.GetLocationTimeline(ctx, userID, startTime, endTime)
}

func (ls *LocationService) GetLocationSummary(ctx context.Context, userID, period string) (*models.LocationSummary, error) {
	return ls.locationRepo.GetLocationSummary(ctx, userID, period)
}

// ==================== GEOFENCING METHODS ====================

func (ls *LocationService) GetGeofenceEvents(ctx context.Context, userID string, page, pageSize int) (*models.GeofenceEventsResponse, error) {
	events, total, err := ls.locationRepo.GetGeofenceEvents(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}

	response := &models.GeofenceEventsResponse{
		Events: events,
		Meta: models.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
		},
	}

	return response, nil
}

func (ls *LocationService) GetGeofenceEvent(ctx context.Context, userID, eventID string) (*models.GeofenceEvent, error) {
	event, err := ls.locationRepo.GetGeofenceEvent(ctx, eventID)
	if err != nil {
		return nil, errors.New("event not found")
	}
	if event.UserID != userID {
		return nil, errors.New("access denied")
	}

	return event, nil
}

func (ls *LocationService) TestGeofence(ctx context.Context, userID string, request models.GeofenceTestRequest) (*models.GeofenceTestResult, error) {
	// Get current location
	location, err := ls.locationRepo.GetCurrentLocation(ctx, userID)
	if err != nil {
		return nil, errors.New("current location not found")
	}

	// Test geofence
	isInside := utils.IsWithinGeofence(location.Latitude, location.Longitude, utils.GeofenceCircle{
		Center: utils.Coordinate{
			Latitude:  request.Latitude,
			Longitude: request.Longitude,
		},
		Radius: request.Radius,
	})

	distance := utils.CalculateDistance(location.Latitude, location.Longitude, request.Latitude, request.Longitude)

	result := &models.GeofenceTestResult{
		IsInside: isInside,
		Distance: distance,
		Location: *location,
	}

	return result, nil
}

func (ls *LocationService) GetGeofenceStatus(ctx context.Context, userID string) (*models.GeofenceStatus, error) {
	return ls.locationRepo.GetGeofenceStatus(ctx, userID)
}

// ==================== DATA MANAGEMENT METHODS ====================

func (ls *LocationService) ExportLocationData(ctx context.Context, userID string, request models.LocationExportRequest) (*models.LocationExport, error) {
	export := &models.LocationExport{
		UserID:    userID,
		DataTypes: request.DataTypes,
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Format:    request.Format,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	err := ls.locationRepo.CreateLocationExport(ctx, export)
	if err != nil {
		return nil, err
	}

	// Start background export job
	go ls.processLocationExport(ctx, export.ID.Hex())

	return export, nil
}

func (ls *LocationService) GetExportStatus(ctx context.Context, userID, exportID string) (*models.LocationExport, error) {
	export, err := ls.locationRepo.GetLocationExport(ctx, exportID)
	if err != nil {
		return nil, errors.New("export not found")
	}
	if export.UserID != userID {
		return nil, errors.New("access denied")
	}

	return export, nil
}

func (ls *LocationService) DownloadLocationExport(ctx context.Context, userID, exportID string) ([]byte, string, error) {
	export, err := ls.locationRepo.GetLocationExport(ctx, exportID)
	if err != nil {
		return nil, "", errors.New("export not found")
	}
	if export.UserID != userID {
		return nil, "", errors.New("access denied")
	}
	if export.Status != "completed" {
		return nil, "", errors.New("export not ready")
	}

	// Download file from storage
	data, err := ls.downloadExportFile(export.FileURL)
	if err != nil {
		return nil, "", err
	}

	filename := "location_export_" + exportID + "." + export.Format
	return data, filename, nil
}

func (ls *LocationService) PurgeLocationData(ctx context.Context, userID string, request models.LocationPurgeRequest) (*models.LocationPurgeResult, error) {
	result, err := ls.locationRepo.PurgeLocationData(ctx, userID, request)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (ls *LocationService) GetDataUsage(ctx context.Context, userID string) (*models.DataUsage, error) {
	return ls.locationRepo.GetDataUsage(ctx, userID)
}

// ==================== EMERGENCY METHODS ====================

func (ls *LocationService) ShareEmergencyLocation(ctx context.Context, userID string, request models.EmergencyLocationRequest) (*models.EmergencyLocationShare, error) {
	share := &models.EmergencyLocationShare{
		UserID:        userID,
		ShareCode:     utils.GenerateEmergencyCode(),
		EmergencyType: request.EmergencyType,
		Duration:      request.Duration,
		SharedWith:    request.SharedWith,
		ExpiresAt:     time.Now().Add(time.Duration(request.Duration) * time.Second),
		IsActive:      true,
		CreatedAt:     time.Now(),
	}

	err := ls.locationRepo.CreateEmergencyLocationShare(ctx, share)
	if err != nil {
		return nil, err
	}

	return share, nil
}

func (ls *LocationService) GetLastKnownLocation(ctx context.Context, requesterID, targetUserID string) (*models.Location, error) {
	// Check permissions or emergency access
	hasPermission, err := ls.hasLocationPermission(ctx, requesterID, targetUserID)
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, errors.New("access denied")
	}

	return ls.locationRepo.GetLastKnownLocation(ctx, targetUserID)
}

func (ls *LocationService) SendLocationPing(ctx context.Context, userID string, request models.LocationPingRequest) (*models.LocationPing, error) {
	// Get current location
	location, err := ls.locationRepo.GetCurrentLocation(ctx, userID)
	if err != nil {
		return nil, errors.New("current location not found")
	}

	ping := &models.LocationPing{
		UserID:       userID,
		TargetUserID: request.TargetUserID,
		Location:     *location,
		Message:      request.Message,
		ExpiresAt:    time.Now().Add(time.Duration(request.Duration) * time.Second),
		CreatedAt:    time.Now(),
	}

	err = ls.locationRepo.CreateLocationPing(ctx, ping)
	if err != nil {
		return nil, err
	}

	return ping, nil
}

func (ls *LocationService) GetLocationPing(ctx context.Context, userID, pingID string) (*models.LocationPing, error) {
	ping, err := ls.locationRepo.GetLocationPing(ctx, pingID)
	if err != nil {
		return nil, errors.New("ping not found")
	}
	if ping.UserID != userID && ping.TargetUserID != userID {
		return nil, errors.New("access denied")
	}

	return ping, nil
}

// ==================== CALIBRATION METHODS ====================

func (ls *LocationService) GetLocationAccuracy(ctx context.Context, userID string) (*models.LocationAccuracy, error) {
	return ls.locationRepo.GetLocationAccuracy(ctx, userID)
}

func (ls *LocationService) CalibrateLocation(ctx context.Context, userID string, request models.LocationCalibrationRequest) (*models.LocationCalibrationResult, error) {
	result := &models.LocationCalibrationResult{
		UserID:       userID,
		Accuracy:     request.AccuracyTarget,
		Provider:     request.Provider,
		Success:      true,
		Improvements: []string{"GPS accuracy improved", "Network positioning optimized"},
		CreatedAt:    time.Now(),
	}

	err := ls.locationRepo.SaveCalibrationResult(ctx, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (ls *LocationService) GetLocationProviders(ctx context.Context, userID string) (*models.LocationProviders, error) {
	return ls.locationRepo.GetLocationProviders(ctx, userID)
}

func (ls *LocationService) UpdateLocationProviders(ctx context.Context, userID string, update models.LocationProvidersUpdate) (*models.LocationProviders, error) {
	err := ls.locationRepo.UpdateLocationProviders(ctx, userID, update)
	if err != nil {
		return nil, err
	}

	return ls.locationRepo.GetLocationProviders(ctx, userID)
}

// ==================== BATTERY OPTIMIZATION METHODS ====================

func (ls *LocationService) GetBatteryOptimization(ctx context.Context, userID string) (*models.BatteryOptimization, error) {
	return ls.locationRepo.GetBatteryOptimization(ctx, userID)
}

func (ls *LocationService) UpdateBatteryOptimization(ctx context.Context, userID string, update models.BatteryOptimizationUpdate) (*models.BatteryOptimization, error) {
	err := ls.locationRepo.UpdateBatteryOptimization(ctx, userID, update)
	if err != nil {
		return nil, err
	}

	return ls.locationRepo.GetBatteryOptimization(ctx, userID)
}

func (ls *LocationService) GetBatteryUsage(ctx context.Context, userID string) (*models.BatteryUsage, error) {
	return ls.locationRepo.GetBatteryUsage(ctx, userID)
}

func (ls *LocationService) SetPowerMode(ctx context.Context, userID, mode string) (*models.PowerModeResult, error) {
	result := &models.PowerModeResult{
		UserID:    userID,
		Mode:      mode,
		Applied:   true,
		Changes:   []string{"Update frequency adjusted", "GPS accuracy modified"},
		CreatedAt: time.Now(),
	}

	err := ls.locationRepo.SetPowerMode(ctx, userID, mode)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ==================== HELPER METHODS ====================

func (ls *LocationService) hasLocationPermission(ctx context.Context, requesterID, targetUserID string) (bool, error) {
	// User can always see their own location
	if requesterID == targetUserID {
		return true, nil
	}

	// Check if users are in the same circle
	requesterCircles, err := ls.circleRepo.GetUserCircles(ctx, requesterID)
	if err != nil {
		return false, err
	}

	targetCircles, err := ls.circleRepo.GetUserCircles(ctx, targetUserID)
	if err != nil {
		return false, err
	}

	// Check for common circles with location sharing enabled
	for _, reqCircle := range requesterCircles {
		for _, targetCircle := range targetCircles {
			if reqCircle.ID == targetCircle.ID && targetCircle.Settings.LocationSharing {
				return true, nil
			}
		}
	}

	return false, nil
}

func (ls *LocationService) handleGeofenceEvents(ctx context.Context, userID string, prevLocation, newLocation models.Location, circles []models.Circle) {
	// Get all geofences (places) for the user
	places, _, err := ls.placeRepo.GetUserPlaces(ctx, userID, models.GetPlacesRequest{})
	if err != nil {
		logrus.Error("Failed to get user places: ", err)
		return
	}

	// Convert places to geofence circles
	var geofences []utils.GeofenceCircle
	for _, place := range places {
		geofences = append(geofences, utils.GeofenceCircle{
			Center: utils.Coordinate{
				Latitude:  place.Latitude,
				Longitude: place.Longitude,
			},
			Radius: float64(place.Radius),
		})
	}

	// Calculate geofence events
	events := utils.CalculateGeofenceEvents(
		prevLocation.Latitude, prevLocation.Longitude,
		newLocation.Latitude, newLocation.Longitude,
		geofences,
	)

	// Process each event
	for _, event := range events {
		if event.GeofenceIndex < len(places) {
			place := places[event.GeofenceIndex]

			// Create place event
			placeEvent := models.WSPlaceEvent{
				UserID:    userID,
				PlaceID:   place.ID.Hex(),
				PlaceName: place.Name,
				EventType: event.EventType,
				Timestamp: time.Now(),
			}

			// Broadcast to circles if notifications enabled
			if (event.EventType == "enter" && place.Notifications.OnArrival) ||
				(event.EventType == "exit" && place.Notifications.OnDeparture) {

				var circleIDs []string
				for _, circle := range circles {
					circleIDs = append(circleIDs, circle.ID.Hex())
				}

				ls.websocketHub.BroadcastPlaceEvent(userID, circleIDs, placeEvent)
			}

			// Handle place visit tracking
			go ls.handlePlaceVisit(ctx, userID, place.ID.Hex(), event.EventType)
		}
	}
}

func (ls *LocationService) handlePlaceVisit(ctx context.Context, userID, placeID, eventType string) {
	if eventType == "enter" {
		// Check for existing active visit
		activeVisit, err := ls.placeRepo.GetActiveVisit(ctx, userID, placeID)
		if err != nil {
			logrus.Error("Failed to check active visit: ", err)
			return
		}

		if activeVisit == nil {
			// Create new visit
			visit := models.PlaceVisit{
				PlaceID:     utils.ObjectIDFromHex(placeID),
				UserID:      utils.ObjectIDFromHex(userID),
				ArrivalTime: time.Now(),
				IsOngoing:   true,
			}

			err = ls.placeRepo.CreateVisit(ctx, &visit)
			if err != nil {
				logrus.Error("Failed to create place visit: ", err)
			}
		}
	} else if eventType == "exit" {
		// End active visit
		activeVisit, err := ls.placeRepo.GetActiveVisit(ctx, userID, placeID)
		if err != nil {
			logrus.Error("Failed to check active visit: ", err)
			return
		}

		if activeVisit != nil {
			departureTime := time.Now()
			duration := int64(departureTime.Sub(activeVisit.ArrivalTime).Seconds())

			err = ls.placeRepo.UpdateVisit(ctx, activeVisit.ID.Hex(), map[string]interface{}{
				"departureTime": departureTime,
				"duration":      duration,
				"isOngoing":     false,
			})

			if err != nil {
				logrus.Error("Failed to update place visit: ", err)
			}
		}
	}
}

func (ls *LocationService) broadcastLocationUpdate(userID string, location models.Location, circles []models.Circle) {
	var circleIDs []string
	for _, circle := range circles {
		if circle.Settings.LocationSharing {
			circleIDs = append(circleIDs, circle.ID.Hex())
		}
	}

	if len(circleIDs) > 0 {
		ls.websocketHub.BroadcastLocationUpdate(userID, circleIDs, location)
	}
}

func (ls *LocationService) getAddressFromCoordinates(lat, lon float64) string {
	// In a production app, you would call Google Maps Geocoding API here
	// For now, return a placeholder
	return "Address not available"
}

func (ls *LocationService) calculateTripStats(ctx context.Context, tripID string) (*models.TripStats, error) {
	// Get trip locations
	locations, err := ls.locationRepo.GetTripLocations(ctx, tripID)
	if err != nil {
		return nil, err
	}

	if len(locations) < 2 {
		return &models.TripStats{}, nil
	}

	var totalDistance float64
	var maxSpeed float64
	var totalTime int64
	var speedSum float64
	var speedCount int

	for i := 1; i < len(locations); i++ {
		prev := locations[i-1]
		curr := locations[i]

		// Calculate distance between points
		distance := utils.CalculateDistance(prev.Latitude, prev.Longitude, curr.Latitude, curr.Longitude)
		totalDistance += distance

		// Calculate time difference
		timeDiff := curr.ServerTime.Sub(prev.ServerTime).Seconds()
		totalTime += int64(timeDiff)

		// Track speed
		if curr.Speed > maxSpeed {
			maxSpeed = curr.Speed
		}
		if curr.Speed > 0 {
			speedSum += curr.Speed
			speedCount++
		}
	}

	avgSpeed := 0.0
	if speedCount > 0 {
		avgSpeed = speedSum / float64(speedCount)
	}

	return &models.TripStats{
		TotalDistance: totalDistance,
		TotalTime:     totalTime,
		MaxSpeed:      maxSpeed,
		AvgSpeed:      avgSpeed,
		StartTime:     locations[0].ServerTime,
		EndTime:       locations[len(locations)-1].ServerTime,
	}, nil
}

func (ls *LocationService) calculateDrivingStats(ctx context.Context, sessionID string) *models.DrivingSessionStats {
	// Placeholder for driving statistics calculation
	return &models.DrivingSessionStats{
		TotalDistance:   0,
		MaxSpeed:        0,
		AverageSpeed:    0,
		TimeMoving:      0,
		TimeStationary:  0,
		BatteryConsumed: 0,
	}
}

func (ls *LocationService) processLocationExport(ctx context.Context, exportID string) {
	// Background job to process location data export
	// This would typically involve:
	// 1. Fetching all requested data
	// 2. Formatting according to the requested format
	// 3. Uploading to storage
	// 4. Updating export status
	logrus.Info("Processing location export: ", exportID)
}

func (ls *LocationService) downloadExportFile(fileURL string) ([]byte, error) {
	// Download file from storage (S3, Google Cloud, etc.)
	// Placeholder implementation
	return []byte("export data"), nil
}
