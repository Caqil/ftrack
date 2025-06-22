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

type ReportRepository struct {
	collection *mongo.Collection
}

func NewReportRepository(db *mongo.Database) *ReportRepository {
	return &ReportRepository{
		collection: db.Collection("message_reports"),
	}
}

func (rr *ReportRepository) Create(ctx context.Context, report *models.MessageReport) error {
	report.ID = primitive.NewObjectID()
	report.CreatedAt = time.Now()
	report.UpdatedAt = time.Now()
	if report.Status == "" {
		report.Status = "pending"
	}

	_, err := rr.collection.InsertOne(ctx, report)
	return err
}

func (rr *ReportRepository) GetByID(ctx context.Context, id string) (*models.MessageReport, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid report ID")
	}

	var report models.MessageReport
	err = rr.collection.FindOne(ctx, bson.M{
		"_id":       objectID,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&report)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("report not found")
		}
		return nil, err
	}

	return &report, nil
}

func (rr *ReportRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid report ID")
	}

	update["updatedAt"] = time.Now()

	result, err := rr.collection.UpdateOne(
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
		return errors.New("report not found")
	}

	return nil
}

func (rr *ReportRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid report ID")
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	result, err := rr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("report not found")
	}

	return nil
}

func (rr *ReportRepository) GetReports(ctx context.Context, req models.GetReportsRequest) ([]models.MessageReport, int64, error) {
	filter := bson.M{
		"isDeleted": bson.M{"$ne": true},
	}

	if req.Status != "" {
		filter["status"] = req.Status
	}

	if req.Severity != "" {
		filter["severity"] = req.Severity
	}

	// Get total count
	total, err := rr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get reports
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := rr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var reports []models.MessageReport
	err = cursor.All(ctx, &reports)
	return reports, total, err
}

func (rr *ReportRepository) ReportExists(ctx context.Context, messageID, userID string) (bool, error) {
	messageObjectID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return false, errors.New("invalid message ID")
	}

	reportedByObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID")
	}

	filter := bson.M{
		"messageId":  messageObjectID,
		"reportedBy": reportedByObjectID,
		"isDeleted":  bson.M{"$ne": true},
	}

	count, err := rr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (rr *ReportRepository) GetReportsByMessage(ctx context.Context, messageID string) ([]models.MessageReport, error) {
	messageObjectID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return nil, errors.New("invalid message ID")
	}

	filter := bson.M{
		"messageId": messageObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	opts := options.Find().SetSort(bson.D{{"createdAt", -1}})

	cursor, err := rr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reports []models.MessageReport
	err = cursor.All(ctx, &reports)
	return reports, err
}

func (rr *ReportRepository) GetReportsByUser(ctx context.Context, userID string, page, pageSize int) ([]models.MessageReport, int64, error) {
	reportedByObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"reportedBy": reportedByObjectID,
		"isDeleted":  bson.M{"$ne": true},
	}

	// Get total count
	total, err := rr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get reports
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := rr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var reports []models.MessageReport
	err = cursor.All(ctx, &reports)
	return reports, total, err
}

func (rr *ReportRepository) GetPendingReports(ctx context.Context, severity string) ([]models.MessageReport, error) {
	filter := bson.M{
		"status":    "pending",
		"isDeleted": bson.M{"$ne": true},
	}

	if severity != "" {
		filter["severity"] = severity
	}

	opts := options.Find().SetSort(bson.D{{"createdAt", 1}}) // Oldest first

	cursor, err := rr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reports []models.MessageReport
	err = cursor.All(ctx, &reports)
	return reports, err
}

func (rr *ReportRepository) GetReportStats(ctx context.Context, startDate, endDate time.Time) (*models.ReportStats, error) {
	filter := bson.M{
		"createdAt": bson.M{"$gte": startDate, "$lt": endDate},
		"isDeleted": bson.M{"$ne": true},
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: filter}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":            nil,
			"totalReports":   bson.M{"$sum": 1},
			"pendingCount":   bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$status", "pending"}}, 1, 0}}},
			"reviewedCount":  bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$status", "reviewed"}}, 1, 0}}},
			"resolvedCount":  bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$status", "resolved"}}, 1, 0}}},
			"highSeverity":   bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$severity", "high"}}, 1, 0}}},
			"mediumSeverity": bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$severity", "medium"}}, 1, 0}}},
			"lowSeverity":    bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$severity", "low"}}, 1, 0}}},
			"reasonStats":    bson.M{"$push": "$reason"},
		}}},
	}

	cursor, err := rr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalReports   int64    `bson:"totalReports"`
		PendingCount   int64    `bson:"pendingCount"`
		ReviewedCount  int64    `bson:"reviewedCount"`
		ResolvedCount  int64    `bson:"resolvedCount"`
		HighSeverity   int64    `bson:"highSeverity"`
		MediumSeverity int64    `bson:"mediumSeverity"`
		LowSeverity    int64    `bson:"lowSeverity"`
		ReasonStats    []string `bson:"reasonStats"`
	}

	if cursor.Next(ctx) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}

	// Count reasons
	reasonCounts := make(map[string]int64)
	for _, reason := range result.ReasonStats {
		reasonCounts[reason]++
	}

	return &models.ReportStats{
		TotalReports:   result.TotalReports,
		PendingCount:   result.PendingCount,
		ReviewedCount:  result.ReviewedCount,
		ResolvedCount:  result.ResolvedCount,
		HighSeverity:   result.HighSeverity,
		MediumSeverity: result.MediumSeverity,
		LowSeverity:    result.LowSeverity,
		ReasonCounts:   reasonCounts,
	}, nil
}

func (rr *ReportRepository) BulkUpdateStatus(ctx context.Context, reportIDs []string, status string, reviewedBy string) error {
	objectIDs := make([]primitive.ObjectID, 0, len(reportIDs))
	for _, id := range reportIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs = append(objectIDs, objectID)
	}

	if len(objectIDs) == 0 {
		return errors.New("no valid report IDs")
	}

	update := bson.M{
		"status":    status,
		"updatedAt": time.Now(),
	}

	if reviewedBy != "" {
		reviewedByObjectID, err := primitive.ObjectIDFromHex(reviewedBy)
		if err == nil {
			update["reviewedBy"] = reviewedByObjectID
			update["reviewedAt"] = time.Now()
		}
	}

	_, err := rr.collection.UpdateMany(
		ctx,
		bson.M{"_id": bson.M{"$in": objectIDs}},
		bson.M{"$set": update},
	)

	return err
}

func (rr *ReportRepository) CleanupOldReports(ctx context.Context, olderThan time.Time) error {
	filter := bson.M{
		"status":    bson.M{"$in": []string{"resolved", "dismissed"}},
		"updatedAt": bson.M{"$lt": olderThan},
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	_, err := rr.collection.UpdateMany(ctx, filter, bson.M{"$set": update})
	return err
}
