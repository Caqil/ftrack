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

// SendNotification sends a notification (admin only)
// @Summary Send notification
// @Description Send a notification to specific users or all users (admin only)
// @Tags Notifications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.SendNotificationRequest true "Notification data"
// @Success 201 {object} models.APIResponse{data=models.Notification}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /notifications/send [post]
func (nc *NotificationController) SendNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Check if user has admin privileges
	role := c.GetString("role")
	if role != "admin" {
		utils.ForbiddenResponse(c, "Admin access required")
		return
	}

	var req models.SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := nc.notificationService.SendNotification(c.Request.Context(), req)
	if err != nil {
		logrus.Errorf("Send notification failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid notification data")
		case "no recipients":
			utils.BadRequestResponse(c, "No valid recipients found")
		default:
			utils.InternalServerErrorResponse(c, "Failed to send notification")
		}
		return
	}

	utils.CreatedResponse(c, "Notification sent successfully", nil)
}

// GetNotifications gets user's notifications
// @Summary Get notifications
// @Description Get user's notifications with pagination and filtering
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Param type query string false "Notification type filter"
// @Param status query string false "Status filter (read, unread, all)" default(all)
// @Param priority query string false "Priority filter"
// @Success 200 {object} models.APIResponse{data=[]models.Notification}
// @Failure 401 {object} models.APIResponse
// @Router /notifications [get]
func (nc *NotificationController) GetNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	req := models.GetNotificationsRequest{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		Type:     c.Query("type"),
		Status:   c.Query("status"),
		Priority: c.Query("priority"),
	}

	notifications, total, err := nc.notificationService.GetNotifications(c.Request.Context(), req)
	if err != nil {
		logrus.Errorf("Get notifications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notifications")
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Notifications retrieved successfully", notifications, meta)
}

// GetNotification gets a specific notification
// @Summary Get notification by ID
// @Description Get a specific notification by ID
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} models.APIResponse{data=models.Notification}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /notifications/{id} [get]
func (nc *NotificationController) GetNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("id")
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

// MarkAsRead marks notifications as read
// @Summary Mark notifications as read
// @Description Mark one or more notifications as read
// @Tags Notifications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.MarkNotificationAsReadRequest true "Notification IDs to mark as read"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /notifications/read [post]
func (nc *NotificationController) MarkAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.MarkNotificationAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := nc.notificationService.MarkAsRead(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Mark notifications as read failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid notification IDs")
		default:
			utils.InternalServerErrorResponse(c, "Failed to mark notifications as read")
		}
		return
	}

	utils.SuccessResponse(c, "Notifications marked as read successfully", nil)
}

// MarkAsUnread marks notifications as unread
// @Summary Mark notifications as unread
// @Description Mark one or more notifications as unread
// @Tags Notifications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.MarkNotificationAsReadRequest true "Notification IDs to mark as unread"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /notifications/unread [post]
func (nc *NotificationController) MarkAsUnread(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.MarkNotificationAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := nc.notificationService.MarkAsUnread(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Mark notifications as unread failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid notification IDs")
		default:
			utils.InternalServerErrorResponse(c, "Failed to mark notifications as unread")
		}
		return
	}

	utils.SuccessResponse(c, "Notifications marked as unread successfully", nil)
}

// MarkAllAsRead marks all notifications as read
// @Summary Mark all notifications as read
// @Description Mark all user's notifications as read
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param type query string false "Notification type to mark as read"
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /notifications/read-all [post]
func (nc *NotificationController) MarkAllAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}


	err := nc.notificationService.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Mark all notifications as read failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to mark all notifications as read")
		return
	}

	utils.SuccessResponse(c, "All notifications marked as read successfully", nil)
}

// DeleteNotification deletes a notification
// @Summary Delete notification
// @Description Delete a notification
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /notifications/{id} [delete]
func (nc *NotificationController) DeleteNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationID := c.Param("id")
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
			utils.ForbiddenResponse(c, "You can only delete your own notifications")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete notification")
		}
		return
	}

	utils.SuccessResponse(c, "Notification deleted successfully", nil)
}

// DeleteAllNotifications deletes all notifications
// @Summary Delete all notifications
// @Description Delete all user's notifications
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param type query string false "Notification type to delete"
// @Param read query bool false "Delete only read notifications"
// @Success 200 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /notifications [delete]
func (nc *NotificationController) DeleteAllNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationType := c.Query("type")
	readOnly := c.Query("read") == "true"

	err := nc.notificationService.DeleteAllNotifications(c.Request.Context(), userID, notificationType, readOnly)
	if err != nil {
		logrus.Errorf("Delete all notifications failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to delete notifications")
		return
	}

	utils.SuccessResponse(c, "Notifications deleted successfully", nil)
}

// GetUnreadCount gets unread notification count
// @Summary Get unread count
// @Description Get unread notification count by type
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param type query string false "Notification type"
// @Success 200 {object} models.APIResponse{data=models.NotificationCount}
// @Failure 401 {object} models.APIResponse
// @Router /notifications/unread-count [get]
func (nc *NotificationController) GetUnreadCount(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	notificationType := c.Query("type")

	count, err := nc.notificationService.GetUnreadCount(c.Request.Context(), userID, notificationType)
	if err != nil {
		logrus.Errorf("Get unread count failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get unread count")
		return
	}

	utils.SuccessResponse(c, "Unread count retrieved successfully", count)
}

// UpdateNotificationSettings updates user's notification preferences
// @Summary Update notification settings
// @Description Update user's notification preferences
// @Tags Notifications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.NotificationPrefs true "Notification preferences"
// @Success 200 {object} models.APIResponse{data=models.NotificationPrefs}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /notifications/settings [put]
func (nc *NotificationController) UpdateNotificationSettings(c *gin.Context) {
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

	updatedSettings, err := nc.notificationService.UpdateNotificationSettings(c.Request.Context(), userID, settings)
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

// GetNotificationSettings gets user's notification preferences
// @Summary Get notification settings
// @Description Get user's current notification preferences
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.NotificationPrefs}
// @Failure 401 {object} models.APIResponse
// @Router /notifications/settings [get]
func (nc *NotificationController) GetNotificationSettings(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	settings, err := nc.notificationService.GetNotificationSettings(c.Request.Context(), userID)
	if err != nil {
		logrus.Errorf("Get notification settings failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification settings")
		return
	}

	utils.SuccessResponse(c, "Notification settings retrieved successfully", settings)
}

// UpdateDeviceToken updates user's device token for push notifications
// @Summary Update device token
// @Description Update user's device token for push notifications
// @Tags Notifications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{token=string,platform=string} true "Device token data"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /notifications/device-token [put]
func (nc *NotificationController) UpdateDeviceToken(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Token    string `json:"token" binding:"required"`
		Platform string `json:"platform" binding:"required,oneof=ios android web"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Valid device token and platform are required")
		return
	}

	err := nc.notificationService.UpdateDeviceToken(c.Request.Context(), userID, req.Token, req.Platform)
	if err != nil {
		logrus.Errorf("Update device token failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid device token or platform")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update device token")
		}
		return
	}

	utils.SuccessResponse(c, "Device token updated successfully", nil)
}

// TestNotification sends a test notification
// @Summary Send test notification
// @Description Send a test notification to the user
// @Tags Notifications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{type=string,title=string,body=string} true "Test notification data"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /notifications/test [post]
func (nc *NotificationController) TestNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req struct {
		Type  string `json:"type"`
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Type = "test"
		req.Title = "Test Notification"
		req.Body = "This is a test notification from the FTrack app."
	}

	testReq := models.SendNotificationRequest{
		UserIDs:  []string{userID},
		Type:     req.Type,
		Title:    req.Title,
		Body:     req.Body,
		Priority: "normal",
		Channels: models.NotificationChannels{
			Push:  true,
			InApp: true,
		},
	}

	err := nc.notificationService.SendNotification(c.Request.Context(), testReq)
	if err != nil {
		logrus.Errorf("Send test notification failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to send test notification")
		return
	}

	utils.SuccessResponse(c, "Test notification sent successfully", nil)
}

// GetNotificationStats gets notification statistics
// @Summary Get notification statistics
// @Description Get notification statistics for user
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param period query string false "Time period (day, week, month, year)" default(week)
// @Success 200 {object} models.APIResponse{data=models.NotificationStats}
// @Failure 401 {object} models.APIResponse
// @Router /notifications/stats [get]
func (nc *NotificationController) GetNotificationStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.Query("period")
	if period == "" {
		period = "week"
	}

	stats, err := nc.notificationService.GetNotificationStats(c.Request.Context(), userID, period)
	if err != nil {
		logrus.Errorf("Get notification stats failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get notification statistics")
		return
	}

	utils.SuccessResponse(c, "Notification statistics retrieved successfully", stats)
}
