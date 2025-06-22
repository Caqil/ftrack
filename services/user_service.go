// COMPLETE USER SERVICE
// Add these methods to your services/user_service.go file
// This includes all methods needed for the complete user controller

package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserService struct {
	userRepo   *repositories.UserRepository
	validator  *utils.ValidationService
	friendRepo *repositories.FriendRepository // You'll need to create this
	reportRepo *repositories.ReportRepository // You'll need to create this
	exportRepo *repositories.ExportRepository // You'll need to create this
}

func NewUserService(userRepo *repositories.UserRepository) *UserService {
	return &UserService{
		userRepo:  userRepo,
		validator: utils.NewValidationService(),
		// Initialize other repositories as needed
	}
}

// =============================================
// EXISTING METHODS (keep these as they are)
// =============================================

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
	}

	return users, nil
}

// =============================================
// CURRENT USER OPERATIONS
// =============================================

func (us *UserService) GetCurrentUser(ctx context.Context, userID string) (*models.User, error) {
	return us.GetUserProfile(ctx, userID)
}

func (us *UserService) UpdateCurrentUser(ctx context.Context, userID string, req models.UpdateUserRequest) (*models.User, error) {
	return us.UpdateUserProfile(ctx, userID, req)
}

func (us *UserService) DeleteCurrentUser(ctx context.Context, userID string) error {
	// Soft delete - mark as inactive
	update := bson.M{
		"isActive":      false,
		"deactivatedAt": time.Now(),
		"updatedAt":     time.Now(),
	}

	return us.userRepo.Update(ctx, userID, update)
}

// =============================================
// PROFILE PICTURE MANAGEMENT
// =============================================

func (us *UserService) UploadProfilePicture(ctx context.Context, userID string, req models.UploadProfilePictureRequest) (string, error) {
	// Extract file and header from the request
	_, ok := req.File.(io.Reader)
	if !ok {
		return "", errors.New("invalid file")
	}

	// Validate file type and size
	header, ok := req.Header.(interface {
		Filename() string
		Size() int64
	})
	if !ok {
		return "", errors.New("invalid file header")
	}

	filename := header.Filename()
	filesize := header.Size()

	// Check file size (e.g., max 5MB)
	if filesize > 5*1024*1024 {
		return "", errors.New("file too large")
	}

	// Check file type
	ext := strings.ToLower(filepath.Ext(filename))
	allowedTypes := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	typeValid := false
	for _, allowedType := range allowedTypes {
		if ext == allowedType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return "", errors.New("invalid file type")
	}

	// Generate unique filename
	newFilename := fmt.Sprintf("%s_%d%s", userID, time.Now().Unix(), ext)

	// TODO: Implement actual file upload to your storage service
	profilePictureURL := fmt.Sprintf("/uploads/profile_pictures/%s", newFilename)

	// Update user's profile picture URL in database
	update := bson.M{
		"profilePicture": profilePictureURL,
		"updatedAt":      time.Now(),
	}

	err := us.userRepo.Update(ctx, userID, update)
	if err != nil {
		return "", err
	}

	return profilePictureURL, nil
}

func (us *UserService) DeleteProfilePicture(ctx context.Context, userID string) error {
	// Get current user to find existing profile picture
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// If user has a profile picture, delete it from storage
	if user.ProfilePicture != "" {
		// TODO: Implement actual file deletion from your storage service
	}

	// Update user record to remove profile picture
	update := bson.M{
		"profilePicture": "",
		"updatedAt":      time.Now(),
	}

	return us.userRepo.Update(ctx, userID, update)
}

func (us *UserService) GetProfilePicture(ctx context.Context, userID string) (string, error) {
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	return user.ProfilePicture, nil
}

// =============================================
// SETTINGS OPERATIONS
// =============================================

func (us *UserService) GetUserSettings(ctx context.Context, userID string) (*models.UserPreferences, error) {
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &user.Preferences, nil
}

func (us *UserService) UpdateUserSettings(ctx context.Context, userID string, preferences models.UserPreferences) (*models.UserPreferences, error) {
	update := bson.M{
		"preferences": preferences,
		"updatedAt":   time.Now(),
	}

	err := us.userRepo.Update(ctx, userID, update)
	if err != nil {
		return nil, err
	}

	return &preferences, nil
}

func (us *UserService) GetPrivacySettings(ctx context.Context, userID string) (*models.PrivacySettings, error) {
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &user.Preferences.Privacy, nil
}

func (us *UserService) UpdatePrivacySettings(ctx context.Context, userID string, settings models.PrivacySettings) (*models.PrivacySettings, error) {
	// Validate the settings
	if validationErrors := us.validator.ValidateStruct(settings); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Update the user's privacy settings
	update := bson.M{
		"preferences.privacy": settings,
		"updatedAt":           time.Now(),
	}

	err := us.userRepo.Update(ctx, userID, update)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func (us *UserService) GetNotificationSettings(ctx context.Context, userID string) (*models.NotificationPrefs, error) {
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &user.Preferences.Notifications, nil
}

func (us *UserService) UpdateNotificationSettings(ctx context.Context, userID string, notificationPrefs models.NotificationPrefs) (*models.NotificationPrefs, error) {
	update := bson.M{
		"preferences.notifications": notificationPrefs,
		"updatedAt":                 time.Now(),
	}

	err := us.userRepo.Update(ctx, userID, update)
	if err != nil {
		return nil, err
	}

	return &notificationPrefs, nil
}

func (us *UserService) GetLocationSettings(ctx context.Context, userID string) (*models.LocationSharing, error) {
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &user.LocationSharing, nil
}

func (us *UserService) UpdateLocationSettings(ctx context.Context, userID string, settings models.LocationSharing) (*models.LocationSharing, error) {
	// Validate the settings
	if validationErrors := us.validator.ValidateStruct(settings); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Additional validation for precision values
	validPrecisions := []string{"exact", "approximate", "city"}
	precisionValid := false
	for _, valid := range validPrecisions {
		if settings.Precision == valid {
			precisionValid = true
			break
		}
	}
	if !precisionValid {
		return nil, errors.New("validation failed")
	}

	// Update the user's location sharing settings
	update := bson.M{
		"locationSharing": settings,
		"updatedAt":       time.Now(),
	}

	err := us.userRepo.Update(ctx, userID, update)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func (us *UserService) GetDrivingSettings(ctx context.Context, userID string) (*models.DrivingPrefs, error) {
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &user.Preferences.Driving, nil
}

func (us *UserService) UpdateDrivingSettings(ctx context.Context, userID string, drivingPrefs models.DrivingPrefs) (*models.DrivingPrefs, error) {
	update := bson.M{
		"preferences.driving": drivingPrefs,
		"updatedAt":           time.Now(),
	}

	err := us.userRepo.Update(ctx, userID, update)
	if err != nil {
		return nil, err
	}

	return &drivingPrefs, nil
}

// =============================================
// EMERGENCY CONTACTS OPERATIONS
// =============================================

func (us *UserService) GetEmergencyContacts(ctx context.Context, userID string) ([]models.EmergencyContact, error) {
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Return array with single emergency contact for backward compatibility
	// You might want to modify your User model to support multiple emergency contacts
	contacts := []models.EmergencyContact{}
	if user.EmergencyContact.Name != "" {
		contacts = append(contacts, user.EmergencyContact)
	}

	return contacts, nil
}

func (us *UserService) AddEmergencyContact(ctx context.Context, userID string, contact models.EmergencyContact) (*models.EmergencyContact, error) {
	// Validate contact
	if validationErrors := us.validator.ValidateStruct(contact); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	update := bson.M{
		"emergencyContact": contact,
		"updatedAt":        time.Now(),
	}

	err := us.userRepo.Update(ctx, userID, update)
	if err != nil {
		return nil, err
	}

	return &contact, nil
}

func (us *UserService) UpdateEmergencyContact(ctx context.Context, userID string, contactID string, contact models.EmergencyContact) (*models.EmergencyContact, error) {
	// For simplicity, treating contactID as single contact update
	return us.AddEmergencyContact(ctx, userID, contact)
}

func (us *UserService) DeleteEmergencyContact(ctx context.Context, userID string, contactID string) error {
	update := bson.M{
		"emergencyContact": models.EmergencyContact{},
		"updatedAt":        time.Now(),
	}

	return us.userRepo.Update(ctx, userID, update)
}

func (us *UserService) VerifyEmergencyContact(ctx context.Context, userID string, contactID string) error {
	// Implementation depends on your verification system
	// For now, just mark as verified
	update := bson.M{
		"emergencyContact.verified": true,
		"updatedAt":                 time.Now(),
	}

	return us.userRepo.Update(ctx, userID, update)
}

// =============================================
// DEVICE MANAGEMENT OPERATIONS
// =============================================

func (us *UserService) GetUserDevices(ctx context.Context, userID string) ([]models.UserDevice, error) {
	// This would typically come from a separate devices collection
	// For now, return basic device info from user profile
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	devices := []models.UserDevice{}
	if user.DeviceToken != "" {
		device := models.UserDevice{
			ID:         primitive.NewObjectID(),
			UserID:     user.ID,
			DeviceType: "unknown", // You'd store this in device info
			DeviceName: "Default Device",
			DeviceID:   user.DeviceToken,
			PushToken:  user.DeviceToken,
			IsActive:   true,
			LastUsed:   user.UpdatedAt,
			CreatedAt:  user.CreatedAt,
			UpdatedAt:  user.UpdatedAt,
		}
		devices = append(devices, device)
	}

	return devices, nil
}

func (us *UserService) RegisterDevice(ctx context.Context, userID string, deviceReq models.RegisterDeviceRequest) (*models.UserDevice, error) {
	// Validate request
	if validationErrors := us.validator.ValidateStruct(deviceReq); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Update user's device token
	update := bson.M{
		"deviceToken": deviceReq.DeviceToken,
		"updatedAt":   time.Now(),
	}

	err := us.userRepo.Update(ctx, userID, update)
	if err != nil {
		return nil, err
	}

	// Return device info
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	device := &models.UserDevice{
		ID:         primitive.NewObjectID(),
		UserID:     user.ID,
		DeviceType: deviceReq.DeviceType,
		DeviceName: deviceReq.DeviceModel,
		DeviceID:   deviceReq.DeviceToken,
		PushToken:  deviceReq.DeviceToken,
		IsActive:   true,
		LastUsed:   time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return device, nil
}

func (us *UserService) UpdateDevice(ctx context.Context, userID string, deviceID string, deviceReq models.UpdateDeviceRequest) (*models.UserDevice, error) {
	// Simple implementation - update device info
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	device := &models.UserDevice{
		ID:         primitive.NewObjectID(),
		UserID:     user.ID,
		DeviceType: "unknown",
		IsActive:   true,
		LastUsed:   time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Apply updates from request
	if deviceReq.IsActive != nil {
		device.IsActive = *deviceReq.IsActive
	}

	return device, nil
}

func (us *UserService) UnregisterDevice(ctx context.Context, userID string, deviceID string) error {
	update := bson.M{
		"deviceToken": "",
		"updatedAt":   time.Now(),
	}

	return us.userRepo.Update(ctx, userID, update)
}

func (us *UserService) TestPushNotification(ctx context.Context, userID string, deviceID string) error {
	// Implementation would depend on your push notification service
	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.DeviceToken == "" {
		return errors.New("no device token found")
	}

	// TODO: Send actual test push notification
	return nil
}

// =============================================
// SOCIAL FEATURES
// =============================================

func (us *UserService) GetUser(ctx context.Context, userID string, targetUserID string) (*models.User, error) {
	// Validate target user ID
	if targetUserID == "" {
		return nil, errors.New("user ID is required")
	}

	// Get the target user
	targetUser, err := us.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}

	// Check if target user exists and is active
	if !targetUser.IsActive {
		return nil, errors.New("user not found")
	}

	// Get the requesting user to check privacy settings
	requestingUser, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply privacy filtering
	publicUser := us.filterUserForPublicView(targetUser, requestingUser)

	return publicUser, nil
}

func (us *UserService) GetPublicProfile(ctx context.Context, userID string, targetUserID string) (*models.User, error) {
	user, err := us.GetUser(ctx, userID, targetUserID)
	if err != nil {
		return nil, err
	}

	// Additional filtering for public profile view
	user.Email = ""                                   // Never show email in public profile
	user.EmergencyContact = models.EmergencyContact{} // Remove emergency contact
	user.Preferences = models.UserPreferences{}       // Remove preferences

	return user, nil
}

// Helper function to filter user data for public viewing
func (us *UserService) filterUserForPublicView(targetUser *models.User, requestingUser *models.User) *models.User {
	// Create a copy of the user with filtered data
	publicUser := &models.User{
		ID:             targetUser.ID,
		FirstName:      targetUser.FirstName,
		LastName:       targetUser.LastName,
		ProfilePicture: targetUser.ProfilePicture,
		IsOnline:       targetUser.IsOnline,
		CreatedAt:      targetUser.CreatedAt,
		UpdatedAt:      targetUser.UpdatedAt,
	}

	// Apply privacy settings
	if targetUser.Preferences.Privacy.ShowInDirectory {
		publicUser.Email = targetUser.Email
	}

	// Remove sensitive data
	publicUser.Password = ""
	publicUser.Phone = ""
	publicUser.DeviceToken = ""

	// Only show basic location sharing status if allowed
	if targetUser.LocationSharing.Enabled && len(targetUser.LocationSharing.ShareWith) > 0 {
		publicUser.LocationSharing = models.LocationSharing{
			Enabled: true,
		}
	}

	return publicUser
}

// Continue adding these methods to your user_service.go file

// =============================================
// FRIEND SYSTEM OPERATIONS
// =============================================

func (us *UserService) SendFriendRequest(ctx context.Context, userID string, targetUserID string, req models.FriendRequestSend) (*models.FriendRequest, error) {
	// Validate that user isn't sending request to themselves
	if userID == targetUserID {
		return nil, errors.New("cannot send to yourself")
	}

	// Check if target user exists
	_, err := us.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Check if users are already friends
	// TODO: Implement friendship check
	// if us.areAlreadyFriends(ctx, userID, targetUserID) {
	//     return nil, errors.New("already friends")
	// }

	// Check if request already exists
	// TODO: Implement existing request check
	// if us.requestAlreadyExists(ctx, userID, targetUserID) {
	//     return nil, errors.New("request already sent")
	// }

	// Create friend request
	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	targetObjectID, _ := primitive.ObjectIDFromHex(targetUserID)

	friendRequest := &models.FriendRequest{
		ID:         primitive.NewObjectID(),
		FromUserID: userObjectID,
		ToUserID:   targetObjectID,
		Status:     "pending",
		Message:    req.Message,
		CreatedAt:  time.Now(),
	}

	// TODO: Save to friend requests collection
	// err = us.friendRepo.CreateFriendRequest(ctx, friendRequest)
	// if err != nil {
	//     return nil, err
	// }

	return friendRequest, nil
}

func (us *UserService) AcceptFriendRequest(ctx context.Context, userID string, requestID string) (*models.FriendRequest, error) {
	// TODO: Implement friend request acceptance
	// 1. Find the friend request by ID
	// 2. Verify that the current user is the recipient
	// 3. Update status to "accepted"
	// 4. Create friendship records for both users

	requestObjectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return nil, errors.New("request not found")
	}

	// Placeholder implementation
	friendRequest := &models.FriendRequest{
		ID:          requestObjectID,
		Status:      "accepted",
		ResponsedAt: time.Now(),
	}

	return friendRequest, nil
}

func (us *UserService) DeclineFriendRequest(ctx context.Context, userID string, requestID string) (*models.FriendRequest, error) {
	// TODO: Implement friend request decline
	requestObjectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return nil, errors.New("request not found")
	}

	// Placeholder implementation
	friendRequest := &models.FriendRequest{
		ID:          requestObjectID,
		Status:      "declined",
		ResponsedAt: time.Now(),
	}

	return friendRequest, nil
}

func (us *UserService) GetFriendRequests(ctx context.Context, userID string, requestType string) ([]models.FriendRequest, error) {
	// Validate request type
	validTypes := []string{"sent", "received", "all"}
	typeValid := false
	for _, validType := range validTypes {
		if requestType == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return nil, errors.New("invalid request type")
	}

	// TODO: Implement getting friend requests from repository
	// return us.friendRepo.GetFriendRequests(ctx, userID, requestType)

	// Placeholder implementation
	return []models.FriendRequest{}, nil
}

func (us *UserService) GetFriends(ctx context.Context, userID string) ([]models.User, error) {
	// TODO: Implement getting friends list
	// 1. Get all friendship records for the user
	// 2. Get user details for all friends
	// 3. Filter sensitive data

	// Placeholder implementation
	return []models.User{}, nil
}

func (us *UserService) RemoveFriend(ctx context.Context, userID string, friendUserID string) error {
	// TODO: Implement friend removal
	// 1. Verify friendship exists
	// 2. Remove friendship records for both users

	if userID == friendUserID {
		return errors.New("cannot remove yourself")
	}

	// Check if friend exists
	_, err := us.userRepo.GetByID(ctx, friendUserID)
	if err != nil {
		return errors.New("user not found")
	}

	// TODO: Remove friendship records
	return nil
}

// =============================================
// BLOCKING AND REPORTING OPERATIONS
// =============================================

func (us *UserService) GetBlockedUsers(ctx context.Context, userID string) ([]models.BlockedUser, error) {
	// TODO: Implement getting blocked users from repository
	// return us.reportRepo.GetBlockedUsers(ctx, userID)

	// Placeholder implementation
	return []models.BlockedUser{}, nil
}

func (us *UserService) BlockUser(ctx context.Context, userID string, targetUserID string, req models.BlockUserRequest) (*models.BlockedUser, error) {
	if userID == targetUserID {
		return nil, errors.New("cannot block yourself")
	}

	// Check if target user exists
	_, err := us.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// TODO: Check if already blocked
	// TODO: Remove friendship if exists
	// TODO: Create block record

	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	targetObjectID, _ := primitive.ObjectIDFromHex(targetUserID)

	blockedUser := &models.BlockedUser{
		ID:            primitive.NewObjectID(),
		UserID:        userObjectID,
		BlockedUserID: targetObjectID,
		Reason:        req.Reason,
		CreatedAt:     time.Now(),
	}

	// TODO: Save to database
	// err = us.reportRepo.CreateBlockedUser(ctx, blockedUser)

	return blockedUser, nil
}

func (us *UserService) UnblockUser(ctx context.Context, userID string, targetUserID string) error {
	// TODO: Implement user unblocking
	// 1. Find block record
	// 2. Remove block record

	if userID == targetUserID {
		return errors.New("cannot unblock yourself")
	}

	// Check if target user exists
	_, err := us.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return errors.New("user not found")
	}

	// TODO: Remove block record
	return nil
}

func (us *UserService) ReportUser(ctx context.Context, userID string, targetUserID string, req models.ReportUserRequest) (*models.UserReport, error) {
	if userID == targetUserID {
		return nil, errors.New("cannot report yourself")
	}

	// Validate request
	if validationErrors := us.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check if target user exists
	_, err := us.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	targetObjectID, _ := primitive.ObjectIDFromHex(targetUserID)

	report := &models.UserReport{
		ID:             primitive.NewObjectID(),
		ReporterID:     userObjectID,
		ReportedUserID: targetObjectID,
		Reason:         req.Reason,
		Description:    req.Description,
		Status:         "pending",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// TODO: Save to database
	// err = us.reportRepo.CreateUserReport(ctx, report)

	return report, nil
}

// =============================================
// DATA EXPORT AND PRIVACY OPERATIONS
// =============================================

func (us *UserService) ExportUserData(ctx context.Context, userID string, req models.ExportUserDataRequest) (*models.UserDataExport, error) {
	// Validate request
	if validationErrors := us.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check if export already in progress
	// TODO: Check existing exports
	// existingExport, _ := us.exportRepo.GetActiveExport(ctx, userID)
	// if existingExport != nil {
	//     return nil, errors.New("export in progress")
	// }

	userObjectID, _ := primitive.ObjectIDFromHex(userID)

	export := &models.UserDataExport{
		ID:        primitive.NewObjectID(),
		UserID:    userObjectID,
		Status:    "pending",
		DataTypes: req.DataTypes,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt: time.Now(),
	}

	// TODO: Save to database and start background export process
	// err := us.exportRepo.CreateExport(ctx, export)
	// if err != nil {
	//     return nil, err
	// }

	// TODO: Start background export process
	// go us.processDataExport(ctx, export)

	return export, nil
}

func (us *UserService) GetExportStatus(ctx context.Context, userID string) (*models.UserDataExport, error) {
	// TODO: Get latest export for user
	// return us.exportRepo.GetLatestExport(ctx, userID)

	// Placeholder implementation
	return nil, errors.New("no export found")
}

func (us *UserService) DownloadExport(ctx context.Context, userID string, exportID string) (*models.ExportFile, error) {
	// TODO: Validate export ownership and status
	// export, err := us.exportRepo.GetExport(ctx, exportID)
	// if err != nil {
	//     return nil, errors.New("export not found")
	// }

	// if export.UserID.Hex() != userID {
	//     return nil, errors.New("access denied")
	// }

	// if export.Status != "completed" {
	//     return nil, errors.New("export not ready")
	// }

	// TODO: Load file data from storage
	exportFile := &models.ExportFile{
		Data:        []byte{}, // Load from storage
		Filename:    fmt.Sprintf("user_data_export_%s.zip", userID),
		ContentType: "application/zip",
	}

	return exportFile, nil
}

func (us *UserService) RequestDataPurge(ctx context.Context, userID string, req models.DataPurgeRequest) (*models.DataPurgeRequest, error) {
	// Validate request
	if validationErrors := us.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check if purge already requested
	// TODO: Check existing purge requests

	userObjectID, _ := primitive.ObjectIDFromHex(userID)

	purgeRequest := &models.DataPurgeRequest{
		ID:          primitive.NewObjectID(),
		UserID:      userObjectID,
		Reason:      req.Reason,
		DataTypes:   req.DataTypes,
		Status:      "pending",
		ScheduledAt: time.Now().Add(30 * 24 * time.Hour), // 30 days from now
		CreatedAt:   time.Now(),
	}

	// TODO: Save to database
	// err := us.exportRepo.CreatePurgeRequest(ctx, purgeRequest)

	return purgeRequest, nil
}

// =============================================
// STATISTICS OPERATIONS
// =============================================

func (us *UserService) GetUserStats(ctx context.Context, userID string) (*models.UserStatsResponse, error) {
	// TODO: Implement comprehensive user statistics
	// This would aggregate data from multiple collections

	user, err := us.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	stats := &models.UserStatsResponse{
		TotalCircles:   0, // TODO: Count user's circles
		TotalMessages:  0, // TODO: Count user's messages
		TotalTrips:     0, // TODO: Count user's trips
		TotalDistance:  0, // TODO: Calculate total distance
		TotalPlaces:    0, // TODO: Count user's places
		DrivingScore:   0, // TODO: Calculate driving score
		SafetyScore:    0, // TODO: Calculate safety score
		MemberSince:    user.CreatedAt.Format("2006-01-02"),
		LastActiveDate: user.UpdatedAt.Format("2006-01-02"),
	}

	return stats, nil
}

func (us *UserService) GetActivityStats(ctx context.Context, userID string, period string) (*models.UserActivityStats, error) {
	// Validate period
	validPeriods := []string{"day", "week", "month", "year"}
	periodValid := false
	for _, validPeriod := range validPeriods {
		if period == validPeriod {
			periodValid = true
			break
		}
	}
	if !periodValid {
		return nil, errors.New("invalid period")
	}

	// TODO: Implement activity statistics calculation
	stats := &models.UserActivityStats{
		DailyActivity:   []models.ActivityPoint{},
		WeeklyActivity:  []models.ActivityPoint{},
		MonthlyActivity: []models.ActivityPoint{},
		TopLocations:    []models.LocationStat{},
		TopCircles:      []models.CircleStat{},
	}

	return stats, nil
}

func (us *UserService) GetLocationStats(ctx context.Context, userID string, period string) (*models.LocationStatsResponse, error) {
	// Validate period
	validPeriods := []string{"day", "week", "month", "year"}
	periodValid := false
	for _, validPeriod := range validPeriods {
		if period == validPeriod {
			periodValid = true
			break
		}
	}
	if !periodValid {
		return nil, errors.New("invalid period")
	}

	// TODO: Implement location statistics calculation
	stats := &models.LocationStatsResponse{
		TotalDistance:   0,
		TotalTime:       0,
		AverageSpeed:    0,
		MaxSpeed:        0,
		PlacesVisited:   0,
		TripsCount:      0,
		DrivingTime:     0,
		WalkingTime:     0,
		StationaryTime:  0,
		BatteryConsumed: 0,
		CO2Footprint:    0,
		SafetyScore:     0,
	}

	return stats, nil
}

func (us *UserService) GetDrivingStats(ctx context.Context, userID string, period string) (*models.DrivingStatsResponse, error) {
	// Validate period
	validPeriods := []string{"day", "week", "month", "year"}
	periodValid := false
	for _, validPeriod := range validPeriods {
		if period == validPeriod {
			periodValid = true
			break
		}
	}
	if !periodValid {
		return nil, errors.New("invalid period")
	}

	// TODO: Implement driving statistics calculation
	stats := &models.DrivingStatsResponse{
		TotalTrips:     0,
		TotalDistance:  0,
		TotalTime:      0,
		AverageSpeed:   0,
		MaxSpeed:       0,
		SafetyScore:    0,
		HardBraking:    0,
		RapidAccel:     0,
		Speeding:       0,
		PhoneUsage:     0,
		NightDriving:   0,
		HighwayDriving: 0,
		CityDriving:    0,
		RuralDriving:   0,
	}

	return stats, nil
}

func (us *UserService) GetCircleStats(ctx context.Context, userID string) (*models.CircleStatsResponse, error) {
	// TODO: Implement circle statistics calculation
	stats := &models.CircleStatsResponse{
		TotalCircles:     0,
		ActiveCircles:    0,
		CirclesCreated:   0,
		CirclesJoined:    0,
		TotalMembers:     0,
		MessagesShared:   0,
		LocationShares:   0,
		EmergencyAlerts:  0,
		MostActiveCircle: models.CircleStat{},
		RecentActivity:   []models.CircleActivity{},
	}

	return stats, nil
}

// =============================================
// HELPER FUNCTIONS AND ADDITIONAL METHODS
// =============================================

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
	}

	return users, nil
}

func (us *UserService) DeactivateAccount(ctx context.Context, userID string) error {
	return us.userRepo.Update(ctx, userID, bson.M{
		"isActive":      false,
		"deactivatedAt": time.Now(),
		"updatedAt":     time.Now(),
	})
}

func (us *UserService) DeleteAccount(ctx context.Context, userID string) error {
	// In a production app, you might want to soft delete or anonymize data
	// instead of hard delete to comply with regulations
	return us.userRepo.Delete(ctx, userID)
}
