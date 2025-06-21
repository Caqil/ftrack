package services

import (
	"context"
	"errors"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"ftrack/websocket"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageService struct {
	messageRepo  *repositories.MessageRepository
	circleRepo   *repositories.CircleRepository
	userRepo     *repositories.UserRepository
	websocketHub *websocket.Hub
	validator    *utils.ValidationService
}

func NewMessageService(
	messageRepo *repositories.MessageRepository,
	circleRepo *repositories.CircleRepository,
	userRepo *repositories.UserRepository,
	websocketHub *websocket.Hub,
) *MessageService {
	return &MessageService{
		messageRepo:  messageRepo,
		circleRepo:   circleRepo,
		userRepo:     userRepo,
		websocketHub: websocketHub,
		validator:    utils.NewValidationService(),
	}
}

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

	// Check member permissions
	// Get circle to check permissions logic if needed

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
		CircleID: circleObjectID,
		SenderID: userObjectID,
		Type:     req.Type,
		Content:  req.Content,
		Media:    *req.Media,
		Location: *req.Location,
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

	// Broadcast message to circle members via WebSocket
	go ms.broadcastMessage(userID, req.CircleID, message)

	// Update circle last activity
	go ms.circleRepo.UpdateLastActivity(ctx, req.CircleID, userID)

	return &message, nil
}

func (ms *MessageService) GetCircleMessages(ctx context.Context, userID, circleID string, page, pageSize int) ([]models.Message, error) {
	// Check if user is a member of the circle
	isMember, err := ms.circleRepo.IsMember(ctx, circleID, userID)
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
		pageSize = 50
	}

	return ms.messageRepo.GetCircleMessages(ctx, circleID, page, pageSize)
}

func (ms *MessageService) EditMessage(ctx context.Context, userID, messageID string, req models.EditMessageRequest) (*models.Message, error) {
	// Get message
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Check if user is the sender
	if message.SenderID.Hex() != userID {
		return nil, errors.New("permission denied")
	}

	// Check message type (only text messages can be edited)
	if message.Type != "text" {
		return nil, errors.New("only text messages can be edited")
	}

	// Update message
	update := bson.M{
		"content":  req.Content,
		"isEdited": true,
		"editedAt": utils.TimePtr(time.Now()),
	}

	err = ms.messageRepo.Update(ctx, messageID, update)
	if err != nil {
		return nil, err
	}

	return ms.messageRepo.GetByID(ctx, messageID)
}

func (ms *MessageService) DeleteMessage(ctx context.Context, userID, messageID string) error {
	// Get message
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	// Check if user is the sender or circle admin
	if message.SenderID.Hex() != userID {
		// Check if user is admin of the circle
		role, err := ms.circleRepo.GetMemberRole(ctx, message.CircleID.Hex(), userID)
		if err != nil {
			return err
		}

		if role != "admin" {
			return errors.New("permission denied")
		}
	}

	return ms.messageRepo.Delete(ctx, messageID)
}

func (ms *MessageService) MarkAsRead(ctx context.Context, userID string, req models.MarkAsReadRequest) error {
	// Validate request
	if validationErrors := ms.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return errors.New("validation failed")
	}

	return ms.messageRepo.MarkAsRead(ctx, req.MessageIDs, userID)
}

func (ms *MessageService) AddReaction(ctx context.Context, userID, messageID string, req models.AddReactionRequest) error {
	// Validate request
	if validationErrors := ms.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return errors.New("validation failed")
	}

	// Get message to check circle membership
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	// Check if user is a member of the circle
	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("access denied")
	}

	return ms.messageRepo.AddReaction(ctx, messageID, userID, req.Emoji)
}

func (ms *MessageService) RemoveReaction(ctx context.Context, userID, messageID string) error {
	// Get message to check circle membership
	message, err := ms.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	// Check if user is a member of the circle
	isMember, err := ms.circleRepo.IsMember(ctx, message.CircleID.Hex(), userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("access denied")
	}

	return ms.messageRepo.RemoveReaction(ctx, messageID, userID)
}

func (ms *MessageService) GetUnreadCount(ctx context.Context, userID, circleID string) (int64, error) {
	// Check if user is a member of the circle
	isMember, err := ms.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return 0, err
	}

	if !isMember {
		return 0, errors.New("access denied")
	}

	return ms.messageRepo.GetUnreadCount(ctx, circleID, userID)
}

func (ms *MessageService) broadcastMessage(senderID, circleID string, message models.Message) {
	wsMessage := models.WSMessage{
		Type:     websocket.TypeMessage,
		Data:     message,
		CircleID: circleID,
	}

	broadcastMsg := websocket.BroadcastMessage{
		CircleID: circleID,
		Message:  wsMessage,
	}

	// Send to WebSocket hub
	ms.websocketHub.broadcast <- broadcastMsg
}
