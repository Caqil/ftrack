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

type DraftRepository struct {
	collection *mongo.Collection
}

func NewDraftRepository(db *mongo.Database) *DraftRepository {
	return &DraftRepository{
		collection: db.Collection("message_drafts"),
	}
}

func (dr *DraftRepository) Create(ctx context.Context, draft *models.MessageDraft) error {
	draft.ID = primitive.NewObjectID()
	draft.CreatedAt = time.Now()
	draft.UpdatedAt = time.Now()

	_, err := dr.collection.InsertOne(ctx, draft)
	return err
}

func (dr *DraftRepository) GetByID(ctx context.Context, id string) (*models.MessageDraft, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid draft ID")
	}

	var draft models.MessageDraft
	err = dr.collection.FindOne(ctx, bson.M{
		"_id":       objectID,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&draft)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("draft not found")
		}
		return nil, err
	}

	return &draft, nil
}

func (dr *DraftRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid draft ID")
	}

	update["updatedAt"] = time.Now()

	result, err := dr.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":       objectID,
			"isDeleted": bson.M{"$ne": true},
		},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("draft not found")
	}

	return nil
}

func (dr *DraftRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid draft ID")
	}

	// Hard delete for drafts since they're temporary
	result, err := dr.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("draft not found")
	}

	return nil
}

func (dr *DraftRepository) GetUserDrafts(ctx context.Context, userID string, req models.GetDraftsRequest) ([]models.MessageDraft, int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	if req.CircleID != "" {
		circleObjectID, err := primitive.ObjectIDFromHex(req.CircleID)
		if err == nil {
			filter["circleId"] = circleObjectID
		}
	}

	// Get total count
	total, err := dr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get drafts
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{"updatedAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := dr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var drafts []models.MessageDraft
	err = cursor.All(ctx, &drafts)
	return drafts, total, err
}

func (dr *DraftRepository) GetByCircle(ctx context.Context, userID, circleID string) ([]models.MessageDraft, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"circleId":  circleObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	opts := options.Find().SetSort(bson.D{{"updatedAt", -1}})

	cursor, err := dr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var drafts []models.MessageDraft
	err = cursor.All(ctx, &drafts)
	return drafts, err
}

func (dr *DraftRepository) DeleteOldDrafts(ctx context.Context, olderThan time.Time) error {
	filter := bson.M{
		"updatedAt": bson.M{"$lt": olderThan},
		"autoSave":  true, // Only auto-saved drafts
	}

	_, err := dr.collection.DeleteMany(ctx, filter)
	return err
}

func (dr *DraftRepository) CountUserDrafts(ctx context.Context, userID string) (int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	return dr.collection.CountDocuments(ctx, filter)
}

func (dr *DraftRepository) GetRecentDrafts(ctx context.Context, userID string, limit int) ([]models.MessageDraft, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	opts := options.Find().
		SetSort(bson.D{{"updatedAt", -1}}).
		SetLimit(int64(limit))

	cursor, err := dr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var drafts []models.MessageDraft
	err = cursor.All(ctx, &drafts)
	return drafts, err
}