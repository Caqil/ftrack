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
	// In a production app, you would call Google Maps Geocoding API here
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

	// Broadcast location update to circle members via WebSocket
	if len(circles) > 0 {
		go ls.broadcastLocationUpdate(userID, location, circles)
	}

	// Update user's last seen
	go ls.userRepo.UpdateLastSeen(ctx, userID)

	return &location, nil
}

func (ls *LocationService) GetCurrentLocation(ctx context.Context, requesterID, targetUserID string) (*models.Location, error) {
	// Check if requester has permission to see target user's location
	hasPermission, err := ls.hasLocationPermission(ctx, requesterID, targetUserID)
	if err != nil {
		return nil, err
	}

	if !hasPermission {
		return nil, errors.New("permission denied")
	}

	return ls.locationRepo.GetCurrentLocation(ctx, targetUserID)
}

func (ls *LocationService) GetLocationHistory(ctx context.Context, requesterID string, req models.LocationHistoryRequest) ([]models.Location, error) {
	// Check permission
	hasPermission, err := ls.hasLocationPermission(ctx, requesterID, req.UserID)
	if err != nil {
		return nil, err
	}

	if !hasPermission {
		return nil, errors.New("permission denied")
	}

	// Validate time range
	if req.EndDate.Before(req.StartDate) {
		return nil, errors.New("end time must be after start time")
	}

	if req.Limit <= 0 || req.Limit > 1000 {
		req.Limit = 100
	}

	return ls.locationRepo.GetLocationHistory(ctx, req.UserID, req.StartDate, req.EndDate, req.Limit)
}

func (ls *LocationService) GetCircleLocations(ctx context.Context, userID, circleID string) (map[string]*models.Location, error) {
	// Check if user is a member of the circle
	isMember, err := ls.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// Get circle
	circle, err := ls.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return nil, err
	}

	// Get member user IDs
	var memberIDs []string
	for _, member := range circle.Members {
		if member.Status == "active" {
			memberIDs = append(memberIDs, member.UserID.Hex())
		}
	}

	// Get latest locations for all members
	return ls.locationRepo.GetLatestLocationsForUsers(ctx, memberIDs)
}

func (ls *LocationService) CreatePlace(ctx context.Context, userID string, req models.CreatePlaceRequest) (*models.Place, error) {
	// Validate request
	if validationErrors := ls.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Create place
	place := models.Place{
		UserID:        utils.ObjectIDFromHex(userID),
		Name:          req.Name,
		Address:       req.Address,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		Radius:        req.Radius,
		Category:      req.Category,
		Icon:          req.Icon,
		Color:         req.Color,
		Notifications: req.Notifications,
		Stats: models.PlaceStats{
			VisitCount: 0,
		},
	}

	// Set circle ID if provided
	if req.CircleID != "" {
		circleObjectID := utils.ObjectIDFromHex(req.CircleID)
		place.CircleID = circleObjectID
		place.IsShared = true
	}

	// Set default values
	if place.Radius == 0 {
		place.Radius = 100 // Default 100 meters
	}
	if place.Color == "" {
		place.Color = utils.GenerateRandomColor()
	}

	err := ls.placeRepo.Create(ctx, &place)
	if err != nil {
		return nil, err
	}

	return &place, nil
}

func (ls *LocationService) GetUserPlaces(ctx context.Context, userID string) ([]models.Place, error) {
	return ls.placeRepo.GetUserPlaces(ctx, userID)
}

func (ls *LocationService) DeletePlace(ctx context.Context, userID, placeID string) error {
	// Get place to check ownership
	place, err := ls.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return err
	}

	if place.UserID.Hex() != userID {
		return errors.New("permission denied")
	}

	return ls.placeRepo.Delete(ctx, placeID)
}

func (ls *LocationService) CheckGeofences(ctx context.Context, userID string, lat, lon float64) ([]models.WSPlaceEvent, error) {
	// Get user's places
	places, err := ls.placeRepo.GetUserPlaces(ctx, userID)
	if err != nil {
		return nil, err
	}

	var events []models.WSPlaceEvent
	for _, place := range places {
		distance := utils.CalculateDistance(lat, lon, place.Latitude, place.Longitude)
		if distance <= float64(place.Radius) {
			event := models.WSPlaceEvent{
				UserID:    userID,
				PlaceID:   place.ID.Hex(),
				PlaceName: place.Name,
				EventType: "arrival",
				Timestamp: time.Now(),
			}
			events = append(events, event)
		}
	}

	return events, nil
}

// Helper methods
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

	// Check for common circles
	for _, reqCircle := range requesterCircles {
		for _, targetCircle := range targetCircles {
			if reqCircle.ID == targetCircle.ID {
				return true, nil
			}
		}
	}

	return false, nil
}

func (ls *LocationService) handleGeofenceEvents(ctx context.Context, userID string, prevLocation, newLocation models.Location, circles []models.Circle) {
	// Get all geofences (places) for the user
	places, err := ls.placeRepo.GetUserPlaces(ctx, userID)
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
