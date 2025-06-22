package repositories

import (
	"context"
	"errors"
	"time"

	"ftrack/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FriendRepository struct {
	db                       *mongo.Database
	friendRequestsCollection *mongo.Collection
	friendshipsCollection    *mongo.Collection
}

func NewFriendRepository(db *mongo.Database) *FriendRepository {
	return &FriendRepository{
		db:                       db,
		friendRequestsCollection: db.Collection("friend_requests"),
		friendshipsCollection:    db.Collection("friendships"),
	}
}

// Friend Requests
func (fr *FriendRepository) CreateFriendRequest(ctx context.Context, request *models.FriendRequest) error {
	request.ID = primitive.NewObjectID()
	request.CreatedAt = time.Now()

	_, err := fr.friendRequestsCollection.InsertOne(ctx, request)
	return err
}

func (fr *FriendRepository) GetFriendRequest(ctx context.Context, requestID string) (*models.FriendRequest, error) {
	objectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return nil, errors.New("invalid request ID")
	}

	var request models.FriendRequest
	err = fr.friendRequestsCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("request not found")
		}
		return nil, err
	}

	return &request, nil
}

func (fr *FriendRepository) GetFriendRequests(ctx context.Context, userID string, requestType string) ([]models.FriendRequest, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var filter bson.M
	switch requestType {
	case "sent":
		filter = bson.M{"fromUserId": userObjectID, "status": "pending"}
	case "received":
		filter = bson.M{"toUserId": userObjectID, "status": "pending"}
	case "all":
		filter = bson.M{
			"$or": []bson.M{
				{"fromUserId": userObjectID},
				{"toUserId": userObjectID},
			},
		}
	default:
		return nil, errors.New("invalid request type")
	}

	opts := options.Find().SetSort(bson.D{{"createdAt", -1}})
	cursor, err := fr.friendRequestsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var requests []models.FriendRequest
	err = cursor.All(ctx, &requests)
	return requests, err
}

func (fr *FriendRepository) UpdateFriendRequestStatus(ctx context.Context, requestID, status string) error {
	objectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return errors.New("invalid request ID")
	}

	update := bson.M{
		"status":      status,
		"responsedAt": time.Now(),
	}

	result, err := fr.friendRequestsCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("request not found")
	}

	return nil
}

func (fr *FriendRepository) CheckExistingRequest(ctx context.Context, fromUserID, toUserID string) (bool, error) {
	fromObjectID, _ := primitive.ObjectIDFromHex(fromUserID)
	toObjectID, _ := primitive.ObjectIDFromHex(toUserID)

	filter := bson.M{
		"$or": []bson.M{
			{"fromUserId": fromObjectID, "toUserId": toObjectID},
			{"fromUserId": toObjectID, "toUserId": fromObjectID},
		},
		"status": "pending",
	}

	count, err := fr.friendRequestsCollection.CountDocuments(ctx, filter)
	return count > 0, err
}

// Friendships
func (fr *FriendRepository) CreateFriendship(ctx context.Context, user1ID, user2ID string) error {
	user1ObjectID, _ := primitive.ObjectIDFromHex(user1ID)
	user2ObjectID, _ := primitive.ObjectIDFromHex(user2ID)

	friendship := &models.Friendship{
		ID:        primitive.NewObjectID(),
		User1ID:   user1ObjectID,
		User2ID:   user2ObjectID,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := fr.friendshipsCollection.InsertOne(ctx, friendship)
	return err
}

func (fr *FriendRepository) GetFriends(ctx context.Context, userID string) ([]string, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"$or": []bson.M{
			{"user1Id": userObjectID},
			{"user2Id": userObjectID},
		},
		"status": "active",
	}

	cursor, err := fr.friendshipsCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var friendships []models.Friendship
	err = cursor.All(ctx, &friendships)
	if err != nil {
		return nil, err
	}

	var friendIDs []string
	for _, friendship := range friendships {
		if friendship.User1ID.Hex() == userID {
			friendIDs = append(friendIDs, friendship.User2ID.Hex())
		} else {
			friendIDs = append(friendIDs, friendship.User1ID.Hex())
		}
	}

	return friendIDs, nil
}

func (fr *FriendRepository) RemoveFriendship(ctx context.Context, user1ID, user2ID string) error {
	user1ObjectID, _ := primitive.ObjectIDFromHex(user1ID)
	user2ObjectID, _ := primitive.ObjectIDFromHex(user2ID)

	filter := bson.M{
		"$or": []bson.M{
			{"user1Id": user1ObjectID, "user2Id": user2ObjectID},
			{"user1Id": user2ObjectID, "user2Id": user1ObjectID},
		},
	}

	result, err := fr.friendshipsCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("friendship not found")
	}

	return nil
}

func (fr *FriendRepository) AreFriends(ctx context.Context, user1ID, user2ID string) (bool, error) {
	user1ObjectID, _ := primitive.ObjectIDFromHex(user1ID)
	user2ObjectID, _ := primitive.ObjectIDFromHex(user2ID)

	filter := bson.M{
		"$or": []bson.M{
			{"user1Id": user1ObjectID, "user2Id": user2ObjectID},
			{"user1Id": user2ObjectID, "user2Id": user1ObjectID},
		},
		"status": "active",
	}

	count, err := fr.friendshipsCollection.CountDocuments(ctx, filter)
	return count > 0, err
}
