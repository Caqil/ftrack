package services

import (
	"context"
	"errors"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PlaceService struct {
	placeRepo  *repositories.PlaceRepository
	circleRepo *repositories.CircleRepository
}

func NewPlaceService(placeRepo *repositories.PlaceRepository, circleRepo *repositories.CircleRepository) *PlaceService {
	return &PlaceService{
		placeRepo:  placeRepo,
		circleRepo: circleRepo,
	}
}

// ==================== BASIC OPERATIONS ====================

func (ps *PlaceService) CreatePlace(ctx context.Context, userID string, req models.CreatePlaceRequest) (*models.Place, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Validate coordinates and radius
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		return nil, errors.New("invalid coordinates")
	}
	if req.Radius < 10 || req.Radius > 5000 {
		return nil, errors.New("radius must be between 10 and 5000 meters")
	}

	place := &models.Place{
		UserID:        userObjectID,
		Name:          req.Name,
		Description:   req.Description,
		Address:       req.Address,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		Radius:        req.Radius,
		Category:      req.Category,
		Color:         req.Color,
		Icon:          req.Icon,
		IsPublic:      req.IsPublic,
		IsShared:      req.IsShared,
		IsActive:      true,
		IsFavorite:    false,
		Tags:          req.Tags,
		Priority:      req.Priority,
		Notifications: req.Notifications,
		Hours:         req.Hours,
		Geofence:      req.Geofence,
		Metadata:      req.Metadata,
	}

	// Initialize sharing settings
	place.Sharing = models.PlaceSharing{
		IsPublic:   req.IsPublic,
		SharedWith: []models.PlaceMember{},
		Permissions: models.PlaceSharingPermissions{
			CanView:   true,
			CanEdit:   false,
			CanShare:  false,
			CanDelete: false,
		},
	}

	err = ps.placeRepo.Create(ctx, place)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Place created: %s for user %s", place.Name, userID)
	return place, nil
}
func (ps *PlaceService) GetPlace(ctx context.Context, userID, placeID string) (*models.Place, error) {
	place, err := ps.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return nil, err
	}

	// Check permissions
	if place.UserID.Hex() != userID && !place.IsPublic {
		// Check if user has access through circles
		hasAccess, err := ps.hasPlaceAccess(ctx, userID, place)
		if err != nil || !hasAccess {
			return nil, errors.New("access denied")
		}
	}

	return place, nil
}

func (ps *PlaceService) GetUserPlaces(ctx context.Context, userID string, req models.GetPlacesRequest) (*models.PlacesResponse, error) {
	places, total, err := ps.placeRepo.GetUserPlaces(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	var placeResponses []models.PlaceResponse
	for _, place := range places {
		response := models.PlaceResponse{
			Place: place,
		}

		// Calculate distance if coordinates provided
		if req.Latitude != 0 && req.Longitude != 0 {
			response.Distance = utils.CalculateDistance(
				req.Latitude, req.Longitude,
				place.Latitude, place.Longitude,
			)
		}

		placeResponses = append(placeResponses, response)
	}

	// Calculate pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	return &models.PlacesResponse{
		Places: placeResponses,
		Meta: models.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
		},
	}, nil
}

func (ps *PlaceService) UpdatePlace(ctx context.Context, userID, placeID string, req models.UpdatePlaceRequest) (*models.Place, error) {
	place, err := ps.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if place.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	// Build update map
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.Latitude != nil && req.Longitude != nil {
		// Validate coordinates
		if *req.Latitude < -90 || *req.Latitude > 90 || *req.Longitude < -180 || *req.Longitude > 180 {
			return nil, errors.New("invalid coordinates")
		}
		updates["latitude"] = *req.Latitude
		updates["longitude"] = *req.Longitude
	}
	if req.Radius != nil {
		if *req.Radius < 10 || *req.Radius > 5000 {
			return nil, errors.New("radius must be between 10 and 5000 meters")
		}
		updates["radius"] = *req.Radius
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}
	if req.Icon != nil {
		updates["icon"] = *req.Icon
	}
	if req.IsPublic != nil {
		updates["isPublic"] = *req.IsPublic
	}
	if req.IsShared != nil {
		updates["isShared"] = *req.IsShared
	}
	if req.IsActive != nil {
		updates["isActive"] = *req.IsActive
	}
	if req.IsFavorite != nil {
		updates["isFavorite"] = *req.IsFavorite
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Notifications != nil {
		updates["notifications"] = *req.Notifications
	}
	if req.Hours != nil {
		updates["hours"] = *req.Hours
	}
	if req.Metadata != nil {
		updates["metadata"] = *req.Metadata
	}

	err = ps.placeRepo.Update(ctx, placeID, updates)
	if err != nil {
		return nil, err
	}

	// Return updated place
	return ps.placeRepo.GetByID(ctx, placeID)
}

func (ps *PlaceService) DeletePlace(ctx context.Context, userID, placeID string) error {
	place, err := ps.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return err
	}

	// Check ownership
	if place.UserID.Hex() != userID {
		return errors.New("access denied")
	}

	err = ps.placeRepo.Delete(ctx, placeID)
	if err != nil {
		return err
	}

	logrus.Infof("Place deleted: %s by user %s", place.Name, userID)
	return nil
}

func (ps *PlaceService) SearchPlaces(ctx context.Context, req models.SearchPlacesRequest) (*models.PlaceSearchResponse, error) {
	places, total, err := ps.placeRepo.SearchPlaces(ctx, req)
	if err != nil {
		return nil, err
	}

	var placeResponses []models.PlaceResponse
	for _, place := range places {
		response := models.PlaceResponse{
			Place: place,
		}

		// Calculate distance if coordinates provided
		if req.Latitude != 0 && req.Longitude != 0 {
			response.Distance = utils.CalculateDistance(
				req.Latitude, req.Longitude,
				place.Latitude, place.Longitude,
			)
		}

		placeResponses = append(placeResponses, response)
	}

	// Generate search suggestions (simplified)
	suggestions := ps.generateSearchSuggestions(req.Query)

	return &models.PlaceSearchResponse{
		Places: placeResponses,
		Meta: models.PaginationMeta{
			Page:       req.Page,
			PageSize:   req.PageSize,
			Total:      total,
			TotalPages: int((total + int64(req.PageSize) - 1) / int64(req.PageSize)),
		},
		Suggestions: suggestions,
	}, nil
}

// ==================== CATEGORY OPERATIONS ====================

func (ps *PlaceService) CreateCategory(ctx context.Context, userID, name, description, icon, color string) (*models.PlaceCategory, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	category := &models.PlaceCategory{
		Name:        name,
		Description: description,
		Icon:        icon,
		Color:       color,
		IsDefault:   false,
		UserID:      userObjectID,
	}

	err = ps.placeRepo.CreateCategory(ctx, category)
	if err != nil {
		return nil, err
	}

	return category, nil
}

func (ps *PlaceService) GetCategories(ctx context.Context, userID string) ([]models.PlaceCategory, error) {
	return ps.placeRepo.GetCategories(ctx, userID)
}

// ==================== VISIT OPERATIONS ====================

func (ps *PlaceService) RecordVisit(ctx context.Context, userID, placeID string, notes string, rating int) (*models.PlaceVisit, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	placeObjectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return nil, errors.New("invalid place ID")
	}

	visit := &models.PlaceVisit{
		PlaceID:     placeObjectID,
		UserID:      userObjectID,
		ArrivalTime: time.Now(),
		IsOngoing:   true,
		Notes:       notes,
		Rating:      rating,
	}

	err = ps.placeRepo.CreateVisit(ctx, visit)
	if err != nil {
		return nil, err
	}

	return visit, nil
}

func (ps *PlaceService) GetPlaceVisits(ctx context.Context, userID, placeID string, page, pageSize int) ([]models.PlaceVisit, int64, error) {
	// Check if user has access to this place
	place, err := ps.placeRepo.GetByID(ctx, placeID)
	if err != nil {
		return nil, 0, err
	}

	if place.UserID.Hex() != userID && !place.IsPublic {
		hasAccess, err := ps.hasPlaceAccess(ctx, userID, place)
		if err != nil || !hasAccess {
			return nil, 0, errors.New("access denied")
		}
	}

	return ps.placeRepo.GetPlaceVisits(ctx, placeID, page, pageSize)
}

// ==================== REVIEW OPERATIONS ====================

func (ps *PlaceService) CreateReview(ctx context.Context, userID, placeID string, rating int, title, comment string, isPublic bool) (*models.PlaceReview, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	placeObjectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return nil, errors.New("invalid place ID")
	}

	if rating < 1 || rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	review := &models.PlaceReview{
		PlaceID:  placeObjectID,
		UserID:   userObjectID,
		Rating:   rating,
		Title:    title,
		Comment:  comment,
		IsPublic: isPublic,
	}

	err = ps.placeRepo.CreateReview(ctx, review)
	if err != nil {
		return nil, err
	}

	return review, nil
}

func (ps *PlaceService) GetPlaceReviews(ctx context.Context, placeID string, page, pageSize int) ([]models.PlaceReview, int64, error) {
	return ps.placeRepo.GetPlaceReviews(ctx, placeID, page, pageSize)
}

// ==================== CHECKIN OPERATIONS ====================

func (ps *PlaceService) CheckIn(ctx context.Context, userID, placeID, message string, isPublic bool, location models.Location) (*models.PlaceCheckin, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	placeObjectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return nil, errors.New("invalid place ID")
	}

	checkin := &models.PlaceCheckin{
		PlaceID:  placeObjectID,
		UserID:   userObjectID,
		Message:  message,
		IsPublic: isPublic,
		Location: location,
	}

	err = ps.placeRepo.CreateCheckin(ctx, checkin)
	if err != nil {
		return nil, err
	}

	return checkin, nil
}

func (ps *PlaceService) GetPlaceCheckins(ctx context.Context, placeID string, page, pageSize int) ([]models.PlaceCheckin, int64, error) {
	return ps.placeRepo.GetPlaceCheckins(ctx, placeID, page, pageSize)
}

// ==================== AUTOMATION OPERATIONS ====================

func (ps *PlaceService) CreateAutomationRule(ctx context.Context, userID, placeID, name, ruleType string, conditions []models.RuleCondition, actions []models.RuleAction) (*models.AutomationRule, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var placeObjectID *primitive.ObjectID
	if placeID != "" {
		pID, err := primitive.ObjectIDFromHex(placeID)
		if err != nil {
			return nil, errors.New("invalid place ID")
		}

		// Verify user owns the place
		place, err := ps.placeRepo.GetByID(ctx, placeID)
		if err != nil {
			return nil, err
		}

		if place.UserID.Hex() != userID {
			return nil, errors.New("access denied")
		}
		placeObjectID = &pID
	}

	// Add placeID to conditions and actions that need it
	for i := range conditions {
		if conditions[i].Type == "place" && placeObjectID != nil {
			conditions[i].PlaceID = placeObjectID
		}
	}

	for i := range actions {
		if (actions[i].Type == "place_notification" || actions[i].Type == "place_action") && placeObjectID != nil {
			actions[i].PlaceID = placeObjectID
		}
	}

	rule := &models.AutomationRule{
		UserID:       userObjectID,
		Name:         name,
		Type:         ruleType,
		IsActive:     true,
		Conditions:   conditions,
		Actions:      actions,
		TriggerCount: 0,
	}

	// Set CircleID if it's a place-related rule and place has circle
	if placeObjectID != nil {
		place, _ := ps.placeRepo.GetByID(ctx, placeID)
		if place != nil && !place.CircleID.IsZero() {
			rule.CircleID = &place.CircleID
		}
	}

	err = ps.placeRepo.CreateAutomationRule(ctx, rule)
	if err != nil {
		return nil, err
	}

	return rule, nil
}

func (ps *PlaceService) GetAutomationRules(ctx context.Context, userID, placeID string) ([]models.AutomationRule, error) {
	// If placeID is provided, verify access
	if placeID != "" {
		place, err := ps.placeRepo.GetByID(ctx, placeID)
		if err != nil {
			return nil, err
		}

		if place.UserID.Hex() != userID {
			return nil, errors.New("access denied")
		}
	}

	return ps.placeRepo.GetAutomationRules(ctx, userID, placeID)
}

// ==================== HELPER METHODS ====================

func (ps *PlaceService) hasPlaceAccess(ctx context.Context, userID string, place *models.Place) (bool, error) {
	if place.IsPublic {
		return true, nil
	}

	if place.IsShared && !place.CircleID.IsZero() {
		circle, err := ps.circleRepo.GetByID(ctx, place.CircleID.Hex())
		if err != nil {
			return false, err
		}

		for _, memberID := range circle.Members {
			if memberID.UserID.Hex() == userID {
				return true, nil
			}
		}
	}

	// Check direct sharing
	for _, member := range place.Sharing.SharedWith {
		if member.UserID.Hex() == userID {
			return true, nil
		}
	}

	return false, nil
}

func (ps *PlaceService) ValidateCoordinates(lat, lon float64) error {
	if lat < -90 || lat > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

func (ps *PlaceService) ValidateRadius(radius int) error {
	if radius < 10 || radius > 5000 {
		return errors.New("radius must be between 10 and 5000 meters")
	}
	return nil
}

func (ps *PlaceService) GetPlaceDistance(place *models.Place, lat, lon float64) float64 {
	return utils.CalculateDistance(lat, lon, place.Latitude, place.Longitude)
}

func (ps *PlaceService) IsUserInPlace(place *models.Place, userLat, userLon float64) bool {
	distance := ps.GetPlaceDistance(place, userLat, userLon)
	return distance <= float64(place.Radius)
}

// ==================== GEOFENCE INTEGRATION METHODS ====================

// Add these methods to services/place_service.go for geofence integration

func (ps *PlaceService) CheckGeofenceEntry(ctx context.Context, userID string, lat, lon float64) ([]models.Place, error) {
	_, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Get all user's places within a reasonable search radius (5km)
	places, _, err := ps.placeRepo.GetUserPlaces(ctx, userID, models.GetPlacesRequest{
		Latitude:  lat,
		Longitude: lon,
		Radius:    5000, // 5km search radius
		IsActive:  &[]bool{true}[0],
	})
	if err != nil {
		return nil, err
	}

	var enteredPlaces []models.Place
	for _, place := range places {
		if ps.IsUserInPlace(&place, lat, lon) {
			enteredPlaces = append(enteredPlaces, place)
		}
	}

	return enteredPlaces, nil
}

func (ps *PlaceService) GetNearbyPlaces(ctx context.Context, userID string, lat, lon, radius float64, limit int) ([]models.PlaceResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	req := models.GetPlacesRequest{
		Latitude:  lat,
		Longitude: lon,
		Radius:    radius,
		PageSize:  limit,
		IsActive:  &[]bool{true}[0],
		SortBy:    "distance",
	}

	// Get both user's places and public places
	userPlaces, _, err := ps.placeRepo.GetUserPlaces(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	// Get public places in the area
	publicReq := models.SearchPlacesRequest{
		Latitude:  lat,
		Longitude: lon,
		Radius:    radius,
		PageSize:  limit,
	}
	publicPlaces, _, err := ps.placeRepo.SearchPlaces(ctx, publicReq)
	if err != nil {
		return nil, err
	}

	// Combine and deduplicate
	placeMap := make(map[string]models.Place)
	for _, place := range userPlaces {
		placeMap[place.ID.Hex()] = place
	}
	for _, place := range publicPlaces {
		placeMap[place.ID.Hex()] = place
	}

	var responses []models.PlaceResponse
	for _, place := range placeMap {
		distance := ps.GetPlaceDistance(&place, lat, lon)
		responses = append(responses, models.PlaceResponse{
			Place:    place,
			Distance: distance,
		})
	}

	// Sort by distance
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].Distance < responses[j].Distance
	})

	// Limit results
	if len(responses) > limit {
		responses = responses[:limit]
	}

	return responses, nil
}

// ==================== STATISTICS AND ANALYTICS METHODS ====================

// Add these methods to services/place_service.go

func (ps *PlaceService) GetPlaceStatistics(ctx context.Context, userID, placeID string) (map[string]interface{}, error) {
	place, err := ps.GetPlace(ctx, userID, placeID)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"visitCount":      place.Stats.VisitCount,
		"totalDuration":   place.Stats.TotalDuration,
		"averageDuration": place.Stats.AverageDuration,
		"lastVisit":       place.Stats.LastVisit,
		"reviewCount":     place.Stats.ReviewCount,
		"averageRating":   place.Stats.AverageRating,
		"checkinCount":    place.Stats.CheckinCount,
		"popularTimes":    place.Stats.PopularTimes,
	}

	return stats, nil
}

func (ps *PlaceService) GetUserPlaceStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	req := models.GetPlacesRequest{
		Page:     1,
		PageSize: 1000, // Get all places
	}

	placesResponse, err := ps.GetUserPlaces(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	totalPlaces := len(placesResponse.Places)
	totalVisits := int64(0)
	totalDuration := int64(0)
	favoriteCount := 0
	activeCount := 0

	for _, placeResponse := range placesResponse.Places {
		place := placeResponse.Place
		totalVisits += place.Stats.VisitCount
		totalDuration += place.Stats.TotalDuration

		if place.IsFavorite {
			favoriteCount++
		}
		if place.IsActive {
			activeCount++
		}
	}

	averageDuration := int64(0)
	if totalVisits > 0 {
		averageDuration = totalDuration / totalVisits
	}

	stats := map[string]interface{}{
		"totalPlaces":     totalPlaces,
		"totalVisits":     totalVisits,
		"totalDuration":   totalDuration,
		"averageDuration": averageDuration,
		"favoriteCount":   favoriteCount,
		"activeCount":     activeCount,
	}

	return stats, nil
}
func (ps *PlaceService) generateSearchSuggestions(query string) []string {
	// Simple suggestion generation - in production, this could use ML or more sophisticated algorithms
	suggestions := []string{}

	if strings.Contains(strings.ToLower(query), "coffee") {
		suggestions = append(suggestions, "coffee shop", "cafe", "starbucks")
	}
	if strings.Contains(strings.ToLower(query), "food") {
		suggestions = append(suggestions, "restaurant", "fast food", "dining")
	}
	if strings.Contains(strings.ToLower(query), "gas") {
		suggestions = append(suggestions, "gas station", "fuel", "petrol")
	}

	return suggestions
}
