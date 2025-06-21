package controllers

import (
	"ftrack/models"
	"ftrack/services"
	"ftrack/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type MessageController struct {
	messageService *services.MessageService
}

func NewMessageController(messageService *services.MessageService) *MessageController {
	return &MessageController{
		messageService: messageService,
	}
}

// SendMessage sends a message to a circle
// @Summary Send message
// @Description Send a message to a circle
// @Tags Messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.SendMessageRequest true "Message data"
// @Success 201 {object} models.APIResponse{data=models.Message}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /messages [post]
func (mc *MessageController) SendMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	message, err := mc.messageService.SendMessage(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Send message failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid message data")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to send messages to this circle")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "message too long":
			utils.BadRequestResponse(c, "Message content is too long")
		default:
			utils.InternalServerErrorResponse(c, "Failed to send message")
		}
		return
	}

	utils.CreatedResponse(c, "Message sent successfully", message)
}

// GetMessages gets messages from a circle
// @Summary Get circle messages
// @Description Get messages from a specific circle with pagination
// @Tags Messages
// @Security BearerAuth
// @Produce json
// @Param circleId path string true "Circle ID"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(50)
// @Param before query string false "Get messages before this message ID"
// @Param after query string false "Get messages after this message ID"
// @Success 200 {object} models.APIResponse{data=[]models.Message}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /messages/circle/{circleId} [get]
func (mc *MessageController) GetMessages(c *gin.Context) {
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

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 50
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	req := models.GetMessagesRequest{
		CircleID: circleID,
		Page:     page,
		PageSize: pageSize,
		Before:   c.Query("before"),
		After:    c.Query("after"),
	}

	messages, total, err := mc.messageService.GetMessages(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get messages failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get messages")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Messages retrieved successfully", messages, meta)
}

// GetMessage gets a specific message by ID
// @Summary Get message by ID
// @Description Get a specific message by its ID
// @Tags Messages
// @Security BearerAuth
// @Produce json
// @Param id path string true "Message ID"
// @Success 200 {object} models.APIResponse{data=models.Message}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /messages/{id} [get]
func (mc *MessageController) GetMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("id")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	message, err := mc.messageService.GetMessage(c.Request.Context(), userID, messageID)
	if err != nil {
		logrus.Errorf("Get message failed: %v", err)

		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get message")
		}
		return
	}

	utils.SuccessResponse(c, "Message retrieved successfully", message)
}

// EditMessage edits an existing message
// @Summary Edit message
// @Description Edit an existing message (only by sender)
// @Tags Messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Message ID"
// @Param request body models.EditMessageRequest true "Updated message data"
// @Success 200 {object} models.APIResponse{data=models.Message}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /messages/{id} [put]
func (mc *MessageController) EditMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("id")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	var req models.EditMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	message, err := mc.messageService.EditMessage(c.Request.Context(), userID, messageID, req)
	if err != nil {
		logrus.Errorf("Edit message failed: %v", err)

		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only edit your own messages")
		case "message too old":
			utils.BadRequestResponse(c, "Message is too old to edit")
		case "only text messages can be edited":
			utils.BadRequestResponse(c, "Only text messages can be edited")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid message content")
		default:
			utils.InternalServerErrorResponse(c, "Failed to edit message")
		}
		return
	}

	utils.SuccessResponse(c, "Message edited successfully", message)
}

// DeleteMessage deletes a message
// @Summary Delete message
// @Description Delete a message (by sender or circle admin)
// @Tags Messages
// @Security BearerAuth
// @Produce json
// @Param id path string true "Message ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /messages/{id} [delete]
func (mc *MessageController) DeleteMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("id")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	err := mc.messageService.DeleteMessage(c.Request.Context(), userID, messageID)
	if err != nil {
		logrus.Errorf("Delete message failed: %v", err)

		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "permission denied":
			utils.ForbiddenResponse(c, "You don't have permission to delete this message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete message")
		}
		return
	}

	utils.SuccessResponse(c, "Message deleted successfully", nil)
}

// MarkAsRead marks messages as read
// @Summary Mark messages as read
// @Description Mark one or more messages as read
// @Tags Messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.MarkAsReadRequest true "Message IDs to mark as read"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /messages/read [post]
func (mc *MessageController) MarkAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.MarkAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := mc.messageService.MarkAsRead(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Mark as read failed: %v", err)

		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid message IDs")
		default:
			utils.InternalServerErrorResponse(c, "Failed to mark messages as read")
		}
		return
	}

	utils.SuccessResponse(c, "Messages marked as read successfully", nil)
}

// AddReaction adds a reaction to a message
// @Summary Add reaction
// @Description Add a reaction (emoji) to a message
// @Tags Messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Message ID"
// @Param request body models.AddReactionRequest true "Reaction data"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /messages/{id}/reactions [post]
func (mc *MessageController) AddReaction(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("id")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	var req models.AddReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := mc.messageService.AddReaction(c.Request.Context(), userID, messageID, req)
	if err != nil {
		logrus.Errorf("Add reaction failed: %v", err)

		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid reaction")
		default:
			utils.InternalServerErrorResponse(c, "Failed to add reaction")
		}
		return
	}

	utils.SuccessResponse(c, "Reaction added successfully", nil)
}

// RemoveReaction removes a reaction from a message
// @Summary Remove reaction
// @Description Remove a reaction from a message
// @Tags Messages
// @Security BearerAuth
// @Produce json
// @Param id path string true "Message ID"
// @Param emoji path string true "Emoji to remove"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Router /messages/{id}/reactions/{emoji} [delete]
func (mc *MessageController) RemoveReaction(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("id")
	emoji := c.Param("emoji")
	if messageID == "" || emoji == "" {
		utils.BadRequestResponse(c, "Message ID and emoji are required")
		return
	}

	err := mc.messageService.RemoveReaction(c.Request.Context(), userID, messageID, emoji)
	if err != nil {
		logrus.Errorf("Remove reaction failed: %v", err)

		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "reaction not found":
			utils.NotFoundResponse(c, "Reaction")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only remove your own reactions")
		default:
			utils.InternalServerErrorResponse(c, "Failed to remove reaction")
		}
		return
	}

	utils.SuccessResponse(c, "Reaction removed successfully", nil)
}

// UploadMedia uploads media for messages
// @Summary Upload media
// @Description Upload media file for use in messages
// @Tags Messages
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Media file"
// @Param type formData string false "Media type (image, video, audio, document)"
// @Success 201 {object} models.APIResponse{data=models.MessageMedia}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /messages/media [post]
func (mc *MessageController) UploadMedia(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.BadRequestResponse(c, "File is required")
		return
	}
	defer file.Close()

	mediaType := c.PostForm("type")
	if mediaType == "" {
		mediaType = "image" // Default to image
	}

	req := models.UploadMediaRequest{
		File:      file,
		Header:    header,
		MediaType: mediaType,
	}

	media, err := mc.messageService.UploadMedia(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Upload media failed: %v", err)

		switch err.Error() {
		case "invalid file type":
			utils.BadRequestResponse(c, "Invalid file type")
		case "file too large":
			utils.BadRequestResponse(c, "File size exceeds limit")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid media data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to upload media")
		}
		return
	}

	utils.CreatedResponse(c, "Media uploaded successfully", media)
}

// SearchMessages searches messages in circles
// @Summary Search messages
// @Description Search messages across user's circles
// @Tags Messages
// @Security BearerAuth
// @Produce json
// @Param q query string true "Search query"
// @Param circleId query string false "Circle ID to search in"
// @Param type query string false "Message type filter"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} models.APIResponse{data=[]models.Message}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Router /messages/search [get]
func (mc *MessageController) SearchMessages(c *gin.Context) {
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

	if len(query) < 3 {
		utils.BadRequestResponse(c, "Search query must be at least 3 characters")
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

	req := models.SearchMessagesRequest{
		Query:    query,
		CircleID: c.Query("circleId"),
		Type:     c.Query("type"),
		Page:     page,
		PageSize: pageSize,
	}

	messages, total, err := mc.messageService.SearchMessages(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Search messages failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to search messages")
		}
		return
	}

	meta := utils.CreatePaginationMeta(page, pageSize, total)
	utils.SuccessResponseWithMeta(c, "Messages searched successfully", messages, meta)
}

// GetUnreadCount gets unread message count
// @Summary Get unread count
// @Description Get unread message count for user's circles
// @Tags Messages
// @Security BearerAuth
// @Produce json
// @Param circleId query string false "Circle ID to get count for"
// @Success 200 {object} models.APIResponse{data=models.UnreadCount}
// @Failure 401 {object} models.APIResponse
// @Router /messages/unread-count [get]
func (mc *MessageController) GetUnreadCount(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Query("circleId")

	count, err := mc.messageService.GetUnreadCount(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get unread count failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get unread count")
		}
		return
	}

	utils.SuccessResponse(c, "Unread count retrieved successfully", count)
}

// GetMessageStats gets message statistics
// @Summary Get message statistics
// @Description Get message statistics for user or circle
// @Tags Messages
// @Security BearerAuth
// @Produce json
// @Param circleId query string false "Circle ID for circle stats"
// @Param period query string false "Time period (day, week, month, year)" default(week)
// @Success 200 {object} models.APIResponse{data=models.MessageStats}
// @Failure 401 {object} models.APIResponse
// @Failure 403 {object} models.APIResponse
// @Router /messages/stats [get]
func (mc *MessageController) GetMessageStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	circleID := c.Query("circleId")
	period := c.Query("period")
	if period == "" {
		period = "week"
	}

	stats, err := mc.messageService.GetMessageStats(c.Request.Context(), userID, circleID, period)
	if err != nil {
		logrus.Errorf("Get message stats failed: %v", err)

		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get message statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Message statistics retrieved successfully", stats)
}
