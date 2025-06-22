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
	collection        *mongo.Collection
	forwardCollection *mongo.Collection
	db                *mongo.Database
}

func NewMessageRepository(db *mongo.Database) *MessageRepository {
	return &MessageRepository{
		collection:        db.Collection("messages"),
		forwardCollection: db.Collection("message_forwards"),
		db:                db,
	}
}

// =============================================================================
// BASIC CRUD OPERATIONS
// =============================================================================

func (mr *MessageRepository) Create(ctx context.Context, message *models.Message) error {
	message.ID = primitive.NewObjectID()
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()
	if message.Status == "" {
		message.Status = "sent"
	}

	_, err := mr.collection.InsertOne(ctx, message)
	return err
}

func (mr *MessageRepository) GetByID(ctx context.Context, id string) (*models.Message, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid message ID")
	}

	var message models.Message
	err = mr.collection.FindOne(ctx, bson.M{
		"_id":       objectID,
		"isDeleted": bson.M{"$ne": true},
	}).Decode(&message)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("message not found")
		}
		return nil, err
	}

	return &message, nil
}

func (mr *MessageRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid message ID")
	}

	update["updatedAt"] = time.Now()

	result, err := mr.collection.UpdateOne(
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
		return errors.New("message not found")
	}

	return nil
}

func (mr *MessageRepository) Delete(ctx context.Context, id string) error {
	return mr.SoftDelete(ctx, id)
}

func (mr *MessageRepository) SoftDelete(ctx context.Context, id string) error {
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

func (mr *MessageRepository) Hide(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid message ID")
	}

	update := bson.M{
		"isHidden":  true,
		"hiddenAt":  time.Now(),
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

// =============================================================================
// MESSAGE RETRIEVAL
// =============================================================================

func (mr *MessageRepository) GetCircleMessages(ctx context.Context, circleID string, page, pageSize int) ([]models.Message, error) {
	objectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	filter := bson.M{
		"circleId":  objectID,
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
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

func (mr *MessageRepository) GetCircleMessagesPaginated(ctx context.Context, req models.GetMessagesRequest) ([]models.Message, int64, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(req.CircleID)
	if err != nil {
		return nil, 0, errors.New("invalid circle ID")
	}

	filter := bson.M{
		"circleId":  circleObjectID,
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	// Add cursor-based pagination if before/after specified
	if req.Before != "" {
		beforeObjectID, err := primitive.ObjectIDFromHex(req.Before)
		if err == nil {
			filter["_id"] = bson.M{"$lt": beforeObjectID}
		}
	}

	if req.After != "" {
		afterObjectID, err := primitive.ObjectIDFromHex(req.After)
		if err == nil {
			filter["_id"] = bson.M{"$gt": afterObjectID}
		}
	}

	// Get total count
	total, err := mr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get messages
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	return messages, total, err
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
		"isHidden":  bson.M{"$ne": true},
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

// =============================================================================
// REPLIES AND THREADING
// =============================================================================

func (mr *MessageRepository) GetReplies(ctx context.Context, messageID string, page, pageSize int) ([]models.Message, int64, error) {
	replyToObjectID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return nil, 0, errors.New("invalid message ID")
	}

	filter := bson.M{
		"replyTo":   replyToObjectID,
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	// Get total count
	total, err := mr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get replies
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", 1}}). // Replies in chronological order
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var replies []models.Message
	err = cursor.All(ctx, &replies)
	return replies, total, err
}

func (mr *MessageRepository) IncrementReplyCount(ctx context.Context, messageID string) error {
	objectID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return errors.New("invalid message ID")
	}

	_, err = mr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$inc": bson.M{"replyCount": 1},
			"$set": bson.M{"updatedAt": time.Now()},
		},
	)

	return err
}

// =============================================================================
// REACTIONS
// =============================================================================

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

	// Remove existing reaction from this user with same emoji first
	_, err = mr.collection.UpdateOne(
		ctx,
		bson.M{"_id": messageObjectID},
		bson.M{"$pull": bson.M{"reactions": bson.M{
			"userId": userObjectID,
			"emoji":  emoji,
		}}},
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

func (mr *MessageRepository) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
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
			"$pull": bson.M{"reactions": bson.M{
				"userId": userObjectID,
				"emoji":  emoji,
			}},
			"$set": bson.M{"updatedAt": time.Now()},
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

func (mr *MessageRepository) GetReactionUsers(ctx context.Context, messageID, emoji string) ([]models.UserInfo, error) {
	messageObjectID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return nil, errors.New("invalid message ID")
	}

	// Use aggregation to get user details for reactions
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"_id": messageObjectID}}},
		{{"$unwind", "$reactions"}},
		{{"$match", bson.M{"reactions.emoji": emoji}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "reactions.userId",
			"foreignField": "_id",
			"as":           "user",
		}}},
		{{"$unwind", "$user"}},
		{{Key: "$project", Value: bson.M{
			"_id":       0,
			"id":        "$user._id",
			"firstName": "$user.firstName",
			"lastName":  "$user.lastName",
			"avatar":    "$user.avatar",
		}}},
	}

	cursor, err := mr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.UserInfo
	err = cursor.All(ctx, &users)
	return users, err
}

// =============================================================================
// READ STATUS AND DELIVERY
// =============================================================================

func (mr *MessageRepository) MarkAsRead(ctx context.Context, messageIDs []string, userID string) error {
	objectIDs := make([]primitive.ObjectID, 0, len(messageIDs))
	for _, id := range messageIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs = append(objectIDs, objectID)
	}

	if len(objectIDs) == 0 {
		return errors.New("no valid message IDs")
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

func (mr *MessageRepository) MarkAsUnread(ctx context.Context, messageID, userID string) error {
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
			"$pull": bson.M{"readBy": bson.M{"userId": userObjectID}},
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

func (mr *MessageRepository) BulkMarkAsRead(ctx context.Context, messageIDs []string, userID string) (int, error) {
	objectIDs := make([]primitive.ObjectID, 0, len(messageIDs))
	for _, id := range messageIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs = append(objectIDs, objectID)
	}

	if len(objectIDs) == 0 {
		return 0, errors.New("no valid message IDs")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, errors.New("invalid user ID")
	}

	readStatus := models.MessageReadStatus{
		UserID: userObjectID,
		ReadAt: time.Now(),
	}

	filter := bson.M{
		"_id":           bson.M{"$in": objectIDs},
		"readBy.userId": bson.M{"$ne": userObjectID}, // Only update if not already read
	}

	update := bson.M{
		"$addToSet": bson.M{"readBy": readStatus},
		"$set":      bson.M{"updatedAt": time.Now()},
	}

	result, err := mr.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return int(result.ModifiedCount), nil
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
		"isHidden":      bson.M{"$ne": true},
	}

	count, err := mr.collection.CountDocuments(ctx, filter)
	return count, err
}

func (mr *MessageRepository) GetDeliveryStatus(ctx context.Context, messageID string) (*models.DeliveryStatusResponse, error) {
	messageObjectID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return nil, errors.New("invalid message ID")
	}

	// Use aggregation to get delivery status with user details
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"_id": messageObjectID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "circles",
			"localField":   "circleId",
			"foreignField": "_id",
			"as":           "circle",
		}}},
		{{Key: "$unwind", Value: "$circle"}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "circle.members",
			"foreignField": "_id",
			"as":           "members",
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":      1,
			"status":   1,
			"readBy":   1,
			"members":  1,
			"senderId": 1,
		}}},
	}

	cursor, err := mr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		ID       primitive.ObjectID         `bson:"_id"`
		Status   string                     `bson:"status"`
		ReadBy   []models.MessageReadStatus `bson:"readBy"`
		Members  []models.User              `bson:"members"`
		SenderID primitive.ObjectID         `bson:"senderId"`
	}

	if cursor.Next(ctx) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("message not found")
	}

	// Calculate delivery statistics
	totalRecipients := 0
	deliveredCount := 0
	details := make([]models.DeliveryStatusDetail, 0)

	readByMap := make(map[string]time.Time)
	for _, readStatus := range result.ReadBy {
		readByMap[readStatus.UserID.Hex()] = readStatus.ReadAt
	}

	for _, member := range result.Members {
		// Skip sender
		if member.ID == result.SenderID {
			continue
		}

		totalRecipients++
		detail := models.DeliveryStatusDetail{
			UserID: member.ID.Hex(),
			Status: "sent",
		}

		if readTime, isRead := readByMap[member.ID.Hex()]; isRead {
			detail.Status = "read"
			detail.DeliveredAt = readTime
			deliveredCount++
		}

		details = append(details, detail)
	}

	return &models.DeliveryStatusResponse{
		MessageID: messageID,
		Status:    result.Status,
		Delivered: deliveredCount,
		Total:     totalRecipients,
		Details:   details,
	}, nil
}

func (mr *MessageRepository) GetReadReceipts(ctx context.Context, messageID string) (*models.ReadReceiptsResponse, error) {
	messageObjectID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return nil, errors.New("invalid message ID")
	}

	// Use aggregation to get read receipts with user details
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"_id": messageObjectID}}},
		{{Key: "$unwind", Value: "$readBy"}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "readBy.userId",
			"foreignField": "_id",
			"as":           "user",
		}}},
		{{Key: "$unwind", Value: "$user"}},
		{{Key: "$project", Value: bson.M{
			"_id":       0,
			"userId":    "$user._id",
			"firstName": "$user.firstName",
			"lastName":  "$user.lastName",
			"avatar":    "$user.avatar",
			"readAt":    "$readBy.readAt",
		}}},
		{{"$sort", bson.M{"readAt": 1}}},
	}

	cursor, err := mr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var readReceipts []models.ReadReceiptInfo
	err = cursor.All(ctx, &readReceipts)
	if err != nil {
		return nil, err
	}

	// Get total circle members count
	message, err := mr.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	totalMembers, err := mr.getCircleMemberCount(ctx, message.CircleID.Hex(), message.SenderID.Hex())
	if err != nil {
		totalMembers = len(readReceipts) // Fallback
	}

	return &models.ReadReceiptsResponse{
		MessageID: messageID,
		ReadBy:    readReceipts,
		ReadCount: len(readReceipts),
		Total:     totalMembers,
	}, nil
}

// =============================================================================
// FORWARDING
// =============================================================================

func (mr *MessageRepository) RecordForward(ctx context.Context, forward *models.MessageForward) error {
	forward.ID = primitive.NewObjectID()
	_, err := mr.forwardCollection.InsertOne(ctx, forward)
	return err
}

func (mr *MessageRepository) GetForwardHistory(ctx context.Context, messageID string) ([]models.ForwardInfo, error) {
	filter := bson.M{"originalMessageId": messageID}
	opts := options.Find().SetSort(bson.D{{Key: "forwardedAt", Value: -1}})

	cursor, err := mr.forwardCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var forwards []models.MessageForward
	err = cursor.All(ctx, &forwards)
	if err != nil {
		return nil, err
	}

	// Convert to ForwardInfo
	forwardInfos := make([]models.ForwardInfo, len(forwards))
	for i, forward := range forwards {
		forwardInfos[i] = models.ForwardInfo{
			ForwardedTo: forward.ForwardedTo,
			ForwardedBy: forward.ForwardedBy,
			ForwardType: forward.ForwardType,
			Comment:     forward.Comment,
			ForwardedAt: forward.ForwardedAt,
		}
	}

	return forwardInfos, nil
}

// =============================================================================
// MEDIA ACCESS
// =============================================================================

func (mr *MessageRepository) CheckMediaAccess(ctx context.Context, mediaID string, circleIDs []string) (bool, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	filter := bson.M{
		"circleId":  bson.M{"$in": circleObjectIDs},
		"media.id":  mediaID,
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	count, err := mr.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// =============================================================================
// ANALYTICS AND STATISTICS
// =============================================================================

func (mr *MessageRepository) GetMessageStats(ctx context.Context, circleIDs []string, startDate, endDate time.Time) (*models.RawMessageStats, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	filter := bson.M{
		"circleId":  bson.M{"$in": circleObjectIDs},
		"createdAt": bson.M{"$gte": startDate, "$lt": endDate},
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	// Aggregation pipeline for comprehensive stats
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{
			Key: "$group",
			Value: bson.M{
				"_id":            nil,
				"totalMessages":  bson.M{"$sum": 1},
				"messagesByType": bson.M{"$push": "$type"},
				"totalReactions": bson.M{"$sum": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$reactions", []interface{}{}}}}},
				"totalReplies":   bson.M{"$sum": bson.M{"$ifNull": []interface{}{"$replyCount", 0}}},
				"senders":        bson.M{"$addToSet": "$senderId"},
				"hourStats":      bson.M{"$push": bson.M{"$hour": "$createdAt"}},
				"dayStats":       bson.M{"$push": bson.M{"$dayOfWeek": "$createdAt"}},
			},
		}},
		{{
			Key: "$project",
			Value: bson.M{
				"totalMessages":  1,
				"totalReactions": 1,
				"totalReplies":   1,
				"activeUsers":    bson.M{"$size": "$senders"},
				"messagesByType": bson.M{
					"$arrayToObject": bson.M{
						"$map": bson.M{
							"input": bson.M{
								"$setUnion": []interface{}{"$messagesByType"},
							},
							"as": "type",
							"in": bson.M{
								"k": "$$type",
								"v": bson.M{
									"$size": bson.M{
										"$filter": bson.M{
											"input": "$messagesByType",
											"cond":  bson.M{"$eq": []interface{}{"$$this", "$$type"}},
										},
									},
								},
							},
						},
					},
				},
				"busiestHour": bson.M{
					"$arrayElemAt": []interface{}{
						bson.M{
							"$map": bson.M{
								"input": bson.M{"$range": []interface{}{0, 24}},
								"as":    "hour",
								"in": bson.M{
									"hour": "$$hour",
									"count": bson.M{
										"$size": bson.M{
											"$filter": bson.M{
												"input": "$hourStats",
												"cond":  bson.M{"$eq": []interface{}{"$$this", "$$hour"}},
											},
										},
									},
								},
							},
						},
						0,
					},
				},
			},
		}},
	}

	cursor, err := mr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalMessages  int64            `bson:"totalMessages"`
		MessagesByType map[string]int64 `bson:"messagesByType"`
		TotalReactions int64            `bson:"totalReactions"`
		TotalReplies   int64            `bson:"totalReplies"`
		ActiveUsers    int64            `bson:"activeUsers"`
		BusiestHour    struct {
			Hour  int `bson:"hour"`
			Count int `bson:"count"`
		} `bson:"busiestHour"`
	}

	if cursor.Next(ctx) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}

	// Calculate busiest day (simplified)
	busiestDay := "Monday" // This would need more complex aggregation

	return &models.RawMessageStats{
		TotalMessages:  result.TotalMessages,
		MessagesByType: result.MessagesByType,
		TotalReactions: result.TotalReactions,
		TotalReplies:   result.TotalReplies,
		ActiveUsers:    result.ActiveUsers,
		BusiestHour:    result.BusiestHour.Hour,
		BusiestDay:     busiestDay,
	}, nil
}

func (mr *MessageRepository) GetMessageActivity(ctx context.Context, circleIDs []string, startDate, endDate time.Time, granularity string) ([]models.ActivityDataPoint, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	var groupBy bson.M
	if granularity == "hour" {
		groupBy = bson.M{
			"year":  bson.M{"$year": "$createdAt"},
			"month": bson.M{"$month": "$createdAt"},
			"day":   bson.M{"$dayOfMonth": "$createdAt"},
			"hour":  bson.M{"$hour": "$createdAt"},
		}
		dateFormat = "%Y-%m-%d %H:00:00"
	} else {
		groupBy = bson.M{
			"year":  bson.M{"$year": "$createdAt"},
			"month": bson.M{"$month": "$createdAt"},
			"day":   bson.M{"$dayOfMonth": "$createdAt"},
		}
		dateFormat = "%Y-%m-%d"
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"circleId":  bson.M{"$in": circleObjectIDs},
			"createdAt": bson.M{"$gte": startDate, "$lt": endDate},
			"isDeleted": bson.M{"$ne": true},
			"isHidden":  bson.M{"$ne": true},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":      groupBy,
			"messages": bson.M{"$sum": 1},
			"users":    bson.M{"$addToSet": "$senderId"},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":       0,
			"timestamp": bson.M{"$dateFromParts": "$_id"},
			"messages":  1,
			"users":     bson.M{"$size": "$users"},
		}}},
		{{Key: "$sort", Value: bson.M{"timestamp": 1}}},
	}

	cursor, err := mr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var dataPoints []models.ActivityDataPoint
	err = cursor.All(ctx, &dataPoints)
	return dataPoints, err
}

func (mr *MessageRepository) GetMessageTrends(ctx context.Context, circleIDs []string, startDate, endDate time.Time, trendType string) ([]models.TrendDataPoint, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	var pipeline mongo.Pipeline

	switch trendType {
	case "volume":
		pipeline = mongo.Pipeline{
			{{Key: "$match", Value: bson.M{
				"circleId":  bson.M{"$in": circleObjectIDs},
				"createdAt": bson.M{"$gte": startDate, "$lt": endDate},
				"isDeleted": bson.M{"$ne": true},
				"isHidden":  bson.M{"$ne": true},
			}}},
			{{Key: "$group", Value: bson.M{
				"_id": bson.M{
					"year":  bson.M{"$year": "$createdAt"},
					"month": bson.M{"$month": "$createdAt"},
					"day":   bson.M{"$dayOfMonth": "$createdAt"},
				},
				"value": bson.M{"$sum": 1},
			}}},
			{{Key: "$project", Value: bson.M{
				"_id":   0,
				"date":  bson.M{"$dateFromParts": "$_id"},
				"value": 1,
			}}},
			{{Key: "$sort", Value: bson.M{"date": 1}}},
		}

	case "engagement":
		pipeline = mongo.Pipeline{
			{{Key: "$match", Value: bson.M{
				"circleId":  bson.M{"$in": circleObjectIDs},
				"createdAt": bson.M{"$gte": startDate, "$lt": endDate},
				"isDeleted": bson.M{"$ne": true},
				"isHidden":  bson.M{"$ne": true},
			}}},
			{{Key: "$group", Value: bson.M{
				"_id": bson.M{
					"year":  bson.M{"$year": "$createdAt"},
					"month": bson.M{"$month": "$createdAt"},
					"day":   bson.M{"$dayOfMonth": "$createdAt"},
				},
				"reactions": bson.M{"$sum": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$reactions", []interface{}{}}}}},
				"replies":   bson.M{"$sum": bson.M{"$ifNull": []interface{}{"$replyCount", 0}}},
				"messages":  bson.M{"$sum": 1},
			}}},
			{{Key: "$project", Value: bson.M{
				"_id":   0,
				"date":  bson.M{"$dateFromParts": "$_id"},
				"value": bson.M{"$divide": []interface{}{bson.M{"$add": []interface{}{"$reactions", "$replies"}}, "$messages"}},
			}}},
			{{Key: "$sort", Value: bson.M{"date": 1}}},
		}

	default:
		return nil, errors.New("invalid trend type")
	}

	cursor, err := mr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var trends []models.TrendDataPoint
	err = cursor.All(ctx, &trends)
	if err != nil {
		return nil, err
	}

	// Calculate change rates
	for i := 1; i < len(trends); i++ {
		if trends[i-1].Value > 0 {
			trends[i].Change = ((trends[i].Value - trends[i-1].Value) / trends[i-1].Value) * 100
		}
	}

	return trends, nil
}

func (mr *MessageRepository) GetPopularMessages(ctx context.Context, circleIDs []string, startDate, endDate time.Time, metric string, limit int) ([]models.PopularMessageInfo, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	var sortField string
	var scoreCalculation bson.M

	switch metric {
	case "reactions":
		sortField = "reactionCount"
		scoreCalculation = bson.M{"$size": bson.M{"$ifNull": []interface{}{"$reactions", []interface{}{}}}}
	case "replies":
		sortField = "replyCount"
		scoreCalculation = bson.M{"$ifNull": []interface{}{"$replyCount", 0}}
	case "views":
		sortField = "viewCount"
		scoreCalculation = bson.M{"$size": bson.M{"$ifNull": []interface{}{"$readBy", []interface{}{}}}}
	default:
		return nil, errors.New("invalid metric")
	}

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"circleId":  bson.M{"$in": circleObjectIDs},
			"createdAt": bson.M{"$gte": startDate, "$lt": endDate},
			"isDeleted": bson.M{"$ne": true},
			"isHidden":  bson.M{"$ne": true},
		}}},
		{{"$addFields", bson.M{
			"reactionCount": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$reactions", []interface{}{}}}},
			"replyCount":    bson.M{"$ifNull": []interface{}{"$replyCount", 0}},
			"viewCount":     bson.M{"$size": bson.M{"$ifNull": []interface{}{"$readBy", []interface{}{}}}},
			"score":         scoreCalculation,
		}}},
		{{"$sort", bson.M{sortField: -1}}},
		{{"$limit", limit}},
	}

	cursor, err := mr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		models.Message `bson:",inline"`
		ReactionCount  int     `bson:"reactionCount"`
		ReplyCount     int     `bson:"replyCount"`
		ViewCount      int     `bson:"viewCount"`
		Score          float64 `bson:"score"`
	}

	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}

	popularMessages := make([]models.PopularMessageInfo, len(results))
	for i, result := range results {
		popularMessages[i] = models.PopularMessageInfo{
			Message:       result.Message,
			Score:         result.Score,
			ReactionCount: result.ReactionCount,
			ReplyCount:    result.ReplyCount,
			ViewCount:     result.ViewCount,
		}
	}

	return popularMessages, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func (mr *MessageRepository) getCircleMemberCount(ctx context.Context, circleID, excludeUserID string) (int, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return 0, errors.New("invalid circle ID")
	}

	excludeUserObjectID, err := primitive.ObjectIDFromHex(excludeUserID)
	if err != nil {
		return 0, errors.New("invalid user ID")
	}

	// Get circle from circles collection
	circleCollection := mr.db.Collection("circles")
	var circle struct {
		Members []primitive.ObjectID `bson:"members"`
	}

	err = circleCollection.FindOne(ctx, bson.M{"_id": circleObjectID}).Decode(&circle)
	if err != nil {
		return 0, err
	}

	// Count members excluding the specified user
	count := 0
	for _, memberID := range circle.Members {
		if memberID != excludeUserObjectID {
			count++
		}
	}

	return count, nil
}

// Additional utility methods for complex queries

func (mr *MessageRepository) SearchMessages(ctx context.Context, query string, circleIDs []string, limit int) ([]models.Message, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	filter := bson.M{
		"circleId":  bson.M{"$in": circleObjectIDs},
		"content":   bson.M{"$regex": query, "$options": "i"},
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetLimit(int64(limit))

	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	return messages, err
}

func (mr *MessageRepository) GetMessagesByType(ctx context.Context, circleIDs []string, messageType string, limit int) ([]models.Message, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	filter := bson.M{
		"circleId":  bson.M{"$in": circleObjectIDs},
		"type":      messageType,
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetLimit(int64(limit))

	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	return messages, err
}

func (mr *MessageRepository) GetMessagesWithMedia(ctx context.Context, circleIDs []string, mediaType string, limit int) ([]models.Message, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	filter := bson.M{
		"circleId":  bson.M{"$in": circleObjectIDs},
		"media.url": bson.M{"$exists": true, "$ne": ""},
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	if mediaType != "" {
		filter["media.type"] = mediaType
	}

	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetLimit(int64(limit))

	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	return messages, err
}

// Batch operations

func (mr *MessageRepository) BulkDelete(ctx context.Context, messageIDs []string) error {
	objectIDs := make([]primitive.ObjectID, 0, len(messageIDs))
	for _, id := range messageIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs = append(objectIDs, objectID)
	}

	if len(objectIDs) == 0 {
		return errors.New("no valid message IDs")
	}

	update := bson.M{
		"isDeleted": true,
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	_, err := mr.collection.UpdateMany(
		ctx,
		bson.M{"_id": bson.M{"$in": objectIDs}},
		bson.M{"$set": update},
	)

	return err
}

func (mr *MessageRepository) BulkUpdateStatus(ctx context.Context, messageIDs []string, status string) error {
	objectIDs := make([]primitive.ObjectID, 0, len(messageIDs))
	for _, id := range messageIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs = append(objectIDs, objectID)
	}

	if len(objectIDs) == 0 {
		return errors.New("no valid message IDs")
	}

	update := bson.M{
		"status":    status,
		"updatedAt": time.Now(),
	}

	_, err := mr.collection.UpdateMany(
		ctx,
		bson.M{"_id": bson.M{"$in": objectIDs}},
		bson.M{"$set": update},
	)

	return err
}
