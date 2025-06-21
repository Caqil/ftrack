package repositories

import (
	"context"
	"errors"
	"ftrack/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EmergencyRepository struct {
	collection         *mongo.Collection
	settingsCollection *mongo.Collection
}

func NewEmergencyRepository(db *mongo.Database) *EmergencyRepository {
	return &EmergencyRepository{
		collection:         db.Collection("emergencies"),
		settingsCollection: db.Collection("emergency_settings"),
	}
}

func (er *EmergencyRepository) Create(ctx context.Context, emergency *models.Emergency) error {
	emergency.ID = primitive.NewObjectID()
	emergency.CreatedAt = time.Now()
	emergency.UpdatedAt = time.Now()
	emergency.Status = models.EmergencyStatusActive

	// Initialize timeline
	event := models.EmergencyEvent{
		ID:          primitive.NewObjectID(),
		Type:        "created",
		Description: "Emergency alert created",
		Actor:       emergency.UserID,
		Timestamp:   time.Now(),
	}
	emergency.Timeline = []models.EmergencyEvent{event}

	_, err := er.collection.InsertOne(ctx, emergency)
	return err
}

func (er *EmergencyRepository) GetByID(ctx context.Context, id string) (*models.Emergency, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid emergency ID")
	}

	var emergency models.Emergency
	err = er.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&emergency)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("emergency not found")
		}
		return nil, err
	}

	return &emergency, nil
}

func (er *EmergencyRepository) GetUserEmergencies(ctx context.Context, userID string) ([]models.Emergency, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	opts := options.Find().SetSort(bson.D{{"createdAt", -1}})
	cursor, err := er.collection.Find(ctx, bson.M{"userId": objectID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var emergencies []models.Emergency
	err = cursor.All(ctx, &emergencies)
	return emergencies, err
}

func (er *EmergencyRepository) GetActiveEmergencies(ctx context.Context) ([]models.Emergency, error) {
	filter := bson.M{"status": models.EmergencyStatusActive}
	opts := options.Find().SetSort(bson.D{{"createdAt", -1}})

	cursor, err := er.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var emergencies []models.Emergency
	err = cursor.All(ctx, &emergencies)
	return emergencies, err
}

func (er *EmergencyRepository) GetCircleEmergencies(ctx context.Context, circleID string) ([]models.Emergency, error) {
	objectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	filter := bson.M{"circleId": objectID}
	opts := options.Find().SetSort(bson.D{{"createdAt", -1}})

	cursor, err := er.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var emergencies []models.Emergency
	err = cursor.All(ctx, &emergencies)
	return emergencies, err
}

func (er *EmergencyRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid emergency ID")
	}

	update["updatedAt"] = time.Now()

	result, err := er.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("emergency not found")
	}

	return nil
}

func (er *EmergencyRepository) AddTimelineEvent(ctx context.Context, emergencyID string, event models.EmergencyEvent) error {
	objectID, err := primitive.ObjectIDFromHex(emergencyID)
	if err != nil {
		return errors.New("invalid emergency ID")
	}

	event.ID = primitive.NewObjectID()
	event.Timestamp = time.Now()

	result, err := er.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$push": bson.M{"timeline": event},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("emergency not found")
	}

	return nil
}

func (er *EmergencyRepository) ResolveEmergency(ctx context.Context, emergencyID, resolvedBy, resolution string) error {
	objectID, err := primitive.ObjectIDFromHex(emergencyID)
	if err != nil {
		return errors.New("invalid emergency ID")
	}

	resolvedByObjectID, err := primitive.ObjectIDFromHex(resolvedBy)
	if err != nil {
		return errors.New("invalid resolver ID")
	}

	now := time.Now()
	update := bson.M{
		"status":     models.EmergencyStatusResolved,
		"resolvedBy": resolvedByObjectID,
		"resolvedAt": now,
		"resolution": resolution,
		"updatedAt":  now,
	}

	// Add timeline event
	event := models.EmergencyEvent{
		ID:          primitive.NewObjectID(),
		Type:        "resolved",
		Description: "Emergency resolved: " + resolution,
		Actor:       resolvedByObjectID,
		Timestamp:   now,
	}

	result, err := er.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$set":  update,
			"$push": bson.M{"timeline": event},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("emergency not found")
	}

	return nil
}

func (er *EmergencyRepository) AddMedia(ctx context.Context, emergencyID string, media models.EmergencyMedia) error {
	objectID, err := primitive.ObjectIDFromHex(emergencyID)
	if err != nil {
		return errors.New("invalid emergency ID")
	}

	media.ID = primitive.NewObjectID()
	media.UploadedAt = time.Now()

	result, err := er.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$push": bson.M{"media": media},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("emergency not found")
	}

	return nil
}

// Emergency Settings
func (er *EmergencyRepository) GetUserSettings(ctx context.Context, userID string) (*models.EmergencySettings, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var settings models.EmergencySettings
	err = er.settingsCollection.FindOne(ctx, bson.M{"userId": objectID}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default settings
			return &models.EmergencySettings{
				UserID:              objectID,
				CrashDetection:      true,
				FallDetection:       false,
				AutoCallEmergency:   false,
				AutoNotifyContacts:  true,
				CountdownDuration:   30,
				ShareLocationAlways: true,
				UpdatedAt:           time.Now(),
			}, nil
		}
		return nil, err
	}

	return &settings, nil
}

func (er *EmergencyRepository) UpdateUserSettings(ctx context.Context, userID string, settings *models.EmergencySettings) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	settings.UserID = objectID
	settings.UpdatedAt = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err = er.settingsCollection.ReplaceOne(
		ctx,
		bson.M{"userId": objectID},
		settings,
		opts,
	)

	return err
}

func (er *EmergencyRepository) GetEmergencyStats(ctx context.Context, startTime, endTime time.Time) (*models.EmergencyStats, error) {
	// Use aggregation to calculate stats
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"createdAt": bson.M{
					"$gte": startTime,
					"$lte": endTime,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":   nil,
				"total": bson.M{"$sum": 1},
				"active": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$status", models.EmergencyStatusActive}},
							1,
							0,
						},
					},
				},
				"resolved": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$status", models.EmergencyStatusResolved}},
							1,
							0,
						},
					},
				},
				"falseAlarms": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$status", models.EmergencyStatusFalseAlarm}},
							1,
							0,
						},
					},
				},
				"avgResponseTime": bson.M{
					"$avg": bson.M{
						"$cond": []interface{}{
							bson.M{"$ne": []interface{}{"$response.responseTime", nil}},
							"$response.responseTime",
							0,
						},
					},
				},
			},
		},
	}

	cursor, err := er.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		Total           int64   `bson:"total"`
		Active          int64   `bson:"active"`
		Resolved        int64   `bson:"resolved"`
		FalseAlarms     int64   `bson:"falseAlarms"`
		AvgResponseTime float64 `bson:"avgResponseTime"`
	}

	if cursor.Next(ctx) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}

	return &models.EmergencyStats{
		Total:           result.Total,
		Active:          result.Active,
		ResolvedToday:   result.Resolved,
		FalseAlarms:     result.FalseAlarms,
		AvgResponseTime: result.AvgResponseTime,
	}, nil
}
