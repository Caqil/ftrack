// repositories/notification_repository.go
package repositories

import (
	"context"
	"fmt"
	"ftrack/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NotificationRepository struct {
	db                       *mongo.Database
	notificationCollection   *mongo.Collection
	pushSettingsCollection   *mongo.Collection
	pushDeviceCollection     *mongo.Collection
	preferencesCollection    *mongo.Collection
	emailSettingsCollection  *mongo.Collection
	emailTemplatesCollection *mongo.Collection
	smsSettingsCollection    *mongo.Collection
	inAppSettingsCollection  *mongo.Collection
	channelsCollection       *mongo.Collection
	rulesCollection          *mongo.Collection
	dndCollection            *mongo.Collection
	templatesCollection      *mongo.Collection
	subscriptionsCollection  *mongo.Collection
}

func NewNotificationRepository(db *mongo.Database) *NotificationRepository {
	return &NotificationRepository{
		db:                       db,
		notificationCollection:   db.Collection("notifications"),
		pushSettingsCollection:   db.Collection("push_settings"),
		pushDeviceCollection:     db.Collection("push_devices"),
		preferencesCollection:    db.Collection("notification_preferences"),
		emailSettingsCollection:  db.Collection("email_settings"),
		emailTemplatesCollection: db.Collection("email_templates"),
		smsSettingsCollection:    db.Collection("sms_settings"),
		inAppSettingsCollection:  db.Collection("in_app_settings"),
		channelsCollection:       db.Collection("notification_channels"),
		rulesCollection:          db.Collection("notification_rules"),
		dndCollection:            db.Collection("dnd_settings"),
		templatesCollection:      db.Collection("notification_templates"),
		subscriptionsCollection:  db.Collection("notification_subscriptions"),
	}
}

// ========================
// Core Notification CRUD
// ========================

func (nr *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()

	_, err := nr.notificationCollection.InsertOne(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

func (nr *NotificationRepository) GetByID(ctx context.Context, id string) (*models.Notification, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid notification ID: %w", err)
	}

	var notification models.Notification
	err = nr.notificationCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&notification)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notification, nil
}

func (nr *NotificationRepository) Update(ctx context.Context, notification *models.Notification) error {
	notification.UpdatedAt = time.Now()

	filter := bson.M{"_id": notification.ID}
	update := bson.M{"$set": notification}

	_, err := nr.notificationCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	return nil
}

func (nr *NotificationRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid notification ID: %w", err)
	}

	_, err = nr.notificationCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	return nil
}

// ========================
// User Notification Queries
// ========================

func (nr *NotificationRepository) GetUserNotifications(ctx context.Context, userID string, page, pageSize int, notificationType, status string) ([]models.Notification, int64, error) {
	filter := bson.M{"user_id": userID, "is_archived": bson.M{"$ne": true}}

	if notificationType != "" {
		filter["type"] = notificationType
	}
	if status != "" {
		filter["status"] = status
	}

	// Count total documents
	total, err := nr.notificationCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Calculate skip value
	skip := (page - 1) * pageSize

	// Find options with pagination and sorting
	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := nr.notificationCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find notifications: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, 0, fmt.Errorf("failed to decode notifications: %w", err)
	}

	return notifications, total, nil
}

func (nr *NotificationRepository) GetNotificationsByPriority(ctx context.Context, userID, priority string, page, pageSize int) ([]models.Notification, int64, error) {
	filter := bson.M{
		"user_id":     userID,
		"priority":    priority,
		"is_archived": bson.M{"$ne": true},
	}

	total, err := nr.notificationCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	skip := (page - 1) * pageSize
	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := nr.notificationCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find notifications: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, 0, fmt.Errorf("failed to decode notifications: %w", err)
	}

	return notifications, total, nil
}

func (nr *NotificationRepository) GetCircleNotifications(ctx context.Context, userID, circleID string, page, pageSize int) ([]models.Notification, int64, error) {
	filter := bson.M{
		"user_id":     userID,
		"circle_id":   circleID,
		"is_archived": bson.M{"$ne": true},
	}

	total, err := nr.notificationCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	skip := (page - 1) * pageSize
	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := nr.notificationCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find notifications: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, 0, fmt.Errorf("failed to decode notifications: %w", err)
	}

	return notifications, total, nil
}

func (nr *NotificationRepository) GetArchivedNotifications(ctx context.Context, userID string, page, pageSize int) ([]models.Notification, int64, error) {
	filter := bson.M{
		"user_id":     userID,
		"is_archived": true,
	}

	total, err := nr.notificationCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	skip := (page - 1) * pageSize
	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := nr.notificationCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find notifications: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, 0, fmt.Errorf("failed to decode notifications: %w", err)
	}

	return notifications, total, nil
}

// ========================
// Badge Count Methods
// ========================

func (nr *NotificationRepository) GetNotificationCount(ctx context.Context, userID, status string) (int64, error) {
	filter := bson.M{
		"user_id":     userID,
		"is_archived": bson.M{"$ne": true},
	}

	if status != "" {
		filter["status"] = status
	}

	count, err := nr.notificationCollection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	return count, nil
}

func (nr *NotificationRepository) GetNotificationCountsByType(ctx context.Context, userID string) (map[string]int, error) {
	pipeline := []bson.M{
		{"$match": bson.M{
			"user_id":     userID,
			"is_archived": bson.M{"$ne": true},
		}},
		{"$group": bson.M{
			"_id":   "$type",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := nr.notificationCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate by type: %w", err)
	}
	defer cursor.Close(ctx)

	counts := make(map[string]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		counts[result.ID] = result.Count
	}

	return counts, nil
}

func (nr *NotificationRepository) GetNotificationCountsByCircle(ctx context.Context, userID string) (map[string]int, error) {
	pipeline := []bson.M{
		{"$match": bson.M{
			"user_id":     userID,
			"is_archived": bson.M{"$ne": true},
			"circle_id":   bson.M{"$ne": nil},
		}},
		{"$group": bson.M{
			"_id":   "$circle_id",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := nr.notificationCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate by circle: %w", err)
	}
	defer cursor.Close(ctx)

	counts := make(map[string]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		counts[result.ID] = result.Count
	}

	return counts, nil
}

func (nr *NotificationRepository) GetNotificationCountsByPriority(ctx context.Context, userID string) (map[string]int, error) {
	pipeline := []bson.M{
		{"$match": bson.M{
			"user_id":     userID,
			"is_archived": bson.M{"$ne": true},
		}},
		{"$group": bson.M{
			"_id":   "$priority",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := nr.notificationCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate by priority: %w", err)
	}
	defer cursor.Close(ctx)

	counts := make(map[string]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		counts[result.ID] = result.Count
	}

	return counts, nil
}

// ========================
// Push Settings
// ========================

func (nr *NotificationRepository) GetPushSettings(ctx context.Context, userID string) (*models.PushSettings, error) {
	var settings models.PushSettings
	err := nr.pushSettingsCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("failed to get push settings: %w", err)
	}

	return &settings, nil
}

func (nr *NotificationRepository) CreatePushSettings(ctx context.Context, settings *models.PushSettings) error {
	settings.CreatedAt = time.Now()
	settings.UpdatedAt = time.Now()

	_, err := nr.pushSettingsCollection.InsertOne(ctx, settings)
	if err != nil {
		return fmt.Errorf("failed to create push settings: %w", err)
	}

	return nil
}

func (nr *NotificationRepository) UpdatePushSettings(ctx context.Context, settings *models.PushSettings) error {
	settings.UpdatedAt = time.Now()

	filter := bson.M{"user_id": settings.UserID}
	update := bson.M{"$set": settings}

	_, err := nr.pushSettingsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update push settings: %w", err)
	}

	return nil
}

// ========================
// Push Devices
// ========================

func (nr *NotificationRepository) GetSMSSettings(ctx context.Context, userID string) (*models.SMSSettings, error) {
	var settings models.SMSSettings
	err := nr.smsSettingsCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("failed to get SMS settings: %w", err)
	}

	return &settings, nil
}

func (nr *NotificationRepository) CreateSMSSettings(ctx context.Context, settings *models.SMSSettings) error {
	settings.CreatedAt = time.Now()
	settings.UpdatedAt = time.Now()

	_, err := nr.smsSettingsCollection.InsertOne(ctx, settings)
	if err != nil {
		return fmt.Errorf("failed to create SMS settings: %w", err)
	}

	return nil
}

func (nr *NotificationRepository) UpdateSMSSettings(ctx context.Context, settings *models.SMSSettings) error {
	settings.UpdatedAt = time.Now()

	filter := bson.M{"user_id": settings.UserID}
	update := bson.M{"$set": settings}

	_, err := nr.smsSettingsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update SMS settings: %w", err)
	}

	return nil
}
func (nr *NotificationRepository) GetDeviceByToken(ctx context.Context, deviceToken string) (*models.PushDevice, error) {
	var device models.PushDevice
	err := nr.pushDeviceCollection.FindOne(ctx, bson.M{"device_token": deviceToken}).Decode(&device)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("device not found")
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return &device, nil
}

func (nr *NotificationRepository) CreatePushDevice(ctx context.Context, device *models.PushDevice) error {
	device.CreatedAt = time.Now()
	device.UpdatedAt = time.Now()

	_, err := nr.pushDeviceCollection.InsertOne(ctx, device)
	if err != nil {
		return fmt.Errorf("failed to create push device: %w", err)
	}

	return nil
}

func (nr *NotificationRepository) GetPushDevice(ctx context.Context, deviceID string) (*models.PushDevice, error) {
	objectID, err := primitive.ObjectIDFromHex(deviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid device ID: %w", err)
	}

	var device models.PushDevice
	err = nr.pushDeviceCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&device)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("device not found")
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return &device, nil
}

func (nr *NotificationRepository) UpdatePushDevice(ctx context.Context, device *models.PushDevice) error {
	device.UpdatedAt = time.Now()

	filter := bson.M{"_id": device.ID}
	update := bson.M{"$set": device}

	_, err := nr.pushDeviceCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update push device: %w", err)
	}

	return nil
}

func (nr *NotificationRepository) DeletePushDevice(ctx context.Context, deviceID string) error {
	objectID, err := primitive.ObjectIDFromHex(deviceID)
	if err != nil {
		return fmt.Errorf("invalid device ID: %w", err)
	}

	_, err = nr.pushDeviceCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to delete push device: %w", err)
	}

	return nil
}

func (nr *NotificationRepository) GetUserPushDevices(ctx context.Context, userID string) ([]models.PushDevice, error) {
	filter := bson.M{"user_id": userID, "is_active": true}

	cursor, err := nr.pushDeviceCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find push devices: %w", err)
	}
	defer cursor.Close(ctx)

	var devices []models.PushDevice
	if err = cursor.All(ctx, &devices); err != nil {
		return nil, fmt.Errorf("failed to decode push devices: %w", err)
	}

	return devices, nil
}

// ========================
// Notification Preferences
// ========================

func (nr *NotificationRepository) GetNotificationPreferences(ctx context.Context, userID string) (*models.NotificationPreferences, error) {
	var preferences models.NotificationPreferences
	err := nr.preferencesCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("failed to get preferences: %w", err)
	}

	return &preferences, nil
}

func (nr *NotificationRepository) CreateNotificationPreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	preferences.CreatedAt = time.Now()
	preferences.UpdatedAt = time.Now()

	_, err := nr.preferencesCollection.InsertOne(ctx, preferences)
	if err != nil {
		return fmt.Errorf("failed to create preferences: %w", err)
	}

	return nil
}

func (nr *NotificationRepository) UpdateNotificationPreferences(ctx context.Context, preferences *models.NotificationPreferences) error {
	preferences.UpdatedAt = time.Now()

	filter := bson.M{"user_id": preferences.UserID}
	update := bson.M{"$set": preferences}

	_, err := nr.preferencesCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update preferences: %w", err)
	}

	return nil
}

// ========================
// Email Settings
// ========================

func (nr *NotificationRepository) GetEmailSettings(ctx context.Context, userID string) (*models.EmailSettings, error) {
	var settings models.EmailSettings
	err := nr.emailSettingsCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("failed to get email settings: %w", err)
	}

	return &settings, nil
}

func (nr *NotificationRepository) CreateEmailSettings(ctx context.Context, settings *models.EmailSettings) error {
	settings.CreatedAt = time.Now()
	settings.UpdatedAt = time.Now()

	_, err := nr.emailSettingsCollection.InsertOne(ctx, settings)
	if err != nil {
		return fmt.Errorf("failed to create email settings: %w", err)
	}

	return nil
}

func (nr *NotificationRepository) UpdateEmailSettings(ctx context.Context, settings *models.EmailSettings) error {
	settings.UpdatedAt = time.Now()

	filter := bson.M{"user_id": settings.UserID}
	update := bson.M{"$set": settings}

	_, err := nr.emailSettingsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update email settings: %w", err)
	}

	return nil
}

// ========================
// Email Templates
// ========================

func (nr *NotificationRepository) GetEmailTemplates(ctx context.Context, userID string) ([]models.EmailTemplate, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"user_id": userID},
			{"is_default": true},
		},
	}

	cursor, err := nr.emailTemplatesCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find email templates: %w", err)
	}
	defer cursor.Close(ctx)

	var templates []models.EmailTemplate
	if err = cursor.All(ctx, &templates); err != nil {
		return nil, fmt.Errorf("failed to decode email templates: %w", err)
	}

	return templates, nil
}

func (nr *NotificationRepository) GetEmailTemplate(ctx context.Context, templateID string) (*models.EmailTemplate, error) {
	objectID, err := primitive.ObjectIDFromHex(templateID)
	if err != nil {
		return nil, fmt.Errorf("invalid template ID: %w", err)
	}

	var template models.EmailTemplate
	err = nr.emailTemplatesCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&template)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &template, nil
}

func (nr *NotificationRepository) UpdateEmailTemplate(ctx context.Context, template *models.EmailTemplate) error {
	template.UpdatedAt = time.Now()

	filter := bson.M{"_id": template.ID}
	update := bson.M{"$set": template}

	_, err := nr.emailTemplatesCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update email template: %w", err)
	}

	return nil
}

// ========================
// Index Creation
// ========================

func (nr *NotificationRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		// Notification indexes
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "circle_id", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	}

	_, err := nr.notificationCollection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create notification indexes: %w", err)
	}

	// Push settings indexes
	pushSettingsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = nr.pushSettingsCollection.Indexes().CreateMany(ctx, pushSettingsIndexes)
	if err != nil {
		return fmt.Errorf("failed to create push settings indexes: %w", err)
	}

	// Push device indexes
	pushDeviceIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "device_token", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = nr.pushDeviceCollection.Indexes().CreateMany(ctx, pushDeviceIndexes)
	if err != nil {
		return fmt.Errorf("failed to create push device indexes: %w", err)
	}

	// Preferences indexes
	preferencesIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = nr.preferencesCollection.Indexes().CreateMany(ctx, preferencesIndexes)
	if err != nil {
		return fmt.Errorf("failed to create preferences indexes: %w", err)
	}

	// Email settings indexes
	emailSettingsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "email_address", Value: 1}},
		},
	}

	_, err = nr.emailSettingsCollection.Indexes().CreateMany(ctx, emailSettingsIndexes)
	if err != nil {
		return fmt.Errorf("failed to create email settings indexes: %w", err)
	}

	return nil
}

// ========================
// Cleanup Methods
// ========================

func (nr *NotificationRepository) CleanupExpiredNotifications(ctx context.Context) error {
	filter := bson.M{
		"expires_at": bson.M{"$lt": time.Now()},
	}

	_, err := nr.notificationCollection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired notifications: %w", err)
	}

	return nil
}

func (nr *NotificationRepository) CleanupOldNotifications(ctx context.Context, userID string, olderThan time.Time) (int64, error) {
	filter := bson.M{
		"created_at": bson.M{"$lt": olderThan},
	}

	if userID != "" {
		filter["user_id"] = userID
	}

	result, err := nr.notificationCollection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old notifications: %w", err)
	}

	return result.DeletedCount, nil
}

// ========================
// Analytics Methods
// ========================

func (nr *NotificationRepository) GetNotificationStatsByDateRange(ctx context.Context, userID string, startDate, endDate time.Time) (*models.NotificationStats, error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	if userID != "" {
		filter["user_id"] = userID
	}

	// Count total notifications
	total, err := nr.notificationCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count total notifications: %w", err)
	}

	// Count by status
	sentFilter := filter
	sentFilter["status"] = bson.M{"$in": []string{"read", "unread"}}
	sent, _ := nr.notificationCollection.CountDocuments(ctx, sentFilter)

	deliveredFilter := filter
	deliveredFilter["status"] = bson.M{"$in": []string{"read", "unread"}}
	delivered, _ := nr.notificationCollection.CountDocuments(ctx, deliveredFilter)

	readFilter := filter
	readFilter["status"] = "read"
	read, _ := nr.notificationCollection.CountDocuments(ctx, readFilter)

	stats := &models.NotificationStats{
		Total:     total,
		Sent:      sent,
		Delivered: delivered,
		Read:      read,
		Failed:    0, // Would need additional logic to track failures
		ByType:    make(map[string]int64),
		ByChannel: make(map[string]int64),
	}

	return stats, nil
}

// ========================
// Search and Advanced Queries
// ========================

func (nr *NotificationRepository) SearchNotifications(ctx context.Context, userID, searchTerm string, page, pageSize int) ([]models.Notification, int64, error) {
	filter := bson.M{
		"user_id": userID,
		"$or": []bson.M{
			{"title": bson.M{"$regex": searchTerm, "$options": "i"}},
			{"message": bson.M{"$regex": searchTerm, "$options": "i"}},
		},
	}

	total, err := nr.notificationCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	skip := (page - 1) * pageSize
	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := nr.notificationCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search notifications: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, 0, fmt.Errorf("failed to decode search results: %w", err)
	}

	return notifications, total, nil
}

func (nr *NotificationRepository) GetNotificationsByDateRange(ctx context.Context, userID string, startDate, endDate time.Time, page, pageSize int) ([]models.Notification, int64, error) {
	filter := bson.M{
		"user_id": userID,
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	total, err := nr.notificationCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications by date range: %w", err)
	}

	skip := (page - 1) * pageSize
	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := nr.notificationCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find notifications by date range: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, 0, fmt.Errorf("failed to decode notifications: %w", err)
	}

	return notifications, total, nil
}
