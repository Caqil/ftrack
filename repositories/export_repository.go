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

type ExportRepository struct {
	collection *mongo.Collection
}

func NewExportRepository(db *mongo.Database) *ExportRepository {
	return &ExportRepository{
		collection: db.Collection("message_exports"),
	}
}

func (er *ExportRepository) Create(ctx context.Context, export *models.MessageExport) error {
	export.ID = primitive.NewObjectID()
	export.CreatedAt = time.Now()
	export.UpdatedAt = time.Now()
	if export.Status == "" {
		export.Status = "processing"
	}
	if export.Progress == 0 {
		export.Progress = 0
	}

	_, err := er.collection.InsertOne(ctx, export)
	return err
}

func (er *ExportRepository) GetByID(ctx context.Context, id string) (*models.MessageExport, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid export ID")
	}

	var export models.MessageExport
	err = er.collection.FindOne(ctx, bson.M{
		"_id":       objectID,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&export)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("export not found")
		}
		return nil, err
	}

	return &export, nil
}

func (er *ExportRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid export ID")
	}

	update["updatedAt"] = time.Now()

	result, err := er.collection.UpdateOne(
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
		return errors.New("export not found")
	}

	return nil
}

func (er *ExportRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid export ID")
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	result, err := er.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("export not found")
	}

	return nil
}

func (er *ExportRepository) GetUserExports(ctx context.Context, userID string, page, pageSize int) ([]models.MessageExport, int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	// Get total count
	total, err := er.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get exports
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := er.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var exports []models.MessageExport
	err = cursor.All(ctx, &exports)
	return exports, total, err
}

func (er *ExportRepository) GetPendingExports(ctx context.Context) ([]models.MessageExport, error) {
	filter := bson.M{
		"status":    "processing",
		"isDeleted": bson.M{"$ne": true},
	}

	opts := options.Find().SetSort(bson.D{{"createdAt", 1}}) // Oldest first

	cursor, err := er.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var exports []models.MessageExport
	err = cursor.All(ctx, &exports)
	return exports, err
}

func (er *ExportRepository) UpdateProgress(ctx context.Context, id string, progress int) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid export ID")
	}

	update := bson.M{
		"progress":  progress,
		"updatedAt": time.Now(),
	}

	if progress >= 100 {
		update["status"] = "completed"
	}

	result, err := er.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("export not found")
	}

	return nil
}

func (er *ExportRepository) MarkAsCompleted(ctx context.Context, id string, fileURL string, fileSize int64, messageCount int) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid export ID")
	}

	update := bson.M{
		"status":       "completed",
		"progress":     100,
		"fileUrl":      fileURL,
		"fileSize":     fileSize,
		"messageCount": messageCount,
		"updatedAt":    time.Now(),
	}

	result, err := er.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("export not found")
	}

	return nil
}

func (er *ExportRepository) MarkAsFailed(ctx context.Context, id string, errorMsg string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid export ID")
	}

	update := bson.M{
		"status":    "failed",
		"errorMsg":  errorMsg,
		"updatedAt": time.Now(),
	}

	result, err := er.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("export not found")
	}

	return nil
}

func (er *ExportRepository) GetExportsByCircle(ctx context.Context, circleID string, page, pageSize int) ([]models.MessageExport, int64, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, 0, errors.New("invalid circle ID")
	}

	filter := bson.M{
		"circleId":  circleObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	// Get total count
	total, err := er.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get exports
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := er.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var exports []models.MessageExport
	err = cursor.All(ctx, &exports)
	return exports, total, err
}

func (er *ExportRepository) GetExpiredExports(ctx context.Context, beforeTime time.Time) ([]models.MessageExport, error) {
	filter := bson.M{
		"expiresAt": bson.M{"$lt": beforeTime},
		"status":    "completed",
		"isDeleted": bson.M{"$ne": true},
	}

	cursor, err := er.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var exports []models.MessageExport
	err = cursor.All(ctx, &exports)
	return exports, err
}

func (er *ExportRepository) CleanupExpiredExports(ctx context.Context, beforeTime time.Time) error {
	filter := bson.M{
		"expiresAt": bson.M{"$lt": beforeTime},
		"isDeleted": bson.M{"$ne": true},
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	_, err := er.collection.UpdateMany(ctx, filter, bson.M{"$set": update})
	return err
}

func (er *ExportRepository) GetExportStats(ctx context.Context, startDate, endDate time.Time) (*models.ExportStats, error) {
	filter := bson.M{
		"createdAt": bson.M{"$gte": startDate, "$lt": endDate},
		"isDeleted": bson.M{"$ne": true},
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{
			"_id":             nil,
			"totalExports":    bson.M{"$sum": 1},
			"completedCount":  bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$status", "completed"}}, 1, 0}}},
			"processingCount": bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$status", "processing"}}, 1, 0}}},
			"failedCount":     bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$status", "failed"}}, 1, 0}}},
			"totalSize":       bson.M{"$sum": "$fileSize"},
			"totalMessages":   bson.M{"$sum": "$messageCount"},
			"formats":         bson.M{"$push": "$format"},
		}}},
	}

	cursor, err := er.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalExports    int64    `bson:"totalExports"`
		CompletedCount  int64    `bson:"completedCount"`
		ProcessingCount int64    `bson:"processingCount"`
		FailedCount     int64    `bson:"failedCount"`
		TotalSize       int64    `bson:"totalSize"`
		TotalMessages   int64    `bson:"totalMessages"`
		Formats         []string `bson:"formats"`
	}

	if cursor.Next(ctx) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}

	// Count formats
	formatCounts := make(map[string]int64)
	for _, format := range result.Formats {
		formatCounts[format]++
	}

	return &models.ExportStats{
		TotalExports:    result.TotalExports,
		CompletedCount:  result.CompletedCount,
		ProcessingCount: result.ProcessingCount,
		FailedCount:     result.FailedCount,
		TotalSize:       result.TotalSize,
		TotalMessages:   result.TotalMessages,
		FormatCounts:    formatCounts,
	}, nil
}

func (er *ExportRepository) GetActiveExportsCount(ctx context.Context, userID string) (int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"status":    "processing",
		"isDeleted": bson.M{"$ne": true},
	}

	return er.collection.CountDocuments(ctx, filter)
}

func (er *ExportRepository) ExtendExpiry(ctx context.Context, id string, newExpiryTime time.Time) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid export ID")
	}

	update := bson.M{
		"expiresAt": newExpiryTime,
		"updatedAt": time.Now(),
	}

	result, err := er.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":       objectID,
			"status":    "completed",
			"isDeleted": bson.M{"$ne": true},
		},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("export not found or not completed")
	}

	return nil
}
