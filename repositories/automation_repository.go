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

type AutomationRepository struct {
	collection *mongo.Collection
}

func NewAutomationRepository(db *mongo.Database) *AutomationRepository {
	return &AutomationRepository{
		collection: db.Collection("automation_rules"),
	}
}

func (ar *AutomationRepository) Create(ctx context.Context, rule *models.AutomationRule) error {
	rule.ID = primitive.NewObjectID()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	if rule.TriggerCount == 0 {
		rule.TriggerCount = 0
	}

	_, err := ar.collection.InsertOne(ctx, rule)
	return err
}

func (ar *AutomationRepository) GetByID(ctx context.Context, id string) (*models.AutomationRule, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid automation rule ID")
	}

	var rule models.AutomationRule
	err = ar.collection.FindOne(ctx, bson.M{
		"_id":       objectID,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&rule)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("automation rule not found")
		}
		return nil, err
	}

	return &rule, nil
}

func (ar *AutomationRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid automation rule ID")
	}

	update["updatedAt"] = time.Now()

	result, err := ar.collection.UpdateOne(
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
		return errors.New("automation rule not found")
	}

	return nil
}

func (ar *AutomationRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid automation rule ID")
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	result, err := ar.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("automation rule not found")
	}

	return nil
}

func (ar *AutomationRepository) GetUserRules(ctx context.Context, userID string, req models.GetAutomationRulesRequest) ([]models.AutomationRule, int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	if req.RuleType != "" {
		filter["type"] = req.RuleType
	}

	if req.Status != "" {
		if req.Status == "active" {
			filter["isActive"] = true
		} else if req.Status == "inactive" {
			filter["isActive"] = false
		}
	}

	// Get total count
	total, err := ar.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get rules
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := ar.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var rules []models.AutomationRule
	err = cursor.All(ctx, &rules)
	return rules, total, err
}

func (ar *AutomationRepository) GetUserRuleCount(ctx context.Context, userID string) (int64, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"isDeleted": bson.M{"$ne": true},
	}

	return ar.collection.CountDocuments(ctx, filter)
}

func (ar *AutomationRepository) GetActiveRulesForCircle(ctx context.Context, circleID string) ([]models.AutomationRule, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	filter := bson.M{
		"$or": []bson.M{
			{"circleId": circleObjectID},
			{"circleId": bson.M{"$exists": false}}, // Global rules
		},
		"isActive":  true,
		"isDeleted": bson.M{"$ne": true},
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}) // Older rules first

	cursor, err := ar.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []models.AutomationRule
	err = cursor.All(ctx, &rules)
	return rules, err
}

func (ar *AutomationRepository) IncrementTriggerCount(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid automation rule ID")
	}

	now := time.Now()
	_, err = ar.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$inc": bson.M{"triggerCount": 1},
			"$set": bson.M{
				"lastTriggered": &now,
				"updatedAt":     now,
			},
		},
	)

	return err
}

func (ar *AutomationRepository) GetRulesByType(ctx context.Context, ruleType string, isActive bool) ([]models.AutomationRule, error) {
	filter := bson.M{
		"type":      ruleType,
		"isActive":  isActive,
		"isDeleted": bson.M{"$ne": true},
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})

	cursor, err := ar.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []models.AutomationRule
	err = cursor.All(ctx, &rules)
	return rules, err
}

func (ar *AutomationRepository) GetMostTriggeredRules(ctx context.Context, limit int, timeframe time.Duration) ([]models.AutomationRule, error) {
	since := time.Now().Add(-timeframe)

	filter := bson.M{
		"isActive":      true,
		"isDeleted":     bson.M{"$ne": true},
		"lastTriggered": bson.M{"$gte": since},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "triggerCount", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := ar.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []models.AutomationRule
	err = cursor.All(ctx, &rules)
	return rules, err
}

func (ar *AutomationRepository) GetRuleStats(ctx context.Context, userID string, startDate, endDate time.Time) (*models.AutomationStats, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":    userObjectID,
		"createdAt": bson.M{"$gte": startDate, "$lt": endDate},
		"isDeleted": bson.M{"$ne": true},
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{
			"_id":           nil,
			"totalRules":    bson.M{"$sum": 1},
			"activeRules":   bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$isActive", true}}, 1, 0}}},
			"inactiveRules": bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$isActive", false}}, 1, 0}}},
			"totalTriggers": bson.M{"$sum": "$triggerCount"},
			"ruleTypes":     bson.M{"$push": "$type"},
		}}},
	}

	cursor, err := ar.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalRules    int64    `bson:"totalRules"`
		ActiveRules   int64    `bson:"activeRules"`
		InactiveRules int64    `bson:"inactiveRules"`
		TotalTriggers int64    `bson:"totalTriggers"`
		RuleTypes     []string `bson:"ruleTypes"`
	}

	if cursor.Next(ctx) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}

	// Count rule types
	typeCounts := make(map[string]int64)
	for _, ruleType := range result.RuleTypes {
		typeCounts[ruleType]++
	}

	return &models.AutomationStats{
		TotalRules:    result.TotalRules,
		ActiveRules:   result.ActiveRules,
		InactiveRules: result.InactiveRules,
		TotalTriggers: result.TotalTriggers,
		TypeCounts:    typeCounts,
	}, nil
}

func (ar *AutomationRepository) BulkToggleStatus(ctx context.Context, ruleIDs []string, isActive bool) error {
	objectIDs := make([]primitive.ObjectID, 0, len(ruleIDs))
	for _, id := range ruleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs = append(objectIDs, objectID)
	}

	if len(objectIDs) == 0 {
		return errors.New("no valid rule IDs")
	}

	update := bson.M{
		"isActive":  isActive,
		"updatedAt": time.Now(),
	}

	_, err := ar.collection.UpdateMany(
		ctx,
		bson.M{"_id": bson.M{"$in": objectIDs}},
		bson.M{"$set": update},
	)

	return err
}

func (ar *AutomationRepository) CleanupInactiveRules(ctx context.Context, inactiveDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -inactiveDays)

	filter := bson.M{
		"isActive": false,
		"$or": []bson.M{
			{"lastTriggered": bson.M{"$lt": cutoffDate}},
			{"lastTriggered": bson.M{"$exists": false}, "createdAt": bson.M{"$lt": cutoffDate}},
		},
		"isDeleted": bson.M{"$ne": true},
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	_, err := ar.collection.UpdateMany(ctx, filter, bson.M{"$set": update})
	return err
}

func (ar *AutomationRepository) GetScheduledRules(ctx context.Context, currentTime time.Time) ([]models.AutomationRule, error) {
	// This would be for time-based triggers
	filter := bson.M{
		"type":      "schedule",
		"isActive":  true,
		"isDeleted": bson.M{"$ne": true},
	}

	// Add time-based conditions here based on your schedule logic
	// For example, rules that should trigger at specific times

	cursor, err := ar.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rules []models.AutomationRule
	err = cursor.All(ctx, &rules)
	return rules, err
}
