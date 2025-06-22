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

type ScheduleRepository struct {
	collection *mongo.Collection
}

func NewScheduleRepository(db *mongo.Database) *ScheduleRepository {
	return &ScheduleRepository{
		collection: db.Collection("scheduled_messages"),
	}
}

func (sr *ScheduleRepository) Create(ctx context.Context, scheduledMessage *models.ScheduledMessage) error {
	scheduledMessage.ID = primitive.NewObjectID()
	scheduledMessage.CreatedAt = time.Now()
	scheduledMessage.UpdatedAt = time.Now()
	if scheduledMessage.Status == "" {
		scheduledMessage.Status = "pending"
	}

	_, err := sr.collection.InsertOne(ctx, scheduledMessage)
	return err
}

func (sr *ScheduleRepository) GetByID(ctx context.Context, id string) (*models.ScheduledMessage, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid scheduled message ID")
	}

	var scheduledMessage models.ScheduledMessage
	err = sr.collection.FindOne(ctx, bson.M{
		"_id":       objectID,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&scheduledMessage)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("scheduled message not found")
		}
		return nil, err
	}

	return &scheduledMessage, nil
}

func (sr *ScheduleRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid scheduled message ID")
	}

	update["updatedAt"] = time.Now()

	result, err := sr.collection.UpdateOne(
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
		return errors.New("scheduled message not found")
	}

	return nil
}

func (sr *ScheduleRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid scheduled message ID")
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	result, err := sr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("scheduled message not found")
	}

	return nil
}

func (sr *ScheduleRepository) GetUserScheduledMessages(ctx context.Context, userID string, req models.GetScheduledMessagesRequest) ([]models.ScheduledMessage, int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	if req.Status != "" {
		filter["status"] = req.Status
	}

	// Get total count
	total, err := sr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get scheduled messages
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{Key: "scheduledAt", Value: 1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := sr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var messages []models.ScheduledMessage
	err = cursor.All(ctx, &messages)
	return messages, total, err
}

func (sr *ScheduleRepository) GetPendingMessages(ctx context.Context, beforeTime time.Time) ([]models.ScheduledMessage, error) {
	filter := bson.M{
		"status":      "pending",
		"scheduledAt": bson.M{"$lte": beforeTime},
		"isDeleted":   bson.M{"$ne": true},
	}

	opts := options.Find().SetSort(bson.D{{Key: "scheduledAt", Value: 1}})

	cursor, err := sr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.ScheduledMessage
	err = cursor.All(ctx, &messages)
	return messages, err
}

func (sr *ScheduleRepository) MarkAsSent(ctx context.Context, id string, messageID primitive.ObjectID) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid scheduled message ID")
	}

	sentAt := time.Now()
	update := bson.M{
		"status":    "sent",
		"sentAt":    &sentAt,
		"messageId": &messageID,
		"updatedAt": sentAt,
	}

	result, err := sr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("scheduled message not found")
	}

	return nil
}

func (sr *ScheduleRepository) MarkAsFailed(ctx context.Context, id string, errorMsg string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid scheduled message ID")
	}

	update := bson.M{
		"status":    "failed",
		"errorMsg":  errorMsg,
		"updatedAt": time.Now(),
	}

	result, err := sr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("scheduled message not found")
	}

	return nil
}

func (sr *ScheduleRepository) GetByCircle(ctx context.Context, circleID string, status string) ([]models.ScheduledMessage, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	filter := bson.M{
		"circleId":  circleObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	if status != "" {
		filter["status"] = status
	}

	opts := options.Find().SetSort(bson.D{{Key: "scheduledAt", Value: 1}})

	cursor, err := sr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.ScheduledMessage
	err = cursor.All(ctx, &messages)
	return messages, err
}

func (sr *ScheduleRepository) CleanupOldMessages(ctx context.Context, olderThan time.Time) error {
	filter := bson.M{
		"$or": []bson.M{
			{"status": "sent", "sentAt": bson.M{"$lt": olderThan}},
			{"status": "failed", "updatedAt": bson.M{"$lt": olderThan}},
			{"status": "cancelled", "updatedAt": bson.M{"$lt": olderThan}},
		},
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	_, err := sr.collection.UpdateMany(ctx, filter, bson.M{"$set": update})
	return err
}

func (sr *ScheduleRepository) GetUpcomingMessages(ctx context.Context, userID string, hours int) ([]models.ScheduledMessage, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	now := time.Now()
	future := now.Add(time.Duration(hours) * time.Hour)

	filter := bson.M{
		"userId":      userObjectID,
		"status":      "pending",
		"scheduledAt": bson.M{"$gte": now, "$lte": future},
		"isDeleted":   bson.M{"$ne": true},
	}

	opts := options.Find().SetSort(bson.D{{Key: "scheduledAt", Value: 1}})

	cursor, err := sr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.ScheduledMessage
	err = cursor.All(ctx, &messages)
	return messages, err
}

func (sr *ScheduleRepository) CountPendingMessages(ctx context.Context, userID string) (int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"status":    "pending",
		"isDeleted": bson.M{"$ne": true},
	}

	return sr.collection.CountDocuments(ctx, filter)
}
