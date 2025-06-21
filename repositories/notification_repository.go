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

type NotificationRepository struct {
	collection           *mongo.Collection
	templateCollection   *mongo.Collection
	preferenceCollection *mongo.Collection
}

func NewNotificationRepository(db *mongo.Database) *NotificationRepository {
	return &NotificationRepository{
		collection:           db.Collection("notifications"),
		templateCollection:   db.Collection("notification_templates"),
		preferenceCollection: db.Collection("notification_preferences"),
	}
}

func (nr *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	notification.ID = primitive.NewObjectID()
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()
	notification.Status = "pending"

	_, err := nr.collection.InsertOne(ctx, notification)
	return err
}

func (nr *NotificationRepository) GetByID(ctx context.Context, id string) (*models.Notification, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid notification ID")
	}

	var notification models.Notification
	err = nr.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&notification)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("notification not found")
		}
		return nil, err
	}

	return &notification, nil
}

func (nr *NotificationRepository) GetUserNotifications(ctx context.Context, userID string, page, pageSize int) ([]models.Notification, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := nr.collection.Find(ctx, bson.M{"userId": objectID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	err = cursor.All(ctx, &notifications)
	return notifications, err
}

func (nr *NotificationRepository) GetPendingNotifications(ctx context.Context, limit int) ([]models.Notification, error) {
	filter := bson.M{
		"status": "pending",
		"$or": []bson.M{
			{"scheduledFor": bson.M{"$exists": false}},
			{"scheduledFor": bson.M{"$lte": time.Now()}},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{"createdAt", 1}}).
		SetLimit(int64(limit))

	cursor, err := nr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	err = cursor.All(ctx, &notifications)
	return notifications, err
}

func (nr *NotificationRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid notification ID")
	}

	update["updatedAt"] = time.Now()

	result, err := nr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("notification not found")
	}

	return nil
}

func (nr *NotificationRepository) MarkAsRead(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid notification ID")
	}

	update := bson.M{
		"isRead":    true,
		"readAt":    time.Now(),
		"updatedAt": time.Now(),
	}

	result, err := nr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("notification not found")
	}

	return nil
}

func (nr *NotificationRepository) MarkAllAsRead(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId": objectID,
		"isRead": false,
	}

	update := bson.M{
		"isRead":    true,
		"readAt":    time.Now(),
		"updatedAt": time.Now(),
	}

	_, err = nr.collection.UpdateMany(ctx, filter, bson.M{"$set": update})
	return err
}

func (nr *NotificationRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid notification ID")
	}

	result, err := nr.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("notification not found")
	}

	return nil
}

func (nr *NotificationRepository) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, errors.New("invalid user ID")
	}

	count, err := nr.collection.CountDocuments(ctx, bson.M{
		"userId": objectID,
		"isRead": false,
	})

	return count, err
}

func (nr *NotificationRepository) DeleteExpired(ctx context.Context) (int64, error) {
	filter := bson.M{
		"expiresAt": bson.M{
			"$exists": true,
			"$lt":     time.Now(),
		},
	}

	result, err := nr.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// Notification Preferences
func (nr *NotificationRepository) GetUserPreferences(ctx context.Context, userID string) (*models.NotificationPreference, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var prefs models.NotificationPreference
	err = nr.preferenceCollection.FindOne(ctx, bson.M{"userId": objectID}).Decode(&prefs)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default preferences
			return &models.NotificationPreference{
				UserID:        objectID,
				GlobalEnabled: true,
				Categories:    make(map[string]models.NotificationCategoryPref),
				CirclePrefs:   make(map[string]models.NotificationChannels),
				UpdatedAt:     time.Now(),
			}, nil
		}
		return nil, err
	}

	return &prefs, nil
}

func (nr *NotificationRepository) UpdateUserPreferences(ctx context.Context, userID string, prefs *models.NotificationPreference) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	prefs.UserID = objectID
	prefs.UpdatedAt = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err = nr.preferenceCollection.ReplaceOne(
		ctx,
		bson.M{"userId": objectID},
		prefs,
		opts,
	)

	return err
}
