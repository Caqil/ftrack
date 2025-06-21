package repositories

import (
	"context"
	"errors"
	"ftrack/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CircleRepository struct {
	collection *mongo.Collection
}

func NewCircleRepository(db *mongo.Database) *CircleRepository {
	return &CircleRepository{
		collection: db.Collection("circles"),
	}
}

func (cr *CircleRepository) Create(ctx context.Context, circle *models.Circle) error {
	circle.ID = primitive.NewObjectID()
	circle.CreatedAt = time.Now()
	circle.UpdatedAt = time.Now()

	// Initialize default settings
	if circle.Settings.MaxMembers == 0 {
		circle.Settings.MaxMembers = 20
	}

	_, err := cr.collection.InsertOne(ctx, circle)
	return err
}

func (cr *CircleRepository) GetByID(ctx context.Context, id string) (*models.Circle, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	var circle models.Circle
	err = cr.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&circle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("circle not found")
		}
		return nil, err
	}

	return &circle, nil
}

func (cr *CircleRepository) GetByInviteCode(ctx context.Context, inviteCode string) (*models.Circle, error) {
	var circle models.Circle
	err := cr.collection.FindOne(ctx, bson.M{"inviteCode": inviteCode}).Decode(&circle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("circle not found")
		}
		return nil, err
	}

	return &circle, nil
}

func (cr *CircleRepository) GetUserCircles(ctx context.Context, userID string) ([]models.Circle, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"members.userId": objectID,
	}

	cursor, err := cr.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var circles []models.Circle
	err = cursor.All(ctx, &circles)
	return circles, err
}

func (cr *CircleRepository) AddMember(ctx context.Context, circleID string, member models.CircleMember) error {
	objectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	member.JoinedAt = time.Now()
	member.LastActivity = time.Now()

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$push": bson.M{"members": member},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle not found")
	}

	return nil
}

func (cr *CircleRepository) RemoveMember(ctx context.Context, circleID, userID string) error {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{"_id": circleObjectID},
		bson.M{
			"$pull": bson.M{"members": bson.M{"userId": userObjectID}},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle not found")
	}

	return nil
}

func (cr *CircleRepository) UpdateMemberPermissions(ctx context.Context, circleID, userID string, permissions models.MemberPermissions) error {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":            circleObjectID,
			"members.userId": userObjectID,
		},
		bson.M{
			"$set": bson.M{
				"members.$.permissions": permissions,
				"updatedAt":             time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle or member not found")
	}

	return nil
}

func (cr *CircleRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	update["updatedAt"] = time.Now()

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle not found")
	}

	return nil
}

func (cr *CircleRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	result, err := cr.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("circle not found")
	}

	return nil
}

func (cr *CircleRepository) IsMember(ctx context.Context, circleID, userID string) (bool, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return false, errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID")
	}

	count, err := cr.collection.CountDocuments(ctx, bson.M{
		"_id":            circleObjectID,
		"members.userId": userObjectID,
	})

	return count > 0, err
}

func (cr *CircleRepository) GetMemberRole(ctx context.Context, circleID, userID string) (string, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return "", errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "", errors.New("invalid user ID")
	}

	// Use aggregation to get specific member info
	pipeline := []bson.M{
		{"$match": bson.M{"_id": circleObjectID}},
		{"$unwind": "$members"},
		{"$match": bson.M{"members.userId": userObjectID}},
		{"$project": bson.M{"_id": 0, "role": "$members.role"}},
	}

	cursor, err := cr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return "", err
	}
	defer cursor.Close(ctx)

	var result struct {
		Role string `bson:"role"`
	}

	if cursor.Next(ctx) {
		err = cursor.Decode(&result)
		if err != nil {
			return "", err
		}
		return result.Role, nil
	}

	return "", errors.New("member not found")
}

func (cr *CircleRepository) UpdateLastActivity(ctx context.Context, circleID, userID string) error {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = cr.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":            circleObjectID,
			"members.userId": userObjectID,
		},
		bson.M{
			"$set": bson.M{
				"members.$.lastActivity": time.Now(),
				"stats.lastActivity":     time.Now(),
				"updatedAt":              time.Now(),
			},
		},
	)

	return err
}

// UpdateMemberRole updates a member's role in the circle
func (cr *CircleRepository) UpdateMemberRole(ctx context.Context, circleID, userID, role string) error {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":            circleObjectID,
			"members.userId": userObjectID,
		},
		bson.M{
			"$set": bson.M{
				"members.$.role": role,
				"updatedAt":      time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle or member not found")
	}

	return nil
}

// GetInvitationByID gets an invitation by its ID
func (cr *CircleRepository) GetInvitationByID(ctx context.Context, invitationID string) (*models.CircleInvitation, error) {
	objectID, err := primitive.ObjectIDFromHex(invitationID)
	if err != nil {
		return nil, errors.New("invalid invitation ID")
	}

	// You'll need to create a separate collection for invitations
	invitationCollection := cr.collection.Database().Collection("circle_invitations")

	var invitation models.CircleInvitation
	err = invitationCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&invitation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invitation not found")
		}
		return nil, err
	}

	return &invitation, nil
}

// UpdateInvitationStatus updates the status of an invitation
func (cr *CircleRepository) UpdateInvitationStatus(ctx context.Context, invitationID, status string) error {
	objectID, err := primitive.ObjectIDFromHex(invitationID)
	if err != nil {
		return errors.New("invalid invitation ID")
	}

	invitationCollection := cr.collection.Database().Collection("circle_invitations")

	result, err := invitationCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$set": bson.M{
				"status":    status,
				"updatedAt": time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("invitation not found")
	}

	return nil
}

// GetUserInvitations gets all invitations for a user
func (cr *CircleRepository) GetUserInvitations(ctx context.Context, userID primitive.ObjectID, status string) ([]models.CircleInvitation, error) {
	invitationCollection := cr.collection.Database().Collection("circle_invitations")

	filter := bson.M{"inviteeId": userID}

	// Add status filter if provided
	if status != "" {
		filter["status"] = status
	}

	cursor, err := invitationCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.CircleInvitation
	err = cursor.All(ctx, &invitations)
	if err != nil {
		return nil, err
	}

	return invitations, nil
}

// CreateInvitation creates a new circle invitation
func (cr *CircleRepository) CreateInvitation(ctx context.Context, invitation *models.CircleInvitation) error {
	invitationCollection := cr.collection.Database().Collection("circle_invitations")

	invitation.ID = primitive.NewObjectID()
	invitation.CreatedAt = time.Now()
	invitation.UpdatedAt = time.Now()
	invitation.Status = "pending"

	// Set expiration time (7 days from now)
	invitation.ExpiresAt = time.Now().AddDate(0, 0, 7)

	_, err := invitationCollection.InsertOne(ctx, invitation)
	return err
}
