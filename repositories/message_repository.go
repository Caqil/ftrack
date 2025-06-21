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

type MessageRepository struct {
	collection     *mongo.Collection
	roomCollection *mongo.Collection
}

func NewMessageRepository(db *mongo.Database) *MessageRepository {
	return &MessageRepository{
		collection:     db.Collection("messages"),
		roomCollection: db.Collection("chat_rooms"),
	}
}

func (mr *MessageRepository) Create(ctx context.Context, message *models.Message) error {
	message.ID = primitive.NewObjectID()
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()
	message.Status = "sent"

	_, err := mr.collection.InsertOne(ctx, message)
	return err
}

func (mr *MessageRepository) GetByID(ctx context.Context, id string) (*models.Message, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid message ID")
	}

	var message models.Message
	err = mr.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("message not found")
		}
		return nil, err
	}

	return &message, nil
}

func (mr *MessageRepository) GetCircleMessages(ctx context.Context, circleID string, page, pageSize int) ([]models.Message, error) {
	objectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	filter := bson.M{
		"circleId":  objectID,
		"isDeleted": bson.M{"$ne": true},
	}

	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	return messages, err
}

func (mr *MessageRepository) GetMessagesSince(ctx context.Context, circleID string, since time.Time) ([]models.Message, error) {
	objectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	filter := bson.M{
		"circleId":  objectID,
		"createdAt": bson.M{"$gt": since},
		"isDeleted": bson.M{"$ne": true},
	}

	opts := options.Find().SetSort(bson.D{{"createdAt", 1}})
	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	return messages, err
}

func (mr *MessageRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid message ID")
	}

	update["updatedAt"] = time.Now()

	result, err := mr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("message not found")
	}

	return nil
}

func (mr *MessageRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid message ID")
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
		return errors.New("message not found")
	}

	return nil
}

func (mr *MessageRepository) MarkAsRead(ctx context.Context, messageIDs []string, userID string) error {
	objectIDs := make([]primitive.ObjectID, len(messageIDs))
	for i, id := range messageIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs[i] = objectID
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	readStatus := models.MessageReadStatus{
		UserID: userObjectID,
		ReadAt: time.Now(),
	}

	filter := bson.M{"_id": bson.M{"$in": objectIDs}}
	update := bson.M{
		"$addToSet": bson.M{"readBy": readStatus},
		"$set":      bson.M{"updatedAt": time.Now()},
	}

	_, err = mr.collection.UpdateMany(ctx, filter, update)
	return err
}

func (mr *MessageRepository) AddReaction(ctx context.Context, messageID, userID, emoji string) error {
	messageObjectID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return errors.New("invalid message ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	reaction := models.MessageReaction{
		UserID:  userObjectID,
		Emoji:   emoji,
		AddedAt: time.Now(),
	}

	// Remove existing reaction from this user first
	_, err = mr.collection.UpdateOne(
		ctx,
		bson.M{"_id": messageObjectID},
		bson.M{"$pull": bson.M{"reactions": bson.M{"userId": userObjectID}}},
	)

	if err != nil {
		return err
	}

	// Add new reaction
	result, err := mr.collection.UpdateOne(
		ctx,
		bson.M{"_id": messageObjectID},
		bson.M{
			"$push": bson.M{"reactions": reaction},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("message not found")
	}

	return nil
}

func (mr *MessageRepository) RemoveReaction(ctx context.Context, messageID, userID string) error {
	messageObjectID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return errors.New("invalid message ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	result, err := mr.collection.UpdateOne(
		ctx,
		bson.M{"_id": messageObjectID},
		bson.M{
			"$pull": bson.M{"reactions": bson.M{"userId": userObjectID}},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("message not found")
	}

	return nil
}

func (mr *MessageRepository) GetUnreadCount(ctx context.Context, circleID, userID string) (int64, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return 0, errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"circleId":      circleObjectID,
		"senderId":      bson.M{"$ne": userObjectID}, // Not sent by user
		"readBy.userId": bson.M{"$ne": userObjectID}, // Not read by user
		"isDeleted":     bson.M{"$ne": true},
	}

	count, err := mr.collection.CountDocuments(ctx, filter)
	return count, err
}
