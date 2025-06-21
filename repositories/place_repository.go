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

type PlaceRepository struct {
	collection      *mongo.Collection
	visitCollection *mongo.Collection
}

func NewPlaceRepository(db *mongo.Database) *PlaceRepository {
	return &PlaceRepository{
		collection:      db.Collection("places"),
		visitCollection: db.Collection("place_visits"),
	}
}

func (pr *PlaceRepository) Create(ctx context.Context, place *models.Place) error {
	place.ID = primitive.NewObjectID()
	place.CreatedAt = time.Now()
	place.UpdatedAt = time.Now()

	// Initialize default values
	if place.Radius == 0 {
		place.Radius = 100 // Default 100 meters
	}
	if place.Category == "" {
		place.Category = "other"
	}

	_, err := pr.collection.InsertOne(ctx, place)
	return err
}

func (pr *PlaceRepository) GetByID(ctx context.Context, id string) (*models.Place, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
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

func (pr *PlaceRepository) GetUserPlaces(ctx context.Context, userID string) ([]models.Place, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	cursor, err := pr.collection.Find(ctx, bson.M{"userId": objectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var places []models.Place
	err = cursor.All(ctx, &places)
	return places, err
}

func (pr *PlaceRepository) GetCirclePlaces(ctx context.Context, circleID string) ([]models.Place, error) {
	objectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	filter := bson.M{
		"$or": []bson.M{
			{"circleId": objectID},
			{"isShared": true},
		},
	}

	cursor, err := pr.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var places []models.Place
	err = cursor.All(ctx, &places)
	return places, err
}

func (pr *PlaceRepository) GetPlacesInRadius(ctx context.Context, lat, lon, radiusM float64) ([]models.Place, error) {
	// Simple distance calculation using MongoDB aggregation
	pipeline := []bson.M{
		{
			"$addFields": bson.M{
				"distance": bson.M{
					"$sqrt": bson.M{
						"$add": []bson.M{
							{
								"$pow": []interface{}{
									bson.M{"$multiply": []interface{}{
										bson.M{"$subtract": []interface{}{"$latitude", lat}},
										111000, // Approximate meters per degree latitude
									}},
									2,
								},
							},
							{
								"$pow": []interface{}{
									bson.M{"$multiply": []interface{}{
										bson.M{"$subtract": []interface{}{"$longitude", lon}},
										111000, // Approximate meters per degree longitude (at equator)
									}},
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
				"distance": bson.M{"$lte": radiusM},
			},
		},
		{
			"$sort": bson.M{"distance": 1},
		},
	}

	cursor, err := pr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var places []models.Place
	err = cursor.All(ctx, &places)
	return places, err
}

func (pr *PlaceRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid place ID")
	}

	update["updatedAt"] = time.Now()

	result, err := pr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("place not found")
	}

	return nil
}

func (pr *PlaceRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
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

func (pr *PlaceRepository) UpdateStats(ctx context.Context, placeID string, stats models.PlaceStats) error {
	objectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return errors.New("invalid place ID")
	}

	_, err = pr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$set": bson.M{
				"stats":     stats,
				"updatedAt": time.Now(),
			},
		},
	)

	return err
}

// Place Visit methods
func (pr *PlaceRepository) CreateVisit(ctx context.Context, visit *models.PlaceVisit) error {
	visit.ID = primitive.NewObjectID()
	visit.CreatedAt = time.Now()

	_, err := pr.visitCollection.InsertOne(ctx, visit)
	return err
}

func (pr *PlaceRepository) UpdateVisit(ctx context.Context, visitID string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(visitID)
	if err != nil {
		return errors.New("invalid visit ID")
	}

	result, err := pr.visitCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("visit not found")
	}

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
			return nil, nil // No active visit found
		}
		return nil, err
	}

	return &visit, nil
}

func (pr *PlaceRepository) GetPlaceVisits(ctx context.Context, placeID string, limit int) ([]models.PlaceVisit, error) {
	objectID, err := primitive.ObjectIDFromHex(placeID)
	if err != nil {
		return nil, errors.New("invalid place ID")
	}

	opts := options.Find().
		SetSort(bson.D{{"arrivalTime", -1}}).
		SetLimit(int64(limit))

	cursor, err := pr.visitCollection.Find(ctx, bson.M{"placeId": objectID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var visits []models.PlaceVisit
	err = cursor.All(ctx, &visits)
	return visits, err
}
