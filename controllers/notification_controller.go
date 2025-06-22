package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type NotificationController struct {
	notificationService *services.NotificationService
}

func NewNotificationController(notificationService *services.NotificationService) *NotificationController {
	return &NotificationController{
		notificationService: notificationService,
	}
}

// ========================
// Basic Notification Operations
// ========================

// GetNotifications gets user's notifications with pagination and filtering
// @Summary Get notifications
// @Description Get user's notifications with pagination and filtering
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Param type query string false "Notification type filter"
// @Param status query string false "Notification status filter"
// @Success 200 {object} models.APIResponse{data=models.PaginatedNotifications}
// @Failure 401 {object} models.APIResponse
// @Router /notifications [get]
func (nc *NotificationController) GetNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	notificationType := c.Query("type")
	status := c.Query("status")

	req := models.GetNotificationsRequest{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		Type:     notificationType,
		Status:   status,
	}

	notifications, err := nc.notificationService.GetNotifications(c.Request.Context(), req)
	if err != nil {
		logrus.Errorf("Get notifications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notifications")
		return
	}

	utils.SuccessResponse(c, "Notifications retrieved successfully", notifications)
}

// GetNotification gets a specific notification by ID
// @Summary Get notification
// @Description Get a specific notification by ID
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param notificationId path string true "Notification ID"
// @Success 200 {object} models.APIResponse{data=models.Notification}
// @Failure 404 {object} models.APIResponse
// @Router /notifications/{notificationId} [get]
func (nc *NotificationController) GetNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		utils.BadRequestResponse(c, "Notification ID is required")
		return
	}

	notification, err := nc.notificationService.GetNotification(c.Request.Context(), userID, notificationID)
	if err != nil {
		logrus.Errorf("Get notification failed: %v", err)
		switch err.Error() {
		case "notification not found":
			utils.NotFoundResponse(c, "Notification")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this notification")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get notification")
		}
		return
	}

	utils.SuccessResponse(c, "Notification retrieved successfully", notification)
}

// MarkAsRead marks a notification as read
// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param notificationId path string true "Notification ID"
// @Success 200 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /notifications/{notificationId}/read [put]
func (nc *NotificationController) MarkAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		utils.BadRequestResponse(c, "Notification ID is required")
		return
	}

	err := nc.notificationService.MarkAsRead(c.Request.Context(), userID, notificationID)
	if err != nil {
		logrus.Errorf("Mark as read failed: %v", err)
		switch err.Error() {
		case "notification not found":
			utils.NotFoundResponse(c, "Notification")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this notification")
		default:
			utils.InternalServerErrorResponse(c, "Failed to mark notification as read")
		}
		return
	}

	utils.SuccessResponse(c, "Notification marked as read", nil)
}

// MarkAsUnread marks a notification as unread
// @Summary Mark notification as unread
// @Description Mark a specific notification as unread
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param notificationId path string true "Notification ID"
// @Success 200 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /notifications/{notificationId}/unread [put]
func (nc *NotificationController) MarkAsUnread(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		utils.BadRequestResponse(c, "Notification ID is required")
		return
	}

	err := nc.notificationService.MarkAsUnread(c.Request.Context(), userID, notificationID)
	if err != nil {
		logrus.Errorf("Mark as unread failed: %v", err)
		switch err.Error() {
		case "notification not found":
			utils.NotFoundResponse(c, "Notification")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this notification")
		default:
			utils.InternalServerErrorResponse(c, "Failed to mark notification as unread")
		}
		return
	}

	utils.SuccessResponse(c, "Notification marked as unread", nil)
}

// DeleteNotification deletes a notification
// @Summary Delete notification
// @Description Delete a specific notification
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param notificationId path string true "Notification ID"
// @Success 200 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /notifications/{notificationId} [delete]
func (nc *NotificationController) DeleteNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		utils.BadRequestResponse(c, "Notification ID is required")
		return
	}

	err := nc.notificationService.DeleteNotification(c.Request.Context(), userID, notificationID)
	if err != nil {
		logrus.Errorf("Delete notification failed: %v", err)
		switch err.Error() {
		case "notification not found":
			utils.NotFoundResponse(c, "Notification")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this notification")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete notification")
		}
		return
	}

	utils.SuccessResponse(c, "Notification deleted successfully", nil)
}

// ========================
// Bulk Operations
// ========================

// BulkMarkAsRead marks multiple notifications as read
func (nc *NotificationController) BulkMarkAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.BulkNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := nc.notificationService.BulkMarkAsRead(c.Request.Context(), userID, req.NotificationIDs)
	if err != nil {
		logrus.Errorf("Bulk mark as read failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to mark notifications as read")
		return
	}

	utils.SuccessResponse(c, "Notifications marked as read", result)
}

// BulkMarkAsUnread marks multiple notifications as unread
func (nc *NotificationController) BulkMarkAsUnread(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.BulkNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := nc.notificationService.BulkMarkAsUnread(c.Request.Context(), userID, req.NotificationIDs)
	if err != nil {
		logrus.Errorf("Bulk mark as unread failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to mark notifications as unread")
		return
	}

	utils.SuccessResponse(c, "Notifications marked as unread", result)
}

// BulkDeleteNotifications deletes multiple notifications
func (nc *NotificationController) BulkDeleteNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.BulkNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := nc.notificationService.BulkDeleteNotifications(c.Request.Context(), userID, req.NotificationIDs)
	if err != nil {
		logrus.Errorf("Bulk delete notifications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to delete notifications")
		return
	}

	utils.SuccessResponse(c, "Notifications deleted successfully", result)
}

// BulkArchiveNotifications archives multiple notifications
func (nc *NotificationController) BulkArchiveNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.BulkNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := nc.notificationService.BulkArchiveNotifications(c.Request.Context(), userID, req.NotificationIDs)
	if err != nil {
		logrus.Errorf("Bulk archive notifications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to archive notifications")
		return
	}

	utils.SuccessResponse(c, "Notifications archived successfully", result)
}

// ========================
// Filtering Operations
// ========================

// GetUnreadNotifications gets unread notifications
func (nc *NotificationController) GetUnreadNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	notifications, err := nc.notificationService.GetUnreadNotifications(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get unread notifications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get unread notifications")
		return
	}

	utils.SuccessResponse(c, "Unread notifications retrieved successfully", notifications)
}

// GetReadNotifications gets read notifications
func (nc *NotificationController) GetReadNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	notifications, err := nc.notificationService.GetReadNotifications(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get read notifications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get read notifications")
		return
	}

	utils.SuccessResponse(c, "Read notifications retrieved successfully", notifications)
}

// GetNotificationsByType gets notifications by type
func (nc *NotificationController) GetNotificationsByType(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationType := c.Param("type")
	if notificationType == "" {
		utils.BadRequestResponse(c, "Notification type is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	notifications, err := nc.notificationService.GetNotificationsByType(c.Request.Context(), userID, notificationType, page, pageSize)
	if err != nil {
		logrus.Errorf("Get notifications by type failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notifications by type")
		return
	}

	utils.SuccessResponse(c, "Notifications retrieved successfully", notifications)
}

// GetNotificationsByPriority gets notifications by priority
func (nc *NotificationController) GetNotificationsByPriority(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	priority := c.Param("priority")
	if priority == "" {
		utils.BadRequestResponse(c, "Priority is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	notifications, err := nc.notificationService.GetNotificationsByPriority(c.Request.Context(), userID, priority, page, pageSize)
	if err != nil {
		logrus.Errorf("Get notifications by priority failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notifications by priority")
		return
	}

	utils.SuccessResponse(c, "Notifications retrieved successfully", notifications)
}

// GetCircleNotifications gets notifications for a specific circle
func (nc *NotificationController) GetCircleNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Param("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	notifications, err := nc.notificationService.GetCircleNotifications(c.Request.Context(), userID, circleID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get circle notifications failed: %v", err)
		switch err.Error() {
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get circle notifications")
		}
		return
	}

	utils.SuccessResponse(c, "Circle notifications retrieved successfully", notifications)
}

// GetArchivedNotifications gets archived notifications
func (nc *NotificationController) GetArchivedNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	notifications, err := nc.notificationService.GetArchivedNotifications(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get archived notifications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get archived notifications")
		return
	}

	utils.SuccessResponse(c, "Archived notifications retrieved successfully", notifications)
}

// ========================
// Push Notification Management
// ========================

// GetPushSettings gets push notification settings
func (nc *NotificationController) GetPushSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := nc.notificationService.GetPushSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get push settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get push settings")
		return
	}

	utils.SuccessResponse(c, "Push settings retrieved successfully", settings)
}

// UpdatePushSettings updates push notification settings
func (nc *NotificationController) UpdatePushSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdatePushSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := nc.notificationService.UpdatePushSettings(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update push settings failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid settings data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update push settings")
		}
		return
	}

	utils.SuccessResponse(c, "Push settings updated successfully", settings)
}

// SendTestNotification sends a test push notification
func (nc *NotificationController) SendTestNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.TestNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := nc.notificationService.SendTestNotification(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Send test notification failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to send test notification")
		return
	}

	utils.SuccessResponse(c, "Test notification sent successfully", nil)
}

// RegisterPushDevice registers a device for push notifications
func (nc *NotificationController) RegisterPushDevice(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	device, err := nc.notificationService.RegisterPushDevice(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Register push device failed: %v", err)
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

// UpdatePushDevice updates a push device
func (nc *NotificationController) UpdatePushDevice(c *gin.Context) {
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
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	device, err := nc.notificationService.UpdatePushDevice(c.Request.Context(), userID, deviceID, req)
	if err != nil {
		logrus.Errorf("Update push device failed: %v", err)
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

// UnregisterPushDevice unregisters a push device
func (nc *NotificationController) UnregisterPushDevice(c *gin.Context) {
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

	err := nc.notificationService.UnregisterPushDevice(c.Request.Context(), userID, deviceID)
	if err != nil {
		logrus.Errorf("Unregister push device failed: %v", err)
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

// GetPushDevices gets user's registered push devices
func (nc *NotificationController) GetPushDevices(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	devices, err := nc.notificationService.GetPushDevices(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get push devices failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get push devices")
		return
	}

	utils.SuccessResponse(c, "Push devices retrieved successfully", devices)
}

// ========================
// Notification Preferences
// ========================

// GetNotificationPreferences gets user's notification preferences
func (nc *NotificationController) GetNotificationPreferences(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	preferences, err := nc.notificationService.GetNotificationPreferences(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get notification preferences failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification preferences")
		return
	}

	utils.SuccessResponse(c, "Notification preferences retrieved successfully", preferences)
}

// UpdateNotificationPreferences updates user's notification preferences
func (nc *NotificationController) UpdateNotificationPreferences(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateNotificationPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	preferences, err := nc.notificationService.UpdateNotificationPreferences(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update notification preferences failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid preferences data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update notification preferences")
		}
		return
	}

	utils.SuccessResponse(c, "Notification preferences updated successfully", preferences)
}

// GetNotificationTypes gets available notification types
func (nc *NotificationController) GetNotificationTypes(c *gin.Context) {
	types, err := nc.notificationService.GetNotificationTypes(c.Request.Context())
	if err != nil {
		logrus.Errorf("Get notification types failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification types")
		return
	}

	utils.SuccessResponse(c, "Notification types retrieved successfully", types)
}

// UpdateTypePreferences updates preferences for a specific notification type
func (nc *NotificationController) UpdateTypePreferences(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationType := c.Param("type")
	if notificationType == "" {
		utils.BadRequestResponse(c, "Notification type is required")
		return
	}

	var req models.UpdateTypePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	preferences, err := nc.notificationService.UpdateTypePreferences(c.Request.Context(), userID, notificationType, req)
	if err != nil {
		logrus.Errorf("Update type preferences failed: %v", err)
		switch err.Error() {
		case "invalid type":
			utils.BadRequestResponse(c, "Invalid notification type")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid preferences data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update type preferences")
		}
		return
	}

	utils.SuccessResponse(c, "Type preferences updated successfully", preferences)
}

// GetNotificationSchedule gets user's notification schedule
func (nc *NotificationController) GetNotificationSchedule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	schedule, err := nc.notificationService.GetNotificationSchedule(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get notification schedule failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification schedule")
		return
	}

	utils.SuccessResponse(c, "Notification schedule retrieved successfully", schedule)
}

// UpdateNotificationSchedule updates user's notification schedule
func (nc *NotificationController) UpdateNotificationSchedule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateNotificationScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	schedule, err := nc.notificationService.UpdateNotificationSchedule(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update notification schedule failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid schedule data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update notification schedule")
		}
		return
	}

	utils.SuccessResponse(c, "Notification schedule updated successfully", schedule)
}

// ========================
// Email Notifications
// ========================

// GetEmailSettings gets email notification settings
func (nc *NotificationController) GetEmailSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := nc.notificationService.GetEmailSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get email settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get email settings")
		return
	}

	utils.SuccessResponse(c, "Email settings retrieved successfully", settings)
}

// UpdateEmailSettings updates email notification settings
func (nc *NotificationController) UpdateEmailSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateEmailSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := nc.notificationService.UpdateEmailSettings(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update email settings failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid email settings data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update email settings")
		}
		return
	}

	utils.SuccessResponse(c, "Email settings updated successfully", settings)
}

// VerifyEmailAddress verifies an email address
func (nc *NotificationController) VerifyEmailAddress(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := nc.notificationService.VerifyEmailAddress(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Verify email address failed: %v", err)
		switch err.Error() {
		case "invalid verification code":
			utils.BadRequestResponse(c, "Invalid verification code")
		case "verification code expired":
			utils.BadRequestResponse(c, "Verification code expired")
		default:
			utils.InternalServerErrorResponse(c, "Failed to verify email address")
		}
		return
	}

	utils.SuccessResponse(c, "Email address verified successfully", result)
}

// GetEmailTemplates gets email templates
func (nc *NotificationController) GetEmailTemplates(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templates, err := nc.notificationService.GetEmailTemplates(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get email templates failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get email templates")
		return
	}

	utils.SuccessResponse(c, "Email templates retrieved successfully", templates)
}

// UpdateEmailTemplate updates an email template
func (nc *NotificationController) UpdateEmailTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templateID := c.Param("templateId")
	if templateID == "" {
		utils.BadRequestResponse(c, "Template ID is required")
		return
	}

	var req models.UpdateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	template, err := nc.notificationService.UpdateEmailTemplate(c.Request.Context(), userID, templateID, req)
	if err != nil {
		logrus.Errorf("Update email template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Email template")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this template")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid template data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update email template")
		}
		return
	}

	utils.SuccessResponse(c, "Email template updated successfully", template)
}

// SendTestEmail sends a test email
func (nc *NotificationController) SendTestEmail(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.TestEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := nc.notificationService.SendTestEmail(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Send test email failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to send test email")
		return
	}

	utils.SuccessResponse(c, "Test email sent successfully", nil)
}

// ========================
// SMS Notifications
// ========================

// GetSMSSettings gets SMS notification settings
func (nc *NotificationController) GetSMSSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := nc.notificationService.GetSMSSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get SMS settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get SMS settings")
		return
	}

	utils.SuccessResponse(c, "SMS settings retrieved successfully", settings)
}

// UpdateSMSSettings updates SMS notification settings
func (nc *NotificationController) UpdateSMSSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateSMSSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := nc.notificationService.UpdateSMSSettings(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update SMS settings failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid SMS settings data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update SMS settings")
		}
		return
	}

	utils.SuccessResponse(c, "SMS settings updated successfully", settings)
}

// VerifyPhoneNumber verifies a phone number
func (nc *NotificationController) VerifyPhoneNumber(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.VerifyPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := nc.notificationService.VerifyPhoneNumber(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Verify phone number failed: %v", err)
		switch err.Error() {
		case "invalid verification code":
			utils.BadRequestResponse(c, "Invalid verification code")
		case "verification code expired":
			utils.BadRequestResponse(c, "Verification code expired")
		default:
			utils.InternalServerErrorResponse(c, "Failed to verify phone number")
		}
		return
	}

	utils.SuccessResponse(c, "Phone number verified successfully", result)
}

// SendTestSMS sends a test SMS
func (nc *NotificationController) SendTestSMS(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.TestSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := nc.notificationService.SendTestSMS(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Send test SMS failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to send test SMS")
		return
	}

	utils.SuccessResponse(c, "Test SMS sent successfully", nil)
}

// GetSMSUsage gets SMS usage statistics
func (nc *NotificationController) GetSMSUsage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	usage, err := nc.notificationService.GetSMSUsage(c.Request.Context(), userID, days)
	if err != nil {
		logrus.Errorf("Get SMS usage failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get SMS usage")
		return
	}

	utils.SuccessResponse(c, "SMS usage retrieved successfully", usage)
}

// ========================
// In-App Notifications
// ========================

// GetInAppSettings gets in-app notification settings
func (nc *NotificationController) GetInAppSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := nc.notificationService.GetInAppSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get in-app settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get in-app settings")
		return
	}

	utils.SuccessResponse(c, "In-app settings retrieved successfully", settings)
}

// UpdateInAppSettings updates in-app notification settings
func (nc *NotificationController) UpdateInAppSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateInAppSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	settings, err := nc.notificationService.UpdateInAppSettings(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update in-app settings failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid in-app settings data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update in-app settings")
		}
		return
	}

	utils.SuccessResponse(c, "In-app settings updated successfully", settings)
}

// GetNotificationBadges gets notification badge counts
func (nc *NotificationController) GetNotificationBadges(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	badges, err := nc.notificationService.GetNotificationBadges(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get notification badges failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification badges")
		return
	}

	utils.SuccessResponse(c, "Notification badges retrieved successfully", badges)
}

// ClearNotificationBadges clears notification badges
func (nc *NotificationController) ClearNotificationBadges(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.ClearBadgesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body, clear all badges
		req.BadgeTypes = []string{}
	}

	err := nc.notificationService.ClearNotificationBadges(c.Request.Context(), userID, req.BadgeTypes)
	if err != nil {
		logrus.Errorf("Clear notification badges failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to clear notification badges")
		return
	}

	utils.SuccessResponse(c, "Notification badges cleared successfully", nil)
}

// GetNotificationSounds gets available notification sounds
func (nc *NotificationController) GetNotificationSounds(c *gin.Context) {
	sounds, err := nc.notificationService.GetNotificationSounds(c.Request.Context())
	if err != nil {
		logrus.Errorf("Get notification sounds failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification sounds")
		return
	}

	utils.SuccessResponse(c, "Notification sounds retrieved successfully", sounds)
}

// UpdateNotificationSounds updates notification sound preferences
func (nc *NotificationController) UpdateNotificationSounds(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateSoundPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	preferences, err := nc.notificationService.UpdateNotificationSounds(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update notification sounds failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid sound preferences data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update notification sounds")
		}
		return
	}

	utils.SuccessResponse(c, "Notification sound preferences updated successfully", preferences)
}

// ========================
// Notification Channels
// ========================

// GetNotificationChannels gets notification channels
func (nc *NotificationController) GetNotificationChannels(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	channels, err := nc.notificationService.GetNotificationChannels(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get notification channels failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification channels")
		return
	}

	utils.SuccessResponse(c, "Notification channels retrieved successfully", channels)
}

// CreateNotificationChannel creates a new notification channel
func (nc *NotificationController) CreateNotificationChannel(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	channel, err := nc.notificationService.CreateNotificationChannel(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create notification channel failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid channel data")
		case "channel already exists":
			utils.ConflictResponse(c, "Channel already exists")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create notification channel")
		}
		return
	}

	utils.CreatedResponse(c, "Notification channel created successfully", channel)
}

// UpdateNotificationChannel updates a notification channel
func (nc *NotificationController) UpdateNotificationChannel(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	channelID := c.Param("channelId")
	if channelID == "" {
		utils.BadRequestResponse(c, "Channel ID is required")
		return
	}

	var req models.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	channel, err := nc.notificationService.UpdateNotificationChannel(c.Request.Context(), userID, channelID, req)
	if err != nil {
		logrus.Errorf("Update notification channel failed: %v", err)
		switch err.Error() {
		case "channel not found":
			utils.NotFoundResponse(c, "Channel")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this channel")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid channel data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update notification channel")
		}
		return
	}

	utils.SuccessResponse(c, "Notification channel updated successfully", channel)
}

// DeleteNotificationChannel deletes a notification channel
func (nc *NotificationController) DeleteNotificationChannel(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	channelID := c.Param("channelId")
	if channelID == "" {
		utils.BadRequestResponse(c, "Channel ID is required")
		return
	}

	err := nc.notificationService.DeleteNotificationChannel(c.Request.Context(), userID, channelID)
	if err != nil {
		logrus.Errorf("Delete notification channel failed: %v", err)
		switch err.Error() {
		case "channel not found":
			utils.NotFoundResponse(c, "Channel")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this channel")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete notification channel")
		}
		return
	}

	utils.SuccessResponse(c, "Notification channel deleted successfully", nil)
}

// TestNotificationChannel tests a notification channel
func (nc *NotificationController) TestNotificationChannel(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	channelID := c.Param("channelId")
	if channelID == "" {
		utils.BadRequestResponse(c, "Channel ID is required")
		return
	}

	result, err := nc.notificationService.TestNotificationChannel(c.Request.Context(), userID, channelID)
	if err != nil {
		logrus.Errorf("Test notification channel failed: %v", err)
		switch err.Error() {
		case "channel not found":
			utils.NotFoundResponse(c, "Channel")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this channel")
		default:
			utils.InternalServerErrorResponse(c, "Failed to test notification channel")
		}
		return
	}

	utils.SuccessResponse(c, "Channel test completed", result)
}

// ========================
// Notification Rules
// ========================

// GetNotificationRules gets notification rules
func (nc *NotificationController) GetNotificationRules(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	rules, err := nc.notificationService.GetNotificationRules(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get notification rules failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification rules")
		return
	}

	utils.SuccessResponse(c, "Notification rules retrieved successfully", rules)
}

// CreateNotificationRule creates a new notification rule
func (nc *NotificationController) CreateNotificationRule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	rule, err := nc.notificationService.CreateNotificationRule(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create notification rule failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid rule data")
		case "rule limit reached":
			utils.BadRequestResponse(c, "Rule limit reached")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create notification rule")
		}
		return
	}

	utils.CreatedResponse(c, "Notification rule created successfully", rule)
}

// GetNotificationRule gets a specific notification rule
func (nc *NotificationController) GetNotificationRule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	ruleID := c.Param("ruleId")
	if ruleID == "" {
		utils.BadRequestResponse(c, "Rule ID is required")
		return
	}

	rule, err := nc.notificationService.GetNotificationRule(c.Request.Context(), userID, ruleID)
	if err != nil {
		logrus.Errorf("Get notification rule failed: %v", err)
		switch err.Error() {
		case "rule not found":
			utils.NotFoundResponse(c, "Rule")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this rule")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get notification rule")
		}
		return
	}

	utils.SuccessResponse(c, "Notification rule retrieved successfully", rule)
}

// UpdateNotificationRule updates a notification rule
func (nc *NotificationController) UpdateNotificationRule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	ruleID := c.Param("ruleId")
	if ruleID == "" {
		utils.BadRequestResponse(c, "Rule ID is required")
		return
	}

	var req models.UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	rule, err := nc.notificationService.UpdateNotificationRule(c.Request.Context(), userID, ruleID, req)
	if err != nil {
		logrus.Errorf("Update notification rule failed: %v", err)
		switch err.Error() {
		case "rule not found":
			utils.NotFoundResponse(c, "Rule")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this rule")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid rule data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update notification rule")
		}
		return
	}

	utils.SuccessResponse(c, "Notification rule updated successfully", rule)
}

// DeleteNotificationRule deletes a notification rule
func (nc *NotificationController) DeleteNotificationRule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	ruleID := c.Param("ruleId")
	if ruleID == "" {
		utils.BadRequestResponse(c, "Rule ID is required")
		return
	}

	err := nc.notificationService.DeleteNotificationRule(c.Request.Context(), userID, ruleID)
	if err != nil {
		logrus.Errorf("Delete notification rule failed: %v", err)
		switch err.Error() {
		case "rule not found":
			utils.NotFoundResponse(c, "Rule")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this rule")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete notification rule")
		}
		return
	}

	utils.SuccessResponse(c, "Notification rule deleted successfully", nil)
}

// TestNotificationRule tests a notification rule
func (nc *NotificationController) TestNotificationRule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	ruleID := c.Param("ruleId")
	if ruleID == "" {
		utils.BadRequestResponse(c, "Rule ID is required")
		return
	}

	result, err := nc.notificationService.TestNotificationRule(c.Request.Context(), userID, ruleID)
	if err != nil {
		logrus.Errorf("Test notification rule failed: %v", err)
		switch err.Error() {
		case "rule not found":
			utils.NotFoundResponse(c, "Rule")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this rule")
		default:
			utils.InternalServerErrorResponse(c, "Failed to test notification rule")
		}
		return
	}

	utils.SuccessResponse(c, "Rule test completed", result)
}

// ========================
// Do Not Disturb
// ========================

// GetDoNotDisturbStatus gets do not disturb status
func (nc *NotificationController) GetDoNotDisturbStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	status, err := nc.notificationService.GetDoNotDisturbStatus(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get DND status failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get do not disturb status")
		return
	}

	utils.SuccessResponse(c, "Do not disturb status retrieved successfully", status)
}

// EnableDoNotDisturb enables do not disturb mode
func (nc *NotificationController) EnableDoNotDisturb(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.EnableDNDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body, enable indefinitely
		req.Duration = 0
	}

	status, err := nc.notificationService.EnableDoNotDisturb(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Enable DND failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to enable do not disturb")
		return
	}

	utils.SuccessResponse(c, "Do not disturb enabled successfully", status)
}

// DisableDoNotDisturb disables do not disturb mode
func (nc *NotificationController) DisableDoNotDisturb(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	status, err := nc.notificationService.DisableDoNotDisturb(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Disable DND failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to disable do not disturb")
		return
	}

	utils.SuccessResponse(c, "Do not disturb disabled successfully", status)
}

// GetQuietHours gets quiet hours settings
func (nc *NotificationController) GetQuietHours(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	quietHours, err := nc.notificationService.GetQuietHours(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get quiet hours failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get quiet hours")
		return
	}

	utils.SuccessResponse(c, "Quiet hours retrieved successfully", quietHours)
}

// UpdateQuietHours updates quiet hours settings
func (nc *NotificationController) UpdateQuietHours(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateQuietHoursRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	quietHours, err := nc.notificationService.UpdateQuietHours(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update quiet hours failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid quiet hours data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update quiet hours")
		}
		return
	}

	utils.SuccessResponse(c, "Quiet hours updated successfully", quietHours)
}

// GetDNDExceptions gets do not disturb exceptions
func (nc *NotificationController) GetDNDExceptions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	exceptions, err := nc.notificationService.GetDNDExceptions(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get DND exceptions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get DND exceptions")
		return
	}

	utils.SuccessResponse(c, "DND exceptions retrieved successfully", exceptions)
}

// UpdateDNDExceptions updates do not disturb exceptions
func (nc *NotificationController) UpdateDNDExceptions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.UpdateDNDExceptionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	exceptions, err := nc.notificationService.UpdateDNDExceptions(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Update DND exceptions failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid exceptions data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update DND exceptions")
		}
		return
	}

	utils.SuccessResponse(c, "DND exceptions updated successfully", exceptions)
}

// ========================
// Notification Templates
// ========================

// GetNotificationTemplates gets notification templates
func (nc *NotificationController) GetNotificationTemplates(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templateType := c.Query("type")
	category := c.Query("category")

	templates, err := nc.notificationService.GetNotificationTemplates(c.Request.Context(), userID, templateType, category)
	if err != nil {
		logrus.Errorf("Get notification templates failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification templates")
		return
	}

	utils.SuccessResponse(c, "Notification templates retrieved successfully", templates)
}

// CreateNotificationTemplate creates a new notification template
func (nc *NotificationController) CreateNotificationTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	template, err := nc.notificationService.CreateNotificationTemplate(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create notification template failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid template data")
		case "template already exists":
			utils.ConflictResponse(c, "Template already exists")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create notification template")
		}
		return
	}

	utils.CreatedResponse(c, "Notification template created successfully", template)
}

// GetNotificationTemplate gets a specific notification template
func (nc *NotificationController) GetNotificationTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templateID := c.Param("templateId")
	if templateID == "" {
		utils.BadRequestResponse(c, "Template ID is required")
		return
	}

	template, err := nc.notificationService.GetNotificationTemplate(c.Request.Context(), userID, templateID)
	if err != nil {
		logrus.Errorf("Get notification template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this template")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get notification template")
		}
		return
	}

	utils.SuccessResponse(c, "Notification template retrieved successfully", template)
}

// UpdateNotificationTemplate updates a notification template
func (nc *NotificationController) UpdateNotificationTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templateID := c.Param("templateId")
	if templateID == "" {
		utils.BadRequestResponse(c, "Template ID is required")
		return
	}

	var req models.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	template, err := nc.notificationService.UpdateNotificationTemplate(c.Request.Context(), userID, templateID, req)
	if err != nil {
		logrus.Errorf("Update notification template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this template")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid template data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update notification template")
		}
		return
	}

	utils.SuccessResponse(c, "Notification template updated successfully", template)
}

// DeleteNotificationTemplate deletes a notification template
func (nc *NotificationController) DeleteNotificationTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templateID := c.Param("templateId")
	if templateID == "" {
		utils.BadRequestResponse(c, "Template ID is required")
		return
	}

	err := nc.notificationService.DeleteNotificationTemplate(c.Request.Context(), userID, templateID)
	if err != nil {
		logrus.Errorf("Delete notification template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this template")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete notification template")
		}
		return
	}

	utils.SuccessResponse(c, "Notification template deleted successfully", nil)
}

// PreviewNotificationTemplate previews a notification template
func (nc *NotificationController) PreviewNotificationTemplate(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	templateID := c.Param("templateId")
	if templateID == "" {
		utils.BadRequestResponse(c, "Template ID is required")
		return
	}

	var req models.PreviewTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	preview, err := nc.notificationService.PreviewNotificationTemplate(c.Request.Context(), userID, templateID, req)
	if err != nil {
		logrus.Errorf("Preview notification template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this template")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid preview data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to preview notification template")
		}
		return
	}

	utils.SuccessResponse(c, "Template preview generated successfully", preview)
}

// ========================
// Analytics
// ========================

// GetNotificationStats gets notification statistics
func (nc *NotificationController) GetNotificationStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	groupBy := c.DefaultQuery("groupBy", "day")

	stats, err := nc.notificationService.GetNotificationStats(c.Request.Context(), userID, days, groupBy)
	if err != nil {
		logrus.Errorf("Get notification stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification stats")
		return
	}

	utils.SuccessResponse(c, "Notification stats retrieved successfully", stats)
}

// GetDeliveryStats gets notification delivery statistics
func (nc *NotificationController) GetDeliveryStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	channel := c.Query("channel")

	stats, err := nc.notificationService.GetDeliveryStats(c.Request.Context(), userID, days, channel)
	if err != nil {
		logrus.Errorf("Get delivery stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get delivery stats")
		return
	}

	utils.SuccessResponse(c, "Delivery stats retrieved successfully", stats)
}

// GetEngagementStats gets notification engagement statistics
func (nc *NotificationController) GetEngagementStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	notificationType := c.Query("type")

	stats, err := nc.notificationService.GetEngagementStats(c.Request.Context(), userID, days, notificationType)
	if err != nil {
		logrus.Errorf("Get engagement stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get engagement stats")
		return
	}

	utils.SuccessResponse(c, "Engagement stats retrieved successfully", stats)
}

// GetNotificationTrends gets notification trends
func (nc *NotificationController) GetNotificationTrends(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	metric := c.DefaultQuery("metric", "count")

	trends, err := nc.notificationService.GetNotificationTrends(c.Request.Context(), userID, days, metric)
	if err != nil {
		logrus.Errorf("Get notification trends failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification trends")
		return
	}

	utils.SuccessResponse(c, "Notification trends retrieved successfully", trends)
}

// GetNotificationPerformance gets notification performance metrics
func (nc *NotificationController) GetNotificationPerformance(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	compareWith, _ := strconv.Atoi(c.DefaultQuery("compareWith", "0"))

	performance, err := nc.notificationService.GetNotificationPerformance(c.Request.Context(), userID, days, compareWith)
	if err != nil {
		logrus.Errorf("Get notification performance failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification performance")
		return
	}

	utils.SuccessResponse(c, "Notification performance retrieved successfully", performance)
}

// ========================
// History and Audit
// ========================

// GetNotificationHistory gets notification history
func (nc *NotificationController) GetNotificationHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	notificationType := c.Query("type")

	req := models.GetHistoryRequest{
		UserID:    userID,
		Page:      page,
		PageSize:  pageSize,
		StartDate: startDate,
		EndDate:   endDate,
		Type:      notificationType,
	}

	history, err := nc.notificationService.GetNotificationHistory(c.Request.Context(), req)
	if err != nil {
		logrus.Errorf("Get notification history failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification history")
		return
	}

	utils.SuccessResponse(c, "Notification history retrieved successfully", history)
}

// GetDeliveryHistory gets delivery history for a specific notification
func (nc *NotificationController) GetDeliveryHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		utils.BadRequestResponse(c, "Notification ID is required")
		return
	}

	history, err := nc.notificationService.GetDeliveryHistory(c.Request.Context(), userID, notificationID)
	if err != nil {
		logrus.Errorf("Get delivery history failed: %v", err)
		switch err.Error() {
		case "notification not found":
			utils.NotFoundResponse(c, "Notification")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this notification")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get delivery history")
		}
		return
	}

	utils.SuccessResponse(c, "Delivery history retrieved successfully", history)
}

// ExportNotificationHistory exports notification history
func (nc *NotificationController) ExportNotificationHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.ExportHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}
	req.UserID = userID

	exportResult, err := nc.notificationService.ExportNotificationHistory(c.Request.Context(), req)
	if err != nil {
		logrus.Errorf("Export notification history failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid export parameters")
		default:
			utils.InternalServerErrorResponse(c, "Failed to export notification history")
		}
		return
	}

	utils.SuccessResponse(c, "Export started successfully", exportResult)
}

// DownloadNotificationExport downloads exported notification history
func (nc *NotificationController) DownloadNotificationExport(c *gin.Context) {
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

	fileData, filename, err := nc.notificationService.DownloadNotificationExport(c.Request.Context(), userID, exportID)
	if err != nil {
		logrus.Errorf("Download notification export failed: %v", err)
		switch err.Error() {
		case "export not found":
			utils.NotFoundResponse(c, "Export")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this export")
		case "export not ready":
			utils.BadRequestResponse(c, "Export is not ready for download")
		default:
			utils.InternalServerErrorResponse(c, "Failed to download export")
		}
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")
	c.Data(200, "application/octet-stream", fileData)
}

// CleanupOldNotifications cleans up old notifications
func (nc *NotificationController) CleanupOldNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Check if user has admin privileges for system-wide cleanup
	role := c.GetString("role")

	days, _ := strconv.Atoi(c.DefaultQuery("days", "90"))
	dryRun := c.DefaultQuery("dryRun", "false") == "true"

	result, err := nc.notificationService.CleanupOldNotifications(c.Request.Context(), userID, role, days, dryRun)
	if err != nil {
		logrus.Errorf("Cleanup old notifications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to cleanup old notifications")
		return
	}

	utils.SuccessResponse(c, "Cleanup completed successfully", result)
}

// ========================
// Subscriptions
// ========================

// GetNotificationSubscriptions gets notification subscriptions
func (nc *NotificationController) GetNotificationSubscriptions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	subscriptions, err := nc.notificationService.GetNotificationSubscriptions(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get notification subscriptions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification subscriptions")
		return
	}

	utils.SuccessResponse(c, "Notification subscriptions retrieved successfully", subscriptions)
}

// CreateNotificationSubscription creates a new notification subscription
func (nc *NotificationController) CreateNotificationSubscription(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	subscription, err := nc.notificationService.CreateNotificationSubscription(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create notification subscription failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid subscription data")
		case "subscription already exists":
			utils.ConflictResponse(c, "Subscription already exists")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create notification subscription")
		}
		return
	}

	utils.CreatedResponse(c, "Notification subscription created successfully", subscription)
}

// UpdateNotificationSubscription updates a notification subscription
func (nc *NotificationController) UpdateNotificationSubscription(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	subscriptionID := c.Param("subscriptionId")
	if subscriptionID == "" {
		utils.BadRequestResponse(c, "Subscription ID is required")
		return
	}

	var req models.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	subscription, err := nc.notificationService.UpdateNotificationSubscription(c.Request.Context(), userID, subscriptionID, req)
	if err != nil {
		logrus.Errorf("Update notification subscription failed: %v", err)
		switch err.Error() {
		case "subscription not found":
			utils.NotFoundResponse(c, "Subscription")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this subscription")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid subscription data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update notification subscription")
		}
		return
	}

	utils.SuccessResponse(c, "Notification subscription updated successfully", subscription)
}

// DeleteNotificationSubscription deletes a notification subscription
func (nc *NotificationController) DeleteNotificationSubscription(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	subscriptionID := c.Param("subscriptionId")
	if subscriptionID == "" {
		utils.BadRequestResponse(c, "Subscription ID is required")
		return
	}

	err := nc.notificationService.DeleteNotificationSubscription(c.Request.Context(), userID, subscriptionID)
	if err != nil {
		logrus.Errorf("Delete notification subscription failed: %v", err)
		switch err.Error() {
		case "subscription not found":
			utils.NotFoundResponse(c, "Subscription")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this subscription")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete notification subscription")
		}
		return
	}

	utils.SuccessResponse(c, "Notification subscription deleted successfully", nil)
}

// GetNotificationTopics gets available notification topics
func (nc *NotificationController) GetNotificationTopics(c *gin.Context) {
	topics, err := nc.notificationService.GetNotificationTopics(c.Request.Context())
	if err != nil {
		logrus.Errorf("Get notification topics failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification topics")
		return
	}

	utils.SuccessResponse(c, "Notification topics retrieved successfully", topics)
}

// ========================
// Actions
// ========================

// GetNotificationActions gets available notification actions
func (nc *NotificationController) GetNotificationActions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationType := c.Query("type")

	actions, err := nc.notificationService.GetNotificationActions(c.Request.Context(), userID, notificationType)
	if err != nil {
		logrus.Errorf("Get notification actions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification actions")
		return
	}

	utils.SuccessResponse(c, "Notification actions retrieved successfully", actions)
}

// ExecuteNotificationAction executes a notification action
func (nc *NotificationController) ExecuteNotificationAction(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("notificationId")
	actionID := c.Param("actionId")

	if notificationID == "" {
		utils.BadRequestResponse(c, "Notification ID is required")
		return
	}

	if actionID == "" {
		utils.BadRequestResponse(c, "Action ID is required")
		return
	}

	var req models.ExecuteActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Some actions might not need a body
		req = models.ExecuteActionRequest{}
	}

	result, err := nc.notificationService.ExecuteNotificationAction(c.Request.Context(), userID, notificationID, actionID, req)
	if err != nil {
		logrus.Errorf("Execute notification action failed: %v", err)
		switch err.Error() {
		case "notification not found":
			utils.NotFoundResponse(c, "Notification")
		case "action not found":
			utils.NotFoundResponse(c, "Action")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this notification")
		case "action not available":
			utils.BadRequestResponse(c, "Action is not available for this notification")
		default:
			utils.InternalServerErrorResponse(c, "Failed to execute notification action")
		}
		return
	}

	utils.SuccessResponse(c, "Action executed successfully", result)
}

// SnoozeNotification snoozes a notification
func (nc *NotificationController) SnoozeNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		utils.BadRequestResponse(c, "Notification ID is required")
		return
	}

	var req models.SnoozeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := nc.notificationService.SnoozeNotification(c.Request.Context(), userID, notificationID, req)
	if err != nil {
		logrus.Errorf("Snooze notification failed: %v", err)
		switch err.Error() {
		case "notification not found":
			utils.NotFoundResponse(c, "Notification")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this notification")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid snooze parameters")
		default:
			utils.InternalServerErrorResponse(c, "Failed to snooze notification")
		}
		return
	}

	utils.SuccessResponse(c, "Notification snoozed successfully", result)
}

// PinNotification pins a notification
func (nc *NotificationController) PinNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		utils.BadRequestResponse(c, "Notification ID is required")
		return
	}

	err := nc.notificationService.PinNotification(c.Request.Context(), userID, notificationID)
	if err != nil {
		logrus.Errorf("Pin notification failed: %v", err)
		switch err.Error() {
		case "notification not found":
			utils.NotFoundResponse(c, "Notification")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this notification")
		case "already pinned":
			utils.BadRequestResponse(c, "Notification is already pinned")
		default:
			utils.InternalServerErrorResponse(c, "Failed to pin notification")
		}
		return
	}

	utils.SuccessResponse(c, "Notification pinned successfully", nil)
}

// UnpinNotification unpins a notification
func (nc *NotificationController) UnpinNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("notificationId")
	if notificationID == "" {
		utils.BadRequestResponse(c, "Notification ID is required")
		return
	}

	err := nc.notificationService.UnpinNotification(c.Request.Context(), userID, notificationID)
	if err != nil {
		logrus.Errorf("Unpin notification failed: %v", err)
		switch err.Error() {
		case "notification not found":
			utils.NotFoundResponse(c, "Notification")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this notification")
		case "not pinned":
			utils.BadRequestResponse(c, "Notification is not pinned")
		default:
			utils.InternalServerErrorResponse(c, "Failed to unpin notification")
		}
		return
	}

	utils.SuccessResponse(c, "Notification unpinned successfully", nil)
}
