package controllers

import (
	"fmt"
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

// =============================================
// CURRENT USER ENDPOINTS
// =============================================

// GetCurrentUser gets the authenticated user's information
// @Summary Get current user
// @Description Get authenticated user's complete information
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.User}
// @Failure 401 {object} models.APIResponse
// @Router /users/me [get]
func (uc *UserController) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	user, err := uc.userService.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get current user failed: %v", err)
		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get user information")
		}
		return
	}

	utils.SuccessResponse(c, "User information retrieved successfully", user)
}

// UpdateCurrentUser updates the authenticated user's information
// @Summary Update current user
// @Description Update authenticated user's information
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.UpdateUserRequest true "Updated user data"
// @Success 200 {object} models.APIResponse{data=models.User}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me [put]
func (uc *UserController) UpdateCurrentUser(c *gin.Context) {
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

	user, err := uc.userService.UpdateCurrentUser(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update current user failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid user data")
		case "no fields to update":
			utils.BadRequestResponse(c, "No fields provided for update")
		case "user not found":
			utils.NotFoundResponse(c, "User")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update user information")
		}
		return
	}

	utils.SuccessResponse(c, "User information updated successfully", user)
}

// DeleteCurrentUser deletes the authenticated user's account
// @Summary Delete current user
// @Description Delete authenticated user's account
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me [delete]
func (uc *UserController) DeleteCurrentUser(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	err := uc.userService.DeleteCurrentUser(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Delete current user failed: %v", err)
		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete user account")
		}
		return
	}

	utils.SuccessResponse(c, "User account deleted successfully", nil)
}

// GetProfile gets user's profile
// @Summary Get user profile
// @Description Get authenticated user's profile information
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.User}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/profile [get]
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
// @Router /users/me/profile [put]
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

// =============================================
// PROFILE PICTURE MANAGEMENT
// =============================================

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
// @Router /users/me/profile-picture [post]
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
// @Router /users/me/profile-picture [delete]
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

// GetProfilePicture gets user's profile picture URL
// @Summary Get profile picture
// @Description Get the authenticated user's profile picture URL
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=object{profilePicture=string}}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/profile-picture [get]
func (uc *UserController) GetProfilePicture(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	profilePictureURL, err := uc.userService.GetProfilePicture(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get profile picture failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get profile picture")
		return
	}

	response := map[string]interface{}{
		"profilePicture": profilePictureURL,
	}

	utils.SuccessResponse(c, "Profile picture retrieved successfully", response)
}

// =============================================
// USER PREFERENCES AND SETTINGS
// =============================================

// GetUserSettings gets user's settings
// @Summary Get user settings
// @Description Get authenticated user's preferences and settings
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.UserPreferences}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/settings [get]
func (uc *UserController) GetUserSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := uc.userService.GetUserSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get user settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get user settings")
		return
	}

	utils.SuccessResponse(c, "User settings retrieved successfully", settings)
}

// UpdateUserSettings updates user's settings
// @Summary Update user settings
// @Description Update authenticated user's preferences and settings
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.UserPreferences true "User preferences"
// @Success 200 {object} models.APIResponse{data=models.UserPreferences}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me/settings [put]
func (uc *UserController) UpdateUserSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var preferences models.UserPreferences
	if err := c.ShouldBindJSON(&preferences); err != nil {
		utils.BadRequestResponse(c, "Invalid settings data")
		return
	}

	updatedSettings, err := uc.userService.UpdateUserSettings(c.Request.Context(), userID, preferences)
	if err != nil {
		logrus.Errorf("Update user settings failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid settings data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update user settings")
		}
		return
	}

	utils.SuccessResponse(c, "User settings updated successfully", updatedSettings)
}

// GetPrivacySettings gets user's privacy settings
// @Summary Get privacy settings
// @Description Get authenticated user's privacy preferences
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.PrivacySettings}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/settings/privacy [get]
func (uc *UserController) GetPrivacySettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := uc.userService.GetPrivacySettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get privacy settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get privacy settings")
		return
	}

	utils.SuccessResponse(c, "Privacy settings retrieved successfully", settings)
}

// UpdatePrivacySettings updates user's privacy settings
// @Summary Update privacy settings
// @Description Update authenticated user's privacy preferences
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.PrivacySettings true "Privacy settings"
// @Success 200 {object} models.APIResponse{data=models.PrivacySettings}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me/settings/privacy [put]
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

// GetNotificationSettings gets user's notification settings
// @Summary Get notification settings
// @Description Get authenticated user's notification preferences
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.NotificationPrefs}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/settings/notifications [get]
func (uc *UserController) GetNotificationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := uc.userService.GetNotificationSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get notification settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification settings")
		return
	}

	utils.SuccessResponse(c, "Notification settings retrieved successfully", settings)
}

// UpdateNotificationSettings updates user's notification settings
// @Summary Update notification settings
// @Description Update authenticated user's notification preferences
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.NotificationPrefs true "Notification settings"
// @Success 200 {object} models.APIResponse{data=models.NotificationPrefs}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me/settings/notifications [put]
func (uc *UserController) UpdateNotificationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var settings models.NotificationPrefs
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.BadRequestResponse(c, "Invalid notification settings")
		return
	}

	updatedSettings, err := uc.userService.UpdateNotificationSettings(c.Request.Context(), userID, settings)
	if err != nil {
		logrus.Errorf("Update notification settings failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid notification settings")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update notification settings")
		}
		return
	}

	utils.SuccessResponse(c, "Notification settings updated successfully", updatedSettings)
}

// GetLocationSettings gets user's location settings
// @Summary Get location settings
// @Description Get authenticated user's location sharing preferences
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.LocationSharing}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/settings/location [get]
func (uc *UserController) GetLocationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := uc.userService.GetLocationSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get location settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get location settings")
		return
	}

	utils.SuccessResponse(c, "Location settings retrieved successfully", settings)
}

// UpdateLocationSettings updates user's location settings
// @Summary Update location settings
// @Description Update authenticated user's location sharing preferences
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.LocationSharing true "Location sharing settings"
// @Success 200 {object} models.APIResponse{data=models.LocationSharing}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me/settings/location [put]
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

// GetDrivingSettings gets user's driving settings
// @Summary Get driving settings
// @Description Get authenticated user's driving detection preferences
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.DrivingPrefs}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/settings/driving [get]
func (uc *UserController) GetDrivingSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := uc.userService.GetDrivingSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get driving settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get driving settings")
		return
	}

	utils.SuccessResponse(c, "Driving settings retrieved successfully", settings)
}

// UpdateDrivingSettings updates user's driving settings
// @Summary Update driving settings
// @Description Update authenticated user's driving detection preferences
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.DrivingPrefs true "Driving settings"
// @Success 200 {object} models.APIResponse{data=models.DrivingPrefs}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me/settings/driving [put]
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

// Continue adding these methods to your user_controller.go file after the previous methods

// =============================================
// EMERGENCY CONTACTS
// =============================================

// GetEmergencyContacts gets user's emergency contacts
// @Summary Get emergency contacts
// @Description Get authenticated user's emergency contacts
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.EmergencyContact}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/emergency-contacts [get]
func (uc *UserController) GetEmergencyContacts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	contacts, err := uc.userService.GetEmergencyContacts(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get emergency contacts failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get emergency contacts")
		return
	}

	utils.SuccessResponse(c, "Emergency contacts retrieved successfully", contacts)
}

// AddEmergencyContact adds a new emergency contact
// @Summary Add emergency contact
// @Description Add a new emergency contact for the authenticated user
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.EmergencyContact true "Emergency contact data"
// @Success 201 {object} models.APIResponse{data=models.EmergencyContact}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me/emergency-contacts [post]
func (uc *UserController) AddEmergencyContact(c *gin.Context) {
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

	addedContact, err := uc.userService.AddEmergencyContact(c.Request.Context(), userID, contact)
	if err != nil {
		logrus.Errorf("Add emergency contact failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid emergency contact data")
		case "contact limit exceeded":
			utils.BadRequestResponse(c, "Maximum number of emergency contacts reached")
		default:
			utils.InternalServerErrorResponse(c, "Failed to add emergency contact")
		}
		return
	}

	utils.CreatedResponse(c, "Emergency contact added successfully", addedContact)
}

// UpdateEmergencyContact updates an emergency contact
// @Summary Update emergency contact
// @Description Update an existing emergency contact
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param contactId path string true "Contact ID"
// @Param request body models.EmergencyContact true "Updated emergency contact data"
// @Success 200 {object} models.APIResponse{data=models.EmergencyContact}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/emergency-contacts/{contactId} [put]
func (uc *UserController) UpdateEmergencyContact(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	contactID := c.Param("contactId")
	if contactID == "" {
		utils.BadRequestResponse(c, "Contact ID is required")
		return
	}

	var contact models.EmergencyContact
	if err := c.ShouldBindJSON(&contact); err != nil {
		utils.BadRequestResponse(c, "Invalid emergency contact data")
		return
	}

	updatedContact, err := uc.userService.UpdateEmergencyContact(c.Request.Context(), userID, contactID, contact)
	if err != nil {
		logrus.Errorf("Update emergency contact failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid emergency contact data")
		case "contact not found":
			utils.NotFoundResponse(c, "Emergency contact")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update emergency contact")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency contact updated successfully", updatedContact)
}

// DeleteEmergencyContact deletes an emergency contact
// @Summary Delete emergency contact
// @Description Delete an existing emergency contact
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param contactId path string true "Contact ID"
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/emergency-contacts/{contactId} [delete]
func (uc *UserController) DeleteEmergencyContact(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	contactID := c.Param("contactId")
	if contactID == "" {
		utils.BadRequestResponse(c, "Contact ID is required")
		return
	}

	err := uc.userService.DeleteEmergencyContact(c.Request.Context(), userID, contactID)
	if err != nil {
		logrus.Errorf("Delete emergency contact failed: %v", err)
		switch err.Error() {
		case "contact not found":
			utils.NotFoundResponse(c, "Emergency contact")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete emergency contact")
		}
		return
	}

	utils.SuccessResponse(c, "Emergency contact deleted successfully", nil)
}

// VerifyEmergencyContact verifies an emergency contact
// @Summary Verify emergency contact
// @Description Send verification request to emergency contact
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param contactId path string true "Contact ID"
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/emergency-contacts/{contactId}/verify [post]
func (uc *UserController) VerifyEmergencyContact(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	contactID := c.Param("contactId")
	if contactID == "" {
		utils.BadRequestResponse(c, "Contact ID is required")
		return
	}

	err := uc.userService.VerifyEmergencyContact(c.Request.Context(), userID, contactID)
	if err != nil {
		logrus.Errorf("Verify emergency contact failed: %v", err)
		switch err.Error() {
		case "contact not found":
			utils.NotFoundResponse(c, "Emergency contact")
		case "already verified":
			utils.BadRequestResponse(c, "Contact is already verified")
		default:
			utils.InternalServerErrorResponse(c, "Failed to verify emergency contact")
		}
		return
	}

	utils.SuccessResponse(c, "Verification request sent successfully", nil)
}

// =============================================
// DEVICE MANAGEMENT
// =============================================

// GetUserDevices gets user's registered devices
// @Summary Get user devices
// @Description Get authenticated user's registered devices
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.UserDevice}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/devices [get]
func (uc *UserController) GetUserDevices(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	devices, err := uc.userService.GetUserDevices(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get user devices failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get user devices")
		return
	}

	utils.SuccessResponse(c, "User devices retrieved successfully", devices)
}

// RegisterDevice registers a new device
// @Summary Register device
// @Description Register a new device for push notifications
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.RegisterDeviceRequest true "Device registration data"
// @Success 201 {object} models.APIResponse{data=models.UserDevice}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me/devices/register [post]
func (uc *UserController) RegisterDevice(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid device data")
		return
	}

	device, err := uc.userService.RegisterDevice(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Register device failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid device data")
		case "device already registered":
			utils.ConflictResponse(c, "Device already registered")
		default:
			utils.InternalServerErrorResponse(c, "Failed to register device")
		}
		return
	}

	utils.CreatedResponse(c, "Device registered successfully", device)
}

// UpdateDevice updates a device
// @Summary Update device
// @Description Update device information
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param deviceId path string true "Device ID"
// @Param request body models.UpdateDeviceRequest true "Updated device data"
// @Success 200 {object} models.APIResponse{data=models.UserDevice}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/devices/{deviceId} [put]
func (uc *UserController) UpdateDevice(c *gin.Context) {
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

	var req models.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid device data")
		return
	}

	device, err := uc.userService.UpdateDevice(c.Request.Context(), userID, deviceID, req)
	if err != nil {
		logrus.Errorf("Update device failed: %v", err)
		switch err.Error() {
		case "device not found":
			utils.NotFoundResponse(c, "Device")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this device")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid device data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update device")
		}
		return
	}

	utils.SuccessResponse(c, "Device updated successfully", device)
}

// UnregisterDevice unregisters a device
// @Summary Unregister device
// @Description Remove device registration
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param deviceId path string true "Device ID"
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/devices/{deviceId} [delete]
func (uc *UserController) UnregisterDevice(c *gin.Context) {
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

	err := uc.userService.UnregisterDevice(c.Request.Context(), userID, deviceID)
	if err != nil {
		logrus.Errorf("Unregister device failed: %v", err)
		switch err.Error() {
		case "device not found":
			utils.NotFoundResponse(c, "Device")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this device")
		default:
			utils.InternalServerErrorResponse(c, "Failed to unregister device")
		}
		return
	}

	utils.SuccessResponse(c, "Device unregistered successfully", nil)
}

// TestPushNotification sends a test push notification
// @Summary Test push notification
// @Description Send a test push notification to a specific device
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param deviceId path string true "Device ID"
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/devices/{deviceId}/push-test [post]
func (uc *UserController) TestPushNotification(c *gin.Context) {
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

	err := uc.userService.TestPushNotification(c.Request.Context(), userID, deviceID)
	if err != nil {
		logrus.Errorf("Test push notification failed: %v", err)
		switch err.Error() {
		case "device not found":
			utils.NotFoundResponse(c, "Device")
		case "no device token found":
			utils.BadRequestResponse(c, "Device has no push token")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this device")
		default:
			utils.InternalServerErrorResponse(c, "Failed to send test notification")
		}
		return
	}

	utils.SuccessResponse(c, "Test notification sent successfully", nil)
}

// =============================================
// SOCIAL FEATURES
// =============================================

// SearchUsers searches for users
// @Summary Search users
// @Description Search for users by name or email
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Search limit" default(20)
// @Success 200 {object} models.APIResponse{data=[]models.UserSearchResult}
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

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 20
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

// GetUserByID gets another user's public profile
// @Summary Get user by ID
// @Description Get another user's public profile information
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} models.APIResponse{data=models.User}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/{userId} [get]
func (uc *UserController) GetUserByID(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		utils.BadRequestResponse(c, "User ID is required")
		return
	}

	user, err := uc.userService.GetUser(c.Request.Context(), userID, targetUserID)
	if err != nil {
		logrus.Errorf("Get user by ID failed: %v", err)
		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "access denied":
			utils.ForbiddenResponse(c, "Access to this user profile is restricted")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get user profile")
		}
		return
	}

	utils.SuccessResponse(c, "User profile retrieved successfully", user)
}

// GetPublicProfile gets another user's public profile
// @Summary Get public profile
// @Description Get another user's public profile information
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} models.APIResponse{data=models.User}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/{userId}/profile [get]
func (uc *UserController) GetPublicProfile(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		utils.BadRequestResponse(c, "User ID is required")
		return
	}

	user, err := uc.userService.GetPublicProfile(c.Request.Context(), userID, targetUserID)
	if err != nil {
		logrus.Errorf("Get public profile failed: %v", err)
		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "access denied":
			utils.ForbiddenResponse(c, "Access to this user profile is restricted")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get public profile")
		}
		return
	}

	utils.SuccessResponse(c, "Public profile retrieved successfully", user)
}

// SendFriendRequest sends a friend request
// @Summary Send friend request
// @Description Send a friend request to another user
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param request body models.FriendRequestSend true "Friend request data"
// @Success 201 {object} models.APIResponse{data=models.FriendRequest}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/{userId}/friend-request [post]
func (uc *UserController) SendFriendRequest(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		utils.BadRequestResponse(c, "User ID is required")
		return
	}

	var req models.FriendRequestSend
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request data")
		return
	}

	friendRequest, err := uc.userService.SendFriendRequest(c.Request.Context(), userID, targetUserID, req)
	if err != nil {
		logrus.Errorf("Send friend request failed: %v", err)
		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "cannot send to yourself":
			utils.BadRequestResponse(c, "Cannot send friend request to yourself")
		case "already friends":
			utils.BadRequestResponse(c, "Already friends with this user")
		case "request already sent":
			utils.BadRequestResponse(c, "Friend request already sent")
		case "user blocked":
			utils.ForbiddenResponse(c, "Cannot send friend request to this user")
		default:
			utils.InternalServerErrorResponse(c, "Failed to send friend request")
		}
		return
	}

	utils.CreatedResponse(c, "Friend request sent successfully", friendRequest)
}

// Continue adding these methods to your user_controller.go file

// AcceptFriendRequest accepts a friend request
// @Summary Accept friend request
// @Description Accept a friend request
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param requestId path string true "Friend Request ID"
// @Success 200 {object} models.APIResponse{data=models.FriendRequest}
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/friend-requests/{requestId}/accept [put]
func (uc *UserController) AcceptFriendRequest(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	requestID := c.Param("requestId")
	if requestID == "" {
		utils.BadRequestResponse(c, "Friend request ID is required")
		return
	}

	friendRequest, err := uc.userService.AcceptFriendRequest(c.Request.Context(), userID, requestID)
	if err != nil {
		logrus.Errorf("Accept friend request failed: %v", err)
		switch err.Error() {
		case "request not found":
			utils.NotFoundResponse(c, "Friend request")
		case "access denied":
			utils.ForbiddenResponse(c, "Cannot accept this friend request")
		case "already processed":
			utils.BadRequestResponse(c, "Friend request already processed")
		default:
			utils.InternalServerErrorResponse(c, "Failed to accept friend request")
		}
		return
	}

	utils.SuccessResponse(c, "Friend request accepted successfully", friendRequest)
}

// DeclineFriendRequest declines a friend request
// @Summary Decline friend request
// @Description Decline a friend request
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param requestId path string true "Friend Request ID"
// @Success 200 {object} models.APIResponse{data=models.FriendRequest}
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/friend-requests/{requestId}/decline [put]
func (uc *UserController) DeclineFriendRequest(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	requestID := c.Param("requestId")
	if requestID == "" {
		utils.BadRequestResponse(c, "Friend request ID is required")
		return
	}

	friendRequest, err := uc.userService.DeclineFriendRequest(c.Request.Context(), userID, requestID)
	if err != nil {
		logrus.Errorf("Decline friend request failed: %v", err)
		switch err.Error() {
		case "request not found":
			utils.NotFoundResponse(c, "Friend request")
		case "access denied":
			utils.ForbiddenResponse(c, "Cannot decline this friend request")
		case "already processed":
			utils.BadRequestResponse(c, "Friend request already processed")
		default:
			utils.InternalServerErrorResponse(c, "Failed to decline friend request")
		}
		return
	}

	utils.SuccessResponse(c, "Friend request declined successfully", friendRequest)
}

// GetFriendRequests gets user's friend requests
// @Summary Get friend requests
// @Description Get pending friend requests for the authenticated user
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param type query string false "Request type" Enums(sent, received, all) default(received)
// @Success 200 {object} models.APIResponse{data=[]models.FriendRequest}
// @Failure 401 {object} models.APIResponse
// @Router /users/friend-requests [get]
func (uc *UserController) GetFriendRequests(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	requestType := c.DefaultQuery("type", "received")

	requests, err := uc.userService.GetFriendRequests(c.Request.Context(), userID, requestType)
	if err != nil {
		logrus.Errorf("Get friend requests failed: %v", err)
		switch err.Error() {
		case "invalid request type":
			utils.BadRequestResponse(c, "Invalid request type. Use 'sent', 'received', or 'all'")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get friend requests")
		}
		return
	}

	utils.SuccessResponse(c, "Friend requests retrieved successfully", requests)
}

// GetFriends gets user's friends
// @Summary Get friends
// @Description Get the authenticated user's friends list
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.User}
// @Failure 401 {object} models.APIResponse
// @Router /users/friends [get]
func (uc *UserController) GetFriends(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	friends, err := uc.userService.GetFriends(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get friends failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get friends list")
		return
	}

	utils.SuccessResponse(c, "Friends list retrieved successfully", friends)
}

// RemoveFriend removes a friend
// @Summary Remove friend
// @Description Remove a user from friends list
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/friends/{userId} [delete]
func (uc *UserController) RemoveFriend(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	friendUserID := c.Param("userId")
	if friendUserID == "" {
		utils.BadRequestResponse(c, "User ID is required")
		return
	}

	err := uc.userService.RemoveFriend(c.Request.Context(), userID, friendUserID)
	if err != nil {
		logrus.Errorf("Remove friend failed: %v", err)
		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "not friends":
			utils.BadRequestResponse(c, "You are not friends with this user")
		default:
			utils.InternalServerErrorResponse(c, "Failed to remove friend")
		}
		return
	}

	utils.SuccessResponse(c, "Friend removed successfully", nil)
}

// =============================================
// BLOCKING AND REPORTING
// =============================================

// GetBlockedUsers gets user's blocked users
// @Summary Get blocked users
// @Description Get list of users blocked by the authenticated user
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.BlockedUser}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/moderation/blocked [get]
func (uc *UserController) GetBlockedUsers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	blockedUsers, err := uc.userService.GetBlockedUsers(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get blocked users failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get blocked users")
		return
	}

	utils.SuccessResponse(c, "Blocked users retrieved successfully", blockedUsers)
}

// BlockUser blocks a user
// @Summary Block user
// @Description Block another user
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param request body models.BlockUserRequest true "Block reason"
// @Success 201 {object} models.APIResponse{data=models.BlockedUser}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/moderation/block/{userId} [post]
func (uc *UserController) BlockUser(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		utils.BadRequestResponse(c, "User ID is required")
		return
	}

	var req models.BlockUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request data")
		return
	}

	blockedUser, err := uc.userService.BlockUser(c.Request.Context(), userID, targetUserID, req)
	if err != nil {
		logrus.Errorf("Block user failed: %v", err)
		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "cannot block yourself":
			utils.BadRequestResponse(c, "Cannot block yourself")
		case "already blocked":
			utils.BadRequestResponse(c, "User is already blocked")
		default:
			utils.InternalServerErrorResponse(c, "Failed to block user")
		}
		return
	}

	utils.CreatedResponse(c, "User blocked successfully", blockedUser)
}

// UnblockUser unblocks a user
// @Summary Unblock user
// @Description Unblock a previously blocked user
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/moderation/block/{userId} [delete]
func (uc *UserController) UnblockUser(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		utils.BadRequestResponse(c, "User ID is required")
		return
	}

	err := uc.userService.UnblockUser(c.Request.Context(), userID, targetUserID)
	if err != nil {
		logrus.Errorf("Unblock user failed: %v", err)
		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "not blocked":
			utils.BadRequestResponse(c, "User is not blocked")
		default:
			utils.InternalServerErrorResponse(c, "Failed to unblock user")
		}
		return
	}

	utils.SuccessResponse(c, "User unblocked successfully", nil)
}

// ReportUser reports a user
// @Summary Report user
// @Description Report another user for inappropriate behavior
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param request body models.ReportUserRequest true "Report data"
// @Success 201 {object} models.APIResponse{data=models.UserReport}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/moderation/report/{userId} [post]
func (uc *UserController) ReportUser(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		utils.BadRequestResponse(c, "User ID is required")
		return
	}

	var req models.ReportUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid report data")
		return
	}

	report, err := uc.userService.ReportUser(c.Request.Context(), userID, targetUserID, req)
	if err != nil {
		logrus.Errorf("Report user failed: %v", err)
		switch err.Error() {
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "cannot report yourself":
			utils.BadRequestResponse(c, "Cannot report yourself")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid report data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to report user")
		}
		return
	}

	utils.CreatedResponse(c, "User reported successfully", report)
}

// =============================================
// DATA EXPORT AND PRIVACY
// =============================================

// ExportUserData requests user data export
// @Summary Export user data
// @Description Request export of all user data
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.ExportUserDataRequest true "Export options"
// @Success 202 {object} models.APIResponse{data=models.UserDataExport}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me/data/export [get]
func (uc *UserController) ExportUserData(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.ExportUserDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use default export options if no body provided
		req = models.ExportUserDataRequest{
			DataTypes: []string{"profile", "locations", "circles", "messages"},
		}
	}

	export, err := uc.userService.ExportUserData(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Export user data failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid export options")
		case "export in progress":
			utils.BadRequestResponse(c, "Data export already in progress")
		default:
			utils.InternalServerErrorResponse(c, "Failed to initiate data export")
		}
		return
	}

	utils.AcceptedResponse(c, "Data export initiated successfully", export)
}

// GetExportStatus gets export status
// @Summary Get export status
// @Description Get status of data export request
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.UserDataExport}
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/data/export/status [get]
func (uc *UserController) GetExportStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	export, err := uc.userService.GetExportStatus(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get export status failed: %v", err)
		switch err.Error() {
		case "no export found":
			utils.NotFoundResponse(c, "Data export")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get export status")
		}
		return
	}

	utils.SuccessResponse(c, "Export status retrieved successfully", export)
}

// DownloadExport downloads exported data
// @Summary Download export
// @Description Download exported user data
// @Tags Users
// @Security BearerAuth
// @Produce application/octet-stream
// @Param exportId path string true "Export ID"
// @Success 200 {file} file
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /users/me/data/download/{exportId} [post]
func (uc *UserController) DownloadExport(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	exportID := c.Param("exportId")
	if exportID == "" {
		utils.BadRequestResponse(c, "Export ID is required")
		return
	}

	exportFile, err := uc.userService.DownloadExport(c.Request.Context(), userID, exportID)
	if err != nil {
		logrus.Errorf("Download export failed: %v", err)
		switch err.Error() {
		case "export not found":
			utils.NotFoundResponse(c, "Export")
		case "export not ready":
			utils.BadRequestResponse(c, "Export is not ready for download")
		case "export expired":
			utils.BadRequestResponse(c, "Export has expired")
		case "access denied":
			utils.ForbiddenResponse(c, "Access denied to this export")
		default:
			utils.InternalServerErrorResponse(c, "Failed to download export")
		}
		return
	}

	c.Header("Content-Type", exportFile.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", exportFile.Filename))
	c.Data(200, exportFile.ContentType, exportFile.Data)
}

// RequestDataPurge requests data purge
// @Summary Request data purge
// @Description Request permanent deletion of all user data
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.DataPurgeRequest true "Purge request"
// @Success 202 {object} models.APIResponse{data=models.DataPurgeRequest}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /users/me/data/purge [delete]
func (uc *UserController) RequestDataPurge(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.DataPurgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid purge request data")
		return
	}

	purgeRequest, err := uc.userService.RequestDataPurge(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Request data purge failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid purge request data")
		case "purge already requested":
			utils.BadRequestResponse(c, "Data purge already requested")
		default:
			utils.InternalServerErrorResponse(c, "Failed to request data purge")
		}
		return
	}

	utils.AcceptedResponse(c, "Data purge request submitted successfully", purgeRequest)
}

// =============================================
// ACCOUNT STATISTICS
// =============================================

// GetUserStats gets user statistics
// @Summary Get user statistics
// @Description Get comprehensive user statistics
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.UserStatsResponse}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/stats [get]
func (uc *UserController) GetUserStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	stats, err := uc.userService.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get user stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get user statistics")
		return
	}

	utils.SuccessResponse(c, "User statistics retrieved successfully", stats)
}

// GetActivityStats gets user activity statistics
// @Summary Get activity statistics
// @Description Get user activity statistics over time
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param period query string false "Time period" Enums(day, week, month, year) default(month)
// @Success 200 {object} models.APIResponse{data=models.UserActivityStats}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/stats/activity [get]
func (uc *UserController) GetActivityStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.DefaultQuery("period", "month")

	stats, err := uc.userService.GetActivityStats(c.Request.Context(), userID, period)
	if err != nil {
		logrus.Errorf("Get activity stats failed: %v", err)
		switch err.Error() {
		case "invalid period":
			utils.BadRequestResponse(c, "Invalid period. Use 'day', 'week', 'month', or 'year'")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get activity statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Activity statistics retrieved successfully", stats)
}

// GetLocationStats gets user location statistics
// @Summary Get location statistics
// @Description Get user location and travel statistics
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param period query string false "Time period" Enums(day, week, month, year) default(month)
// @Success 200 {object} models.APIResponse{data=models.LocationStatsResponse}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/stats/location [get]
func (uc *UserController) GetLocationStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.DefaultQuery("period", "month")

	stats, err := uc.userService.GetLocationStats(c.Request.Context(), userID, period)
	if err != nil {
		logrus.Errorf("Get location stats failed: %v", err)
		switch err.Error() {
		case "invalid period":
			utils.BadRequestResponse(c, "Invalid period. Use 'day', 'week', 'month', or 'year'")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get location statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Location statistics retrieved successfully", stats)
}

// GetDrivingStats gets user driving statistics
// @Summary Get driving statistics
// @Description Get user driving behavior and safety statistics
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param period query string false "Time period" Enums(day, week, month, year) default(month)
// @Success 200 {object} models.APIResponse{data=models.DrivingStatsResponse}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/stats/driving [get]
func (uc *UserController) GetDrivingStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.DefaultQuery("period", "month")

	stats, err := uc.userService.GetDrivingStats(c.Request.Context(), userID, period)
	if err != nil {
		logrus.Errorf("Get driving stats failed: %v", err)
		switch err.Error() {
		case "invalid period":
			utils.BadRequestResponse(c, "Invalid period. Use 'day', 'week', 'month', or 'year'")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get driving statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Driving statistics retrieved successfully", stats)
}

// GetCircleStats gets user circle statistics
// @Summary Get circle statistics
// @Description Get user circle participation statistics
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.CircleStatsResponse}
// @Failure 401 {object} models.APIResponse
// @Router /users/me/stats/circles [get]
func (uc *UserController) GetCircleStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	stats, err := uc.userService.GetCircleStats(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get circle stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get circle statistics")
		return
	}

	utils.SuccessResponse(c, "Circle statistics retrieved successfully", stats)
}
