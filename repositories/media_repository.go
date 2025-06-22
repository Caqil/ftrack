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

type MediaRepository struct {
	collection *mongo.Collection
}

func NewMediaRepository(db *mongo.Database) *MediaRepository {
	return &MediaRepository{
		collection: db.Collection("message_media"),
	}
}

func (mr *MediaRepository) Create(ctx context.Context, media *models.MessageMediaExtended) error {
	media.ID = primitive.NewObjectID()
	media.CreatedAt = time.Now()
	media.UpdatedAt = time.Now()

	_, err := mr.collection.InsertOne(ctx, media)
	return err
}

func (mr *MediaRepository) GetByID(ctx context.Context, id string) (*models.MessageMedia, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid media ID")
	}

	var media models.MessageMedia
	err = mr.collection.FindOne(ctx, bson.M{
		"_id":       objectID,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&media)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("media not found")
		}
		return nil, err
	}

	return &media, nil
}

func (mr *MediaRepository) Update(ctx context.Context, id string, media *models.MessageMedia) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid media ID")
	}

	media.UpdatedAt = time.Now()

	result, err := mr.collection.ReplaceOne(
		ctx,
		bson.M{
			"_id":       objectID,
			"isDeleted": bson.M{"$ne": true},
		},
		media,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("media not found")
	}

	return nil
}

func (mr *MediaRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid media ID")
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	result, err := mr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("media not found")
	}

	return nil
}

func (mr *MediaRepository) GetByUser(ctx context.Context, userID string, page, pageSize int) ([]models.MessageMedia, int64, error) {
	filter := bson.M{
		"uploadedBy": userID,
		"isDeleted":  bson.M{"$ne": true},
	}

	// Get total count
	total, err := mr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get media
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var media []models.MessageMedia
	err = cursor.All(ctx, &media)
	return media, total, err
}

func (mr *MediaRepository) GetByType(ctx context.Context, mediaType string, circleIDs []string, page, pageSize int) ([]models.MessageMedia, int64, error) {
	filter := bson.M{
		"type":      mediaType,
		"isDeleted": bson.M{"$ne": true},
	}

	if len(circleIDs) > 0 {
		filter["circleId"] = bson.M{"$in": circleIDs}
	}

	// Get total count
	total, err := mr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get media
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var media []models.MessageMedia
	err = cursor.All(ctx, &media)
	return media, total, err
}

func (mr *MediaRepository) Search(ctx context.Context, req models.SearchMediaRequest, circleIDs []string) ([]models.MessageMediaExtended, int64, error) {
	filter := bson.M{
		"isDeleted": bson.M{"$ne": true},
	}

	if len(circleIDs) > 0 {
		filter["circleId"] = bson.M{"$in": circleIDs}
	}

	if req.MediaType != "" {
		filter["type"] = req.MediaType
	}

	if req.CircleID != "" {
		filter["circleId"] = req.CircleID
	}

	// Get total count
	total, err := mr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get media
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var media []models.MessageMediaExtended
	err = cursor.All(ctx, &media)
	return media, total, err
}
