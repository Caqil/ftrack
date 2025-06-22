// COMPLETE USER REPOSITORY
// Add these methods to your repositories/user_repository.go file

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

type UserRepository struct {
	db         *mongo.Database
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		db:         db,
		collection: db.Collection("users"),
	}
}

// =============================================
// BASIC CRUD OPERATIONS
// =============================================

func (ur *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := ur.collection.InsertOne(ctx, user)
	return err
}

func (ur *UserRepository) GetByID(ctx context.Context, userID string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var user models.User
	err = ur.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := ur.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	err := ur.collection.FindOne(ctx, bson.M{"phoneNumber": phone}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) GetByVerificationToken(ctx context.Context, token string) (*models.User, error) {
	var user models.User
	err := ur.collection.FindOne(ctx, bson.M{"verificationToken": token}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) GetByResetToken(ctx context.Context, token string) (*models.User, error) {
	var user models.User
	err := ur.collection.FindOne(ctx, bson.M{
		"resetToken":     token,
		"resetExpiresAt": bson.M{"$gt": time.Now()},
	}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found or token expired")
		}
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) Update(ctx context.Context, userID string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update["updatedAt"] = time.Now()

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// =============================================
// AUTHENTICATION AND SECURITY OPERATIONS
// =============================================

func (ur *UserRepository) UpdateLastSeen(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"lastSeen":  time.Now(),
		"updatedAt": time.Now(),
	}

	_, err = ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	return err
}

func (ur *UserRepository) SetOffline(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"isOnline":  false,
		"lastSeen":  time.Now(),
		"updatedAt": time.Now(),
	}

	_, err = ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	return err
}

func (ur *UserRepository) IsAccountLocked(ctx context.Context, userID string) (bool, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID")
	}

	var user models.User
	err = ur.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return false, err
	}

	// Check if account is locked and lock hasn't expired
	if !user.LockedUntil.IsZero() && user.LockedUntil.After(time.Now()) {
		return true, nil
	}

	return false, nil
}

func (ur *UserRepository) LockAccount(ctx context.Context, userID string, duration time.Duration) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	lockUntil := time.Now().Add(duration)
	update := bson.M{
		"lockedUntil": lockUntil,
		"updatedAt":   time.Now(),
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) UnlockAccount(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$unset": bson.M{"lockedUntil": ""},
		"$set": bson.M{
			"loginAttempts": 0,
			"updatedAt":     time.Now(),
		},
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		update,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) IncrementLoginAttempts(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$inc": bson.M{"loginAttempts": 1},
		"$set": bson.M{"updatedAt": time.Now()},
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		update,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) ResetLoginAttempts(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"loginAttempts": 0,
		"updatedAt":     time.Now(),
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) SetVerificationToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"verificationToken":     token,
		"verificationExpiresAt": expiresAt,
		"updatedAt":             time.Now(),
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) ClearVerificationToken(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$unset": bson.M{
			"verificationToken":     "",
			"verificationExpiresAt": "",
		},
		"$set": bson.M{"updatedAt": time.Now()},
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		update,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) SetResetToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"resetToken":     token,
		"resetExpiresAt": expiresAt,
		"updatedAt":      time.Now(),
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) ClearResetToken(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$unset": bson.M{
			"resetToken":     "",
			"resetExpiresAt": "",
		},
		"$set": bson.M{"updatedAt": time.Now()},
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		update,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$set": bson.M{
			"password":      hashedPassword,
			"passwordHash":  hashedPassword, // Alternative field name if used
			"loginAttempts": 0,              // Reset login attempts on password change
			"updatedAt":     time.Now(),
		},
		"$unset": bson.M{
			"resetToken":     "",
			"resetExpiresAt": "",
			"lockedUntil":    "",
		},
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		update,
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) Delete(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	result, err := ur.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// =============================================
// SEARCH AND QUERY OPERATIONS
// =============================================

func (ur *UserRepository) SearchUsers(ctx context.Context, query string, limit int) ([]models.User, error) {
	filter := bson.M{
		"$and": []bson.M{
			{
				"$or": []bson.M{
					{"firstName": bson.M{"$regex": query, "$options": "i"}},
					{"lastName": bson.M{"$regex": query, "$options": "i"}},
					{"email": bson.M{"$regex": query, "$options": "i"}},
				},
			},
			{"isActive": true},
			{"preferences.privacy.showInDirectory": true},
		},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{"firstName", 1}, {"lastName", 1}})

	cursor, err := ur.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	err = cursor.All(ctx, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (ur *UserRepository) SearchUsersAdvanced(ctx context.Context, req models.SearchUsersAdvancedRequest) ([]models.User, error) {
	filter := bson.M{
		"isActive":                            true,
		"preferences.privacy.showInDirectory": true,
	}

	// Add text search
	if req.Query != "" {
		filter["$or"] = []bson.M{
			{"firstName": bson.M{"$regex": req.Query, "$options": "i"}},
			{"lastName": bson.M{"$regex": req.Query, "$options": "i"}},
			{"email": bson.M{"$regex": req.Query, "$options": "i"}},
		}
	}

	// Add filters
	if req.Filters.IsOnline != nil {
		filter["isOnline"] = *req.Filters.IsOnline
	}

	if req.Filters.HasProfilePic != nil {
		if *req.Filters.HasProfilePic {
			filter["profilePicture"] = bson.M{"$ne": ""}
		} else {
			filter["profilePicture"] = ""
		}
	}

	// Exclude specific user IDs
	if len(req.ExcludeIDs) > 0 {
		excludeObjectIDs := make([]primitive.ObjectID, 0, len(req.ExcludeIDs))
		for _, id := range req.ExcludeIDs {
			if objectID, err := primitive.ObjectIDFromHex(id); err == nil {
				excludeObjectIDs = append(excludeObjectIDs, objectID)
			}
		}
		if len(excludeObjectIDs) > 0 {
			filter["_id"] = bson.M{"$nin": excludeObjectIDs}
		}
	}

	// Set up options
	opts := options.Find().SetLimit(int64(req.Limit))

	// Add sorting
	switch req.SortBy {
	case "name":
		opts.SetSort(bson.D{{"firstName", 1}, {"lastName", 1}})
	case "recent":
		opts.SetSort(bson.D{{"createdAt", -1}})
	default: // relevance
		opts.SetSort(bson.D{{"firstName", 1}})
	}

	cursor, err := ur.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	err = cursor.All(ctx, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (ur *UserRepository) GetUsersByIDs(ctx context.Context, userIDs []string) ([]models.User, error) {
	objectIDs := make([]primitive.ObjectID, 0, len(userIDs))
	for _, id := range userIDs {
		if objectID, err := primitive.ObjectIDFromHex(id); err == nil {
			objectIDs = append(objectIDs, objectID)
		}
	}

	if len(objectIDs) == 0 {
		return []models.User{}, nil
	}

	filter := bson.M{
		"_id":      bson.M{"$in": objectIDs},
		"isActive": true,
	}

	cursor, err := ur.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	err = cursor.All(ctx, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// =============================================
// DEVICE AND STATUS OPERATIONS
// =============================================

func (ur *UserRepository) UpdateDeviceToken(ctx context.Context, userID, deviceToken, deviceType string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"deviceToken": deviceToken,
		"deviceType":  deviceType,
		"updatedAt":   time.Now(),
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) UpdateOnlineStatus(ctx context.Context, userID string, isOnline bool) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"isOnline":  isOnline,
		"lastSeen":  time.Now(),
		"updatedAt": time.Now(),
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) UpdateLastActivity(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"lastActivity": time.Now(),
		"updatedAt":    time.Now(),
	}

	_, err = ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	return err
}

// =============================================
// VERIFICATION OPERATIONS
// =============================================

func (ur *UserRepository) MarkEmailVerified(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"isVerified": true,
		"verifiedAt": time.Now(),
		"updatedAt":  time.Now(),
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (ur *UserRepository) MarkPhoneVerified(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"phoneVerified": true,
		"updatedAt":     time.Now(),
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// =============================================
// STATISTICS AND ANALYTICS
// =============================================

func (ur *UserRepository) GetUserCount(ctx context.Context) (int64, error) {
	return ur.collection.CountDocuments(ctx, bson.M{"isActive": true})
}

func (ur *UserRepository) GetOnlineUserCount(ctx context.Context) (int64, error) {
	return ur.collection.CountDocuments(ctx, bson.M{
		"isActive": true,
		"isOnline": true,
	})
}

func (ur *UserRepository) GetVerifiedUserCount(ctx context.Context) (int64, error) {
	return ur.collection.CountDocuments(ctx, bson.M{
		"isActive":   true,
		"isVerified": true,
	})
}

func (ur *UserRepository) GetNewUsersCount(ctx context.Context, since time.Time) (int64, error) {
	return ur.collection.CountDocuments(ctx, bson.M{
		"isActive":  true,
		"createdAt": bson.M{"$gte": since},
	})
}

func (ur *UserRepository) GetActiveUsersCount(ctx context.Context, since time.Time) (int64, error) {
	return ur.collection.CountDocuments(ctx, bson.M{
		"isActive":     true,
		"lastActivity": bson.M{"$gte": since},
	})
}

// =============================================
// BULK OPERATIONS
// =============================================

func (ur *UserRepository) BulkUpdateUsers(ctx context.Context, userIDs []string, update bson.M) error {
	objectIDs := make([]primitive.ObjectID, 0, len(userIDs))
	for _, id := range userIDs {
		if objectID, err := primitive.ObjectIDFromHex(id); err == nil {
			objectIDs = append(objectIDs, objectID)
		}
	}

	if len(objectIDs) == 0 {
		return errors.New("no valid user IDs provided")
	}

	update["updatedAt"] = time.Now()

	result, err := ur.collection.UpdateMany(
		ctx,
		bson.M{"_id": bson.M{"$in": objectIDs}},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("no users found")
	}

	return nil
}

func (ur *UserRepository) BulkDeactivateUsers(ctx context.Context, userIDs []string) error {
	return ur.BulkUpdateUsers(ctx, userIDs, bson.M{
		"isActive":      false,
		"deactivatedAt": time.Now(),
	})
}

// =============================================
// ADMIN OPERATIONS
// =============================================

func (ur *UserRepository) GetUsersPaginated(ctx context.Context, page, limit int, filter bson.M) ([]models.User, int64, error) {
	if filter == nil {
		filter = bson.M{}
	}

	// Get total count
	total, err := ur.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Get users
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{"createdAt", -1}})

	cursor, err := ur.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	err = cursor.All(ctx, &users)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (ur *UserRepository) GetUsersByRole(ctx context.Context, role string) ([]models.User, error) {
	filter := bson.M{
		"role":     role,
		"isActive": true,
	}

	cursor, err := ur.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	err = cursor.All(ctx, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (ur *UserRepository) UpdateUserRole(ctx context.Context, userID, role string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"role":      role,
		"updatedAt": time.Now(),
	}

	result, err := ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// =============================================
// CLEANUP AND MAINTENANCE
// =============================================

func (ur *UserRepository) CleanupInactiveUsers(ctx context.Context, inactiveSince time.Time) (int64, error) {
	filter := bson.M{
		"isActive":     true,
		"lastActivity": bson.M{"$lt": inactiveSince},
	}

	update := bson.M{
		"$set": bson.M{
			"isActive":      false,
			"deactivatedAt": time.Now(),
			"updatedAt":     time.Now(),
		},
	}

	result, err := ur.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

func (ur *UserRepository) DeleteUnverifiedUsers(ctx context.Context, createdBefore time.Time) (int64, error) {
	filter := bson.M{
		"isVerified": false,
		"createdAt":  bson.M{"$lt": createdBefore},
	}

	result, err := ur.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// =============================================
// INDEXES
// =============================================

func (ur *UserRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"email", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{"phoneNumber", 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys:    bson.D{{"deviceToken", 1}},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys:    bson.D{{"verificationToken", 1}},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys:    bson.D{{"resetToken", 1}},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys:    bson.D{{"firstName", "text"}, {"lastName", "text"}, {"email", "text"}},
			Options: options.Index().SetDefaultLanguage("english"),
		},
		{
			Keys: bson.D{{"isActive", 1}},
		},
		{
			Keys: bson.D{{"isOnline", 1}},
		},
		{
			Keys: bson.D{{"isVerified", 1}},
		},
		{
			Keys: bson.D{{"lastActivity", -1}},
		},
		{
			Keys: bson.D{{"lastSeen", -1}},
		},
		{
			Keys: bson.D{{"createdAt", -1}},
		},
		{
			Keys: bson.D{{"role", 1}},
		},
		{
			Keys:    bson.D{{"lockedUntil", 1}},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys:    bson.D{{"resetExpiresAt", 1}},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys:    bson.D{{"verificationExpiresAt", 1}},
			Options: options.Index().SetSparse(true),
		},
	}

	_, err := ur.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
