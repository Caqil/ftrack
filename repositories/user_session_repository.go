// repositories/user_session_repository.go
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

type UserSessionRepository struct {
	collection *mongo.Collection
}

func NewUserSessionRepository(db *mongo.Database) *UserSessionRepository {
	return &UserSessionRepository{
		collection: db.Collection("user_sessions"),
	}
}

func (usr *UserSessionRepository) Create(ctx context.Context, session *models.UserSession) error {
	session.ID = primitive.NewObjectID()
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	session.LastUsed = time.Now()

	_, err := usr.collection.InsertOne(ctx, session)
	return err
}

func (usr *UserSessionRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.UserSession, error) {
	var session models.UserSession
	err := usr.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("session not found")
		}
		return nil, err
	}
	return &session, nil
}

func (usr *UserSessionRepository) GetActiveSessions(ctx context.Context, userID primitive.ObjectID) ([]models.UserSession, error) {
	filter := bson.M{
		"userId":    userID,
		"isActive":  true,
		"expiresAt": bson.M{"$gt": time.Now()},
	}

	cursor, err := usr.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"lastUsed": -1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []models.UserSession
	err = cursor.All(ctx, &sessions)
	return sessions, err
}

func (usr *UserSessionRepository) UpdateTokenHash(ctx context.Context, userID string, newTokenHash string) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId":   userObjectID,
		"isActive": true,
	}

	update := bson.M{
		"$set": bson.M{
			"tokenHash":  newTokenHash,
			"lastUsed":   time.Now(),
			"updatedAt":  time.Now(),
		},
	}

	_, err = usr.collection.UpdateOne(ctx, filter, update)
	return err
}

func (usr *UserSessionRepository) InvalidateByTokenHash(ctx context.Context, userID primitive.ObjectID, tokenHash string) error {
	filter := bson.M{
		"userId":    userID,
		"tokenHash": tokenHash,
		"isActive":  true,
	}

	update := bson.M{
		"$set": bson.M{
			"isActive":  false,
			"updatedAt": time.Now(),
		},
	}

	_, err := usr.collection.UpdateOne(ctx, filter, update)
	return err
}

func (usr *UserSessionRepository) InvalidateSession(ctx context.Context, sessionID primitive.ObjectID) error {
	filter := bson.M{"_id": sessionID}
	update := bson.M{
		"$set": bson.M{
			"isActive":  false,
			"updatedAt": time.Now(),
		},
	}

	_, err := usr.collection.UpdateOne(ctx, filter, update)
	return err
}

func (usr *UserSessionRepository) InvalidateAllUserSessions(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{
		"userId":   userID,
		"isActive": true,
	}

	update := bson.M{
		"$set": bson.M{
			"isActive":  false,
			"updatedAt": time.Now(),
		},
	}

	_, err := usr.collection.UpdateMany(ctx, filter, update)
	return err
}

func (usr *UserSessionRepository) CleanupExpiredSessions(ctx context.Context) error {
	filter := bson.M{
		"expiresAt": bson.M{"$lt": time.Now()},
		"isActive":  true,
	}

	update := bson.M{
		"$set": bson.M{
			"isActive":  false,
			"updatedAt": time.Now(),
		},
	}

	_, err := usr.collection.UpdateMany(ctx, filter, update)
	return err
}

func (usr *UserSessionRepository) UpdateLastUsed(ctx context.Context, sessionID primitive.ObjectID) error {
	filter := bson.M{"_id": sessionID}
	update := bson.M{
		"$set": bson.M{
			"lastUsed":  time.Now(),
			"updatedAt": time.Now(),
		},
	}

	_, err := usr.collection.UpdateOne(ctx, filter, update)
	return err
}
