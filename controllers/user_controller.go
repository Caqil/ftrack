package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	userService *services.UserService
}

func NewUserController(userService *services.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

// GetProfile gets user's profile
// @Summary Get user profile
// @Description Get authenticated user's profile information
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.User}
// @Failure 401 {object} models.APIResponse
// @Router /users/profile [get]
func (uc *UserController) GetProfile(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	user, err := uc.userService.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get profile failed: %v", err)

		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get user profile")
		}
		return
	}

	utils.SuccessResponse(c, "Profile retrieved successfully", user)
}

// UpdateProfile updates user's profile
// @Summary Update user profile
// @Description Update authenticated user's profile information
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.UpdateUserRequest true "Updated profile data"
// @Success 200 {object} models.APIResponse{data=models.User}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/profile [put]
func (uc *UserController) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	user, err := uc.userService.UpdateUserProfile(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update profile failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid profile data")
		case "no fields to update":
			utils.BadRequestResponse(c, "No fields provided for update")
		case "user not found":
			utils.NotFoundResponse(c, "User")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update profile")
		}
		return
	}

	utils.SuccessResponse(c, "Profile updated successfully", user)
}

// GetUser gets another user's public profile
// @Summary Get user by ID
// @Description Get another user's public profile information
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.APIResponse{data=models.User}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/{id} [get]
func (uc *UserController) GetUser(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	targetUserID := c.Param("id")
	if targetUserID == "" {
		utils.BadRequestResponse(c, "User ID is required")
		return
	}

	user, err := uc.userService.GetUser(c.Request.Context(), userID, targetUserID)
	if err != nil {
		logrus.Errorf("Get user failed: %v", err)

		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to view this user's profile")
		case "privacy settings":
			utils.ForbiddenResponse(c, "This user's profile is private")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get user")
		}
		return
	}

	utils.SuccessResponse(c, "User retrieved successfully", user)
}

// SearchUsers searches for users
// @Summary Search users
// @Description Search for users by name, email, or phone
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Number of results to return" default(20)
// @Success 200 {object} models.APIResponse{data=[]models.User}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/search [get]
func (uc *UserController) SearchUsers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	query := c.Query("q")
	if query == "" {
		utils.BadRequestResponse(c, "Search query is required")
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	users, err := uc.userService.SearchUsers(c.Request.Context(), query, limit)
	if err != nil {
		logrus.Errorf("Search users failed: %v", err)

		switch err.Error() {
		case "search query must be at least 2 characters":
			utils.BadRequestResponse(c, "Search query must be at least 2 characters")
		default:
			utils.InternalServerErrorResponse(c, "Failed to search users")
		}
		return
	}

	utils.SuccessResponse(c, "Users found successfully", users)
}

// UploadProfilePicture uploads user's profile picture
// @Summary Upload profile picture
// @Description Upload a new profile picture for the authenticated user
// @Tags Users
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Profile picture file"
// @Success 200 {object} models.APIResponse{data=object{profilePicture=string}}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/profile/picture [post]
func (uc *UserController) UploadProfilePicture(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.BadRequestResponse(c, "Profile picture file is required")
		return
	}
	defer file.Close()

	req := models.UploadProfilePictureRequest{
		File:   file,
		Header: header,
	}

	profilePictureURL, err := uc.userService.UploadProfilePicture(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Upload profile picture failed: %v", err)

		switch err.Error() {
		case "invalid file type":
			utils.BadRequestResponse(c, "Invalid file type. Only images are allowed")
		case "file too large":
			utils.BadRequestResponse(c, "File size exceeds the limit")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid file")
		default:
			utils.InternalServerErrorResponse(c, "Failed to upload profile picture")
		}
		return
	}

	response := map[string]interface{}{
		"profilePicture": profilePictureURL,
	}

	utils.SuccessResponse(c, "Profile picture uploaded successfully", response)
}

// DeleteProfilePicture deletes user's profile picture
// @Summary Delete profile picture
// @Description Delete the authenticated user's profile picture
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/profile/picture [delete]
func (uc *UserController) DeleteProfilePicture(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	err := uc.userService.DeleteProfilePicture(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Delete profile picture failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to delete profile picture")
		return
	}

	utils.SuccessResponse(c, "Profile picture deleted successfully", nil)
}

// UpdateLocationSettings updates user's location sharing settings
// @Summary Update location settings
// @Description Update user's location sharing preferences
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.LocationSharing true "Location sharing settings"
// @Success 200 {object} models.APIResponse{data=models.LocationSharing}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/location-settings [put]
func (uc *UserController) UpdateLocationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var settings models.LocationSharing
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.BadRequestResponse(c, "Invalid location settings")
		return
	}

	updatedSettings, err := uc.userService.UpdateLocationSettings(c.Request.Context(), userID, settings)
	if err != nil {
		logrus.Errorf("Update location settings failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid location settings")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update location settings")
		}
		return
	}

	utils.SuccessResponse(c, "Location settings updated successfully", updatedSettings)
}

// UpdatePrivacySettings updates user's privacy settings
// @Summary Update privacy settings
// @Description Update user's privacy preferences
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.PrivacySettings true "Privacy settings"
// @Success 200 {object} models.APIResponse{data=models.PrivacySettings}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/privacy-settings [put]
func (uc *UserController) UpdatePrivacySettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var settings models.PrivacySettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.BadRequestResponse(c, "Invalid privacy settings")
		return
	}

	updatedSettings, err := uc.userService.UpdatePrivacySettings(c.Request.Context(), userID, settings)
	if err != nil {
		logrus.Errorf("Update privacy settings failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid privacy settings")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update privacy settings")
		}
		return
	}

	utils.SuccessResponse(c, "Privacy settings updated successfully", updatedSettings)
}

// UpdateDrivingSettings updates user's driving detection settings
// @Summary Update driving settings
// @Description Update user's driving detection preferences
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.DrivingPrefs true "Driving settings"
// @Success 200 {object} models.APIResponse{data=models.DrivingPrefs}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/driving-settings [put]
func (uc *UserController) UpdateDrivingSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var settings models.DrivingPrefs
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.BadRequestResponse(c, "Invalid driving settings")
		return
	}

	updatedSettings, err := uc.userService.UpdateDrivingSettings(c.Request.Context(), userID, settings)
	if err != nil {
		logrus.Errorf("Update driving settings failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid driving settings")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update driving settings")
		}
		return
	}

	utils.SuccessResponse(c, "Driving settings updated successfully", updatedSettings)
}

// UpdateEmergencyContact updates user's emergency contact
// @Summary Update emergency contact
// @Description Update user's emergency contact information
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.EmergencyContact true "Emergency contact"
// @Success 200 {object} models.APIResponse{data=models.EmergencyContact}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/emergency-contact [put]
func (uc *UserController) UpdateEmergencyContact(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var contact models.EmergencyContact
	if err := c.ShouldBindJSON(&contact); err != nil {
		utils.BadRequestResponse(c, "Invalid emergency contact data")
		return
	}

	updatedContact, err := uc.userService.UpdateEmergencyContact(c.Request.Context(), userID, contact)
	if err != nil {
		logrus.Errorf("Update emergency contact failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid emergency contact data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update emergency contact")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency contact updated successfully", updatedContact)
}

// GetUserStats gets user's activity statistics
// @Summary Get user statistics
// @Description Get user's activity and usage statistics
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param period query string false "Time period (day, week, month, year)" default(month)
// @Success 200 {object} models.APIResponse{data=models.UserStats}
// @Failure 401 {object} models.APIResponse
// @Router /users/stats [get]
func (uc *UserController) GetUserStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.Query("period")
	if period == "" {
		period = "month"
	}

	stats, err := uc.userService.GetUserStats(c.Request.Context(), userID, period)
	if err != nil {
		logrus.Errorf("Get user stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get user statistics")
		return
	}

	utils.SuccessResponse(c, "User statistics retrieved successfully", stats)
}

// DeactivateAccount deactivates user's account
// @Summary Deactivate account
// @Description Temporarily deactivate user's account
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{reason=string,password=string} true "Deactivation data"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/deactivate [post]
func (uc *UserController) DeactivateAccount(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Reason   string `json:"reason"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Password is required for account deactivation")
		return
	}

	err := uc.userService.DeactivateAccount(c.Request.Context(), userID, req.Password, req.Reason)
	if err != nil {
		logrus.Errorf("Deactivate account failed: %v", err)

		switch err.Error() {
		case "invalid password":
			utils.UnauthorizedResponse(c, "Invalid password")
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "account already deactivated":
			utils.BadRequestResponse(c, "Account is already deactivated")
		default:
			utils.InternalServerErrorResponse(c, "Failed to deactivate account")
		}
		return
	}

	utils.SuccessResponse(c, "Account deactivated successfully", nil)
}

// ReactivateAccount reactivates user's account
// @Summary Reactivate account
// @Description Reactivate a previously deactivated account
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{password=string} true "Reactivation data"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/reactivate [post]
func (uc *UserController) ReactivateAccount(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Password is required for account reactivation")
		return
	}

	err := uc.userService.ReactivateAccount(c.Request.Context(), userID, req.Password)
	if err != nil {
		logrus.Errorf("Reactivate account failed: %v", err)

		switch err.Error() {
		case "invalid password":
			utils.UnauthorizedResponse(c, "Invalid password")
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "account already active":
			utils.BadRequestResponse(c, "Account is already active")
		default:
			utils.InternalServerErrorResponse(c, "Failed to reactivate account")
		}
		return
	}

	utils.SuccessResponse(c, "Account reactivated successfully", nil)
}

// DeleteAccount permanently deletes user's account
// @Summary Delete account
// @Description Permanently delete user's account and all associated data
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{password=string,confirmation=string} true "Deletion confirmation"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/delete [post]
func (uc *UserController) DeleteAccount(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Password     string `json:"password" binding:"required"`
		Confirmation string `json:"confirmation" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Password and confirmation are required")
		return
	}

	if req.Confirmation != "DELETE_MY_ACCOUNT" {
		utils.BadRequestResponse(c, "Invalid confirmation. Please type 'DELETE_MY_ACCOUNT'")
		return
	}

	err := uc.userService.DeleteAccount(c.Request.Context(), userID, req.Password)
	if err != nil {
		logrus.Errorf("Delete account failed: %v", err)

		switch err.Error() {
		case "invalid password":
			utils.UnauthorizedResponse(c, "Invalid password")
		case "user not found":
			utils.NotFoundResponse(c, "User")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete account")
		}
		return
	}

	utils.SuccessResponse(c, "Account deleted successfully", nil)
}

// GetDevices gets user's registered devices
// @Summary Get user devices
// @Description Get list of user's registered devices
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.UserDevice}
// @Failure 401 {object} models.APIResponse
// @Router /users/devices [get]
func (uc *UserController) GetDevices(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	devices, err := uc.userService.GetUserDevices(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get devices failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get user devices")
		return
	}

	utils.SuccessResponse(c, "User devices retrieved successfully", devices)
}

// RemoveDevice removes a device from user's account
// @Summary Remove device
// @Description Remove a device from user's account
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param deviceId path string true "Device ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/devices/{deviceId} [delete]
func (uc *UserController) RemoveDevice(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	deviceID := c.Param("deviceId")
	if deviceID == "" {
		utils.BadRequestResponse(c, "Device ID is required")
		return
	}

	err := uc.userService.RemoveDevice(c.Request.Context(), userID, deviceID)
	if err != nil {
		logrus.Errorf("Remove device failed: %v", err)

		switch err.Error() {
		case "device not found":
			utils.NotFoundResponse(c, "Device")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only remove your own devices")
		default:
			utils.InternalServerErrorResponse(c, "Failed to remove device")
		}
		return
	}

	utils.SuccessResponse(c, "Device removed successfully", nil)
}

// ExportData exports user's data
// @Summary Export user data
// @Description Export user's data for backup or transfer
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param format query string false "Export format (json, csv)" default(json)
// @Success 200 {object} models.APIResponse{data=object{downloadUrl=string}}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/export [post]
func (uc *UserController) ExportData(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	if format != "json" && format != "csv" {
		utils.BadRequestResponse(c, "Invalid format. Supported formats: json, csv")
		return
	}

	downloadURL, err := uc.userService.ExportUserData(c.Request.Context(), userID, format)
	if err != nil {
		logrus.Errorf("Export data failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to export user data")
		return
	}

	response := map[string]interface{}{
		"downloadUrl": downloadURL,
		"format":      format,
		"expiresIn":   "24h",
	}

	utils.SuccessResponse(c, "Data export initiated successfully", response)
}
