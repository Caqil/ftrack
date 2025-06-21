package services

import (
	"context"
	"errors"
	"fmt"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PlaceService struct {
	placeRepo  *repositories.PlaceRepository
	circleRepo *repositories.CircleRepository
	userRepo   *repositories.UserRepository
	validator  *utils.ValidationService
}

func NewPlaceService(
	placeRepo *repositories.PlaceRepository,
	circleRepo *repositories.CircleRepository,
	userRepo *repositories.UserRepository,
) *PlaceService {
	return &PlaceService{
		placeRepo:  placeRepo,
		circleRepo: circleRepo,
		userRepo:   userRepo,
		validator:  utils.NewValidationService(),
	}
}

func (ps *PlaceService) CreatePlace(ctx context.Context, userID string, req models.CreatePlaceRequest) (*models.Place, error) {
	// Validate request
	if validationErrors := ps.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Validate coordinates
	if !utils.IsValidCoordinate(req.Latitude, req.Longitude) {
		return nil, errors.New("invalid coordinates")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Create place
	place := models.Place{
		UserID:        userObjectID,
		Name:          req.Name,
		Address:       req.Address,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		Radius:        req.Radius,
		Category:      req.Category,
		Icon:          req.Icon,
		Color:         req.Color,
		Notifications: req.Notifications,
		Detection: models.PlaceDetection{
			IsAutoDetected: false,
			Confidence:     1.0,
		},
		Stats: models.PlaceStats{
			VisitCount:      0,
			TotalTimeSpent:  0,
			AverageStayTime: 0,
			FavoriteRating:  0.0,
		},
		IsShared: false,
	}

	// Set circle ID if provided and validate membership
	if req.CircleID != "" {
		circleObjectID, err := primitive.ObjectIDFromHex(req.CircleID)
		if err != nil {
			return nil, errors.New("invalid circle ID")
		}

		// Check if user is a member of the circle
		isMember, err := ps.circleRepo.IsMember(ctx, req.CircleID, userID)
		if err != nil {
			return nil, err
		}

		if !isMember {
			return nil, errors.New("not a member of the specified circle")
		}

		// Check permissions
		role, err := ps.circleRepo.GetMemberRole(ctx, req.CircleID, userID)
		if err != nil {
			return nil, err
		}

		// Only admins or members with place management permission can create shared places
		if role != "admin" {
			// Check member permissions (this would need to be implemented in circle repo)
			// For now, allow all members to create places
		}

		place.CircleID = circleObjectID
		place.IsShared = true
	}

	// Set default values
	if place.Radius == 0 {
		place.Radius = 100 // Default 100 meters
	}
	if place.Category == "" {
		place.Category = "other"
	}
	if place.Color == "" {
		place.Color = utils.GenerateRandomColor()
	}
	if place.Icon == "" {
		place.Icon = ps.getDefaultIcon(place.Category)
	}

	// Set default notification settings
	if place.Notifications.ExtendedStayMin == 0 {
		place.Notifications.ExtendedStayMin = 60 // 1 hour
	}

	err = ps.placeRepo.Create(ctx, &place)
	if err != nil {
		return nil, err
	}

	return &place, nil
}

func (ps *PlaceService) GetPlace(ctx context.Context, userID, placeID string) (*models.Place, error) {
	place, err := ps.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return nil, err
	}

	// Check permissions
	hasPermission := place.UserID.Hex() == userID

	// If it's a shared place, check circle membership
	if !hasPermission && place.IsShared && !place.CircleID.IsZero() {
		isMember, err := ps.circleRepo.IsMember(ctx, place.CircleID.Hex(), userID)
		if err == nil && isMember {
			hasPermission = true
		}
	}

	if !hasPermission {
		return nil, errors.New("access denied")
	}

	return place, nil
}

func (ps *PlaceService) GetUserPlaces(ctx context.Context, userID string) ([]models.Place, error) {
	return ps.placeRepo.GetUserPlaces(ctx, userID)
}

func (ps *PlaceService) GetCirclePlaces(ctx context.Context, userID, circleID string) ([]models.Place, error) {
	// Check if user is a member of the circle
	isMember, err := ps.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	return ps.placeRepo.GetCirclePlaces(ctx, circleID)
}

func (ps *PlaceService) UpdatePlace(ctx context.Context, userID, placeID string, req models.UpdatePlaceRequest) (*models.Place, error) {
	// Validate request
	if validationErrors := ps.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Get place to check ownership
	place, err := ps.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return nil, err
	}

	// Check permissions - only owner or circle admin can update
	hasPermission := place.UserID.Hex() == userID

	if !hasPermission && place.IsShared && !place.CircleID.IsZero() {
		role, err := ps.circleRepo.GetMemberRole(ctx, place.CircleID.Hex(), userID)
		if err == nil && role == "admin" {
			hasPermission = true
		}
	}

	if !hasPermission {
		return nil, errors.New("permission denied")
	}

	// Build update document
	update := bson.M{}

	if req.Name != nil {
		update["name"] = *req.Name
	}
	if req.Address != nil {
		update["address"] = *req.Address
	}
	if req.Radius != nil {
		if *req.Radius < 10 || *req.Radius > 5000 {
			return nil, errors.New("radius must be between 10 and 5000 meters")
		}
		update["radius"] = *req.Radius
	}
	if req.Category != nil {
		update["category"] = *req.Category
		if req.Icon == nil {
			// Update icon based on new category
			update["icon"] = ps.getDefaultIcon(*req.Category)
		}
	}
	if req.Icon != nil {
		update["icon"] = *req.Icon
	}
	if req.Color != nil {
		update["color"] = *req.Color
	}
	if req.Notifications != nil {
		update["notifications"] = *req.Notifications
	}

	if len(update) == 0 {
		return nil, errors.New("no fields to update")
	}

	err = ps.placeRepo.Update(ctx, placeID, update)
	if err != nil {
		return nil, err
	}

	return ps.placeRepo.GetByID(ctx, placeID)
}

func (ps *PlaceService) DeletePlace(ctx context.Context, userID, placeID string) error {
	// Get place to check ownership
	place, err := ps.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return err
	}

	// Check permissions - only owner or circle admin can delete
	hasPermission := place.UserID.Hex() == userID

	if !hasPermission && place.IsShared && !place.CircleID.IsZero() {
		role, err := ps.circleRepo.GetMemberRole(ctx, place.CircleID.Hex(), userID)
		if err == nil && role == "admin" {
			hasPermission = true
		}
	}

	if !hasPermission {
		return errors.New("permission denied")
	}

	return ps.placeRepo.Delete(ctx, placeID)
}

func (ps *PlaceService) GetPlacesNearby(ctx context.Context, userID string, lat, lon, radiusM float64) ([]models.Place, error) {
	// Validate coordinates
	if !utils.IsValidCoordinate(lat, lon) {
		return nil, errors.New("invalid coordinates")
	}

	if radiusM <= 0 || radiusM > 50000 { // Max 50km radius
		radiusM = 5000 // Default 5km
	}

	return ps.placeRepo.GetPlacesInRadius(ctx, lat, lon, radiusM)
}

func (ps *PlaceService) GetPlaceVisits(ctx context.Context, userID, placeID string, limit int) ([]models.PlaceVisit, error) {
	// Get place to check permissions
	_, err := ps.GetPlace(ctx, userID, placeID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	return ps.placeRepo.GetPlaceVisits(ctx, placeID, limit)
}

func (ps *PlaceService) StartPlaceVisit(ctx context.Context, userID, placeID string) (*models.PlaceVisit, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	placeObjectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return nil, errors.New("invalid place ID")
	}

	// Check if there's already an active visit
	activeVisit, err := ps.placeRepo.GetActiveVisit(ctx, userID, placeID)
	if err != nil {
		return nil, err
	}

	if activeVisit != nil {
		return nil, errors.New("visit already in progress")
	}

	// Create new visit
	visit := models.PlaceVisit{
		PlaceID:     placeObjectID,
		UserID:      userObjectID,
		ArrivalTime: time.Now(),
		IsOngoing:   true,
	}

	err = ps.placeRepo.CreateVisit(ctx, &visit)
	if err != nil {
		return nil, err
	}

	// Update place stats
	go ps.updatePlaceStats(ctx, placeID, "visit_start")

	return &visit, nil
}

func (ps *PlaceService) EndPlaceVisit(ctx context.Context, userID, placeID string) error {
	// Check for active visit
	activeVisit, err := ps.placeRepo.GetActiveVisit(ctx, userID, placeID)
	if err != nil {
		return err
	}

	if activeVisit == nil {
		return errors.New("no active visit found")
	}

	// End the visit
	departureTime := time.Now()
	duration := int64(departureTime.Sub(activeVisit.ArrivalTime).Seconds())

	update := bson.M{
		"departureTime": departureTime,
		"duration":      duration,
		"isOngoing":     false,
	}

	err = ps.placeRepo.UpdateVisit(ctx, activeVisit.ID.Hex(), update)
	if err != nil {
		return err
	}

	// Update place stats
	go ps.updatePlaceStats(ctx, placeID, "visit_end")

	return nil
}

func (ps *PlaceService) DetectFrequentPlaces(ctx context.Context, userID string) ([]models.Place, error) {
	// This would analyze location history to detect frequently visited places
	// For now, return empty array - this would be implemented with ML algorithms
	return []models.Place{}, nil
}

func (ps *PlaceService) SuggestPlaceCategory(ctx context.Context, lat, lon float64) (string, error) {
	// This would use external APIs (Google Places, Foursquare) to suggest category
	// For now, return default category
	return "other", nil
}

func (ps *PlaceService) SharePlace(ctx context.Context, userID, placeID, circleID string) error {
	// Get place to check ownership
	place, err := ps.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return err
	}

	if place.UserID.Hex() != userID {
		return errors.New("permission denied")
	}

	// Check if user is a member of the circle
	isMember, err := ps.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("not a member of the specified circle")
	}

	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	// Update place to be shared
	update := bson.M{
		"circleId": circleObjectID,
		"isShared": true,
		"sharedBy": place.UserID,
	}

	return ps.placeRepo.Update(ctx, placeID, update)
}

func (ps *PlaceService) UnsharePlace(ctx context.Context, userID, placeID string) error {
	// Get place to check ownership
	place, err := ps.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return err
	}

	// Check permissions - only owner or circle admin can unshare
	hasPermission := place.UserID.Hex() == userID

	if !hasPermission && place.IsShared && !place.CircleID.IsZero() {
		role, err := ps.circleRepo.GetMemberRole(ctx, place.CircleID.Hex(), userID)
		if err == nil && role == "admin" {
			hasPermission = true
		}
	}

	if !hasPermission {
		return errors.New("permission denied")
	}

	// Update place to be private
	update := bson.M{
		"isShared": false,
		"$unset": bson.M{
			"circleId": "",
			"sharedBy": "",
		},
	}

	return ps.placeRepo.Update(ctx, placeID, update)
}

// Helper methods
func (ps *PlaceService) getDefaultIcon(category string) string {
	iconMap := map[string]string{
		"home":       "🏠",
		"work":       "🏢",
		"school":     "🏫",
		"gym":        "💪",
		"restaurant": "🍽️",
		"shopping":   "🛒",
		"hospital":   "🏥",
		"gas":        "⛽",
		"park":       "🌳",
		"airport":    "✈️",
		"hotel":      "🏨",
		"church":     "⛪",
		"bank":       "🏦",
		"library":    "📚",
		"cinema":     "🎬",
		"pharmacy":   "💊",
		"other":      "📍",
	}

	if icon, exists := iconMap[category]; exists {
		return icon
	}
	return "📍"
}

func (ps *PlaceService) updatePlaceStats(ctx context.Context, placeID, eventType string) {
	place, err := ps.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return
	}

	stats := place.Stats

	switch eventType {
	case "visit_start":
		stats.VisitCount++
		stats.LastVisit = time.Now()

		// Determine most visited day
		weekday := time.Now().Weekday().String()
		stats.MostVisitedDay = weekday

		// Determine usual arrival time
		arrivalHour := time.Now().Hour()
		stats.UsualArrivalTime = fmt.Sprintf("%02d:00", arrivalHour)

	case "visit_end":
		// Calculate average stay time
		visits, err := ps.placeRepo.GetPlaceVisits(ctx, placeID, 10)
		if err == nil && len(visits) > 0 {
			var totalDuration int64
			completedVisits := 0

			for _, visit := range visits {
				if !visit.IsOngoing && visit.Duration > 0 {
					totalDuration += visit.Duration
					completedVisits++
				}
			}

			if completedVisits > 0 {
				stats.AverageStayTime = totalDuration / int64(completedVisits)
			}
		}
	}

	// Update stats
	ps.placeRepo.UpdateStats(ctx, placeID, stats)
}
