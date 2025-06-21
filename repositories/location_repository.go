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

type LocationRepository struct {
	collection *mongo.Collection
}

func NewLocationRepository(db *mongo.Database) *LocationRepository {
	return &LocationRepository{
		collection: db.Collection("locations"),
	}
}

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

func (lr *LocationRepository) GetLocationHistory(ctx context.Context, userID string, startTime, endTime time.Time, limit int) ([]models.Location, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId": objectID,
		"createdAt": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
	}

	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetLimit(int64(limit))

	cursor, err := lr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var locations []models.Location
	err = cursor.All(ctx, &locations)
	return locations, err
}

func (lr *LocationRepository) GetLocationsInRadius(ctx context.Context, lat, lon, radiusM float64, limit int) ([]models.Location, error) {
	// Using MongoDB's geospatial query
	filter := bson.M{
		"location": bson.M{
			"$near": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{lon, lat}, // MongoDB uses [longitude, latitude]
				},
				"$maxDistance": radiusM,
			},
		},
	}

	opts := options.Find().SetLimit(int64(limit))
	cursor, err := lr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var locations []models.Location
	err = cursor.All(ctx, &locations)
	return locations, err
}

func (lr *LocationRepository) GetUserLocationsInTimeRange(ctx context.Context, userIDs []string, startTime, endTime time.Time) (map[string][]models.Location, error) {
	objectIDs := make([]primitive.ObjectID, len(userIDs))
	for i, id := range userIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs[i] = objectID
	}

	filter := bson.M{
		"userId": bson.M{"$in": objectIDs},
		"createdAt": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
	}

	opts := options.Find().SetSort(bson.D{{"userId", 1}, {"createdAt", -1}})
	cursor, err := lr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var locations []models.Location
	err = cursor.All(ctx, &locations)
	if err != nil {
		return nil, err
	}

	// Group by user ID
	result := make(map[string][]models.Location)
	for _, location := range locations {
		userID := location.UserID.Hex()
		result[userID] = append(result[userID], location)
	}

	return result, nil
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

func (lr *LocationRepository) GetDrivingLocations(ctx context.Context, userID string, startTime, endTime time.Time) ([]models.Location, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    objectID,
		"isDriving": true,
		"createdAt": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
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

func (lr *LocationRepository) GetLatestLocationsForUsers(ctx context.Context, userIDs []string) (map[string]*models.Location, error) {
	objectIDs := make([]primitive.ObjectID, len(userIDs))
	for i, id := range userIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs[i] = objectID
	}

	// Use aggregation to get latest location for each user
	pipeline := []bson.M{
		{"$match": bson.M{"userId": bson.M{"$in": objectIDs}}},
		{"$sort": bson.M{"userId": 1, "createdAt": -1}},
		{"$group": bson.M{
			"_id":    "$userId",
			"latest": bson.M{"$first": "$$ROOT"},
		}},
	}

	cursor, err := lr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[string]*models.Location)
	for cursor.Next(ctx) {
		var item struct {
			ID     primitive.ObjectID `bson:"_id"`
			Latest models.Location    `bson:"latest"`
		}

		err := cursor.Decode(&item)
		if err != nil {
			continue
		}

		result[item.ID.Hex()] = &item.Latest
	}

	return result, nil
}
