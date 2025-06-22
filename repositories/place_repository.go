package repositories

import (
	"context"
	"errors"
	"ftrack/models"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PlaceRepository struct {
	collection           *mongo.Collection
	categoryCollection   *mongo.Collection
	visitCollection      *mongo.Collection
	reviewCollection     *mongo.Collection
	checkinCollection    *mongo.Collection
	collectionCollection *mongo.Collection
	automationCollection *mongo.Collection
	templateCollection   *mongo.Collection
}

func NewPlaceRepository(db *mongo.Database) *PlaceRepository {
	return &PlaceRepository{
		collection:           db.Collection("places"),
		categoryCollection:   db.Collection("place_categories"),
		visitCollection:      db.Collection("place_visits"),
		reviewCollection:     db.Collection("place_reviews"),
		checkinCollection:    db.Collection("place_checkins"),
		collectionCollection: db.Collection("place_collections"),
		automationCollection: db.Collection("automation_rules"),
		templateCollection:   db.Collection("place_templates"),
	}
}

// ==================== PLACE OPERATIONS ====================

func (pr *PlaceRepository) Create(ctx context.Context, place *models.Place) error {
	place.ID = primitive.NewObjectID()
	place.CreatedAt = time.Now()
	place.UpdatedAt = time.Now()

	// Initialize stats and geofence
	place.Stats = models.PlaceStats{
		VisitCount:      0,
		TotalDuration:   0,
		AverageDuration: 0,
		PopularTimes:    make([]int, 24),
		ReviewCount:     0,
		AverageRating:   0,
		CheckinCount:    0,
	}

	if place.Geofence.DwellTime == 0 {
		place.Geofence.DwellTime = 30 // default 30 seconds
	}
	if place.Geofence.ExitDelay == 0 {
		place.Geofence.ExitDelay = 60 // default 60 seconds
	}

	_, err := pr.collection.InsertOne(ctx, place)
	return err
}

func (pr *PlaceRepository) GetByID(ctx context.Context, placeID string) (*models.Place, error) {
	objectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return nil, errors.New("invalid place ID")
	}

	var place models.Place
	err = pr.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&place)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("place not found")
		}
		return nil, err
	}

	return &place, nil
}

func (pr *PlaceRepository) Update(ctx context.Context, placeID string, updates map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return errors.New("invalid place ID")
	}

	updates["updatedAt"] = time.Now()

	_, err = pr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updates},
	)

	return err
}

func (pr *PlaceRepository) Delete(ctx context.Context, placeID string) error {
	objectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return errors.New("invalid place ID")
	}

	result, err := pr.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("place not found")
	}

	return nil
}

func (pr *PlaceRepository) SearchPlaces(ctx context.Context, req models.SearchPlacesRequest) ([]models.Place, int64, error) {
	filter := bson.M{}

	// Text search
	if req.Query != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": req.Query, "$options": "i"}},
			{"description": bson.M{"$regex": req.Query, "$options": "i"}},
			{"address": bson.M{"$regex": req.Query, "$options": "i"}},
			{"tags": bson.M{"$in": []string{req.Query}}},
		}
	}

	// Category filter
	if req.Category != "" {
		filter["category"] = req.Category
	}

	// Geographic filter
	if req.Latitude != 0 && req.Longitude != 0 && req.Radius > 0 {
		filter["location"] = bson.M{
			"$geoWithin": bson.M{
				"$centerSphere": []interface{}{
					[]float64{req.Longitude, req.Latitude},
					req.Radius / 6378100,
				},
			},
		}
	}

	// Tags filter
	if req.Tags != "" {
		tags := strings.Split(req.Tags, ",")
		filter["tags"] = bson.M{"$in": tags}
	}

	// Only show public or accessible places
	filter["$or"] = []bson.M{
		{"isPublic": true},
		{"isActive": true},
	}

	total, err := pr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	skip := (page - 1) * pageSize

	opts := options.Find().
		SetSort(bson.D{{"stats.visitCount", -1}, {"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := pr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var places []models.Place
	err = cursor.All(ctx, &places)
	return places, total, err
}

// ==================== CATEGORY OPERATIONS ====================

func (pr *PlaceRepository) CreateCategory(ctx context.Context, category *models.PlaceCategory) error {
	category.ID = primitive.NewObjectID()
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	_, err := pr.categoryCollection.InsertOne(ctx, category)
	return err
}

func (pr *PlaceRepository) GetCategories(ctx context.Context, userID string) ([]models.PlaceCategory, error) {
	userObjectID, _ := primitive.ObjectIDFromHex(userID)

	filter := bson.M{
		"$or": []bson.M{
			{"isDefault": true},
			{"userId": userObjectID},
		},
	}

	cursor, err := pr.categoryCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var categories []models.PlaceCategory
	err = cursor.All(ctx, &categories)
	return categories, err
}

func (pr *PlaceRepository) UpdateCategory(ctx context.Context, categoryID string, updates map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		return errors.New("invalid category ID")
	}

	updates["updatedAt"] = time.Now()

	_, err = pr.categoryCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updates},
	)

	return err
}

func (pr *PlaceRepository) GetUserPlaces(ctx context.Context, userID string, req models.GetPlacesRequest) ([]models.Place, int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, errors.New("invalid user ID")
	}

	filter := bson.M{"userId": userObjectID}

	// Apply filters
	if req.Category != "" {
		filter["category"] = req.Category
	}
	if req.IsPublic != nil {
		filter["isPublic"] = *req.IsPublic
	}
	if req.IsShared != nil {
		filter["isShared"] = *req.IsShared
	}
	if req.IsActive != nil {
		filter["isActive"] = *req.IsActive
	}
	if req.IsFavorite != nil {
		filter["isFavorite"] = *req.IsFavorite
	}

	// Geographic filter
	if req.Latitude != 0 && req.Longitude != 0 && req.Radius > 0 {
		filter["$geoWithin"] = bson.M{
			"$centerSphere": []interface{}{
				[]float64{req.Longitude, req.Latitude},
				req.Radius / 6378100, // Convert meters to radians
			},
		}
	}

	// Tags filter
	if req.Tags != "" {
		tags := strings.Split(req.Tags, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		filter["tags"] = bson.M{"$in": tags}
	}

	// Count total
	total, err := pr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Setup pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	skip := (page - 1) * pageSize

	// Setup sorting
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "createdAt"
	}
	sortOrder := -1
	if req.SortOrder == "asc" {
		sortOrder = 1
	}

	opts := options.Find().
		SetSort(bson.D{{Key: sortBy, Value: sortOrder}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := pr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var places []models.Place
	err = cursor.All(ctx, &places)
	return places, total, err
}

// GetPlacesInRadius returns all places within a specified radius from a given point
func (pr *PlaceRepository) GetPlacesInRadius(ctx context.Context, lat, lon, radiusM float64) ([]models.Place, error) {
	// MongoDB geospatial query to find places within radius
	filter := bson.M{
		"$and": []bson.M{
			{
				"latitude": bson.M{
					"$gte": lat - (radiusM / 111000), // rough conversion: 1 degree ≈ 111km
					"$lte": lat + (radiusM / 111000),
				},
			},
			{
				"longitude": bson.M{
					"$gte": lon - (radiusM / (111000 * math.Cos(lat*math.Pi/180))),
					"$lte": lon + (radiusM / (111000 * math.Cos(lat*math.Pi/180))),
				},
			},
		},
	}

	cursor, err := pr.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var places []models.Place
	err = cursor.All(ctx, &places)
	if err != nil {
		return nil, err
	}

	// Filter by exact distance using Haversine formula for precision
	var filteredPlaces []models.Place
	for _, place := range places {
		distance := calculateDistance(lat, lon, place.Latitude, place.Longitude)
		if distance <= radiusM {
			filteredPlaces = append(filteredPlaces, place)
		}
	}

	return filteredPlaces, nil
}
func (pr *PlaceRepository) DeleteCategory(ctx context.Context, categoryID string) error {
	objectID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		return errors.New("invalid category ID")
	}

	result, err := pr.categoryCollection.DeleteOne(ctx, bson.M{"_id": objectID, "isDefault": false})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("category not found or cannot delete default category")
	}

	return nil
}

// ==================== VISIT OPERATIONS ====================

func (pr *PlaceRepository) CreateVisit(ctx context.Context, visit *models.PlaceVisit) error {
	visit.ID = primitive.NewObjectID()
	visit.CreatedAt = time.Now()
	visit.UpdatedAt = time.Now()

	_, err := pr.visitCollection.InsertOne(ctx, visit)
	if err != nil {
		return err
	}

	// Update place statistics
	go pr.updatePlaceStatsAfterVisit(ctx, visit.PlaceID.Hex())

	return nil
}

func (pr *PlaceRepository) GetActiveVisit(ctx context.Context, userID, placeID string) (*models.PlaceVisit, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	placeObjectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return nil, errors.New("invalid place ID")
	}

	var visit models.PlaceVisit
	err = pr.visitCollection.FindOne(ctx, bson.M{
		"userId":    userObjectID,
		"placeId":   placeObjectID,
		"isOngoing": true,
	}).Decode(&visit)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &visit, nil
}

func (pr *PlaceRepository) GetPlaceVisits(ctx context.Context, placeID string, page, pageSize int) ([]models.PlaceVisit, int64, error) {
	placeObjectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return nil, 0, errors.New("invalid place ID")
	}

	filter := bson.M{"placeId": placeObjectID}

	total, err := pr.visitCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"arrivalTime", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := pr.visitCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var visits []models.PlaceVisit
	err = cursor.All(ctx, &visits)
	return visits, total, err
}

func (pr *PlaceRepository) UpdateVisit(ctx context.Context, visitID string, updates map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(visitID)
	if err != nil {
		return errors.New("invalid visit ID")
	}

	updates["updatedAt"] = time.Now()

	_, err = pr.visitCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updates},
	)

	return err
}

// ==================== REVIEW OPERATIONS ====================

func (pr *PlaceRepository) CreateReview(ctx context.Context, review *models.PlaceReview) error {
	review.ID = primitive.NewObjectID()
	review.CreatedAt = time.Now()
	review.UpdatedAt = time.Now()

	_, err := pr.reviewCollection.InsertOne(ctx, review)
	if err != nil {
		return err
	}

	// Update place rating statistics
	go pr.updatePlaceRatingStats(ctx, review.PlaceID.Hex())

	return nil
}

func (pr *PlaceRepository) GetPlaceReviews(ctx context.Context, placeID string, page, pageSize int) ([]models.PlaceReview, int64, error) {
	placeObjectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return nil, 0, errors.New("invalid place ID")
	}

	filter := bson.M{"placeId": placeObjectID, "isPublic": true}

	total, err := pr.reviewCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"helpfulCount", -1}, {"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := pr.reviewCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var reviews []models.PlaceReview
	err = cursor.All(ctx, &reviews)
	return reviews, total, err
}

// ==================== CHECKIN OPERATIONS ====================

func (pr *PlaceRepository) CreateCheckin(ctx context.Context, checkin *models.PlaceCheckin) error {
	checkin.ID = primitive.NewObjectID()
	checkin.CreatedAt = time.Now()
	checkin.UpdatedAt = time.Now()

	_, err := pr.checkinCollection.InsertOne(ctx, checkin)
	if err != nil {
		return err
	}

	// Update place checkin statistics
	go pr.updatePlaceCheckinStats(ctx, checkin.PlaceID.Hex())

	return nil
}

func (pr *PlaceRepository) GetPlaceCheckins(ctx context.Context, placeID string, page, pageSize int) ([]models.PlaceCheckin, int64, error) {
	placeObjectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return nil, 0, errors.New("invalid place ID")
	}

	filter := bson.M{"placeId": placeObjectID, "isPublic": true}

	total, err := pr.checkinCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := pr.checkinCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var checkins []models.PlaceCheckin
	err = cursor.All(ctx, &checkins)
	return checkins, total, err
}

// ==================== AUTOMATION OPERATIONS ====================

func (pr *PlaceRepository) CreateAutomationRule(ctx context.Context, rule *models.AutomationRule) error {
	rule.ID = primitive.NewObjectID()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	_, err := pr.automationCollection.InsertOne(ctx, rule)
	return err
}

func (pr *PlaceRepository) GetAutomationRules(ctx context.Context, userID, placeID string) ([]models.AutomationRule, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{"userId": userObjectID}

	// If placeID is provided, filter by place-related conditions
	if placeID != "" {
		placeObjectID, err := primitive.ObjectIDFromHex(placeID)
		if err != nil {
			return nil, errors.New("invalid place ID")
		}

		// Filter rules that have conditions or actions related to this place
		filter["$or"] = []bson.M{
			{"conditions.placeId": placeObjectID},
			{"actions.placeId": placeObjectID},
			{"type": bson.M{"$in": []string{"place_arrival", "place_departure"}}},
		}
	}

	cursor, err := pr.automationCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []models.AutomationRule
	err = cursor.All(ctx, &rules)
	return rules, err
}

// ==================== HELPER METHODS ====================

func (pr *PlaceRepository) updatePlaceStatsAfterVisit(ctx context.Context, placeID string) {
	placeObjectID, _ := primitive.ObjectIDFromHex(placeID)

	// Count total visits
	visitCount, _ := pr.visitCollection.CountDocuments(ctx, bson.M{"placeId": placeObjectID})

	// Calculate average duration
	pipeline := []bson.M{
		{"$match": bson.M{"placeId": placeObjectID, "isOngoing": false}},
		{"$group": bson.M{
			"_id":           nil,
			"avgDuration":   bson.M{"$avg": "$duration"},
			"totalDuration": bson.M{"$sum": "$duration"},
		}},
	}

	cursor, err := pr.visitCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	defer cursor.Close(ctx)

	var result struct {
		AvgDuration   float64 `bson:"avgDuration"`
		TotalDuration int64   `bson:"totalDuration"`
	}

	if cursor.Next(ctx) {
		cursor.Decode(&result)
	}

	// Update place stats
	pr.Update(ctx, placeID, map[string]interface{}{
		"stats.visitCount":      visitCount,
		"stats.averageDuration": int64(result.AvgDuration),
		"stats.totalDuration":   result.TotalDuration,
		"stats.lastVisit":       time.Now(),
	})
}

func (pr *PlaceRepository) updatePlaceRatingStats(ctx context.Context, placeID string) {
	placeObjectID, _ := primitive.ObjectIDFromHex(placeID)

	pipeline := []bson.M{
		{"$match": bson.M{"placeId": placeObjectID}},
		{"$group": bson.M{
			"_id":       nil,
			"avgRating": bson.M{"$avg": "$rating"},
			"count":     bson.M{"$sum": 1},
		}},
	}

	cursor, err := pr.reviewCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	defer cursor.Close(ctx)

	var result struct {
		AvgRating float64 `bson:"avgRating"`
		Count     int     `bson:"count"`
	}

	if cursor.Next(ctx) {
		cursor.Decode(&result)
	}

	pr.Update(ctx, placeID, map[string]interface{}{
		"stats.averageRating": result.AvgRating,
		"stats.reviewCount":   result.Count,
	})
}

func (pr *PlaceRepository) updatePlaceCheckinStats(ctx context.Context, placeID string) {
	placeObjectID, _ := primitive.ObjectIDFromHex(placeID)

	checkinCount, _ := pr.checkinCollection.CountDocuments(ctx, bson.M{"placeId": placeObjectID})

	pr.Update(ctx, placeID, map[string]interface{}{
		"stats.checkinCount": checkinCount,
	})
}

// Helper function to calculate distance between two points using Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusM = 6371000 // Earth radius in meters

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusM * c
}
