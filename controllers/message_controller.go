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

// Basic message operations

// GetMessages gets messages from a circle with pagination
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	before := c.Query("before")
	after := c.Query("after")

	req := models.GetMessagesRequest{
		CircleID: circleID,
		Page:     page,
		PageSize: pageSize,
		Before:   before,
		After:    after,
	}

	messages, err := mc.messageService.GetCircleMessages(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get messages failed: %v", err)
		switch err.Error() {
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get messages")
		}
		return
	}

	utils.SuccessResponse(c, "Messages retrieved successfully", messages)
}

// SendMessage sends a message to a circle
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

// GetMessage gets a specific message by ID
func (mc *MessageController) GetMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
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

// UpdateMessage updates a message
func (mc *MessageController) UpdateMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	var req models.EditMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	message, err := mc.messageService.UpdateMessage(c.Request.Context(), userID, messageID, req)
	if err != nil {
		logrus.Errorf("Update message failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only edit your own messages")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid message content")
		case "edit time expired":
			utils.BadRequestResponse(c, "Message can no longer be edited")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update message")
		}
		return
	}

	utils.SuccessResponse(c, "Message updated successfully", message)
}

// DeleteMessage deletes a message
func (mc *MessageController) DeleteMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
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
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own messages")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete message")
		}
		return
	}

	utils.SuccessResponse(c, "Message deleted successfully", nil)
}

// Message threading and replies

// GetReplies gets replies to a message
func (mc *MessageController) GetReplies(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	replies, err := mc.messageService.GetReplies(c.Request.Context(), userID, messageID, page, pageSize)
	if err != nil {
		logrus.Errorf("Get replies failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get replies")
		}
		return
	}

	utils.SuccessResponse(c, "Replies retrieved successfully", replies)
}

// ReplyToMessage sends a reply to a message
func (mc *MessageController) ReplyToMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	var req models.ReplyMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	reply, err := mc.messageService.ReplyToMessage(c.Request.Context(), userID, messageID, req)
	if err != nil {
		logrus.Errorf("Reply to message failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to reply to this message")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid reply data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to send reply")
		}
		return
	}

	utils.CreatedResponse(c, "Reply sent successfully", reply)
}

// GetReply gets a specific reply
func (mc *MessageController) GetReply(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	replyID := c.Param("replyId")
	if replyID == "" {
		utils.BadRequestResponse(c, "Reply ID is required")
		return
	}

	reply, err := mc.messageService.GetReply(c.Request.Context(), userID, replyID)
	if err != nil {
		logrus.Errorf("Get reply failed: %v", err)
		switch err.Error() {
		case "reply not found":
			utils.NotFoundResponse(c, "Reply")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this reply")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get reply")
		}
		return
	}

	utils.SuccessResponse(c, "Reply retrieved successfully", reply)
}

// UpdateReply updates a reply
func (mc *MessageController) UpdateReply(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	replyID := c.Param("replyId")
	if replyID == "" {
		utils.BadRequestResponse(c, "Reply ID is required")
		return
	}

	var req models.EditMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	reply, err := mc.messageService.UpdateReply(c.Request.Context(), userID, replyID, req)
	if err != nil {
		logrus.Errorf("Update reply failed: %v", err)
		switch err.Error() {
		case "reply not found":
			utils.NotFoundResponse(c, "Reply")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only edit your own replies")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid reply content")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update reply")
		}
		return
	}

	utils.SuccessResponse(c, "Reply updated successfully", reply)
}

// DeleteReply deletes a reply
func (mc *MessageController) DeleteReply(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	replyID := c.Param("replyId")
	if replyID == "" {
		utils.BadRequestResponse(c, "Reply ID is required")
		return
	}

	err := mc.messageService.DeleteReply(c.Request.Context(), userID, replyID)
	if err != nil {
		logrus.Errorf("Delete reply failed: %v", err)
		switch err.Error() {
		case "reply not found":
			utils.NotFoundResponse(c, "Reply")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own replies")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete reply")
		}
		return
	}

	utils.SuccessResponse(c, "Reply deleted successfully", nil)
}

// Message reactions and emojis

// GetReactions gets reactions for a message
func (mc *MessageController) GetReactions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	reactions, err := mc.messageService.GetReactions(c.Request.Context(), userID, messageID)
	if err != nil {
		logrus.Errorf("Get reactions failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get reactions")
		}
		return
	}

	utils.SuccessResponse(c, "Reactions retrieved successfully", reactions)
}

// AddReaction adds a reaction to a message
func (mc *MessageController) AddReaction(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	emoji := c.Param("emoji")
	if messageID == "" || emoji == "" {
		utils.BadRequestResponse(c, "Message ID and emoji are required")
		return
	}

	err := mc.messageService.AddReaction(c.Request.Context(), userID, messageID, emoji)
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
func (mc *MessageController) RemoveReaction(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
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

// GetReactionUsers gets users who reacted with a specific emoji
func (mc *MessageController) GetReactionUsers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	emoji := c.Param("emoji")
	if messageID == "" || emoji == "" {
		utils.BadRequestResponse(c, "Message ID and emoji are required")
		return
	}

	users, err := mc.messageService.GetReactionUsers(c.Request.Context(), userID, messageID, emoji)
	if err != nil {
		logrus.Errorf("Get reaction users failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get reaction users")
		}
		return
	}

	utils.SuccessResponse(c, "Reaction users retrieved successfully", users)
}

// Media handling

// UploadMedia uploads media for messages
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
		mediaType = "image"
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

// GetMedia gets media by ID
func (mc *MessageController) GetMedia(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	mediaID := c.Param("mediaId")
	if mediaID == "" {
		utils.BadRequestResponse(c, "Media ID is required")
		return
	}

	media, err := mc.messageService.GetMedia(c.Request.Context(), userID, mediaID)
	if err != nil {
		logrus.Errorf("Get media failed: %v", err)
		switch err.Error() {
		case "media not found":
			utils.NotFoundResponse(c, "Media")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this media")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get media")
		}
		return
	}

	utils.SuccessResponse(c, "Media retrieved successfully", media)
}

// DeleteMedia deletes media
func (mc *MessageController) DeleteMedia(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	mediaID := c.Param("mediaId")
	if mediaID == "" {
		utils.BadRequestResponse(c, "Media ID is required")
		return
	}

	err := mc.messageService.DeleteMedia(c.Request.Context(), userID, mediaID)
	if err != nil {
		logrus.Errorf("Delete media failed: %v", err)
		switch err.Error() {
		case "media not found":
			utils.NotFoundResponse(c, "Media")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own media")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete media")
		}
		return
	}

	utils.SuccessResponse(c, "Media deleted successfully", nil)
}

// GetMediaThumbnail gets media thumbnail
func (mc *MessageController) GetMediaThumbnail(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	mediaID := c.Param("mediaId")
	if mediaID == "" {
		utils.BadRequestResponse(c, "Media ID is required")
		return
	}

	thumbnail, err := mc.messageService.GetMediaThumbnail(c.Request.Context(), userID, mediaID)
	if err != nil {
		logrus.Errorf("Get media thumbnail failed: %v", err)
		switch err.Error() {
		case "media not found":
			utils.NotFoundResponse(c, "Media")
		case "thumbnail not available":
			utils.NotFoundResponse(c, "Thumbnail")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this media")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get media thumbnail")
		}
		return
	}

	utils.SuccessResponse(c, "Media thumbnail retrieved successfully", thumbnail)
}

// CompressMedia compresses media file
func (mc *MessageController) CompressMedia(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	mediaID := c.Param("mediaId")
	if mediaID == "" {
		utils.BadRequestResponse(c, "Media ID is required")
		return
	}

	var req models.CompressMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	compressedMedia, err := mc.messageService.CompressMedia(c.Request.Context(), userID, mediaID, req)
	if err != nil {
		logrus.Errorf("Compress media failed: %v", err)
		switch err.Error() {
		case "media not found":
			utils.NotFoundResponse(c, "Media")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only compress your own media")
		case "compression failed":
			utils.BadRequestResponse(c, "Media compression failed")
		default:
			utils.InternalServerErrorResponse(c, "Failed to compress media")
		}
		return
	}

	utils.SuccessResponse(c, "Media compressed successfully", compressedMedia)
}

// Message search and filtering

// SearchMessages searches messages across all accessible circles
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	messageType := c.Query("type")
	dateFrom := c.Query("dateFrom")
	dateTo := c.Query("dateTo")

	req := models.SearchMessagesRequest{
		Query:       query,
		Page:        page,
		PageSize:    pageSize,
		MessageType: messageType,
		DateFrom:    dateFrom,
		DateTo:      dateTo,
	}

	results, err := mc.messageService.SearchMessages(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Search messages failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to search messages")
		return
	}

	utils.SuccessResponse(c, "Messages searched successfully", results)
}

// SearchInCircle searches messages within a specific circle
func (mc *MessageController) SearchInCircle(c *gin.Context) {
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

	query := c.Query("q")
	if query == "" {
		utils.BadRequestResponse(c, "Search query is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	req := models.SearchInCircleRequest{
		CircleID: circleID,
		Query:    query,
		Page:     page,
		PageSize: pageSize,
	}

	results, err := mc.messageService.SearchInCircle(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Search in circle failed: %v", err)
		switch err.Error() {
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to search messages")
		}
		return
	}

	utils.SuccessResponse(c, "Circle messages searched successfully", results)
}

// SearchMedia searches for media files
func (mc *MessageController) SearchMedia(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	mediaType := c.Query("type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	circleID := c.Query("circleId")

	req := models.SearchMediaRequest{
		MediaType: mediaType,
		Page:      page,
		PageSize:  pageSize,
		CircleID:  circleID,
	}

	results, err := mc.messageService.SearchMedia(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Search media failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to search media")
		return
	}

	utils.SuccessResponse(c, "Media searched successfully", results)
}

// SearchMentions searches for messages that mention the user
func (mc *MessageController) SearchMentions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	circleID := c.Query("circleId")

	req := models.SearchMentionsRequest{
		Page:     page,
		PageSize: pageSize,
		CircleID: circleID,
	}

	results, err := mc.messageService.SearchMentions(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Search mentions failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to search mentions")
		return
	}

	utils.SuccessResponse(c, "Mentions searched successfully", results)
}

// SearchLinks searches for messages containing links
func (mc *MessageController) SearchLinks(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	circleID := c.Query("circleId")
	domain := c.Query("domain")

	req := models.SearchLinksRequest{
		Page:     page,
		PageSize: pageSize,
		CircleID: circleID,
		Domain:   domain,
	}

	results, err := mc.messageService.SearchLinks(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Search links failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to search links")
		return
	}

	utils.SuccessResponse(c, "Links searched successfully", results)
}

// SearchFiles searches for file attachments
func (mc *MessageController) SearchFiles(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	fileType := c.Query("fileType")
	circleID := c.Query("circleId")

	req := models.SearchFilesRequest{
		Page:     page,
		PageSize: pageSize,
		FileType: fileType,
		CircleID: circleID,
	}

	results, err := mc.messageService.SearchFiles(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Search files failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to search files")
		return
	}

	utils.SuccessResponse(c, "Files searched successfully", results)
}

// Message status and delivery

// MarkAsRead marks a message as read
func (mc *MessageController) MarkAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	err := mc.messageService.MarkAsRead(c.Request.Context(), userID, messageID)
	if err != nil {
		logrus.Errorf("Mark as read failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to mark message as read")
		}
		return
	}

	utils.SuccessResponse(c, "Message marked as read", nil)
}

// MarkAsUnread marks a message as unread
func (mc *MessageController) MarkAsUnread(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	err := mc.messageService.MarkAsUnread(c.Request.Context(), userID, messageID)
	if err != nil {
		logrus.Errorf("Mark as unread failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to mark message as unread")
		}
		return
	}

	utils.SuccessResponse(c, "Message marked as unread", nil)
}

// BulkMarkAsRead marks multiple messages as read
func (mc *MessageController) BulkMarkAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.BulkMarkAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	count, err := mc.messageService.BulkMarkAsRead(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Bulk mark as read failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to mark messages as read")
		return
	}

	utils.SuccessResponse(c, "Messages marked as read successfully", map[string]interface{}{
		"count": count,
	})
}

// GetDeliveryStatus gets delivery status of a message
func (mc *MessageController) GetDeliveryStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	status, err := mc.messageService.GetDeliveryStatus(c.Request.Context(), userID, messageID)
	if err != nil {
		logrus.Errorf("Get delivery status failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get delivery status")
		}
		return
	}

	utils.SuccessResponse(c, "Delivery status retrieved successfully", status)
}

// GetReadReceipts gets read receipts for a message
func (mc *MessageController) GetReadReceipts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	receipts, err := mc.messageService.GetReadReceipts(c.Request.Context(), userID, messageID)
	if err != nil {
		logrus.Errorf("Get read receipts failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get read receipts")
		}
		return
	}

	utils.SuccessResponse(c, "Read receipts retrieved successfully", receipts)
}

// Message forwarding

// ForwardToCircle forwards a message to a circle
func (mc *MessageController) ForwardToCircle(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	circleID := c.Param("circleId")
	if messageID == "" || circleID == "" {
		utils.BadRequestResponse(c, "Message ID and Circle ID are required")
		return
	}

	var req models.ForwardMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body for simple forwarding
		req = models.ForwardMessageRequest{}
	}

	forwardedMessage, err := mc.messageService.ForwardToCircle(c.Request.Context(), userID, messageID, circleID, req)
	if err != nil {
		logrus.Errorf("Forward to circle failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to forward this message or access this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to forward message")
		}
		return
	}

	utils.CreatedResponse(c, "Message forwarded successfully", forwardedMessage)
}

// ForwardToUser forwards a message to a user (direct message)
func (mc *MessageController) ForwardToUser(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	targetUserID := c.Param("userId")
	if messageID == "" || targetUserID == "" {
		utils.BadRequestResponse(c, "Message ID and User ID are required")
		return
	}

	var req models.ForwardMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = models.ForwardMessageRequest{}
	}

	forwardedMessage, err := mc.messageService.ForwardToUser(c.Request.Context(), userID, messageID, targetUserID, req)
	if err != nil {
		logrus.Errorf("Forward to user failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "user not found":
			utils.NotFoundResponse(c, "User")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to forward this message or contact this user")
		default:
			utils.InternalServerErrorResponse(c, "Failed to forward message")
		}
		return
	}

	utils.CreatedResponse(c, "Message forwarded successfully", forwardedMessage)
}

// GetForwardHistory gets forward history of a message
func (mc *MessageController) GetForwardHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	history, err := mc.messageService.GetForwardHistory(c.Request.Context(), userID, messageID)
	if err != nil {
		logrus.Errorf("Get forward history failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get forward history")
		}
		return
	}

	utils.SuccessResponse(c, "Forward history retrieved successfully", history)
}

// Message scheduling

// ScheduleMessage schedules a message to be sent later
func (mc *MessageController) ScheduleMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.ScheduleMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	scheduledMessage, err := mc.messageService.ScheduleMessage(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Schedule message failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid message data or schedule time")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to send messages to this circle")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "invalid schedule time":
			utils.BadRequestResponse(c, "Schedule time must be in the future")
		default:
			utils.InternalServerErrorResponse(c, "Failed to schedule message")
		}
		return
	}

	utils.CreatedResponse(c, "Message scheduled successfully", scheduledMessage)
}

// GetScheduledMessages gets user's scheduled messages
func (mc *MessageController) GetScheduledMessages(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status") // pending, sent, cancelled

	req := models.GetScheduledMessagesRequest{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
	}

	scheduledMessages, err := mc.messageService.GetScheduledMessages(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get scheduled messages failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get scheduled messages")
		return
	}

	utils.SuccessResponse(c, "Scheduled messages retrieved successfully", scheduledMessages)
}

// GetScheduledMessage gets a specific scheduled message
func (mc *MessageController) GetScheduledMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	scheduleID := c.Param("scheduleId")
	if scheduleID == "" {
		utils.BadRequestResponse(c, "Schedule ID is required")
		return
	}

	scheduledMessage, err := mc.messageService.GetScheduledMessage(c.Request.Context(), userID, scheduleID)
	if err != nil {
		logrus.Errorf("Get scheduled message failed: %v", err)
		switch err.Error() {
		case "scheduled message not found":
			utils.NotFoundResponse(c, "Scheduled message")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only view your own scheduled messages")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get scheduled message")
		}
		return
	}

	utils.SuccessResponse(c, "Scheduled message retrieved successfully", scheduledMessage)
}

// UpdateScheduledMessage updates a scheduled message
func (mc *MessageController) UpdateScheduledMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	scheduleID := c.Param("scheduleId")
	if scheduleID == "" {
		utils.BadRequestResponse(c, "Schedule ID is required")
		return
	}

	var req models.UpdateScheduledMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	scheduledMessage, err := mc.messageService.UpdateScheduledMessage(c.Request.Context(), userID, scheduleID, req)
	if err != nil {
		logrus.Errorf("Update scheduled message failed: %v", err)
		switch err.Error() {
		case "scheduled message not found":
			utils.NotFoundResponse(c, "Scheduled message")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own scheduled messages")
		case "cannot update sent message":
			utils.BadRequestResponse(c, "Cannot update already sent message")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid message data or schedule time")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update scheduled message")
		}
		return
	}

	utils.SuccessResponse(c, "Scheduled message updated successfully", scheduledMessage)
}

// CancelScheduledMessage cancels a scheduled message
func (mc *MessageController) CancelScheduledMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	scheduleID := c.Param("scheduleId")
	if scheduleID == "" {
		utils.BadRequestResponse(c, "Schedule ID is required")
		return
	}

	err := mc.messageService.CancelScheduledMessage(c.Request.Context(), userID, scheduleID)
	if err != nil {
		logrus.Errorf("Cancel scheduled message failed: %v", err)
		switch err.Error() {
		case "scheduled message not found":
			utils.NotFoundResponse(c, "Scheduled message")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only cancel your own scheduled messages")
		case "cannot cancel sent message":
			utils.BadRequestResponse(c, "Cannot cancel already sent message")
		default:
			utils.InternalServerErrorResponse(c, "Failed to cancel scheduled message")
		}
		return
	}

	utils.SuccessResponse(c, "Scheduled message cancelled successfully", nil)
}

// Message templates and quick replies

// GetMessageTemplates gets user's message templates
func (mc *MessageController) GetMessageTemplates(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	category := c.Query("category")

	req := models.GetTemplatesRequest{
		Page:     page,
		PageSize: pageSize,
		Category: category,
	}

	templates, err := mc.messageService.GetMessageTemplates(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get message templates failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get message templates")
		return
	}

	utils.SuccessResponse(c, "Message templates retrieved successfully", templates)
}

// CreateMessageTemplate creates a new message template
func (mc *MessageController) CreateMessageTemplate(c *gin.Context) {
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

	template, err := mc.messageService.CreateMessageTemplate(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create message template failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid template data")
		case "template name exists":
			utils.BadRequestResponse(c, "Template with this name already exists")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create message template")
		}
		return
	}

	utils.CreatedResponse(c, "Message template created successfully", template)
}

// GetMessageTemplate gets a specific message template
func (mc *MessageController) GetMessageTemplate(c *gin.Context) {
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

	template, err := mc.messageService.GetMessageTemplate(c.Request.Context(), userID, templateID)
	if err != nil {
		logrus.Errorf("Get message template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only view your own templates")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get message template")
		}
		return
	}

	utils.SuccessResponse(c, "Message template retrieved successfully", template)
}

// UpdateMessageTemplate updates a message template
func (mc *MessageController) UpdateMessageTemplate(c *gin.Context) {
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

	template, err := mc.messageService.UpdateMessageTemplate(c.Request.Context(), userID, templateID, req)
	if err != nil {
		logrus.Errorf("Update message template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own templates")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid template data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update message template")
		}
		return
	}

	utils.SuccessResponse(c, "Message template updated successfully", template)
}

// DeleteMessageTemplate deletes a message template
func (mc *MessageController) DeleteMessageTemplate(c *gin.Context) {
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

	err := mc.messageService.DeleteMessageTemplate(c.Request.Context(), userID, templateID)
	if err != nil {
		logrus.Errorf("Delete message template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own templates")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete message template")
		}
		return
	}

	utils.SuccessResponse(c, "Message template deleted successfully", nil)
}

// UseMessageTemplate uses a template to send a message
func (mc *MessageController) UseMessageTemplate(c *gin.Context) {
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

	var req models.UseTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	message, err := mc.messageService.UseMessageTemplate(c.Request.Context(), userID, templateID, req)
	if err != nil {
		logrus.Errorf("Use message template failed: %v", err)
		switch err.Error() {
		case "template not found":
			utils.NotFoundResponse(c, "Template")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this template or circle")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid template usage data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to use message template")
		}
		return
	}

	utils.CreatedResponse(c, "Message sent using template successfully", message)
}

// Message backup and export

// ExportCircleMessages exports messages from a circle
func (mc *MessageController) ExportCircleMessages(c *gin.Context) {
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

	var req models.ExportMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body for default export
		req = models.ExportMessagesRequest{
			Format: "json",
		}
	}

	exportJob, err := mc.messageService.ExportCircleMessages(c.Request.Context(), userID, circleID, req)
	if err != nil {
		logrus.Errorf("Export circle messages failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid export parameters")
		default:
			utils.InternalServerErrorResponse(c, "Failed to start message export")
		}
		return
	}

	utils.AcceptedResponse(c, "Message export started successfully", exportJob)
}

// GetExportStatus gets the status of a message export
func (mc *MessageController) GetExportStatus(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	exportID := c.Query("exportId")
	if exportID == "" {
		utils.BadRequestResponse(c, "Export ID is required")
		return
	}

	status, err := mc.messageService.GetExportStatus(c.Request.Context(), userID, exportID)
	if err != nil {
		logrus.Errorf("Get export status failed: %v", err)
		switch err.Error() {
		case "export not found":
			utils.NotFoundResponse(c, "Export")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only view your own exports")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get export status")
		}
		return
	}

	utils.SuccessResponse(c, "Export status retrieved successfully", status)
}

// DownloadMessageExport downloads a completed message export
func (mc *MessageController) DownloadMessageExport(c *gin.Context) {
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

	exportData, err := mc.messageService.DownloadMessageExport(c.Request.Context(), userID, exportID)
	if err != nil {
		logrus.Errorf("Download message export failed: %v", err)
		switch err.Error() {
		case "export not found":
			utils.NotFoundResponse(c, "Export")
		case "export not ready":
			utils.BadRequestResponse(c, "Export is not ready for download")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only download your own exports")
		default:
			utils.InternalServerErrorResponse(c, "Failed to download export")
		}
		return
	}

	// Set headers for file download
	c.Header("Content-Disposition", "attachment; filename="+exportData.Filename)
	c.Header("Content-Type", exportData.ContentType)
	c.Data(200, exportData.ContentType, exportData.Data)
}

// ImportMessages imports messages from a file
func (mc *MessageController) ImportMessages(c *gin.Context) {
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

	circleID := c.PostForm("circleId")
	if circleID == "" {
		utils.BadRequestResponse(c, "Circle ID is required")
		return
	}

	req := models.ImportMessagesRequest{
		File:     file,
		Header:   header,
		CircleID: circleID,
	}

	importJob, err := mc.messageService.ImportMessages(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Import messages failed: %v", err)
		switch err.Error() {
		case "invalid file format":
			utils.BadRequestResponse(c, "Invalid file format")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to import messages to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to start message import")
		}
		return
	}

	utils.AcceptedResponse(c, "Message import started successfully", importJob)
}

// Message moderation

// ReportMessage reports a message for moderation
func (mc *MessageController) ReportMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	var req models.ReportMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	report, err := mc.messageService.ReportMessage(c.Request.Context(), userID, messageID, req)
	if err != nil {
		logrus.Errorf("Report message failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this message")
		case "already reported":
			utils.BadRequestResponse(c, "You have already reported this message")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid report data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to report message")
		}
		return
	}

	utils.CreatedResponse(c, "Message reported successfully", report)
}

// FlagMessage flags a message (admin action)
func (mc *MessageController) FlagMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	var req models.FlagMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := mc.messageService.FlagMessage(c.Request.Context(), userID, messageID, req)
	if err != nil {
		logrus.Errorf("Flag message failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have admin permissions")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid flag data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to flag message")
		}
		return
	}

	utils.SuccessResponse(c, "Message flagged successfully", nil)
}

// AdminDeleteMessage deletes a message (admin action)
func (mc *MessageController) AdminDeleteMessage(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		utils.BadRequestResponse(c, "Message ID is required")
		return
	}

	var req models.AdminDeleteMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req = models.AdminDeleteMessageRequest{}
	}

	err := mc.messageService.AdminDeleteMessage(c.Request.Context(), userID, messageID, req)
	if err != nil {
		logrus.Errorf("Admin delete message failed: %v", err)
		switch err.Error() {
		case "message not found":
			utils.NotFoundResponse(c, "Message")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have admin permissions")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete message")
		}
		return
	}

	utils.SuccessResponse(c, "Message deleted successfully", nil)
}

// GetMessageReports gets message reports for moderation
func (mc *MessageController) GetMessageReports(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status")     // pending, reviewed, resolved
	severity := c.Query("severity") // low, medium, high

	req := models.GetReportsRequest{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
		Severity: severity,
	}

	reports, err := mc.messageService.GetMessageReports(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get message reports failed: %v", err)
		switch err.Error() {
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have admin permissions")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get message reports")
		}
		return
	}

	utils.SuccessResponse(c, "Message reports retrieved successfully", reports)
}

// HandleMessageReport handles a message report (admin action)
func (mc *MessageController) HandleMessageReport(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	reportID := c.Param("reportId")
	if reportID == "" {
		utils.BadRequestResponse(c, "Report ID is required")
		return
	}

	var req models.HandleReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := mc.messageService.HandleMessageReport(c.Request.Context(), userID, reportID, req)
	if err != nil {
		logrus.Errorf("Handle message report failed: %v", err)
		switch err.Error() {
		case "report not found":
			utils.NotFoundResponse(c, "Report")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have admin permissions")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid report handling data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to handle message report")
		}
		return
	}

	utils.SuccessResponse(c, "Message report handled successfully", result)
}

// Typing indicators

// StartTyping indicates user started typing in a circle
func (mc *MessageController) StartTyping(c *gin.Context) {
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

	err := mc.messageService.StartTyping(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Start typing failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to start typing")
		}
		return
	}

	utils.SuccessResponse(c, "Typing indicator started", nil)
}

// StopTyping indicates user stopped typing in a circle
func (mc *MessageController) StopTyping(c *gin.Context) {
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

	err := mc.messageService.StopTyping(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Stop typing failed: %v", err)
		// Don't return error for stop typing to avoid issues
	}

	utils.SuccessResponse(c, "Typing indicator stopped", nil)
}

// GetTypingStatus gets typing status in a circle
func (mc *MessageController) GetTypingStatus(c *gin.Context) {
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

	typingUsers, err := mc.messageService.GetTypingStatus(c.Request.Context(), userID, circleID)
	if err != nil {
		logrus.Errorf("Get typing status failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get typing status")
		}
		return
	}

	utils.SuccessResponse(c, "Typing status retrieved successfully", typingUsers)
}

// Message analytics

// GetMessageStats gets message statistics
func (mc *MessageController) GetMessageStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.DefaultQuery("period", "7d") // 1d, 7d, 30d, 90d
	circleID := c.Query("circleId")

	req := models.GetStatsRequest{
		Period:   period,
		CircleID: circleID,
	}

	stats, err := mc.messageService.GetMessageStats(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get message stats failed: %v", err)
		switch err.Error() {
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this data")
		case "invalid period":
			utils.BadRequestResponse(c, "Invalid time period")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get message statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Message statistics retrieved successfully", stats)
}

// GetCircleMessageStats gets message statistics for a specific circle
func (mc *MessageController) GetCircleMessageStats(c *gin.Context) {
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

	period := c.DefaultQuery("period", "7d")

	req := models.GetCircleStatsRequest{
		CircleID: circleID,
		Period:   period,
	}

	stats, err := mc.messageService.GetCircleMessageStats(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get circle message stats failed: %v", err)
		switch err.Error() {
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		case "invalid period":
			utils.BadRequestResponse(c, "Invalid time period")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get circle message statistics")
		}
		return
	}

	utils.SuccessResponse(c, "Circle message statistics retrieved successfully", stats)
}

// GetMessageActivity gets message activity data
func (mc *MessageController) GetMessageActivity(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.DefaultQuery("period", "7d")
	granularity := c.DefaultQuery("granularity", "hour") // hour, day
	circleID := c.Query("circleId")

	req := models.GetActivityRequest{
		Period:      period,
		Granularity: granularity,
		CircleID:    circleID,
	}

	activity, err := mc.messageService.GetMessageActivity(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get message activity failed: %v", err)
		switch err.Error() {
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this data")
		case "invalid parameters":
			utils.BadRequestResponse(c, "Invalid parameters")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get message activity")
		}
		return
	}

	utils.SuccessResponse(c, "Message activity retrieved successfully", activity)
}

// GetMessageTrends gets message trends
func (mc *MessageController) GetMessageTrends(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.DefaultQuery("period", "30d")
	trendType := c.DefaultQuery("type", "volume") // volume, engagement, types

	req := models.GetTrendsRequest{
		Period:    period,
		TrendType: trendType,
	}

	trends, err := mc.messageService.GetMessageTrends(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get message trends failed: %v", err)
		switch err.Error() {
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this data")
		case "invalid parameters":
			utils.BadRequestResponse(c, "Invalid parameters")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get message trends")
		}
		return
	}

	utils.SuccessResponse(c, "Message trends retrieved successfully", trends)
}

// GetPopularMessages gets popular messages
func (mc *MessageController) GetPopularMessages(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	period := c.DefaultQuery("period", "7d")
	metric := c.DefaultQuery("metric", "reactions") // reactions, replies, views
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	circleID := c.Query("circleId")

	req := models.GetPopularMessagesRequest{
		Period:   period,
		Metric:   metric,
		Limit:    limit,
		CircleID: circleID,
	}

	messages, err := mc.messageService.GetPopularMessages(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get popular messages failed: %v", err)
		switch err.Error() {
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this data")
		case "invalid parameters":
			utils.BadRequestResponse(c, "Invalid parameters")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get popular messages")
		}
		return
	}

	utils.SuccessResponse(c, "Popular messages retrieved successfully", messages)
}

// Message automation and bots

// GetAutomationRules gets automation rules
func (mc *MessageController) GetAutomationRules(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	ruleType := c.Query("type") // auto_reply, keyword_trigger, schedule
	status := c.Query("status") // active, inactive

	req := models.GetAutomationRulesRequest{
		Page:     page,
		PageSize: pageSize,
		RuleType: ruleType,
		Status:   status,
	}

	rules, err := mc.messageService.GetAutomationRules(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get automation rules failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get automation rules")
		return
	}

	utils.SuccessResponse(c, "Automation rules retrieved successfully", rules)
}

// CreateAutomationRule creates a new automation rule
func (mc *MessageController) CreateAutomationRule(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.CreateAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	rule, err := mc.messageService.CreateAutomationRule(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Create automation rule failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid automation rule data")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have permission to create automation rules")
		case "rule limit exceeded":
			utils.BadRequestResponse(c, "Maximum number of automation rules exceeded")
		default:
			utils.InternalServerErrorResponse(c, "Failed to create automation rule")
		}
		return
	}

	utils.CreatedResponse(c, "Automation rule created successfully", rule)
}

// UpdateAutomationRule updates an automation rule
func (mc *MessageController) UpdateAutomationRule(c *gin.Context) {
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

	var req models.UpdateAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	rule, err := mc.messageService.UpdateAutomationRule(c.Request.Context(), userID, ruleID, req)
	if err != nil {
		logrus.Errorf("Update automation rule failed: %v", err)
		switch err.Error() {
		case "rule not found":
			utils.NotFoundResponse(c, "Automation rule")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own automation rules")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid automation rule data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update automation rule")
		}
		return
	}

	utils.SuccessResponse(c, "Automation rule updated successfully", rule)
}

// DeleteAutomationRule deletes an automation rule
func (mc *MessageController) DeleteAutomationRule(c *gin.Context) {
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

	err := mc.messageService.DeleteAutomationRule(c.Request.Context(), userID, ruleID)
	if err != nil {
		logrus.Errorf("Delete automation rule failed: %v", err)
		switch err.Error() {
		case "rule not found":
			utils.NotFoundResponse(c, "Automation rule")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own automation rules")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete automation rule")
		}
		return
	}

	utils.SuccessResponse(c, "Automation rule deleted successfully", nil)
}

// TestAutomationRule tests an automation rule
func (mc *MessageController) TestAutomationRule(c *gin.Context) {
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

	var req models.TestAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	result, err := mc.messageService.TestAutomationRule(c.Request.Context(), userID, ruleID, req)
	if err != nil {
		logrus.Errorf("Test automation rule failed: %v", err)
		switch err.Error() {
		case "rule not found":
			utils.NotFoundResponse(c, "Automation rule")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only test your own automation rules")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid test data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to test automation rule")
		}
		return
	}

	utils.SuccessResponse(c, "Automation rule tested successfully", result)
}

// Message drafts

// GetDrafts gets user's message drafts
func (mc *MessageController) GetDrafts(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	circleID := c.Query("circleId")

	req := models.GetDraftsRequest{
		Page:     page,
		PageSize: pageSize,
		CircleID: circleID,
	}

	drafts, err := mc.messageService.GetDrafts(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Get drafts failed: %v", err)
		utils.InternalServerErrorResponse(c, "Failed to get drafts")
		return
	}

	utils.SuccessResponse(c, "Drafts retrieved successfully", drafts)
}

// SaveDraft saves a message draft
func (mc *MessageController) SaveDraft(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	var req models.SaveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	draft, err := mc.messageService.SaveDraft(c.Request.Context(), userID, req)
	if err != nil {
		logrus.Errorf("Save draft failed: %v", err)
		switch err.Error() {
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid draft data")
		case "access denied":
			utils.ForbiddenResponse(c, "You don't have access to this circle")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		default:
			utils.InternalServerErrorResponse(c, "Failed to save draft")
		}
		return
	}

	utils.CreatedResponse(c, "Draft saved successfully", draft)
}

// GetDraft gets a specific draft
func (mc *MessageController) GetDraft(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	draftID := c.Param("draftId")
	if draftID == "" {
		utils.BadRequestResponse(c, "Draft ID is required")
		return
	}

	draft, err := mc.messageService.GetDraft(c.Request.Context(), userID, draftID)
	if err != nil {
		logrus.Errorf("Get draft failed: %v", err)
		switch err.Error() {
		case "draft not found":
			utils.NotFoundResponse(c, "Draft")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only view your own drafts")
		default:
			utils.InternalServerErrorResponse(c, "Failed to get draft")
		}
		return
	}

	utils.SuccessResponse(c, "Draft retrieved successfully", draft)
}

// UpdateDraft updates a draft
func (mc *MessageController) UpdateDraft(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	draftID := c.Param("draftId")
	if draftID == "" {
		utils.BadRequestResponse(c, "Draft ID is required")
		return
	}

	var req models.UpdateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body")
		return
	}

	draft, err := mc.messageService.UpdateDraft(c.Request.Context(), userID, draftID, req)
	if err != nil {
		logrus.Errorf("Update draft failed: %v", err)
		switch err.Error() {
		case "draft not found":
			utils.NotFoundResponse(c, "Draft")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only update your own drafts")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid draft data")
		default:
			utils.InternalServerErrorResponse(c, "Failed to update draft")
		}
		return
	}

	utils.SuccessResponse(c, "Draft updated successfully", draft)
}

// DeleteDraft deletes a draft
func (mc *MessageController) DeleteDraft(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	draftID := c.Param("draftId")
	if draftID == "" {
		utils.BadRequestResponse(c, "Draft ID is required")
		return
	}

	err := mc.messageService.DeleteDraft(c.Request.Context(), userID, draftID)
	if err != nil {
		logrus.Errorf("Delete draft failed: %v", err)
		switch err.Error() {
		case "draft not found":
			utils.NotFoundResponse(c, "Draft")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only delete your own drafts")
		default:
			utils.InternalServerErrorResponse(c, "Failed to delete draft")
		}
		return
	}

	utils.SuccessResponse(c, "Draft deleted successfully", nil)
}

// SendDraft sends a draft as a message
func (mc *MessageController) SendDraft(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	draftID := c.Param("draftId")
	if draftID == "" {
		utils.BadRequestResponse(c, "Draft ID is required")
		return
	}

	message, err := mc.messageService.SendDraft(c.Request.Context(), userID, draftID)
	if err != nil {
		logrus.Errorf("Send draft failed: %v", err)
		switch err.Error() {
		case "draft not found":
			utils.NotFoundResponse(c, "Draft")
		case "access denied":
			utils.ForbiddenResponse(c, "You can only send your own drafts")
		case "circle not found":
			utils.NotFoundResponse(c, "Circle")
		case "validation failed":
			utils.BadRequestResponse(c, "Invalid draft content")
		default:
			utils.InternalServerErrorResponse(c, "Failed to send draft")
		}
		return
	}

	utils.CreatedResponse(c, "Draft sent successfully", message)
}
