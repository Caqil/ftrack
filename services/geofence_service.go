package services

import (
	"context"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"ftrack/websocket"

	"github.com/sirupsen/logrus"
)

type GeofenceService struct {
	placeRepo    *repositories.PlaceRepository
	locationRepo *repositories.LocationRepository
	websocketHub *websocket.Hub
}

func NewGeofenceService(
	placeRepo *repositories.PlaceRepository,
	locationRepo *repositories.LocationRepository,
	websocketHub *websocket.Hub,
) *GeofenceService {
	return &GeofenceService{
		placeRepo:    placeRepo,
		locationRepo: locationRepo,
		websocketHub: websocketHub,
	}
}

func (gs *GeofenceService) CheckGeofences(ctx context.Context, userID string, lat, lon float64) ([]models.WSPlaceEvent, error) {
	// Get user places within a reasonable radius (e.g., 5km)
	places, err := gs.placeRepo.GetPlacesInRadius(ctx, lat, lon, 5000)
	if err != nil {
		return nil, err
	}

	var events []models.WSPlaceEvent
	for _, place := range places {
		// Check if user is within this place's geofence
		isInside := utils.IsWithinGeofence(lat, lon, utils.GeofenceCircle{
			Center: utils.Coordinate{
				Latitude:  place.Latitude,
				Longitude: place.Longitude,
			},
			Radius: float64(place.Radius),
		})

		if isInside {
			event := models.WSPlaceEvent{
				UserID:    userID,
				PlaceID:   place.ID.Hex(),
				PlaceName: place.Name,
				EventType: "arrival",
			}
			events = append(events, event)
		}
	}

	return events, nil
}

func (gs *GeofenceService) ProcessLocationUpdate(ctx context.Context, userID string, newLocation models.Location) {
	// Get previous location
	prevLocation, err := gs.locationRepo.GetCurrentLocation(ctx, userID)
	if err != nil {
		logrus.Debug("No previous location for geofence processing")
		return
	}

	// Get user's places
	places, err := gs.placeRepo.GetUserPlaces(ctx, userID)
	if err != nil {
		logrus.Error("Failed to get user places: ", err)
		return
	}

	// Check each place for entry/exit events
	for _, place := range places {
		geofence := utils.GeofenceCircle{
			Center: utils.Coordinate{
				Latitude:  place.Latitude,
				Longitude: place.Longitude,
			},
			Radius: float64(place.Radius),
		}

		wasInside := utils.IsWithinGeofence(prevLocation.Latitude, prevLocation.Longitude, geofence)
		isInside := utils.IsWithinGeofence(newLocation.Latitude, newLocation.Longitude, geofence)

		// Entry event
		if !wasInside && isInside {
			gs.handlePlaceEvent(ctx, userID, place, "arrival")
		}

		// Exit event
		if wasInside && !isInside {
			gs.handlePlaceEvent(ctx, userID, place, "departure")
		}
	}
}

func (gs *GeofenceService) handlePlaceEvent(ctx context.Context, userID string, place models.Place, eventType string) {
	event := models.WSPlaceEvent{
		UserID:    userID,
		PlaceID:   place.ID.Hex(),
		PlaceName: place.Name,
		EventType: eventType,
	}

	// Check if notifications are enabled for this event type
	shouldNotify := (eventType == "arrival" && place.Notifications.OnArrival) ||
		(eventType == "departure" && place.Notifications.OnDeparture)

	if shouldNotify {
		// Broadcast to circle members
		if place.IsShared && !place.CircleID.IsZero() {
			circleIDs := []string{place.CircleID.Hex()}
			gs.websocketHub.BroadcastPlaceEvent(userID, circleIDs, event)
		}
	}

	logrus.Info("Place event: ", event.EventType, " at ", event.PlaceName, " for user ", userID)
}
