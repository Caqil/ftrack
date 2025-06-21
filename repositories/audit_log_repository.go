package repositories

import (
	"context"
	"ftrack/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuditLogRepository struct {
	collection *mongo.Collection
}

func NewAuditLogRepository(db *mongo.Database) *AuditLogRepository {
	return &AuditLogRepository{
		collection: db.Collection("audit_logs"),
	}
}

func (alr *AuditLogRepository) Create(ctx context.Context, entry *models.AuditLogEntry) error {
	entry.ID = primitive.NewObjectID()
	entry.CreatedAt = time.Now()

	_, err := alr.collection.InsertOne(ctx, entry)
	return err
}

func (alr *AuditLogRepository) GetUserAuditLog(ctx context.Context, userID string, page, pageSize int) ([]models.AuditLogEntry, int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, err
	}

	filter := bson.M{"userId": userObjectID}

	// Get total count
	total, err := alr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * pageSize

	// Find documents with pagination
	opts := options.Find().
		SetSort(bson.M{"createdAt": -1}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := alr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var entries []models.AuditLogEntry
	err = cursor.All(ctx, &entries)
	return entries, total, err
}

func (alr *AuditLogRepository) LogSecurityEvent(ctx context.Context, userID, eventType, description, ipAddress, userAgent, deviceType, severity string, details map[string]interface{}) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	entry := models.AuditLogEntry{
		UserID:      userObjectID,
		EventType:   eventType,
		Description: description,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		DeviceType:  deviceType,
		Severity:    severity,
		Details:     details,
	}

	return alr.Create(ctx, &entry)
}

func (alr *AuditLogRepository) GetSecurityEvents(ctx context.Context, userID string, eventTypes []string, limit int) ([]models.AuditLogEntry, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"userId": userObjectID}
	if len(eventTypes) > 0 {
		filter["eventType"] = bson.M{"$in": eventTypes}
	}

	opts := options.Find().
		SetSort(bson.M{"createdAt": -1}).
		SetLimit(int64(limit))

	cursor, err := alr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var entries []models.AuditLogEntry
	err = cursor.All(ctx, &entries)
	return entries, err
}

func (alr *AuditLogRepository) CleanupOldLogs(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	filter := bson.M{"createdAt": bson.M{"$lt": cutoff}}

	_, err := alr.collection.DeleteMany(ctx, filter)
	return err
}
