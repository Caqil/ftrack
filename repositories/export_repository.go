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
	db                      *mongo.Database
	dataExportsCollection   *mongo.Collection
	purgeRequestsCollection *mongo.Collection
}

func NewExportRepository(db *mongo.Database) *ExportRepository {
	return &ExportRepository{
		db:                      db,
		dataExportsCollection:   db.Collection("data_exports"),
		purgeRequestsCollection: db.Collection("purge_requests"),
	}
}

// Data Exports
func (er *ExportRepository) CreateExport(ctx context.Context, export *models.UserDataExport) error {
	export.ID = primitive.NewObjectID()
	export.CreatedAt = time.Now()

	_, err := er.dataExportsCollection.InsertOne(ctx, export)
	return err
}

func (er *ExportRepository) GetExport(ctx context.Context, exportID string) (*models.UserDataExport, error) {
	objectID, err := primitive.ObjectIDFromHex(exportID)
	if err != nil {
		return nil, errors.New("invalid export ID")
	}

	var export models.UserDataExport
	err = er.dataExportsCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&export)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("export not found")
		}
		return nil, err
	}

	return &export, nil
}

func (er *ExportRepository) GetLatestExport(ctx context.Context, userID string) (*models.UserDataExport, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	opts := options.FindOne().SetSort(bson.D{{"createdAt", -1}})
	var export models.UserDataExport
	err = er.dataExportsCollection.FindOne(ctx, bson.M{"userId": userObjectID}, opts).Decode(&export)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("no export found")
		}
		return nil, err
	}

	return &export, nil
}

func (er *ExportRepository) GetActiveExport(ctx context.Context, userID string) (*models.UserDataExport, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId": userObjectID,
		"status": bson.M{"$in": []string{"pending", "processing"}},
	}

	var export models.UserDataExport
	err = er.dataExportsCollection.FindOne(ctx, filter).Decode(&export)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No active export
		}
		return nil, err
	}

	return &export, nil
}

func (er *ExportRepository) UpdateExportStatus(ctx context.Context, exportID, status string) error {
	objectID, err := primitive.ObjectIDFromHex(exportID)
	if err != nil {
		return errors.New("invalid export ID")
	}

	update := bson.M{
		"status":    status,
		"updatedAt": time.Now(),
	}

	if status == "completed" {
		update["completedAt"] = time.Now()
	}

	result, err := er.dataExportsCollection.UpdateOne(
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

// Data Purge Requests
func (er *ExportRepository) CreatePurgeRequest(ctx context.Context, request *models.DataPurgeRequest) error {
	request.ID = primitive.NewObjectID()
	request.CreatedAt = time.Now()

	_, err := er.purgeRequestsCollection.InsertOne(ctx, request)
	return err
}

func (er *ExportRepository) GetPurgeRequest(ctx context.Context, userID string) (*models.DataPurgeRequest, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	opts := options.FindOne().SetSort(bson.D{{"createdAt", -1}})
	var request models.DataPurgeRequest
	err = er.purgeRequestsCollection.FindOne(ctx, bson.M{"userId": userObjectID}, opts).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No purge request
		}
		return nil, err
	}

	return &request, nil
}
