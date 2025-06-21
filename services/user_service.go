package services

import (
	"context"
	"errors"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"

	"go.mongodb.org/mongo-driver/bson"
)

type UserService struct {
	userRepo  *repositories.UserRepository
	validator *utils.ValidationService
}

func NewUserService(userRepo *repositories.UserRepository) *UserService {
	return &UserService{
		userRepo:  userRepo,
		validator: utils.NewValidationService(),
	}
}

func (us *UserService) GetUserProfile(ctx context.Context, userID string) (*models.User, error) {
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Remove sensitive data
	user.Password = ""
	return user, nil
}

func (us *UserService) UpdateUserProfile(ctx context.Context, userID string, req models.UpdateUserRequest) (*models.User, error) {
	// Validate request
	if validationErrors := us.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Build update document
	update := bson.M{}

	if req.FirstName != nil {
		update["firstName"] = *req.FirstName
	}
	if req.LastName != nil {
		update["lastName"] = *req.LastName
	}
	if req.ProfilePicture != nil {
		update["profilePicture"] = *req.ProfilePicture
	}
	if req.LocationSharing != nil {
		update["locationSharing"] = *req.LocationSharing
	}
	if req.EmergencyContact != nil {
		update["emergencyContact"] = *req.EmergencyContact
	}
	if req.Preferences != nil {
		update["preferences"] = *req.Preferences
	}

	if len(update) == 0 {
		return nil, errors.New("no fields to update")
	}

	err := us.userRepo.Update(ctx, userID, update)
	if err != nil {
		return nil, err
	}

	return us.GetUserProfile(ctx, userID)
}

func (us *UserService) SearchUsers(ctx context.Context, query string, limit int) ([]models.User, error) {
	if len(query) < 2 {
		return nil, errors.New("search query must be at least 2 characters")
	}

	if limit <= 0 || limit > 50 {
		limit = 20
	}

	users, err := us.userRepo.SearchUsers(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	// Remove sensitive data
	for i := range users {
		users[i].Password = ""
		users[i].DeviceToken = ""
	}

	return users, nil
}

func (us *UserService) UpdateDeviceToken(ctx context.Context, userID, deviceToken, deviceType string) error {
	return us.userRepo.UpdateDeviceToken(ctx, userID, deviceToken, deviceType)
}

func (us *UserService) UpdateOnlineStatus(ctx context.Context, userID string, isOnline bool) error {
	return us.userRepo.UpdateOnlineStatus(ctx, userID, isOnline)
}

func (us *UserService) GetUsersByIDs(ctx context.Context, userIDs []string) ([]models.User, error) {
	users, err := us.userRepo.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	// Remove sensitive data
	for i := range users {
		users[i].Password = ""
		users[i].DeviceToken = ""
	}

	return users, nil
}

func (us *UserService) DeactivateAccount(ctx context.Context, userID string) error {
	return us.userRepo.Update(ctx, userID, bson.M{
		"isActive": false,
	})
}

func (us *UserService) DeleteAccount(ctx context.Context, userID string) error {
	// In a production app, you might want to soft delete or anonymize data
	// instead of hard delete to comply with regulations
	return us.userRepo.Delete(ctx, userID)
}
