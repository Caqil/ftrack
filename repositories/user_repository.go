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

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		collection: db.Collection("users"),
	}
}

func (ur *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true
	user.IsOnline = false

	_, err := ur.collection.InsertOne(ctx, user)
	return err
}

func (ur *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
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
	err := ur.collection.FindOne(ctx, bson.M{"phone": phone}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
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

func (ur *UserRepository) UpdateLastSeen(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{
			"lastSeen":  time.Now(),
			"isOnline":  true,
			"updatedAt": time.Now(),
		}},
	)

	return err
}

func (ur *UserRepository) UpdateOnlineStatus(ctx context.Context, id string, isOnline bool) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"isOnline":  isOnline,
		"updatedAt": time.Now(),
	}

	if !isOnline {
		update["lastSeen"] = time.Now()
	}

	_, err = ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	return err
}

func (ur *UserRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
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

func (ur *UserRepository) GetUsersByIDs(ctx context.Context, ids []string) ([]models.User, error) {
	objectIDs := make([]primitive.ObjectID, len(ids))
	for i, id := range ids {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objectIDs[i] = objectID
	}

	cursor, err := ur.collection.Find(ctx, bson.M{"_id": bson.M{"$in": objectIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	err = cursor.All(ctx, &users)
	return users, err
}

func (ur *UserRepository) SearchUsers(ctx context.Context, query string, limit int) ([]models.User, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"firstName": bson.M{"$regex": query, "$options": "i"}},
			{"lastName": bson.M{"$regex": query, "$options": "i"}},
			{"email": bson.M{"$regex": query, "$options": "i"}},
		},
		"isActive": true,
	}

	opts := options.Find().SetLimit(int64(limit))
	cursor, err := ur.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	err = cursor.All(ctx, &users)
	return users, err
}

func (ur *UserRepository) UpdateDeviceToken(ctx context.Context, userID, deviceToken, deviceType string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = ur.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{
			"deviceToken": deviceToken,
			"deviceType":  deviceType,
			"updatedAt":   time.Now(),
		}},
	)

	return err
}

// GetByResetToken gets user by password reset token
func (ur *UserRepository) GetByResetToken(ctx context.Context, token string) (*models.User, error) {
	var user models.User

	// Check if token exists and is not expired
	filter := bson.M{
		"resetToken":        token,
		"resetTokenExpires": bson.M{"$gt": time.Now()}, // Token not expired
	}

	err := ur.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invalid or expired token")
		}
		return nil, err
	}

	return &user, nil
}

// GetByVerificationToken gets user by email verification token
func (ur *UserRepository) GetByVerificationToken(ctx context.Context, token string) (*models.User, error) {
	var user models.User

	// Check if token exists and is not expired
	filter := bson.M{
		"verificationToken":        token,
		"verificationTokenExpires": bson.M{"$gt": time.Now()}, // Token not expired
	}

	err := ur.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invalid or expired token")
		}
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) SetOffline(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"isOnline":  false,
			"updatedAt": time.Now(),
		},
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

func (ur *UserRepository) UpdateVerificationStatus(ctx context.Context, userID string, isVerified bool) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"isVerified": isVerified,
			"updatedAt":  time.Now(),
		},
	}

	if isVerified {
		update["$unset"] = bson.M{
			"verificationToken": "",
			"tokenExpiresAt":    "",
		}
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

func (ur *UserRepository) Enable2FA(ctx context.Context, userID, secret string, backupCodes []string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"twoFactorEnabled": true,
			"twoFactorSecret":  secret,
			"backupCodes":      backupCodes,
			"updatedAt":        time.Now(),
		},
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

func (ur *UserRepository) Disable2FA(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"twoFactorEnabled": false,
			"twoFactorSecret":  "",
			"backupCodes":      []string{},
			"updatedAt":        time.Now(),
		},
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

func (ur *UserRepository) UpdateBackupCodes(ctx context.Context, userID string, backupCodes []string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"backupCodes": backupCodes,
			"updatedAt":   time.Now(),
		},
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

func (ur *UserRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"password":  hashedPassword,
			"updatedAt": time.Now(),
		},
		"$unset": bson.M{
			"resetToken":     "",
			"tokenExpiresAt": "",
		},
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

func (ur *UserRepository) IncrementLoginAttempts(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$inc": bson.M{"loginAttempts": 1},
		"$set": bson.M{"updatedAt": time.Now()},
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

func (ur *UserRepository) ResetLoginAttempts(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"loginAttempts": 0,
			"updatedAt":     time.Now(),
		},
		"$unset": bson.M{"lockedUntil": ""},
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

func (ur *UserRepository) LockAccount(ctx context.Context, userID string, lockDuration time.Duration) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"lockedUntil": time.Now().Add(lockDuration),
			"updatedAt":   time.Now(),
		},
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

func (ur *UserRepository) UnlockAccount(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"loginAttempts": 0,
			"updatedAt":     time.Now(),
		},
		"$unset": bson.M{"lockedUntil": ""},
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

func (ur *UserRepository) IsAccountLocked(ctx context.Context, userID string) (bool, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID")
	}

	var user models.User
	filter := bson.M{"_id": objectID}
	err = ur.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return false, err
	}

	if !user.LockedUntil.IsZero() && time.Now().Before(user.LockedUntil) {
		return true, nil
	}

	return false, nil
}

func (ur *UserRepository) GetByAuthProvider(ctx context.Context, provider, providerID string) (*models.User, error) {
	var user models.User
	filter := bson.M{
		"authProvider":   provider,
		"authProviderId": providerID,
	}

	err := ur.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (ur *UserRepository) UpdateOAuthInfo(ctx context.Context, userID, provider, providerID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"authProvider":   provider,
			"authProviderId": providerID,
			"updatedAt":      time.Now(),
		},
	}

	_, err = ur.collection.UpdateOne(ctx, filter, update)
	return err
}

// Add these methods to your existing repositories/user_repository.go file

// IsAdmin checks if a user has admin privileges
func (ur *UserRepository) IsAdmin(ctx context.Context, userID string) (bool, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID")
	}

	var user models.User
	err = ur.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, errors.New("user not found")
		}
		return false, err
	}

	// Check if user has admin role
	return user.Role == "admin" || user.Role == "superadmin", nil
}

// GetUserRole gets the role of a user
func (ur *UserRepository) GetUserRole(ctx context.Context, userID string) (string, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "", errors.New("invalid user ID")
	}

	var user models.User
	err = ur.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", errors.New("user not found")
		}
		return "", err
	}

	return user.Role, nil
}

// UpdateUserRole updates a user's role (admin only)
func (ur *UserRepository) UpdateUserRole(ctx context.Context, userID, newRole string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"role":      newRole,
			"updatedAt": time.Now(),
		},
	}

	result, err := ur.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// IsSuperAdmin checks if a user has super admin privileges
func (ur *UserRepository) IsSuperAdmin(ctx context.Context, userID string) (bool, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID")
	}

	var user models.User
	err = ur.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, errors.New("user not found")
		}
		return false, err
	}

	return user.Role == "superadmin", nil
}

// GetAdminUsers gets all users with admin privileges
func (ur *UserRepository) GetAdminUsers(ctx context.Context) ([]models.User, error) {
	filter := bson.M{
		"role":     bson.M{"$in": []string{"admin", "superadmin"}},
		"isActive": true,
	}

	cursor, err := ur.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	err = cursor.All(ctx, &users)
	return users, err
}

// HasPermission checks if a user has a specific permission
func (ur *UserRepository) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	user, err := ur.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}

	// Check role-based permissions
	switch user.Role {
	case "superadmin":
		return true, nil // Super admin has all permissions
	case "admin":
		// Admin has most permissions except super admin specific ones
		adminPermissions := map[string]bool{
			"send_notifications": true,
			"manage_users":       true,
			"view_analytics":     true,
			"manage_circles":     true,
			"moderate_content":   true,
			"export_data":        true,
			"manage_settings":    true,
		}
		return adminPermissions[permission], nil
	case "moderator":
		// Moderator has limited permissions
		moderatorPermissions := map[string]bool{
			"moderate_content":     true,
			"view_basic_analytics": true,
		}
		return moderatorPermissions[permission], nil
	default:
		return false, nil // Regular users have no admin permissions
	}
}

// SetUserRole sets a user's role with validation
func (ur *UserRepository) SetUserRole(ctx context.Context, userID, role string) error {
	// Validate role
	validRoles := map[string]bool{
		"user":       true,
		"moderator":  true,
		"admin":      true,
		"superadmin": true,
	}

	if !validRoles[role] {
		return errors.New("invalid role")
	}

	return ur.UpdateUserRole(ctx, userID, role)
}

// DeactivateUser deactivates a user account (admin only)
func (ur *UserRepository) DeactivateUser(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"isActive":      false,
			"deactivatedAt": time.Now(),
			"updatedAt":     time.Now(),
		},
	}

	result, err := ur.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// ReactivateUser reactivates a user account (admin only)
func (ur *UserRepository) ReactivateUser(ctx context.Context, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"isActive":  true,
			"updatedAt": time.Now(),
		},
		"$unset": bson.M{
			"deactivatedAt": "",
		},
	}

	result, err := ur.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// GetUserStatistics gets user statistics (admin only)
func (ur *UserRepository) GetUserStatistics(ctx context.Context) (*models.UserStatistics, error) {
	// Count total users
	totalUsers, err := ur.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	// Count active users
	activeUsers, err := ur.collection.CountDocuments(ctx, bson.M{"isActive": true})
	if err != nil {
		return nil, err
	}

	// Count verified users
	verifiedUsers, err := ur.collection.CountDocuments(ctx, bson.M{"isVerified": true})
	if err != nil {
		return nil, err
	}

	// Count online users (last seen within 15 minutes)
	onlineThreshold := time.Now().Add(-15 * time.Minute)
	onlineUsers, err := ur.collection.CountDocuments(ctx, bson.M{
		"isOnline": true,
		"lastSeen": bson.M{"$gte": onlineThreshold},
	})
	if err != nil {
		return nil, err
	}

	// Count users by role
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":   "$role",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := ur.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	roleStats := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		roleStats[result.ID] = result.Count
	}

	return &models.UserStatistics{
		TotalUsers:    totalUsers,
		ActiveUsers:   activeUsers,
		VerifiedUsers: verifiedUsers,
		OnlineUsers:   onlineUsers,
		UsersByRole:   roleStats,
	}, nil
}
