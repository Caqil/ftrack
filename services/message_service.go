package services

import (
	"context"
	"errors"
	"fmt"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"ftrack/websocket"
	"mime/multipart"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageService struct {
	messageRepo    *repositories.MessageRepository
	circleRepo     *repositories.CircleRepository
	userRepo       *repositories.UserRepository
	mediaRepo      *repositories.MediaRepository
	templateRepo   *repositories.TemplateRepository
	draftRepo      *repositories.DraftRepository
	scheduleRepo   *repositories.ScheduleRepository
	reportRepo     *repositories.ReportRepository
	automationRepo *repositories.AutomationRepository
	exportRepo     *repositories.ExportRepository
	mediaService   *MediaService
	searchService  *SearchService
	websocketHub   *websocket.Hub
	validator      *utils.ValidationService

	redisClient interface{} // For typing indicators and caching
}

func NewMessageService(
	messageRepo *repositories.MessageRepository,
	circleRepo *repositories.CircleRepository,
	userRepo *repositories.UserRepository,
	mediaRepo *repositories.MediaRepository,
	templateRepo *repositories.TemplateRepository,
	draftRepo *repositories.DraftRepository,
	scheduleRepo *repositories.ScheduleRepository,
	reportRepo *repositories.ReportRepository,
	automationRepo *repositories.AutomationRepository,
	exportRepo *repositories.ExportRepository,
	websocketHub *websocket.Hub,
	mediaService *MediaService,
	searchService *SearchService,
	redisClient interface{},
) *MessageService {
	return &MessageService{
		messageRepo:    messageRepo,
		circleRepo:     circleRepo,
		userRepo:       userRepo,
		mediaRepo:      mediaRepo,
		templateRepo:   templateRepo,
		draftRepo:      draftRepo,
		scheduleRepo:   scheduleRepo,
		reportRepo:     reportRepo,
		automationRepo: automationRepo,
		exportRepo:     exportRepo,
		websocketHub:   websocketHub,
		validator:      utils.NewValidationService(),
		mediaService:   mediaService,
		searchService:  searchService,
		redisClient:    redisClient,
	}
}

// =============================================================================
// BASIC MESSAGE OPERATIONS
// =============================================================================

func (ms *MessageService) SendMessage(ctx context.Context, userID string, req models.SendMessageRequest) (*models.Message, error) {
	// Validate request
	if validationErrors := ms.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check if user is a member of the circle
	isMember, err := ms.circleRepo.IsMember(ctx, req.CircleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	circleObjectID, err := primitive.ObjectIDFromHex(req.CircleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	// Create message
	message := models.Message{
		CircleID:  circleObjectID,
		SenderID:  userObjectID,
		Type:      req.Type,
		Content:   req.Content,
		Status:    "sent",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set media if provided
	if req.Media != nil {
		message.Media = *req.Media
	}

	// Set location if provided
	if req.Location != nil {
		message.Location = *req.Location
	}

	// Set reply reference if provided
	if req.ReplyTo != "" {
		replyToObjectID, err := primitive.ObjectIDFromHex(req.ReplyTo)
		if err == nil {
			message.ReplyTo = replyToObjectID
		}
	}

	err = ms.messageRepo.Create(ctx, &message)
	if err != nil {
		return nil, err
	}

	// Process automation rules
	go ms.ProcessAutomationRules(ctx, &message)

	// Broadcast message to circle members via WebSocket
	go ms.broadcastMessage(userID, req.CircleID, message)

	// Update circle last activity
	go ms.circleRepo.UpdateLastActivity(ctx, req.CircleID, userID)

	return &message, nil
}

func (ms *MessageService) GetCircleMessages(ctx context.Context, userID string, req models.GetMessagesRequest) (*models.MessagesResponse, error) {
	// Check if user is a member of the circle
	isMember, err := ms.circleRepo.IsMember(ctx, req.CircleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 50
	}

	messages, total, err := ms.messageRepo.GetCircleMessagesPaginated(ctx, req)
	if err != nil {
		return nil, err
	}

	return &models.MessagesResponse{
		Messages:    messages,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ms *MessageService) GetMessage(ctx context.Context, userID, messageID string) (*models.Message, error) {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to this message
	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	return message, nil
}

func (ms *MessageService) UpdateMessage(ctx context.Context, userID, messageID string, req models.EditMessageRequest) (*models.Message, error) {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Check if user is the sender
	if message.SenderID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	// Check if message can be edited (time limit, etc.)
	if time.Since(message.CreatedAt) > 15*time.Minute {
		return nil, errors.New("edit time expired")
	}

	// Check message type (only text messages can be edited)
	if message.Type != "text" {
		return nil, errors.New("only text messages can be edited")
	}

	// Update message
	update := bson.M{
		"content":   req.Content,
		"isEdited":  true,
		"editedAt":  time.Now(),
		"updatedAt": time.Now(),
	}

	err = ms.messageRepo.Update(ctx, messageID, update)
	if err != nil {
		return nil, err
	}

	// Broadcast edit to circle members
	go ms.broadcastMessageEdit(userID, message.CircleID.Hex(), messageID, req.Content)

	return ms.messageRepo.GetByID(ctx, messageID)
}

func (ms *MessageService) DeleteMessage(ctx context.Context, userID, messageID string) error {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	// Check if user is the sender or circle admin
	if message.SenderID.Hex() != userID {
		role, err := ms.circleRepo.GetMemberRole(ctx, message.CircleID.Hex(), userID)
		if err != nil {
			return err
		}

		if role != "admin" {
			return errors.New("access denied")
		}
	}

	err = ms.messageRepo.SoftDelete(ctx, messageID)
	if err != nil {
		return err
	}

	// Broadcast deletion to circle members
	go ms.broadcastMessageDeletion(userID, message.CircleID.Hex(), messageID)

	return nil
}

// =============================================================================
// MESSAGE THREADING AND REPLIES
// =============================================================================

func (ms *MessageService) GetReplies(ctx context.Context, userID, messageID string, page, pageSize int) (*models.RepliesResponse, error) {
	// Check access to parent message
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	replies, total, err := ms.messageRepo.GetReplies(ctx, messageID, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &models.RepliesResponse{
		Replies:     replies,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		HasNext:     total > int64(page*pageSize),
		HasPrevious: page > 1,
	}, nil
}

func (ms *MessageService) ReplyToMessage(ctx context.Context, userID, messageID string, req models.ReplyMessageRequest) (*models.Message, error) {
	// Check access to parent message
	parentMessage, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, parentMessage.CircleID.Hex(), userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	// Create reply message
	sendReq := models.SendMessageRequest{
		CircleID: parentMessage.CircleID.Hex(),
		Type:     req.Type,
		Content:  req.Content,
		Media:    req.Media,
		Location: req.Location,
		ReplyTo:  messageID,
	}

	reply, err := ms.SendMessage(ctx, userID, sendReq)
	if err != nil {
		return nil, err
	}

	// Update parent message reply count
	go ms.messageRepo.IncrementReplyCount(ctx, messageID)

	return reply, nil
}

func (ms *MessageService) GetReply(ctx context.Context, userID, replyID string) (*models.Message, error) {
	return ms.GetMessage(ctx, userID, replyID)
}

func (ms *MessageService) UpdateReply(ctx context.Context, userID, replyID string, req models.EditMessageRequest) (*models.Message, error) {
	return ms.UpdateMessage(ctx, userID, replyID, req)
}

func (ms *MessageService) DeleteReply(ctx context.Context, userID, replyID string) error {
	return ms.DeleteMessage(ctx, userID, replyID)
}

// =============================================================================
// MESSAGE REACTIONS
// =============================================================================

func (ms *MessageService) GetReactions(ctx context.Context, userID, messageID string) (*models.ReactionsResponse, error) {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	reactions := ms.aggregateReactions(message.Reactions)

	return &models.ReactionsResponse{
		MessageID: messageID,
		Reactions: reactions,
		Total:     len(message.Reactions),
	}, nil
}

func (ms *MessageService) AddReaction(ctx context.Context, userID, messageID, emoji string) error {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("access denied")
	}

	err = ms.messageRepo.AddReaction(ctx, messageID, userID, emoji)
	if err != nil {
		return err
	}

	// Broadcast reaction to circle members
	go ms.broadcastReaction(userID, message.CircleID.Hex(), messageID, emoji, "add")

	return nil
}

func (ms *MessageService) RemoveReaction(ctx context.Context, userID, messageID, emoji string) error {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("access denied")
	}

	err = ms.messageRepo.RemoveReaction(ctx, messageID, userID, emoji)
	if err != nil {
		return err
	}

	// Broadcast reaction removal to circle members
	go ms.broadcastReaction(userID, message.CircleID.Hex(), messageID, emoji, "remove")

	return nil
}

func (ms *MessageService) GetReactionUsers(ctx context.Context, userID, messageID, emoji string) (*models.ReactionUsersResponse, error) {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	users, err := ms.messageRepo.GetReactionUsers(ctx, messageID, emoji)
	if err != nil {
		return nil, err
	}

	return &models.ReactionUsersResponse{
		MessageID: messageID,
		Emoji:     emoji,
		Users:     users,
		Count:     len(users),
	}, nil
}

// =============================================================================
// MEDIA HANDLING
// =============================================================================

func (ms *MessageService) UploadMedia(ctx context.Context, userID string, req models.UploadMediaRequest) (*models.MessageMedia, error) {
	// Validate file type and size
	if !ms.isValidMediaType(req.Header.Header.Get("Content-Type")) {
		return nil, errors.New("invalid file type")
	}

	if req.Header.Size > 50*1024*1024 { // 50MB limit
		return nil, errors.New("file too large")
	}

	// Upload file using media service
	media, err := ms.mediaService.UploadFile(ctx, req.File, req.Header, userID)
	if err != nil {
		return nil, err
	}

	// Create media record
	messageMedia := &models.MessageMedia{
		URL:          media.URL,
		Type:         req.MediaType,
		Size:         req.Header.Size,
		Filename:     req.Header.Filename,
		MimeType:     req.Header.Header.Get("Content-Type"),
		ThumbnailURL: media.ThumbnailURL,
		Duration:     media.Duration,
		Dimensions:   media.Dimensions,
		UploadedBy:   userID,
		UploadedAt:   time.Now(),
	}

	// Convert to MessageMediaExtended if needed
	messageMediaExtended := &models.MessageMediaExtended{
		MessageMedia: *messageMedia,
		// Add any additional fields initialization here if needed
	}

	err = ms.mediaRepo.Create(ctx, messageMediaExtended)
	if err != nil {
		return nil, err
	}

	return messageMedia, nil
}

func (ms *MessageService) GetMedia(ctx context.Context, userID, mediaID string) (*models.MessageMedia, error) {
	media, err := ms.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to this media
	if media.UploadedBy != userID {
		// Check if media is used in a message in a circle the user has access to
		hasAccess, err := ms.checkMediaAccess(ctx, userID, mediaID)
		if err != nil {
			return nil, err
		}

		if !hasAccess {
			return nil, errors.New("access denied")
		}
	}

	return media, nil
}

func (ms *MessageService) DeleteMedia(ctx context.Context, userID, mediaID string) error {
	media, err := ms.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return err
	}

	if media.UploadedBy != userID {
		return errors.New("access denied")
	}

	// Delete from storage
	err = ms.mediaService.DeleteFile(ctx, media.URL)
	if err != nil {
		logrus.Errorf("Failed to delete media file: %v", err)
	}

	// Delete from database
	return ms.mediaRepo.Delete(ctx, mediaID)
}

func (ms *MessageService) GetMediaThumbnail(ctx context.Context, userID, mediaID string) (*models.MediaThumbnail, error) {
	media, err := ms.GetMedia(ctx, userID, mediaID)
	if err != nil {
		return nil, err
	}

	if media.ThumbnailURL == "" {
		return nil, errors.New("thumbnail not available")
	}

	return &models.MediaThumbnail{
		MediaID:      mediaID,
		ThumbnailURL: media.ThumbnailURL,
		Type:         media.Type,
	}, nil
}

func (ms *MessageService) CompressMedia(ctx context.Context, userID, mediaID string, req models.CompressMediaRequest) (*models.MessageMedia, error) {
	media, err := ms.GetMedia(ctx, userID, mediaID)
	if err != nil {
		return nil, err
	}

	if media.UploadedBy != userID {
		return nil, errors.New("access denied")
	}

	compressedMedia, err := ms.mediaService.CompressMedia(ctx, media, req.Quality, req.MaxSize)
	if err != nil {
		return nil, errors.New("compression failed")
	}

	// Update media record
	media.URL = compressedMedia.URL
	media.Size = compressedMedia.Size
	media.UpdatedAt = time.Now()

	err = ms.mediaRepo.Update(ctx, mediaID, media)
	if err != nil {
		return nil, err
	}

	return media, nil
}

// =============================================================================
// MESSAGE SEARCH
// =============================================================================

func (ms *MessageService) SearchMessages(ctx context.Context, userID string, req models.SearchMessagesRequest) (*models.SearchResponse, error) {
	// Get user's accessible circles
	circles, err := ms.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	circleIDs := make([]string, len(circles))
	for i, circle := range circles {
		circleIDs[i] = circle.ID.Hex()
	}

	return ms.searchService.SearchMessages(ctx, req, circleIDs)
}

func (ms *MessageService) SearchInCircle(ctx context.Context, userID string, req models.SearchInCircleRequest) (*models.SearchResponse, error) {
	isMember, err := ms.circleRepo.IsMember(ctx, req.CircleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	return ms.searchService.SearchInCircle(ctx, req)
}

func (ms *MessageService) SearchMedia(ctx context.Context, userID string, req models.SearchMediaRequest) (*models.MediaSearchResponse, error) {
	circles, err := ms.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	circleIDs := make([]string, len(circles))
	for i, circle := range circles {
		circleIDs[i] = circle.ID.Hex()
	}

	return ms.searchService.SearchMedia(ctx, req, circleIDs)
}

func (ms *MessageService) SearchMentions(ctx context.Context, userID string, req models.SearchMentionsRequest) (*models.SearchResponse, error) {
	circles, err := ms.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	circleIDs := make([]string, len(circles))
	for i, circle := range circles {
		circleIDs[i] = circle.ID.Hex()
	}

	return ms.searchService.SearchMentions(ctx, userID, req, circleIDs)
}

func (ms *MessageService) SearchLinks(ctx context.Context, userID string, req models.SearchLinksRequest) (*models.LinksSearchResponse, error) {
	circles, err := ms.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	circleIDs := make([]string, len(circles))
	for i, circle := range circles {
		circleIDs[i] = circle.ID.Hex()
	}

	return ms.searchService.SearchLinks(ctx, req, circleIDs)
}

func (ms *MessageService) SearchFiles(ctx context.Context, userID string, req models.SearchFilesRequest) (*models.FilesSearchResponse, error) {
	circles, err := ms.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	circleIDs := make([]string, len(circles))
	for i, circle := range circles {
		circleIDs[i] = circle.ID.Hex()
	}

	return ms.searchService.SearchFiles(ctx, req, circleIDs)
}

// =============================================================================
// MESSAGE STATUS AND DELIVERY
// =============================================================================

func (ms *MessageService) MarkAsRead(ctx context.Context, userID, messageID string) error {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("access denied")
	}

	err = ms.messageRepo.MarkAsRead(ctx, []string{messageID}, userID)
	if err != nil {
		return err
	}

	// Broadcast read receipt
	go ms.broadcastReadReceipt(userID, message.CircleID.Hex(), messageID)

	return nil
}

func (ms *MessageService) MarkAsUnread(ctx context.Context, userID, messageID string) error {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("access denied")
	}

	return ms.messageRepo.MarkAsUnread(ctx, messageID, userID)
}

func (ms *MessageService) BulkMarkAsRead(ctx context.Context, userID string, req models.BulkMarkAsReadRequest) (int, error) {
	validMessageIDs := []string{}

	// Validate all messages and check access
	for _, messageID := range req.MessageIDs {
		message, err := ms.messageRepo.GetByID(ctx, messageID)
		if err != nil {
			continue // Skip invalid message IDs
		}

		isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
		if err != nil || !isMember {
			continue // Skip messages user doesn't have access to
		}

		validMessageIDs = append(validMessageIDs, messageID)
	}

	if len(validMessageIDs) == 0 {
		return 0, nil
	}

	count, err := ms.messageRepo.BulkMarkAsRead(ctx, validMessageIDs, userID)
	if err != nil {
		return 0, err
	}

	// Broadcast bulk read receipts
	go ms.broadcastBulkReadReceipts(userID, validMessageIDs)

	return count, nil
}

func (ms *MessageService) GetDeliveryStatus(ctx context.Context, userID, messageID string) (*models.DeliveryStatusResponse, error) {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Only sender can check delivery status
	if message.SenderID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	status, err := ms.messageRepo.GetDeliveryStatus(ctx, messageID)
	if err != nil {
		return nil, err
	}

	return status, nil
}

func (ms *MessageService) GetReadReceipts(ctx context.Context, userID, messageID string) (*models.ReadReceiptsResponse, error) {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	receipts, err := ms.messageRepo.GetReadReceipts(ctx, messageID)
	if err != nil {
		return nil, err
	}

	return receipts, nil
}

// =============================================================================
// MESSAGE FORWARDING
// =============================================================================

func (ms *MessageService) ForwardToCircle(ctx context.Context, userID, messageID, circleID string, req models.ForwardMessageRequest) (*models.Message, error) {
	// Check access to original message
	originalMessage, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, originalMessage.CircleID.Hex(), userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	// Check access to target circle
	isMember, err = ms.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	// Create forwarded message
	forwardReq := models.SendMessageRequest{
		CircleID: circleID,
		Type:     originalMessage.Type,
		Content:  originalMessage.Content,
		Media:    &originalMessage.Media,
		Location: &originalMessage.Location,
	}

	if req.Comment != "" {
		forwardReq.Content = fmt.Sprintf("%s\n\n--- Forwarded message ---\n%s", req.Comment, originalMessage.Content)
	}

	forwardedMessage, err := ms.SendMessage(ctx, userID, forwardReq)
	if err != nil {
		return nil, err
	}

	// Record forward history
	go ms.recordForwardHistory(messageID, forwardedMessage.ID.Hex(), userID, circleID)

	return forwardedMessage, nil
}

func (ms *MessageService) ForwardToUser(ctx context.Context, userID, messageID, targetUserID string, req models.ForwardMessageRequest) (*models.Message, error) {
	// Check access to original message
	originalMessage, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, originalMessage.CircleID.Hex(), userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	// Create or get direct message circle between users
	dmCircleID, err := ms.getOrCreateDirectMessageCircle(ctx, userID, targetUserID)
	if err != nil {
		return nil, err
	}

	return ms.ForwardToCircle(ctx, userID, messageID, dmCircleID, req)
}

func (ms *MessageService) GetForwardHistory(ctx context.Context, userID, messageID string) (*models.ForwardHistoryResponse, error) {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	history, err := ms.messageRepo.GetForwardHistory(ctx, messageID)
	if err != nil {
		return nil, err
	}

	return &models.ForwardHistoryResponse{
		MessageID: messageID,
		Forwards:  history,
		Count:     len(history),
	}, nil
}

// =============================================================================
// MESSAGE SCHEDULING
// =============================================================================

func (ms *MessageService) ScheduleMessage(ctx context.Context, userID string, req models.ScheduleMessageRequest) (*models.ScheduledMessage, error) {
	// Validate request
	if validationErrors := ms.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check if scheduled time is in the future
	if req.ScheduledAt.Before(time.Now()) {
		return nil, errors.New("invalid schedule time")
	}

	// Check if user is a member of the circle
	isMember, err := ms.circleRepo.IsMember(ctx, req.CircleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	circleObjectID, _ := primitive.ObjectIDFromHex(req.CircleID)

	scheduledMessage := models.ScheduledMessage{
		UserID:      userObjectID,
		CircleID:    circleObjectID,
		Type:        req.Type,
		Content:     req.Content,
		Media:       req.Media,
		Location:    req.Location,
		ScheduledAt: req.ScheduledAt,
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = ms.scheduleRepo.Create(ctx, &scheduledMessage)
	if err != nil {
		return nil, err
	}

	return &scheduledMessage, nil
}

func (ms *MessageService) GetScheduledMessages(ctx context.Context, userID string, req models.GetScheduledMessagesRequest) (*models.ScheduledMessagesResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	messages, total, err := ms.scheduleRepo.GetUserScheduledMessages(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	return &models.ScheduledMessagesResponse{
		Messages:    messages,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ms *MessageService) GetScheduledMessage(ctx context.Context, userID, scheduleID string) (*models.ScheduledMessage, error) {
	scheduledMessage, err := ms.scheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return nil, err
	}

	if scheduledMessage.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	return scheduledMessage, nil
}

func (ms *MessageService) UpdateScheduledMessage(ctx context.Context, userID, scheduleID string, req models.UpdateScheduledMessageRequest) (*models.ScheduledMessage, error) {
	scheduledMessage, err := ms.scheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return nil, err
	}

	if scheduledMessage.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	if scheduledMessage.Status != "pending" {
		return nil, errors.New("cannot update sent message")
	}

	// Validate scheduled time if provided
	if req.ScheduledAt != nil && req.ScheduledAt.Before(time.Now()) {
		return nil, errors.New("validation failed")
	}

	update := bson.M{"updatedAt": time.Now()}
	if req.Content != "" {
		update["content"] = req.Content
	}
	if req.Media != nil {
		update["media"] = req.Media
	}
	if req.Location != nil {
		update["location"] = req.Location
	}
	if req.ScheduledAt != nil {
		update["scheduledAt"] = req.ScheduledAt
	}

	err = ms.scheduleRepo.Update(ctx, scheduleID, update)
	if err != nil {
		return nil, err
	}

	return ms.scheduleRepo.GetByID(ctx, scheduleID)
}

func (ms *MessageService) CancelScheduledMessage(ctx context.Context, userID, scheduleID string) error {
	scheduledMessage, err := ms.scheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return err
	}

	if scheduledMessage.UserID.Hex() != userID {
		return errors.New("access denied")
	}

	if scheduledMessage.Status != "pending" {
		return errors.New("cannot cancel sent message")
	}

	update := bson.M{
		"status":    "cancelled",
		"updatedAt": time.Now(),
	}

	return ms.scheduleRepo.Update(ctx, scheduleID, update)
}

// =============================================================================
// MESSAGE TEMPLATES
// =============================================================================

func (ms *MessageService) GetMessageTemplates(ctx context.Context, userID string, req models.GetTemplatesRequest) (*models.TemplatesResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	templates, total, err := ms.templateRepo.GetUserTemplates(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	return &models.TemplatesResponse{
		Templates:   templates,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ms *MessageService) CreateMessageTemplate(ctx context.Context, userID string, req models.CreateTemplateRequest) (*models.MessageTemplate, error) {
	if validationErrors := ms.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check if template name already exists for user
	exists, err := ms.templateRepo.NameExists(ctx, userID, req.Name)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, errors.New("template name exists")
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)

	template := models.MessageTemplate{
		UserID:    userObjectID,
		Name:      req.Name,
		Category:  req.Category,
		Content:   req.Content,
		Type:      req.Type,
		Media:     req.Media,
		Variables: req.Variables,
		IsPublic:  req.IsPublic,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = ms.templateRepo.Create(ctx, &template)
	if err != nil {
		return nil, err
	}

	return &template, nil
}

func (ms *MessageService) GetMessageTemplate(ctx context.Context, userID, templateID string) (*models.MessageTemplate, error) {
	template, err := ms.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	if template.UserID.Hex() != userID && !template.IsPublic {
		return nil, errors.New("access denied")
	}

	return template, nil
}

func (ms *MessageService) UpdateMessageTemplate(ctx context.Context, userID, templateID string, req models.UpdateTemplateRequest) (*models.MessageTemplate, error) {
	template, err := ms.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	if template.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	update := bson.M{"updatedAt": time.Now()}
	if req.Name != "" {
		// Check if new name already exists
		exists, err := ms.templateRepo.NameExistsExcluding(ctx, userID, req.Name, templateID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("validation failed")
		}
		update["name"] = req.Name
	}
	if req.Category != "" {
		update["category"] = req.Category
	}
	if req.Content != "" {
		update["content"] = req.Content
	}
	if req.Media != nil {
		update["media"] = req.Media
	}
	if req.Variables != nil {
		update["variables"] = req.Variables
	}
	if req.IsPublic != nil {
		update["isPublic"] = *req.IsPublic
	}

	err = ms.templateRepo.Update(ctx, templateID, update)
	if err != nil {
		return nil, err
	}

	return ms.templateRepo.GetByID(ctx, templateID)
}

func (ms *MessageService) DeleteMessageTemplate(ctx context.Context, userID, templateID string) error {
	template, err := ms.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return err
	}

	if template.UserID.Hex() != userID {
		return errors.New("access denied")
	}

	return ms.templateRepo.Delete(ctx, templateID)
}

func (ms *MessageService) UseMessageTemplate(ctx context.Context, userID, templateID string, req models.UseTemplateRequest) (*models.Message, error) {
	template, err := ms.GetMessageTemplate(ctx, userID, templateID)
	if err != nil {
		return nil, err
	}

	// Check access to circle
	isMember, err := ms.circleRepo.IsMember(ctx, req.CircleID, userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	// Replace variables in content
	content := template.Content
	for variable, value := range req.Variables {
		placeholder := fmt.Sprintf("{%s}", variable)
		content = strings.ReplaceAll(content, placeholder, value)
	}

	// Create message request
	messageReq := models.SendMessageRequest{
		CircleID: req.CircleID,
		Type:     template.Type,
		Content:  content,
		Media:    template.Media,
	}

	message, err := ms.SendMessage(ctx, userID, messageReq)
	if err != nil {
		return nil, err
	}

	// Increment template usage count
	go ms.templateRepo.IncrementUsage(context.Background(), templateID)

	return message, nil
}

// =============================================================================
// MESSAGE DRAFTS
// =============================================================================

func (ms *MessageService) GetDrafts(ctx context.Context, userID string, req models.GetDraftsRequest) (*models.DraftsResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	drafts, total, err := ms.draftRepo.GetUserDrafts(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	return &models.DraftsResponse{
		Drafts:      drafts,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ms *MessageService) SaveDraft(ctx context.Context, userID string, req models.SaveDraftRequest) (*models.MessageDraft, error) {
	if validationErrors := ms.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check access to circle
	isMember, err := ms.circleRepo.IsMember(ctx, req.CircleID, userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	circleObjectID, _ := primitive.ObjectIDFromHex(req.CircleID)

	draft := models.MessageDraft{
		UserID:    userObjectID,
		CircleID:  circleObjectID,
		Type:      req.Type,
		Content:   req.Content,
		Media:     req.Media,
		Location:  req.Location,
		AutoSave:  req.AutoSave,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.ReplyTo != "" {
		replyToObjectID, _ := primitive.ObjectIDFromHex(req.ReplyTo)
		draft.ReplyTo = replyToObjectID
	}

	err = ms.draftRepo.Create(ctx, &draft)
	if err != nil {
		return nil, err
	}

	return &draft, nil
}

func (ms *MessageService) GetDraft(ctx context.Context, userID, draftID string) (*models.MessageDraft, error) {
	draft, err := ms.draftRepo.GetByID(ctx, draftID)
	if err != nil {
		return nil, err
	}

	if draft.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	return draft, nil
}

func (ms *MessageService) UpdateDraft(ctx context.Context, userID, draftID string, req models.UpdateDraftRequest) (*models.MessageDraft, error) {
	draft, err := ms.draftRepo.GetByID(ctx, draftID)
	if err != nil {
		return nil, err
	}

	if draft.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	update := bson.M{"updatedAt": time.Now()}
	if req.Content != "" {
		update["content"] = req.Content
	}
	if req.Media != nil {
		update["media"] = req.Media
	}
	if req.Location != nil {
		update["location"] = req.Location
	}

	err = ms.draftRepo.Update(ctx, draftID, update)
	if err != nil {
		return nil, err
	}

	return ms.draftRepo.GetByID(ctx, draftID)
}

func (ms *MessageService) DeleteDraft(ctx context.Context, userID, draftID string) error {
	draft, err := ms.draftRepo.GetByID(ctx, draftID)
	if err != nil {
		return err
	}

	if draft.UserID.Hex() != userID {
		return errors.New("access denied")
	}

	return ms.draftRepo.Delete(ctx, draftID)
}

func (ms *MessageService) SendDraft(ctx context.Context, userID, draftID string) (*models.Message, error) {
	draft, err := ms.draftRepo.GetByID(ctx, draftID)
	if err != nil {
		return nil, err
	}

	if draft.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	// Create message request from draft
	messageReq := models.SendMessageRequest{
		CircleID: draft.CircleID.Hex(),
		Type:     draft.Type,
		Content:  draft.Content,
		Media:    draft.Media,
		Location: draft.Location,
	}

	if !draft.ReplyTo.IsZero() {
		messageReq.ReplyTo = draft.ReplyTo.Hex()
	}

	message, err := ms.SendMessage(ctx, userID, messageReq)
	if err != nil {
		return nil, err
	}

	// Delete draft after successful send
	go ms.draftRepo.Delete(context.Background(), draftID)

	return message, nil
}

// =============================================================================
// EXPORT/IMPORT
// =============================================================================

func (ms *MessageService) ExportCircleMessages(ctx context.Context, userID, circleID string, req models.ExportMessagesRequest) (*models.MessageExport, error) {
	// Check access to circle
	isMember, err := ms.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	circleObjectID, _ := primitive.ObjectIDFromHex(circleID)

	export := models.MessageExport{
		UserID:       userObjectID,
		CircleID:     circleObjectID,
		Format:       req.Format,
		Status:       "processing",
		Progress:     0,
		DateRange:    req.DateRange,
		IncludeMedia: req.IncludeMedia,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = ms.exportRepo.Create(ctx, &export)
	if err != nil {
		return nil, err
	}

	// Start export process in background
	go ms.processExport(export.ID.Hex())

	return &export, nil
}

func (ms *MessageService) GetExportStatus(ctx context.Context, userID, exportID string) (*models.ExportStatusResponse, error) {
	export, err := ms.exportRepo.GetExport(ctx, exportID)
	if err != nil {
		return nil, err
	}

	if export.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	response := &models.ExportStatusResponse{
		ExportID:     exportID,
		Status:       export.Status,
		Progress:     export.Progress,
		FileURL:      export.FileURL,
		FileSize:     export.FileSize,
		MessageCount: export.MessageCount,
		CreatedAt:    export.CreatedAt,
		ExpiresAt:    export.ExpiresAt,
	}

	return response, nil
}

func (ms *MessageService) DownloadMessageExport(ctx context.Context, userID, exportID string) (*models.ExportDownload, error) {
	export, err := ms.exportRepo.GetByID(ctx, exportID)
	if err != nil {
		return nil, err
	}

	if export.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	if export.Status != "completed" {
		return nil, errors.New("export not ready")
	}

	// Download file from storage
	data, err := ms.mediaService.DownloadFile(ctx, export.FileURL)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("messages_export_%s.%s", exportID, export.Format)
	contentType := "application/octet-stream"
	if export.Format == "json" {
		contentType = "application/json"
	} else if export.Format == "csv" {
		contentType = "text/csv"
	}

	return &models.ExportDownload{
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
	}, nil
}

func (ms *MessageService) ImportMessages(ctx context.Context, userID string, req models.ImportMessagesRequest) (*models.ImportJob, error) {
	// Check access to circle
	isMember, err := ms.circleRepo.IsMember(ctx, req.CircleID, userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	// Validate file format
	if !ms.isValidImportFormat(req.Header.Header.Get("Content-Type")) {
		return nil, errors.New("invalid file format")
	}

	// Create import job
	job := &models.ImportJob{
		UserID:    userID,
		CircleID:  req.CircleID,
		Status:    "processing",
		CreatedAt: time.Now(),
	}

	// Process import in background
	go ms.processImport(job, req.File)

	return job, nil
}

// =============================================================================
// MODERATION
// =============================================================================

func (ms *MessageService) ReportMessage(ctx context.Context, userID, messageID string, req models.ReportMessageRequest) (*models.MessageReport, error) {
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Check access to message
	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	// Check if user already reported this message
	exists, err := ms.reportRepo.ReportExists(ctx, messageID, userID)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, errors.New("already reported")
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	messageObjectID, _ := primitive.ObjectIDFromHex(messageID)

	report := models.MessageReport{
		MessageID:   messageObjectID,
		ReportedBy:  userObjectID,
		Reason:      req.Reason,
		Description: req.Description,
		Status:      "pending",
		Severity:    ms.calculateSeverity(req.Reason),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = ms.reportRepo.Create(ctx, &report)
	if err != nil {
		return nil, err
	}

	return &report, nil
}
func (ms *MessageService) FlagMessage(ctx context.Context, userID, messageID string, req models.FlagMessageRequest) error {
	// Check admin permissions
	isAdmin, err := ms.userRepo.IsAdmin(ctx, userID)
	if err != nil || !isAdmin {
		return errors.New("access denied")
	}

	// Get the message to verify it exists
	_, err = ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	// Flag message
	update := bson.M{
		"flagged":    true,
		"flagReason": req.Reason,
		"flaggedBy":  userID,
		"flaggedAt":  time.Now(),
		"updatedAt":  time.Now(),
	}

	return ms.messageRepo.Update(ctx, messageID, update)
}

func (ms *MessageService) AdminDeleteMessage(ctx context.Context, userID, messageID string, req models.AdminDeleteMessageRequest) error {
	// Check admin permissions
	isAdmin, err := ms.userRepo.IsAdmin(ctx, userID)
	if err != nil || !isAdmin {
		return errors.New("access denied")
	}

	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	// Record admin action
	go ms.recordAdminAction(userID, "delete_message", messageID, req.Reason)

	// Delete message
	err = ms.messageRepo.SoftDelete(ctx, messageID)
	if err != nil {
		return err
	}

	// Notify if requested
	if req.Notify {
		go ms.notifyMessageDeleted(message.SenderID.Hex(), messageID, req.Reason)
	}

	return nil
}

func (ms *MessageService) GetMessageReports(ctx context.Context, userID string, req models.GetReportsRequest) (*models.ReportsResponse, error) {
	// Check admin permissions
	isAdmin, err := ms.userRepo.IsAdmin(ctx, userID)
	if err != nil || !isAdmin {
		return nil, errors.New("access denied")
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	reports, total, err := ms.reportRepo.GetReports(ctx, req)
	if err != nil {
		return nil, err
	}

	return &models.ReportsResponse{
		Reports:     reports,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ms *MessageService) HandleMessageReport(ctx context.Context, userID, reportID string, req models.HandleReportRequest) (*models.ReportHandleResult, error) {
	// Check admin permissions
	isAdmin, err := ms.userRepo.IsAdmin(ctx, userID)
	if err != nil || !isAdmin {
		return nil, errors.New("access denied")
	}

	report, err := ms.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return nil, err
	}

	reviewedByObjectID, _ := primitive.ObjectIDFromHex(userID)
	now := time.Now()

	// Update report
	update := bson.M{
		"status":     "reviewed",
		"resolution": req.Resolution,
		"reviewedBy": reviewedByObjectID,
		"reviewedAt": &now,
		"updatedAt":  now,
	}

	err = ms.reportRepo.Update(ctx, reportID, update)
	if err != nil {
		return nil, err
	}

	// Take action based on decision
	var actionTaken string
	switch req.Action {
	case "hide":
		err = ms.messageRepo.Hide(ctx, report.MessageID.Hex())
		actionTaken = "Message hidden"
	case "delete":
		err = ms.messageRepo.SoftDelete(ctx, report.MessageID.Hex())
		actionTaken = "Message deleted"
	case "warn":
		err = ms.warnUser(ctx, report.MessageID.Hex())
		actionTaken = "User warned"
	case "dismiss":
		actionTaken = "Report dismissed"
	}

	if err != nil {
		return nil, err
	}

	return &models.ReportHandleResult{
		ReportID:    reportID,
		Action:      req.Action,
		Resolution:  req.Resolution,
		ActionTaken: actionTaken,
		HandledBy:   userID,
		HandledAt:   now,
	}, nil
}

// =============================================================================
// TYPING INDICATORS
// =============================================================================

func (ms *MessageService) StartTyping(ctx context.Context, userID, circleID string) error {
	// Check access to circle
	isMember, err := ms.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil || !isMember {
		return errors.New("access denied")
	}

	// Set typing indicator in Redis with expiration
	key := fmt.Sprintf("typing:%s:%s", circleID, userID)
	err = ms.setRedisKey(key, "1", 10*time.Second)
	if err != nil {
		return err
	}

	// Broadcast typing start
	wsMessage := models.WSMessage{
		Type: models.WSTypeTypingStart,
		Data: models.WSTypingData{
			CircleID:  circleID,
			UserID:    userID,
			IsTyping:  true,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	ms.websocketHub.BroadcastMessage(circleID, wsMessage)

	return nil
}

func (ms *MessageService) StopTyping(ctx context.Context, userID, circleID string) error {
	// Remove typing indicator from Redis
	key := fmt.Sprintf("typing:%s:%s", circleID, userID)
	_ = ms.deleteRedisKey(key)

	// Broadcast typing stop
	wsMessage := models.WSMessage{
		Type: models.WSTypeTypingStop,
		Data: models.WSTypingData{
			CircleID:  circleID,
			UserID:    userID,
			IsTyping:  false,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	ms.websocketHub.BroadcastMessage(circleID, wsMessage)

	return nil
}

func (ms *MessageService) GetTypingStatus(ctx context.Context, userID, circleID string) ([]string, error) {
	// Check access to circle
	isMember, err := ms.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	// Get typing users from Redis
	pattern := fmt.Sprintf("typing:%s:*", circleID)
	keys, err := ms.getRedisKeys(pattern)
	if err != nil {
		return []string{}, nil
	}

	typingUsers := make([]string, 0)
	for _, key := range keys {
		parts := strings.Split(key, ":")
		if len(parts) == 3 {
			typingUserID := parts[2]
			if typingUserID != userID { // Don't include self
				typingUsers = append(typingUsers, typingUserID)
			}
		}
	}

	return typingUsers, nil
}

// =============================================================================
// ANALYTICS
// =============================================================================

func (ms *MessageService) GetMessageStats(ctx context.Context, userID string, req models.GetStatsRequest) (*models.MessageStatsResponse, error) {
	// Get user's accessible circles
	circles, err := ms.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	circleIDs := make([]string, len(circles))
	for i, circle := range circles {
		circleIDs[i] = circle.ID.Hex()
	}

	// If specific circle requested, check access
	if req.CircleID != "" {
		hasAccess := false
		for _, circleID := range circleIDs {
			if circleID == req.CircleID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return nil, errors.New("access denied")
		}
		circleIDs = []string{req.CircleID}
	}

	// Calculate date range based on period
	endDate := time.Now()
	var startDate time.Time
	switch req.Period {
	case "1d":
		startDate = endDate.AddDate(0, 0, -1)
	case "7d":
		startDate = endDate.AddDate(0, 0, -7)
	case "30d":
		startDate = endDate.AddDate(0, 0, -30)
	case "90d":
		startDate = endDate.AddDate(0, 0, -90)
	default:
		return nil, errors.New("invalid period")
	}

	stats, err := ms.messageRepo.GetMessageStats(ctx, circleIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Calculate previous period for comparison
	previousEndDate := startDate
	previousStartDate := previousEndDate.Add(-(endDate.Sub(startDate)))
	previousStats, err := ms.messageRepo.GetMessageStats(ctx, circleIDs, previousStartDate, previousEndDate)
	if err != nil {
		logrus.Errorf("Failed to get previous period stats: %v", err)
		previousStats = &models.RawMessageStats{} // Use empty stats if failed
	}

	// Calculate growth rate
	var growthRate float64
	var trend string
	if previousStats.TotalMessages > 0 {
		growthRate = ((float64(stats.TotalMessages) - float64(previousStats.TotalMessages)) / float64(previousStats.TotalMessages)) * 100
		if growthRate > 5 {
			trend = "up"
		} else if growthRate < -5 {
			trend = "down"
		} else {
			trend = "stable"
		}
	} else {
		growthRate = 0
		trend = "stable"
	}

	// Calculate engagement metrics
	var avgReactions, avgReplies float64
	if stats.TotalMessages > 0 {
		avgReactions = float64(stats.TotalReactions) / float64(stats.TotalMessages)
		avgReplies = float64(stats.TotalReplies) / float64(stats.TotalMessages)
	}

	response := &models.MessageStatsResponse{
		Period:         req.Period,
		TotalMessages:  stats.TotalMessages,
		MessagesByType: stats.MessagesByType,
		ActiveUsers:    stats.ActiveUsers,
		BusiestHour:    stats.BusiestHour,
		BusiestDay:     stats.BusiestDay,
		Engagement: models.EngagementStats{
			TotalReactions: stats.TotalReactions,
			TotalReplies:   stats.TotalReplies,
			AvgReactions:   avgReactions,
			AvgReplies:     avgReplies,
		},
		Growth: models.GrowthStats{
			PreviousPeriod: previousStats.TotalMessages,
			GrowthRate:     growthRate,
			Trend:          trend,
		},
	}

	return response, nil
}

func (ms *MessageService) GetCircleMessageStats(ctx context.Context, userID string, req models.GetCircleStatsRequest) (*models.MessageStatsResponse, error) {
	// Check access to circle
	isMember, err := ms.circleRepo.IsMember(ctx, req.CircleID, userID)
	if err != nil || !isMember {
		return nil, errors.New("access denied")
	}

	// Use the same logic as GetMessageStats but for specific circle
	statsReq := models.GetStatsRequest{
		Period:   req.Period,
		CircleID: req.CircleID,
	}

	return ms.GetMessageStats(ctx, userID, statsReq)
}

func (ms *MessageService) GetMessageActivity(ctx context.Context, userID string, req models.GetActivityRequest) (*models.ActivityResponse, error) {
	// Get user's accessible circles
	circles, err := ms.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	circleIDs := make([]string, len(circles))
	for i, circle := range circles {
		circleIDs[i] = circle.ID.Hex()
	}

	// If specific circle requested, check access
	if req.CircleID != "" {
		hasAccess := false
		for _, circleID := range circleIDs {
			if circleID == req.CircleID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return nil, errors.New("access denied")
		}
		circleIDs = []string{req.CircleID}
	}

	// Calculate date range
	endDate := time.Now()
	var startDate time.Time
	switch req.Period {
	case "1d":
		startDate = endDate.AddDate(0, 0, -1)
	case "7d":
		startDate = endDate.AddDate(0, 0, -7)
	case "30d":
		startDate = endDate.AddDate(0, 0, -30)
	case "90d":
		startDate = endDate.AddDate(0, 0, -90)
	default:
		return nil, errors.New("invalid parameters")
	}

	// Validate granularity
	if req.Granularity != "hour" && req.Granularity != "day" {
		return nil, errors.New("invalid parameters")
	}

	activity, err := ms.messageRepo.GetMessageActivity(ctx, circleIDs, startDate, endDate, req.Granularity)
	if err != nil {
		return nil, err
	}

	return &models.ActivityResponse{
		Period:      req.Period,
		Granularity: req.Granularity,
		DataPoints:  activity,
	}, nil
}

func (ms *MessageService) GetMessageTrends(ctx context.Context, userID string, req models.GetTrendsRequest) (*models.TrendsResponse, error) {
	// Get user's accessible circles
	circles, err := ms.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	circleIDs := make([]string, len(circles))
	for i, circle := range circles {
		circleIDs[i] = circle.ID.Hex()
	}

	// Calculate date range
	endDate := time.Now()
	var startDate time.Time
	switch req.Period {
	case "7d":
		startDate = endDate.AddDate(0, 0, -7)
	case "30d":
		startDate = endDate.AddDate(0, 0, -30)
	case "90d":
		startDate = endDate.AddDate(0, 0, -90)
	default:
		return nil, errors.New("invalid parameters")
	}

	// Validate trend type
	if req.TrendType != "volume" && req.TrendType != "engagement" && req.TrendType != "types" {
		return nil, errors.New("invalid parameters")
	}

	trends, err := ms.messageRepo.GetMessageTrends(ctx, circleIDs, startDate, endDate, req.TrendType)
	if err != nil {
		return nil, err
	}

	return &models.TrendsResponse{
		Period:     req.Period,
		TrendType:  req.TrendType,
		DataPoints: trends,
	}, nil
}

func (ms *MessageService) GetPopularMessages(ctx context.Context, userID string, req models.GetPopularMessagesRequest) (*models.PopularMessagesResponse, error) {
	// Get user's accessible circles
	circles, err := ms.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return nil, err
	}

	circleIDs := make([]string, len(circles))
	for i, circle := range circles {
		circleIDs[i] = circle.ID.Hex()
	}

	// If specific circle requested, check access
	if req.CircleID != "" {
		hasAccess := false
		for _, circleID := range circleIDs {
			if circleID == req.CircleID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return nil, errors.New("access denied")
		}
		circleIDs = []string{req.CircleID}
	}

	// Calculate date range
	endDate := time.Now()
	var startDate time.Time
	switch req.Period {
	case "1d":
		startDate = endDate.AddDate(0, 0, -1)
	case "7d":
		startDate = endDate.AddDate(0, 0, -7)
	case "30d":
		startDate = endDate.AddDate(0, 0, -30)
	default:
		return nil, errors.New("invalid parameters")
	}

	// Validate metric
	if req.Metric != "reactions" && req.Metric != "replies" && req.Metric != "views" {
		return nil, errors.New("invalid parameters")
	}

	if req.Limit < 1 || req.Limit > 50 {
		req.Limit = 10
	}

	popularMessages, err := ms.messageRepo.GetPopularMessages(ctx, circleIDs, startDate, endDate, req.Metric, req.Limit)
	if err != nil {
		return nil, err
	}

	return &models.PopularMessagesResponse{
		Period:   req.Period,
		Metric:   req.Metric,
		Messages: popularMessages,
		Count:    len(popularMessages),
	}, nil
}

// =============================================================================
// AUTOMATION
// =============================================================================

func (ms *MessageService) GetAutomationRules(ctx context.Context, userID string, req models.GetAutomationRulesRequest) (*models.AutomationRulesResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	rules, total, err := ms.automationRepo.GetUserRules(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	return &models.AutomationRulesResponse{
		Rules:       rules,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ms *MessageService) CreateAutomationRule(ctx context.Context, userID string, req models.CreateAutomationRuleRequest) (*models.AutomationRule, error) {
	if validationErrors := ms.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	// Check rule limit
	count, err := ms.automationRepo.GetUserRuleCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	if count >= 50 { // Maximum 50 rules per user
		return nil, errors.New("rule limit exceeded")
	}

	// If circle specified, check access
	if req.CircleID != "" {
		isMember, err := ms.circleRepo.IsMember(ctx, req.CircleID, userID)
		if err != nil || !isMember {
			return nil, errors.New("access denied")
		}
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)

	rule := models.AutomationRule{
		UserID:     userObjectID,
		Name:       req.Name,
		Type:       req.Type,
		IsActive:   req.IsActive,
		Conditions: req.Conditions,
		Actions:    req.Actions,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if req.CircleID != "" {
		circleObjectID, _ := primitive.ObjectIDFromHex(req.CircleID)
		rule.CircleID = &circleObjectID
	}

	err = ms.automationRepo.Create(ctx, &rule)
	if err != nil {
		return nil, err
	}

	return &rule, nil
}

func (ms *MessageService) UpdateAutomationRule(ctx context.Context, userID, ruleID string, req models.UpdateAutomationRuleRequest) (*models.AutomationRule, error) {
	rule, err := ms.automationRepo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}

	if rule.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	update := bson.M{"updatedAt": time.Now()}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Conditions != nil {
		update["conditions"] = req.Conditions
	}
	if req.Actions != nil {
		update["actions"] = req.Actions
	}
	if req.IsActive != nil {
		update["isActive"] = *req.IsActive
	}

	err = ms.automationRepo.Update(ctx, ruleID, update)
	if err != nil {
		return nil, err
	}

	return ms.automationRepo.GetByID(ctx, ruleID)
}

func (ms *MessageService) DeleteAutomationRule(ctx context.Context, userID, ruleID string) error {
	rule, err := ms.automationRepo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}

	if rule.UserID.Hex() != userID {
		return errors.New("access denied")
	}

	return ms.automationRepo.Delete(ctx, ruleID)
}

func (ms *MessageService) TestAutomationRule(ctx context.Context, userID, ruleID string, req models.TestAutomationRuleRequest) (*models.AutomationTestResult, error) {
	rule, err := ms.automationRepo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}

	if rule.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	// Test the rule against the provided message
	triggered, matchedConditions, err := ms.evaluateRuleConditions(rule.Conditions, req.TestMessage, req.Context)
	if err != nil {
		return nil, errors.New("validation failed")
	}

	result := &models.AutomationTestResult{
		RuleID:              ruleID,
		TestMessage:         req.TestMessage,
		Triggered:           triggered,
		MatchedConditions:   matchedConditions,
		WouldExecuteActions: triggered,
		TestedAt:            time.Now(),
	}

	if triggered {
		// Simulate what actions would be taken
		result.SimulatedActions = ms.simulateActions(rule.Actions, req.TestMessage, req.Context)
	}

	return result, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func (ms *MessageService) broadcastMessage(senderID, circleID string, message models.Message) {
	wsMessage := models.WSMessage{
		Type: models.WSTypeMessage,
		Data: models.WSMessageData{
			MessageID: message.ID.Hex(),
			CircleID:  message.CircleID.Hex(),
			SenderID:  message.SenderID.Hex(),
			Type:      message.Type,
			Content:   message.Content,
			Media:     &message.Media,
			Timestamp: message.CreatedAt,
		},
		Timestamp: time.Now(),
	}

	ms.websocketHub.BroadcastMessage(circleID, wsMessage)
}

func (ms *MessageService) broadcastMessageEdit(senderID, circleID, messageID, newContent string) {
	wsMessage := models.WSMessage{
		Type: models.WSTypeMessageEdit,
		Data: models.WSMessageEditData{
			MessageID:  messageID,
			CircleID:   circleID,
			SenderID:   senderID,
			NewContent: newContent,
			Timestamp:  time.Now(),
		},
		Timestamp: time.Now(),
	}

	ms.websocketHub.BroadcastMessage(circleID, wsMessage)
}

func (ms *MessageService) broadcastMessageDeletion(senderID, circleID, messageID string) {
	wsMessage := models.WSMessage{
		Type: models.WSTypeMessageDelete,
		Data: models.WSMessageDeleteData{
			MessageID: messageID,
			CircleID:  circleID,
			SenderID:  senderID,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	ms.websocketHub.BroadcastMessage(circleID, wsMessage)
}

func (ms *MessageService) broadcastReaction(userID, circleID, messageID, emoji, action string) {
	wsMessage := models.WSMessage{
		Type: models.WSTypeReaction,
		Data: models.WSReactionData{
			MessageID: messageID,
			CircleID:  circleID,
			UserID:    userID,
			Emoji:     emoji,
			Action:    action,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	ms.websocketHub.BroadcastMessage(circleID, wsMessage)
}

func (ms *MessageService) broadcastReadReceipt(userID, circleID, messageID string) {
	wsMessage := models.WSMessage{
		Type: models.WSTypeReadReceipt,
		Data: models.WSReadReceiptData{
			MessageID: messageID,
			CircleID:  circleID,
			UserID:    userID,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	ms.websocketHub.BroadcastMessage(circleID, wsMessage)
}

func (ms *MessageService) broadcastBulkReadReceipts(userID string, messageIDs []string) {
	// Group messages by circle
	messagesByCircle := make(map[string][]string)

	for _, messageID := range messageIDs {
		message, err := ms.messageRepo.GetByID(context.Background(), messageID)
		if err != nil {
			continue
		}

		circleID := message.CircleID.Hex()
		messagesByCircle[circleID] = append(messagesByCircle[circleID], messageID)
	}

	// Broadcast to each circle
	for circleID, msgIDs := range messagesByCircle {
		wsMessage := models.WSMessage{
			Type: models.WSTypeBulkReadReceipt,
			Data: models.WSBulkReadReceiptData{
				MessageIDs: msgIDs,
				CircleID:   circleID,
				UserID:     userID,
				Timestamp:  time.Now(),
			},
			Timestamp: time.Now(),
		}

		ms.websocketHub.BroadcastMessage(circleID, wsMessage)
	}
}

func (ms *MessageService) aggregateReactions(reactions []models.MessageReaction) map[string]models.ReactionSummary {
	aggregated := make(map[string]models.ReactionSummary)

	for _, reaction := range reactions {
		if summary, exists := aggregated[reaction.Emoji]; exists {
			summary.Count++
			summary.Users = append(summary.Users, reaction.UserID.Hex())
			aggregated[reaction.Emoji] = summary
		} else {
			aggregated[reaction.Emoji] = models.ReactionSummary{
				Emoji: reaction.Emoji,
				Count: 1,
				Users: []string{reaction.UserID.Hex()},
			}
		}
	}

	return aggregated
}

func (ms *MessageService) isValidMediaType(contentType string) bool {
	validTypes := []string{
		"image/jpeg", "image/png", "image/gif", "image/webp",
		"video/mp4", "video/mpeg", "video/quicktime",
		"audio/mpeg", "audio/wav", "audio/ogg",
		"application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}

	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}

	return false
}

func (ms *MessageService) checkMediaAccess(ctx context.Context, userID, mediaID string) (bool, error) {
	// Check if media is used in any message in circles the user has access to
	userCircles, err := ms.circleRepo.GetUserCircles(ctx, userID)
	if err != nil {
		return false, err
	}

	circleIDs := make([]string, len(userCircles))
	for i, circle := range userCircles {
		circleIDs[i] = circle.ID.Hex()
	}

	return ms.messageRepo.CheckMediaAccess(ctx, mediaID, circleIDs)
}

func (ms *MessageService) recordForwardHistory(originalMessageID, forwardedMessageID, userID, circleID string) {
	forward := models.MessageForward{
		OriginalMessageID:  originalMessageID,
		ForwardedMessageID: forwardedMessageID,
		ForwardedBy:        userID,
		ForwardedTo:        circleID,
		ForwardType:        "circle",
		ForwardedAt:        time.Now(),
	}

	err := ms.messageRepo.RecordForward(context.Background(), &forward)
	if err != nil {
		logrus.Errorf("Failed to record forward history: %v", err)
	}
}
func (ms *MessageService) getOrCreateDirectMessageCircle(ctx context.Context, userID1, userID2 string) (string, error) {
	// Check if DM circle already exists between these users
	// Since GetDirectMessageCircle doesn't exist, we need to implement this search manually
	userID1Obj, err := primitive.ObjectIDFromHex(userID1)
	if err != nil {
		return "", errors.New("invalid user ID 1")
	}

	userID2Obj, err := primitive.ObjectIDFromHex(userID2)
	if err != nil {
		return "", errors.New("invalid user ID 2")
	}

	// Search for existing DM circle (circle with only these two users)
	circles1, _ := ms.circleRepo.GetUserCircles(ctx, userID1)
	for _, circle := range circles1 {
		if len(circle.Members) == 2 && circle.Name == "Direct Message" {
			// Check if both users are in this circle
			foundUser1, foundUser2 := false, false
			for _, member := range circle.Members {
				if member.UserID == userID1Obj {
					foundUser1 = true
				}
				if member.UserID == userID2Obj {
					foundUser2 = true
				}
			}
			if foundUser1 && foundUser2 {
				return circle.ID.Hex(), nil
			}
		}
	}

	// Create new DM circle
	circle := models.Circle{
		Name:       "Direct Message",
		AdminID:    userID1Obj,                 // Use AdminID instead of CreatedBy
		InviteCode: utils.GenerateInviteCode(), // Generate invite code
		Settings: models.CircleSettings{
			AutoAcceptInvites:  true,
			RequireApproval:    false,
			MaxMembers:         2, // Only allow 2 members for DM
			LocationSharing:    true,
			DrivingReports:     true,
			EmergencyAlerts:    true,
			AutoCheckIn:        false,
			PlaceNotifications: false,
		},
		Stats: models.CircleStats{
			TotalMembers:  2,
			ActiveMembers: 2,
		},
		Members: []models.CircleMember{}, // Initialize empty, will add below
	}

	// Create CircleMember structs for both users
	member1 := models.CircleMember{
		UserID: userID1Obj,
		Role:   "admin", // First user is admin
		Status: "active",
		Permissions: models.MemberPermissions{
			CanSeeLocation:   true,
			CanSeeDriving:    true,
			CanSendMessages:  true,
			CanManagePlaces:  false,
			CanReceiveAlerts: true,
			CanSendEmergency: true,
		},
		JoinedAt:     time.Now(),
		LastActivity: time.Now(),
	}

	member2 := models.CircleMember{
		UserID: userID2Obj,
		Role:   "member", // Second user is member
		Status: "active",
		Permissions: models.MemberPermissions{
			CanSeeLocation:   true,
			CanSeeDriving:    true,
			CanSendMessages:  true,
			CanManagePlaces:  false,
			CanReceiveAlerts: true,
			CanSendEmergency: true,
		},
		JoinedAt:     time.Now(),
		LastActivity: time.Now(),
	}

	circle.Members = append(circle.Members, member1, member2)

	err = ms.circleRepo.Create(ctx, &circle)
	if err != nil {
		return "", err
	}

	return circle.ID.Hex(), nil
}

// Additional utility functions for automation

func (ms *MessageService) ProcessAutomationRules(ctx context.Context, message *models.Message) {
	// Get automation rules for this circle
	rules, err := ms.automationRepo.GetActiveRulesForCircle(ctx, message.CircleID.Hex())
	if err != nil {
		logrus.Errorf("Failed to get automation rules: %v", err)
		return
	}

	for _, rule := range rules {
		// Skip if rule belongs to message sender (to avoid self-triggering)
		if rule.UserID == message.SenderID {
			continue
		}

		// Check if rule conditions are met
		triggered, _, err := ms.evaluateRuleConditions(rule.Conditions, message.Content, nil)
		if err != nil {
			logrus.Errorf("Failed to evaluate rule conditions: %v", err)
			continue
		}

		if triggered {
			// Execute rule actions
			go ms.executeRuleActions(rule, message)

			// Update rule statistics
			go ms.automationRepo.IncrementTriggerCount(context.Background(), rule.ID.Hex())
		}
	}
}

func (ms *MessageService) evaluateRuleConditions(conditions []models.RuleCondition, messageContent string, context map[string]string) (bool, []string, error) {
	if len(conditions) == 0 {
		return false, nil, nil
	}

	matchedConditions := []string{}
	allMatched := true

	for _, condition := range conditions {
		matched := false

		switch condition.Type {
		case "keyword":
			keyword := condition.Value.(string)
			if !condition.CaseSensitive {
				messageContent = strings.ToLower(messageContent)
				keyword = strings.ToLower(keyword)
			}

			switch condition.Operator {
			case "contains":
				matched = strings.Contains(messageContent, keyword)
			case "equals":
				matched = messageContent == keyword
			case "starts_with":
				matched = strings.HasPrefix(messageContent, keyword)
			case "ends_with":
				matched = strings.HasSuffix(messageContent, keyword)
			}

		case "time":
			currentHour := time.Now().Hour()
			targetHour := int(condition.Value.(float64))

			switch condition.Operator {
			case "equals":
				matched = currentHour == targetHour
			case "greater_than":
				matched = currentHour > targetHour
			case "less_than":
				matched = currentHour < targetHour
			}

		case "message_type":
			// This would need the message type to be passed in context
			if context != nil {
				messageType := context["messageType"]
				matched = messageType == condition.Value.(string)
			}
		}

		if matched {
			matchedConditions = append(matchedConditions, condition.Type)
		} else {
			allMatched = false
		}
	}

	return allMatched, matchedConditions, nil
}

func (ms *MessageService) executeRuleActions(rule models.AutomationRule, triggerMessage *models.Message) {
	for _, action := range rule.Actions {
		switch action.Type {
		case "reply":
			ms.executeReplyAction(action, rule, triggerMessage)
		case "forward":
			ms.executeForwardAction(action, rule, triggerMessage)
		case "notify":
			ms.executeNotifyAction(action, rule, triggerMessage)
		case "mark_read":
			ms.executeMarkReadAction(action, rule, triggerMessage)
		}
	}
}

func (ms *MessageService) executeReplyAction(action models.RuleAction, rule models.AutomationRule, triggerMessage *models.Message) {
	replyContent := action.Config["content"].(string)

	// Replace placeholders in reply content
	replyContent = strings.ReplaceAll(replyContent, "{sender}", triggerMessage.SenderID.Hex())
	replyContent = strings.ReplaceAll(replyContent, "{time}", time.Now().Format("15:04"))

	// Send reply message
	sendReq := models.SendMessageRequest{
		CircleID: triggerMessage.CircleID.Hex(),
		Type:     "text",
		Content:  replyContent,
		ReplyTo:  triggerMessage.ID.Hex(),
	}

	_, err := ms.SendMessage(context.Background(), rule.UserID.Hex(), sendReq)
	if err != nil {
		logrus.Errorf("Failed to execute reply automation: %v", err)
	}
}

func (ms *MessageService) executeForwardAction(action models.RuleAction, rule models.AutomationRule, triggerMessage *models.Message) {
	targetCircleID := action.Config["circleId"].(string)
	comment := ""
	if commentVal, exists := action.Config["comment"]; exists {
		comment = commentVal.(string)
	}

	forwardReq := models.ForwardMessageRequest{
		Comment: comment,
	}

	_, err := ms.ForwardToCircle(context.Background(), rule.UserID.Hex(), triggerMessage.ID.Hex(), targetCircleID, forwardReq)
	if err != nil {
		logrus.Errorf("Failed to execute forward automation: %v", err)
	}
}

func (ms *MessageService) executeNotifyAction(action models.RuleAction, rule models.AutomationRule, triggerMessage *models.Message) {
	notificationMessage := action.Config["message"].(string)

	// Send notification to rule owner
	notification := map[string]interface{}{
		"type":        "automation_triggered",
		"ruleId":      rule.ID.Hex(),
		"message":     notificationMessage,
		"triggeredBy": triggerMessage.ID.Hex(),
		"timestamp":   time.Now(),
	}

	ms.websocketHub.SendNotificationToUser(rule.UserID.Hex(), notification)
}

func (ms *MessageService) executeMarkReadAction(action models.RuleAction, rule models.AutomationRule, triggerMessage *models.Message) {
	err := ms.MarkAsRead(context.Background(), rule.UserID.Hex(), triggerMessage.ID.Hex())
	if err != nil {
		logrus.Errorf("Failed to execute mark read automation: %v", err)
	}
}

func (ms *MessageService) simulateActions(actions []models.RuleAction, testMessage string, context map[string]string) []string {
	simulated := []string{}

	for _, action := range actions {
		switch action.Type {
		case "reply":
			content := action.Config["content"].(string)
			simulated = append(simulated, fmt.Sprintf("Would reply with: %s", content))
		case "forward":
			circleID := action.Config["circleId"].(string)
			simulated = append(simulated, fmt.Sprintf("Would forward to circle: %s", circleID))
		case "notify":
			message := action.Config["message"].(string)
			simulated = append(simulated, fmt.Sprintf("Would send notification: %s", message))
		case "mark_read":
			simulated = append(simulated, "Would mark message as read")
		}
	}

	return simulated
}

// Additional helper functions for processing

func (ms *MessageService) processExport(exportID string) {
	// Implementation for background export processing
	logrus.Infof("Starting export process for ID: %s", exportID)
	// This would handle the actual export logic
}

func (ms *MessageService) processImport(job *models.ImportJob, file multipart.File) {
	// Implementation for background import processing
	logrus.Infof("Starting import process for job: %s", job.ID)
}

func (ms *MessageService) isValidImportFormat(contentType string) bool {
	validTypes := []string{"application/json", "text/csv", "text/plain"}
	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}

func (ms *MessageService) calculateSeverity(reason string) string {
	switch reason {
	case "violence", "harassment":
		return "high"
	case "inappropriate", "false_information":
		return "medium"
	default:
		return "low"
	}
}

func (ms *MessageService) recordAdminAction(adminID, action, targetID, reason string) {
	// Record admin action for audit trail
	logrus.Infof("Admin action recorded: %s by %s on %s - %s", action, adminID, targetID, reason)
}

func (ms *MessageService) notifyMessageDeleted(userID, messageID, reason string) {
	// Send notification to user about deleted message
	notification := map[string]interface{}{
		"type":      "message_deleted",
		"messageId": messageID,
		"reason":    reason,
		"timestamp": time.Now(),
	}

	ms.websocketHub.SendNotificationToUser(userID, notification)
}

func (ms *MessageService) warnUser(ctx context.Context, messageID string) error {
	// Implementation for warning user
	logrus.Infof("Warning user for message: %s", messageID)
	return nil
}

// Redis helper functions (these would need to be implemented based on your Redis client)
func (ms *MessageService) setRedisKey(key, value string, expiration time.Duration) error {
	// Implementation depends on your Redis client
	logrus.Debugf("Setting Redis key: %s with expiration: %v", key, expiration)
	return nil
}

func (ms *MessageService) deleteRedisKey(key string) error {
	// Implementation depends on your Redis client
	logrus.Debugf("Deleting Redis key: %s", key)
	return nil
}

func (ms *MessageService) getRedisKeys(pattern string) ([]string, error) {
	// Implementation depends on your Redis client
	logrus.Debugf("Getting Redis keys with pattern: %s", pattern)
	return []string{}, nil
}
