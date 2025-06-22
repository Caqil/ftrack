package repositories

import (
	"context"
	"errors"
	"ftrack/models"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LocationRepository struct {
	// Core collections
	collection *mongo.Collection

	// Feature-specific collections
	settingsCollection       *mongo.Collection
	sharingCollection        *mongo.Collection
	tempShareCollection      *mongo.Collection
	proximityAlertCollection *mongo.Collection
	tripCollection           *mongo.Collection
	tripShareCollection      *mongo.Collection
	drivingSessionCollection *mongo.Collection
	drivingEventCollection   *mongo.Collection
	drivingReportCollection  *mongo.Collection
	geofenceEventCollection  *mongo.Collection
	exportCollection         *mongo.Collection
	emergencyShareCollection *mongo.Collection
	pingCollection           *mongo.Collection
	calibrationCollection    *mongo.Collection
	batteryOptCollection     *mongo.Collection
}

func NewLocationRepository(db *mongo.Database) *LocationRepository {
	return &LocationRepository{
		collection:               db.Collection("locations"),
		settingsCollection:       db.Collection("location_settings"),
		sharingCollection:        db.Collection("sharing_permissions"),
		tempShareCollection:      db.Collection("temporary_shares"),
		proximityAlertCollection: db.Collection("proximity_alerts"),
		tripCollection:           db.Collection("trips"),
		tripShareCollection:      db.Collection("trip_shares"),
		drivingSessionCollection: db.Collection("driving_sessions"),
		drivingEventCollection:   db.Collection("driving_events"),
		drivingReportCollection:  db.Collection("driving_reports"),
		geofenceEventCollection:  db.Collection("geofence_events"),
		exportCollection:         db.Collection("location_exports"),
		emergencyShareCollection: db.Collection("emergency_location_shares"),
		pingCollection:           db.Collection("location_pings"),
		calibrationCollection:    db.Collection("location_calibrations"),
		batteryOptCollection:     db.Collection("battery_optimizations"),
	}
}

// ==================== BASIC LOCATION METHODS ====================

func (lr *LocationRepository) Create(ctx context.Context, location *models.Location) error {
	location.ID = primitive.NewObjectID()
	location.CreatedAt = time.Now()
	location.ServerTime = time.Now()

	_, err := lr.collection.InsertOne(ctx, location)
	return err
}

func (lr *LocationRepository) GetCurrentLocation(ctx context.Context, userID string) (*models.Location, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	opts := options.FindOne().SetSort(bson.D{{"createdAt", -1}})
	var location models.Location
	err = lr.collection.FindOne(ctx, bson.M{"userId": objectID}, opts).Decode(&location)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("location not found")
		}
		return nil, err
	}

	return &location, nil
}

func (lr *LocationRepository) DeleteOldLocations(ctx context.Context, olderThan time.Time) (int64, error) {
	filter := bson.M{
		"createdAt": bson.M{"$lt": olderThan},
	}

	result, err := lr.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}
func (lr *LocationRepository) GetLocationHistory(ctx context.Context, userID string, startTime, endTime *time.Time, page, pageSize int) ([]models.Location, int64, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, errors.New("invalid user ID")
	}

	filter := bson.M{"userId": objectID}

	// Add time range filter if provided
	if startTime != nil || endTime != nil {
		timeFilter := bson.M{}
		if startTime != nil {
			timeFilter["$gte"] = *startTime
		}
		if endTime != nil {
			timeFilter["$lte"] = *endTime
		}
		filter["createdAt"] = timeFilter
	}

	// Count total documents
	total, err := lr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate pagination
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := lr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var locations []models.Location
	err = cursor.All(ctx, &locations)
	return locations, total, err
}

func (lr *LocationRepository) ClearLocationHistory(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = lr.collection.DeleteMany(ctx, bson.M{"userId": objectID})
	return err
}

func (lr *LocationRepository) GetLastKnownLocation(ctx context.Context, userID string) (*models.Location, error) {
	return lr.GetCurrentLocation(ctx, userID)
}

// ==================== SETTINGS & SHARING METHODS ====================

func (lr *LocationRepository) GetLocationSettings(ctx context.Context, userID string) (*models.LocationSettings, error) {
	var settings models.LocationSettings
	err := lr.settingsCollection.FindOne(ctx, bson.M{"userId": userID}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("settings not found")
		}
		return nil, err
	}
	return &settings, nil
}

func (lr *LocationRepository) UpdateLocationSettings(ctx context.Context, userID string, settings models.LocationSettings) error {
	settings.UpdatedAt = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err := lr.settingsCollection.ReplaceOne(ctx, bson.M{"userId": userID}, settings, opts)
	return err
}

func (lr *LocationRepository) GetSharingPermissions(ctx context.Context, userID string) (*models.SharingPermissions, error) {
	var permissions models.SharingPermissions
	err := lr.sharingCollection.FindOne(ctx, bson.M{"userId": userID}).Decode(&permissions)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("permissions not found")
		}
		return nil, err
	}
	return &permissions, nil
}

func (lr *LocationRepository) UpdateSharingPermissions(ctx context.Context, userID string, permissions models.SharingPermissions) error {
	permissions.UpdatedAt = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err := lr.sharingCollection.ReplaceOne(ctx, bson.M{"userId": userID}, permissions, opts)
	return err
}

func (lr *LocationRepository) CreateTemporaryShare(ctx context.Context, share *models.TemporaryShare) error {
	share.ID = primitive.NewObjectID()
	share.CreatedAt = time.Now()

	_, err := lr.tempShareCollection.InsertOne(ctx, share)
	return err
}

func (lr *LocationRepository) GetTemporaryShares(ctx context.Context, userID string) ([]models.TemporaryShare, error) {
	filter := bson.M{
		"userId":    userID,
		"isActive":  true,
		"expiresAt": bson.M{"$gt": time.Now()},
	}

	cursor, err := lr.tempShareCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var shares []models.TemporaryShare
	err = cursor.All(ctx, &shares)
	return shares, err
}

func (lr *LocationRepository) GetTemporaryShare(ctx context.Context, shareID string) (*models.TemporaryShare, error) {
	objectID, err := primitive.ObjectIDFromHex(shareID)
	if err != nil {
		return nil, errors.New("invalid share ID")
	}

	var share models.TemporaryShare
	err = lr.tempShareCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&share)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("share not found")
		}
		return nil, err
	}
	return &share, nil
}

func (lr *LocationRepository) DeleteTemporaryShare(ctx context.Context, shareID string) error {
	objectID, err := primitive.ObjectIDFromHex(shareID)
	if err != nil {
		return errors.New("invalid share ID")
	}

	result, err := lr.tempShareCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("share not found")
	}
	return nil
}

// ==================== PROXIMITY METHODS ====================

func (lr *LocationRepository) GetNearbyUsers(ctx context.Context, lat, lon, radius float64, circleIDs []string) ([]models.NearbyUser, error) {
	// Convert circleIDs to ObjectIDs
	var circleObjectIDs []primitive.ObjectID
	for _, id := range circleIDs {
		if objectID, err := primitive.ObjectIDFromHex(id); err == nil {
			circleObjectIDs = append(circleObjectIDs, objectID)
		}
	}

	// Aggregation pipeline to get latest locations for users in specified circles
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"circleId":  bson.M{"$in": circleObjectIDs},
				"createdAt": bson.M{"$gte": time.Now().Add(-time.Hour)}, // Only recent locations
			},
		},
		{
			"$sort": bson.M{"userId": 1, "createdAt": -1},
		},
		{
			"$group": bson.M{
				"_id":    "$userId",
				"latest": bson.M{"$first": "$$ROOT"},
			},
		},
		{
			"$addFields": bson.M{
				"distance": bson.M{
					"$sqrt": bson.M{
						"$add": []bson.M{
							{
								"$pow": []interface{}{
									bson.M{"$subtract": []interface{}{"$latest.latitude", lat}},
									2,
								},
							},
							{
								"$pow": []interface{}{
									bson.M{"$subtract": []interface{}{"$latest.longitude", lon}},
									2,
								},
							},
						},
					},
				},
			},
		},
		{
			"$match": bson.M{
				"distance": bson.M{"$lte": radius / 111000}, // Convert meters to degrees (approximate)
			},
		},
		{
			"$sort": bson.M{"distance": 1},
		},
	}

	cursor, err := lr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}

	// Convert to NearbyUser models
	var nearbyUsers []models.NearbyUser
	for _, result := range results {
		if latest, ok := result["latest"].(bson.M); ok {
			nearbyUser := models.NearbyUser{
				UserID:   latest["userId"].(primitive.ObjectID).Hex(),
				Distance: result["distance"].(float64) * 111000, // Convert back to meters
				// Additional fields would be populated by joining with user collection
			}
			nearbyUsers = append(nearbyUsers, nearbyUser)
		}
	}

	return nearbyUsers, nil
}

func (lr *LocationRepository) GetNearbyCircleMembers(ctx context.Context, lat, lon, radius float64, circleID string) ([]models.NearbyUser, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	return lr.GetNearbyUsers(ctx, lat, lon, radius, []string{circleObjectID.Hex()})
}

func (lr *LocationRepository) CreateProximityAlert(ctx context.Context, alert *models.ProximityAlert) error {
	alert.ID = primitive.NewObjectID()
	alert.CreatedAt = time.Now()

	_, err := lr.proximityAlertCollection.InsertOne(ctx, alert)
	return err
}

func (lr *LocationRepository) GetProximityAlerts(ctx context.Context, userID string) ([]models.ProximityAlert, error) {
	filter := bson.M{
		"userId":   userID,
		"isActive": true,
	}

	cursor, err := lr.proximityAlertCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alerts []models.ProximityAlert
	err = cursor.All(ctx, &alerts)
	return alerts, err
}

func (lr *LocationRepository) GetProximityAlert(ctx context.Context, alertID string) (*models.ProximityAlert, error) {
	objectID, err := primitive.ObjectIDFromHex(alertID)
	if err != nil {
		return nil, errors.New("invalid alert ID")
	}

	var alert models.ProximityAlert
	err = lr.proximityAlertCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&alert)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("alert not found")
		}
		return nil, err
	}
	return &alert, nil
}

func (lr *LocationRepository) UpdateProximityAlert(ctx context.Context, alertID string, update models.ProximityAlertUpdate) error {
	objectID, err := primitive.ObjectIDFromHex(alertID)
	if err != nil {
		return errors.New("invalid alert ID")
	}

	updateDoc := bson.M{}
	if update.Radius != nil {
		updateDoc["radius"] = *update.Radius
	}
	if update.AlertType != nil {
		updateDoc["alertType"] = *update.AlertType
	}
	if update.IsActive != nil {
		updateDoc["isActive"] = *update.IsActive
	}

	if len(updateDoc) == 0 {
		return errors.New("no fields to update")
	}

	result, err := lr.proximityAlertCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updateDoc},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("alert not found")
	}
	return nil
}

func (lr *LocationRepository) DeleteProximityAlert(ctx context.Context, alertID string) error {
	objectID, err := primitive.ObjectIDFromHex(alertID)
	if err != nil {
		return errors.New("invalid alert ID")
	}

	result, err := lr.proximityAlertCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("alert not found")
	}
	return nil
}

// ==================== TRIP METHODS ====================

func (lr *LocationRepository) GetTrips(ctx context.Context, userID string, page, pageSize int) ([]models.Trip, int64, error) {
	filter := bson.M{"userId": userID}

	total, err := lr.tripCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := lr.tripCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var trips []models.Trip
	err = cursor.All(ctx, &trips)
	return trips, total, err
}

func (lr *LocationRepository) CreateTrip(ctx context.Context, trip *models.Trip) error {
	trip.ID = primitive.NewObjectID()
	trip.CreatedAt = time.Now()
	trip.UpdatedAt = time.Now()

	_, err := lr.tripCollection.InsertOne(ctx, trip)
	return err
}

func (lr *LocationRepository) GetTrip(ctx context.Context, tripID string) (*models.Trip, error) {
	objectID, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return nil, errors.New("invalid trip ID")
	}

	var trip models.Trip
	err = lr.tripCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&trip)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("trip not found")
		}
		return nil, err
	}
	return &trip, nil
}

func (lr *LocationRepository) UpdateTrip(ctx context.Context, tripID string, update models.TripUpdate) error {
	objectID, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return errors.New("invalid trip ID")
	}

	updateDoc := bson.M{"updatedAt": time.Now()}

	if update.Name != nil {
		updateDoc["name"] = *update.Name
	}
	if update.Description != nil {
		updateDoc["description"] = *update.Description
	}
	if update.Type != nil {
		updateDoc["type"] = *update.Type
	}
	if update.Transportation != nil {
		updateDoc["transportation"] = *update.Transportation
	}
	if update.Purpose != nil {
		updateDoc["purpose"] = *update.Purpose
	}
	if update.EndTime != nil {
		updateDoc["endTime"] = *update.EndTime
	}
	if update.IsActive != nil {
		updateDoc["isActive"] = *update.IsActive
	}
	if update.Stats != nil {
		updateDoc["stats"] = *update.Stats
	}

	result, err := lr.tripCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updateDoc},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("trip not found")
	}
	return nil
}

func (lr *LocationRepository) DeleteTrip(ctx context.Context, tripID string) error {
	objectID, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return errors.New("invalid trip ID")
	}

	result, err := lr.tripCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("trip not found")
	}
	return nil
}

func (lr *LocationRepository) GetTripRoute(ctx context.Context, tripID string) (*models.TripRoute, error) {
	// Get trip to validate existence
	trip, err := lr.GetTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}

	// Get all locations for this trip
	locations, err := lr.GetTripLocations(ctx, tripID)
	if err != nil {
		return nil, err
	}

	route := &models.TripRoute{
		TripID: tripID,
		Points: locations,
	}

	// Calculate total distance and duration if we have locations
	if len(locations) > 0 {
		route.Duration = int64(trip.EndTime.Sub(trip.StartTime).Seconds())
		// Distance calculation would be done here
	}

	return route, nil
}

func (lr *LocationRepository) GetTripLocations(ctx context.Context, tripID string) ([]models.Location, error) {
	trip, err := lr.GetTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}

	objectID, err := primitive.ObjectIDFromHex(trip.UserID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId": objectID,
		"createdAt": bson.M{
			"$gte": trip.StartTime,
		},
	}

	if trip.EndTime != nil {
		filter["createdAt"].(bson.M)["$lte"] = *trip.EndTime
	}

	opts := options.Find().SetSort(bson.D{{"createdAt", 1}})
	cursor, err := lr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var locations []models.Location
	err = cursor.All(ctx, &locations)
	return locations, err
}

func (lr *LocationRepository) CreateTripShare(ctx context.Context, share *models.TripShare) error {
	share.ID = primitive.NewObjectID()
	share.CreatedAt = time.Now()

	_, err := lr.tripShareCollection.InsertOne(ctx, share)
	return err
}

// ==================== DRIVING METHODS ====================

func (lr *LocationRepository) GetDrivingStatus(ctx context.Context, userID string) (*models.DrivingStatus, error) {
	// Get active driving session
	session, err := lr.GetActiveDrivingSession(ctx, userID)
	if err != nil {
		// No active session, return default status
		return &models.DrivingStatus{
			UserID:     userID,
			IsDriving:  false,
			LastUpdate: time.Now(),
		}, nil
	}

	status := &models.DrivingStatus{
		UserID:     userID,
		IsDriving:  true,
		StartTime:  &session.StartTime,
		LastUpdate: time.Now(),
	}

	return status, nil
}

func (lr *LocationRepository) CreateDrivingSession(ctx context.Context, session *models.DrivingSession) error {
	session.ID = primitive.NewObjectID()
	session.CreatedAt = time.Now()

	_, err := lr.drivingSessionCollection.InsertOne(ctx, session)
	return err
}

func (lr *LocationRepository) GetActiveDrivingSession(ctx context.Context, userID string) (*models.DrivingSession, error) {
	var session models.DrivingSession
	err := lr.drivingSessionCollection.FindOne(ctx, bson.M{
		"userId":   userID,
		"isActive": true,
	}).Decode(&session)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("no active session")
		}
		return nil, err
	}
	return &session, nil
}

func (lr *LocationRepository) UpdateDrivingSession(ctx context.Context, sessionID string, session models.DrivingSession) error {
	objectID, err := primitive.ObjectIDFromHex(sessionID)
	if err != nil {
		return errors.New("invalid session ID")
	}

	result, err := lr.drivingSessionCollection.ReplaceOne(
		ctx,
		bson.M{"_id": objectID},
		session,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("session not found")
	}
	return nil
}

func (lr *LocationRepository) GetDrivingSessions(ctx context.Context, userID string, page, pageSize int) ([]models.DrivingSession, int64, error) {
	filter := bson.M{"userId": userID}

	total, err := lr.drivingSessionCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := lr.drivingSessionCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var sessions []models.DrivingSession
	err = cursor.All(ctx, &sessions)
	return sessions, total, err
}

func (lr *LocationRepository) GetDrivingSession(ctx context.Context, sessionID string) (*models.DrivingSession, error) {
	objectID, err := primitive.ObjectIDFromHex(sessionID)
	if err != nil {
		return nil, errors.New("invalid session ID")
	}

	var session models.DrivingSession
	err = lr.drivingSessionCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("session not found")
		}
		return nil, err
	}
	return &session, nil
}

func (lr *LocationRepository) GetDrivingReports(ctx context.Context, userID string, page, pageSize int) ([]models.DrivingReport, int64, error) {
	filter := bson.M{"userId": userID}

	total, err := lr.drivingReportCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := lr.drivingReportCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var reports []models.DrivingReport
	err = cursor.All(ctx, &reports)
	return reports, total, err
}

func (lr *LocationRepository) GetDrivingReport(ctx context.Context, reportID string) (*models.DrivingReport, error) {
	objectID, err := primitive.ObjectIDFromHex(reportID)
	if err != nil {
		return nil, errors.New("invalid report ID")
	}

	var report models.DrivingReport
	err = lr.drivingReportCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&report)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("report not found")
		}
		return nil, err
	}
	return &report, nil
}

func (lr *LocationRepository) GetDrivingScore(ctx context.Context, userID, period string) (*models.DrivingScore, error) {
	// For now, return a placeholder implementation
	// In production, this would calculate scores based on driving events and patterns
	score := &models.DrivingScore{
		UserID:       userID,
		OverallScore: 85,
		Period:       period,
		StartDate:    time.Now().AddDate(0, 0, -30),
		EndDate:      time.Now(),
		CreatedAt:    time.Now(),
	}
	return score, nil
}

func (lr *LocationRepository) GetDrivingEvents(ctx context.Context, userID string, page, pageSize int) ([]models.DrivingEvent, int64, error) {
	filter := bson.M{"userId": userID}

	total, err := lr.drivingEventCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"timestamp", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := lr.drivingEventCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var events []models.DrivingEvent
	err = cursor.All(ctx, &events)
	return events, total, err
}

func (lr *LocationRepository) CreateDrivingEvent(ctx context.Context, event *models.DrivingEvent) error {
	event.ID = primitive.NewObjectID()
	event.CreatedAt = time.Now()

	_, err := lr.drivingEventCollection.InsertOne(ctx, event)
	return err
}

// ==================== ANALYTICS METHODS ====================

func (lr *LocationRepository) GetLocationStats(ctx context.Context, userID, period string) (*models.LocationStats, error) {
	// Calculate start and end dates based on period
	var startDate time.Time
	endDate := time.Now()

	switch period {
	case "day":
		startDate = endDate.AddDate(0, 0, -1)
	case "week":
		startDate = endDate.AddDate(0, 0, -7)
	case "month":
		startDate = endDate.AddDate(0, -1, 0)
	case "year":
		startDate = endDate.AddDate(-1, 0, 0)
	default:
		startDate = endDate.AddDate(0, 0, -7) // Default to week
	}

	// Get locations for the period
	locations, _, err := lr.GetLocationHistory(ctx, userID, &startDate, &endDate, 1, 10000)
	if err != nil {
		return nil, err
	}

	// Calculate statistics
	var totalDistance float64
	var totalTime int64
	placesVisited := make(map[string]bool)

	for i := 1; i < len(locations); i++ {
		prev := locations[i-1]
		curr := locations[i]

		// Calculate distance (simplified)
		distance := lr.calculateDistance(prev.Latitude, prev.Longitude, curr.Latitude, curr.Longitude)
		totalDistance += distance

		// Calculate time
		timeDiff := curr.ServerTime.Sub(prev.ServerTime).Seconds()
		totalTime += int64(timeDiff)

		// Track unique places
		if curr.Address != "" {
			placesVisited[curr.Address] = true
		}
	}

	stats := &models.LocationStats{
		UserID:        userID,
		Period:        period,
		TotalDistance: totalDistance,
		TotalTime:     totalTime,
		PlacesVisited: len(placesVisited),
	}

	return stats, nil
}

func (lr *LocationRepository) GetLocationHeatmap(ctx context.Context, userID, period string) (*models.LocationHeatmap, error) {
	// Get locations for the period
	var startDate time.Time
	endDate := time.Now()

	switch period {
	case "week":
		startDate = endDate.AddDate(0, 0, -7)
	case "month":
		startDate = endDate.AddDate(0, -1, 0)
	default:
		startDate = endDate.AddDate(0, 0, -30)
	}

	locations, _, err := lr.GetLocationHistory(ctx, userID, &startDate, &endDate, 1, 10000)
	if err != nil {
		return nil, err
	}

	// Generate heatmap points
	pointMap := make(map[string]*models.HeatmapPoint)

	for _, location := range locations {
		// Round coordinates to create grid
		latKey := math.Floor(location.Latitude*1000) / 1000
		lonKey := math.Floor(location.Longitude*1000) / 1000
		key := string(rune(latKey)) + "," + string(rune(lonKey))

		if point, exists := pointMap[key]; exists {
			point.Count++
			point.Weight = math.Min(1.0, float64(point.Count)/100.0)
		} else {
			pointMap[key] = &models.HeatmapPoint{
				Latitude:  latKey,
				Longitude: lonKey,
				Count:     1,
				Weight:    0.01,
			}
		}
	}

	// Convert map to slice
	var points []models.HeatmapPoint
	for _, point := range pointMap {
		points = append(points, *point)
	}

	heatmap := &models.LocationHeatmap{
		UserID:    userID,
		Period:    period,
		Points:    points,
		Generated: time.Now(),
	}

	return heatmap, nil
}

func (lr *LocationRepository) GetLocationPatterns(ctx context.Context, userID string) (*models.LocationPatterns, error) {
	// This would analyze location data to identify patterns
	// For now, return a placeholder
	patterns := &models.LocationPatterns{
		UserID:    userID,
		Generated: time.Now(),
	}
	return patterns, nil
}

func (lr *LocationRepository) GetLocationInsights(ctx context.Context, userID string) (*models.LocationInsights, error) {
	// This would generate insights based on location data
	// For now, return a placeholder
	insights := &models.LocationInsights{
		UserID:    userID,
		Generated: time.Now(),
	}
	return insights, nil
}

func (lr *LocationRepository) GetLocationTimeline(ctx context.Context, userID string, startTime, endTime *time.Time) (*models.LocationTimeline, error) {
	locations, _, err := lr.GetLocationHistory(ctx, userID, startTime, endTime, 1, 1000)
	if err != nil {
		return nil, err
	}

	var events []models.TimelineEvent
	for _, location := range locations {
		event := models.TimelineEvent{
			Type:      "location_update",
			Timestamp: location.ServerTime,
			Location:  location,
		}
		events = append(events, event)
	}

	timeline := &models.LocationTimeline{
		UserID:    userID,
		StartDate: *startTime,
		EndDate:   *endTime,
		Events:    events,
	}

	return timeline, nil
}

func (lr *LocationRepository) GetLocationSummary(ctx context.Context, userID, period string) (*models.LocationSummary, error) {
	stats, err := lr.GetLocationStats(ctx, userID, period)
	if err != nil {
		return nil, err
	}

	summary := &models.LocationSummary{
		UserID: userID,
		Period: period,
		Stats:  *stats,
	}

	return summary, nil
}

// ==================== GEOFENCING METHODS ====================

func (lr *LocationRepository) GetGeofenceEvents(ctx context.Context, userID string, page, pageSize int) ([]models.GeofenceEvent, int64, error) {
	filter := bson.M{"userId": userID}

	total, err := lr.geofenceEventCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"timestamp", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := lr.geofenceEventCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var events []models.GeofenceEvent
	err = cursor.All(ctx, &events)
	return events, total, err
}

func (lr *LocationRepository) GetGeofenceEvent(ctx context.Context, eventID string) (*models.GeofenceEvent, error) {
	objectID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return nil, errors.New("invalid event ID")
	}

	var event models.GeofenceEvent
	err = lr.geofenceEventCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("event not found")
		}
		return nil, err
	}
	return &event, nil
}

func (lr *LocationRepository) GetGeofenceStatus(ctx context.Context, userID string) (*models.GeofenceStatus, error) {
	// Count active geofences for the user
	count, err := lr.geofenceEventCollection.CountDocuments(ctx, bson.M{
		"userId":    userID,
		"createdAt": bson.M{"$gte": time.Now().AddDate(0, 0, -1)},
	})
	if err != nil {
		return nil, err
	}

	status := &models.GeofenceStatus{
		UserID:       userID,
		ActiveFences: int(count),
		LastUpdate:   time.Now(),
		Status:       "active",
	}

	return status, nil
}

// ==================== DATA MANAGEMENT METHODS ====================

func (lr *LocationRepository) CreateLocationExport(ctx context.Context, export *models.LocationExport) error {
	export.ID = primitive.NewObjectID()
	export.CreatedAt = time.Now()
	export.UpdatedAt = time.Now()

	_, err := lr.exportCollection.InsertOne(ctx, export)
	return err
}

func (lr *LocationRepository) GetLocationExport(ctx context.Context, exportID string) (*models.LocationExport, error) {
	objectID, err := primitive.ObjectIDFromHex(exportID)
	if err != nil {
		return nil, errors.New("invalid export ID")
	}

	var export models.LocationExport
	err = lr.exportCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&export)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("export not found")
		}
		return nil, err
	}
	return &export, nil
}

func (lr *LocationRepository) PurgeLocationData(ctx context.Context, userID string, request models.LocationPurgeRequest) (*models.LocationPurgeResult, error) {
	result := &models.LocationPurgeResult{}

	filter := bson.M{"userId": userID}

	// Add date range if specified
	if request.StartDate != nil || request.EndDate != nil {
		dateFilter := bson.M{}
		if request.StartDate != nil {
			dateFilter["$gte"] = *request.StartDate
		}
		if request.EndDate != nil {
			dateFilter["$lte"] = *request.EndDate
		}
		filter["createdAt"] = dateFilter
	}

	// Purge different data types
	for _, dataType := range request.DataTypes {
		switch dataType {
		case "locations":
			deleteResult, err := lr.collection.DeleteMany(ctx, filter)
			if err == nil {
				result.LocationsDeleted = int(deleteResult.DeletedCount)
			}
		case "trips":
			deleteResult, err := lr.tripCollection.DeleteMany(ctx, bson.M{"userId": userID})
			if err == nil {
				result.TripsDeleted = int(deleteResult.DeletedCount)
			}
		case "events":
			deleteResult, err := lr.geofenceEventCollection.DeleteMany(ctx, filter)
			if err == nil {
				result.EventsDeleted = int(deleteResult.DeletedCount)
			}
		}
	}

	return result, nil
}

func (lr *LocationRepository) GetDataUsage(ctx context.Context, userID string) (*models.DataUsage, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Count documents in different collections
	locationCount, _ := lr.collection.CountDocuments(ctx, bson.M{"userId": objectID})
	tripCount, _ := lr.tripCollection.CountDocuments(ctx, bson.M{"userId": userID})
	eventCount, _ := lr.geofenceEventCollection.CountDocuments(ctx, bson.M{"userId": userID})

	usage := &models.DataUsage{
		UserID:        userID,
		LocationCount: locationCount,
		TripCount:     tripCount,
		EventCount:    eventCount,
		LastUpdate:    time.Now(),
		RetentionDays: 365, // Default retention
	}

	return usage, nil
}

// ==================== EMERGENCY METHODS ====================

func (lr *LocationRepository) CreateEmergencyLocationShare(ctx context.Context, share *models.EmergencyLocationShare) error {
	share.ID = primitive.NewObjectID()
	share.CreatedAt = time.Now()

	_, err := lr.emergencyShareCollection.InsertOne(ctx, share)
	return err
}

func (lr *LocationRepository) CreateLocationPing(ctx context.Context, ping *models.LocationPing) error {
	ping.ID = primitive.NewObjectID()
	ping.CreatedAt = time.Now()

	_, err := lr.pingCollection.InsertOne(ctx, ping)
	return err
}

func (lr *LocationRepository) GetLocationPing(ctx context.Context, pingID string) (*models.LocationPing, error) {
	objectID, err := primitive.ObjectIDFromHex(pingID)
	if err != nil {
		return nil, errors.New("invalid ping ID")
	}

	var ping models.LocationPing
	err = lr.pingCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&ping)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("ping not found")
		}
		return nil, err
	}
	return &ping, nil
}

// ==================== CALIBRATION METHODS ====================

func (lr *LocationRepository) GetLocationAccuracy(ctx context.Context, userID string) (*models.LocationAccuracy, error) {
	// Get recent locations to calculate accuracy
	recent, _, err := lr.GetLocationHistory(ctx, userID, nil, nil, 1, 100)
	if err != nil {
		return nil, err
	}

	var avgAccuracy float64
	if len(recent) > 0 {
		var totalAccuracy float64
		for _, location := range recent {
			totalAccuracy += location.Accuracy
		}
		avgAccuracy = totalAccuracy / float64(len(recent))
	}

	accuracy := &models.LocationAccuracy{
		UserID:          userID,
		GPSAccuracy:     avgAccuracy,
		NetworkAccuracy: avgAccuracy * 1.5, // Typically less accurate
		LastCalibration: time.Now(),
		Provider:        "gps",
		Score:           85, // Calculate based on accuracy
	}

	return accuracy, nil
}

func (lr *LocationRepository) SaveCalibrationResult(ctx context.Context, result *models.LocationCalibrationResult) error {
	result.CreatedAt = time.Now()

	_, err := lr.calibrationCollection.InsertOne(ctx, result)
	return err
}

func (lr *LocationRepository) GetLocationProviders(ctx context.Context, userID string) (*models.LocationProviders, error) {
	// Return default provider configuration
	providers := &models.LocationProviders{
		UserID: userID,
		GPS: models.ProviderConfig{
			Enabled:     true,
			MinAccuracy: 10.0,
			MaxAge:      30,
		},
		Network: models.ProviderConfig{
			Enabled:     true,
			MinAccuracy: 100.0,
			MaxAge:      60,
		},
		Passive: models.ProviderConfig{
			Enabled:     false,
			MinAccuracy: 500.0,
			MaxAge:      300,
		},
		Priority:  []string{"gps", "network", "passive"},
		UpdatedAt: time.Now(),
	}

	return providers, nil
}

func (lr *LocationRepository) UpdateLocationProviders(ctx context.Context, userID string, update models.LocationProvidersUpdate) error {
	// For now, just return success
	// In production, this would update the user's provider configuration
	return nil
}

// ==================== BATTERY OPTIMIZATION METHODS ====================

func (lr *LocationRepository) GetBatteryOptimization(ctx context.Context, userID string) (*models.BatteryOptimization, error) {
	var optimization models.BatteryOptimization
	err := lr.batteryOptCollection.FindOne(ctx, bson.M{"userId": userID}).Decode(&optimization)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default optimization settings
			optimization = models.BatteryOptimization{
				UserID:          userID,
				PowerMode:       "balanced",
				UpdateFrequency: 30,
				EstimatedUsage:  5,
				UpdatedAt:       time.Now(),
			}
		} else {
			return nil, err
		}
	}
	return &optimization, nil
}

func (lr *LocationRepository) UpdateBatteryOptimization(ctx context.Context, userID string, update models.BatteryOptimizationUpdate) error {
	updateDoc := bson.M{"updatedAt": time.Now()}

	if update.PowerMode != nil {
		updateDoc["powerMode"] = *update.PowerMode
	}
	if update.UpdateFrequency != nil {
		updateDoc["updateFrequency"] = *update.UpdateFrequency
	}
	if update.GPSSettings != nil {
		updateDoc["gpsSettings"] = *update.GPSSettings
	}
	if update.NetworkSettings != nil {
		updateDoc["networkSettings"] = *update.NetworkSettings
	}
	if update.BackgroundMode != nil {
		updateDoc["backgroundMode"] = *update.BackgroundMode
	}

	opts := options.Update().SetUpsert(true)
	_, err := lr.batteryOptCollection.UpdateOne(
		ctx,
		bson.M{"userId": userID},
		bson.M{"$set": updateDoc},
		opts,
	)
	return err
}

func (lr *LocationRepository) GetBatteryUsage(ctx context.Context, userID string) (*models.BatteryUsage, error) {
	// Calculate battery usage based on location update frequency
	// This is a simplified implementation
	usage := &models.BatteryUsage{
		UserID:      userID,
		LastHour:    2.5,  // percentage
		Last24Hours: 15.0, // percentage
		LastWeek:    75.0, // percentage
	}

	return usage, nil
}

func (lr *LocationRepository) SetPowerMode(ctx context.Context, userID, mode string) error {
	updateDoc := bson.M{
		"powerMode": mode,
		"updatedAt": time.Now(),
	}

	opts := options.Update().SetUpsert(true)
	_, err := lr.batteryOptCollection.UpdateOne(
		ctx,
		bson.M{"userId": userID},
		bson.M{"$set": updateDoc},
		opts,
	)
	return err
}

// ==================== HELPER METHODS ====================

func (lr *LocationRepository) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Haversine formula for calculating distance between two points
	const R = 6371000 // Earth's radius in meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
