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

type TemplateRepository struct {
	collection *mongo.Collection
}

func NewTemplateRepository(db *mongo.Database) *TemplateRepository {
	return &TemplateRepository{
		collection: db.Collection("message_templates"),
	}
}

func (tr *TemplateRepository) Create(ctx context.Context, template *models.MessageTemplate) error {
	template.ID = primitive.NewObjectID()
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	if template.UsageCount == 0 {
		template.UsageCount = 0
	}

	_, err := tr.collection.InsertOne(ctx, template)
	return err
}

func (tr *TemplateRepository) GetByID(ctx context.Context, id string) (*models.MessageTemplate, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid template ID")
	}

	var template models.MessageTemplate
	err = tr.collection.FindOne(ctx, bson.M{
		"_id":       objectID,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&template)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("template not found")
		}
		return nil, err
	}

	return &template, nil
}

func (tr *TemplateRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid template ID")
	}

	update["updatedAt"] = time.Now()

	result, err := tr.collection.UpdateOne(
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
		return errors.New("template not found")
	}

	return nil
}

func (tr *TemplateRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid template ID")
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	result, err := tr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("template not found")
	}

	return nil
}

func (tr *TemplateRepository) GetUserTemplates(ctx context.Context, userID string, req models.GetTemplatesRequest) ([]models.MessageTemplate, int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"$or": []bson.M{
			{"userId": userObjectID},
			{"isPublic": true},
		},
		"isDeleted": bson.M{"$ne": true},
	}

	if req.Category != "" {
		filter["category"] = req.Category
	}

	// Get total count
	total, err := tr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get templates
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{"usageCount", -1}, {"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := tr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var templates []models.MessageTemplate
	err = cursor.All(ctx, &templates)
	return templates, total, err
}

func (tr *TemplateRepository) NameExists(ctx context.Context, userID, name string) (bool, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"name":      name,
		"isDeleted": bson.M{"$ne": true},
	}

	count, err := tr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (tr *TemplateRepository) NameExistsExcluding(ctx context.Context, userID, name, excludeID string) (bool, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID")
	}

	excludeObjectID, err := primitive.ObjectIDFromHex(excludeID)
	if err != nil {
		return false, errors.New("invalid template ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"name":      name,
		"_id":       bson.M{"$ne": excludeObjectID},
		"isDeleted": bson.M{"$ne": true},
	}

	count, err := tr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (tr *TemplateRepository) IncrementUsage(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid template ID")
	}

	_, err = tr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$inc": bson.M{"usageCount": 1},
			"$set": bson.M{"updatedAt": time.Now()},
		},
	)

	return err
}

func (tr *TemplateRepository) GetPopularTemplates(ctx context.Context, limit int) ([]models.MessageTemplate, error) {
	filter := bson.M{
		"isPublic":  true,
		"isDeleted": bson.M{"$ne": true},
	}

	opts := options.Find().
		SetSort(bson.D{{"usageCount", -1}}).
		SetLimit(int64(limit))

	cursor, err := tr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []models.MessageTemplate
	err = cursor.All(ctx, &templates)
	return templates, err
}

func (tr *TemplateRepository) GetByCategory(ctx context.Context, category string, page, pageSize int) ([]models.MessageTemplate, int64, error) {
	filter := bson.M{
		"category":  category,
		"isPublic":  true,
		"isDeleted": bson.M{"$ne": true},
	}

	// Get total count
	total, err := tr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get templates
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"usageCount", -1}, {"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := tr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var templates []models.MessageTemplate
	err = cursor.All(ctx, &templates)
	return templates, total, err
}
