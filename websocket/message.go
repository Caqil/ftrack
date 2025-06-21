package websocket

import (
	"context"
	"encoding/json"
	"ftrack/models"
	"ftrack/utils"
	"time"

	"github.com/sirupsen/logrus"
)

type MessageHandler struct {
	hub                *Hub
	locationProcessor  *LocationProcessor
	messageProcessor   *MessageProcessor
	emergencyProcessor *EmergencyProcessor
}

type LocationProcessor struct {
	hub *Hub
}

type MessageProcessor struct {
	hub *Hub
}

type EmergencyProcessor struct {
	hub *Hub
}

func NewMessageHandler(hub *Hub) *MessageHandler {
	return &MessageHandler{
		hub:                hub,
		locationProcessor:  &LocationProcessor{hub: hub},
		messageProcessor:   &MessageProcessor{hub: hub},
		emergencyProcessor: &EmergencyProcessor{hub: hub},
	}
}

// Location message processing
func (lp *LocationProcessor) ProcessLocationUpdate(userID string, data map[string]interface{}) error {
	var locationReq models.WSLocationRequest
	if err := unmarshalMapToStruct(data, &locationReq); err != nil {
		return err
	}

	// Validate location data
	if !utils.IsValidCoordinate(locationReq.Latitude, locationReq.Longitude) {
		return utils.NewValidationError("Invalid coordinates")
	}

	// Convert to internal location model
	location := models.Location{
		Latitude:     locationReq.Latitude,
		Longitude:    locationReq.Longitude,
		Accuracy:     locationReq.Accuracy,
		Speed:        locationReq.Speed,
		Bearing:      locationReq.Bearing,
		BatteryLevel: locationReq.BatteryLevel,
		IsCharging:   locationReq.IsCharging,
		IsDriving:    locationReq.IsDriving,
		IsMoving:     locationReq.IsMoving,
		MovementType: locationReq.MovementType,
		NetworkType:  locationReq.NetworkType,
		Source:       locationReq.Source,
		ServerTime:   time.Now(),
	}

	// Parse device time if provided
	if locationReq.DeviceTime != "" {
		if deviceTime, err := time.Parse(time.RFC3339, locationReq.DeviceTime); err == nil {
			location.DeviceTime = deviceTime
		} else {
			location.DeviceTime = time.Now()
		}
	} else {
		location.DeviceTime = time.Now()
	}

	// Set timezone
	location.Timezone = locationReq.Timezone
	if location.Timezone == "" {
		location.Timezone = "UTC"
	}

	// Process through location service
	if lp.hub.locationService != nil {
		go func() {
			_, err := lp.hub.locationService.UpdateLocation(context.Background(), userID, location)
			if err != nil {
				logrus.Errorf("Failed to process location update for user %s: %v", userID, err)
			}
		}()
	}

	return nil
}

// Message processing
func (mp *MessageProcessor) ProcessMessage(userID string, data map[string]interface{}) error {
	var messageReq models.WSMessageRequest
	if err := unmarshalMapToStruct(data, &messageReq); err != nil {
		return err
	}

	// Validate message data
	if messageReq.CircleID == "" {
		return utils.NewValidationError("Circle ID is required")
	}

	if messageReq.Type == "" {
		messageReq.Type = "text"
	}

	// Convert to internal message request
	req := models.SendMessageRequest{
		CircleID: messageReq.CircleID,
		Type:     messageReq.Type,
		Content:  messageReq.Content,
		Media:    messageReq.Media,
		ReplyTo:  messageReq.ReplyTo,
	}

	// Process through message service
	if mp.hub.messageService != nil {
		go func() {
			message, err := mp.hub.messageService.SendMessage(context.Background(), userID, req)
			if err != nil {
				logrus.Errorf("Failed to process message for user %s: %v", userID, err)
				return
			}

			// Broadcast message to circle members
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

			broadcastMsg := BroadcastMessage{
				RoomID:  messageReq.CircleID,
				Message: wsMessage,
			}

			select {
			case mp.hub.broadcast <- broadcastMsg:
			default:
				logrus.Warn("Broadcast channel full, dropping message")
			}
		}()
	}

	return nil
}

// Emergency processing
func (ep *EmergencyProcessor) ProcessEmergencyAlert(userID string, data map[string]interface{}) error {
	var emergencyReq models.WSEmergencyRequest
	if err := unmarshalMapToStruct(data, &emergencyReq); err != nil {
		return err
	}

	// Validate emergency data
	if emergencyReq.Type == "" {
		return utils.NewValidationError("Emergency type is required")
	}

	if !utils.IsValidCoordinate(emergencyReq.Location.Latitude, emergencyReq.Location.Longitude) {
		return utils.NewValidationError("Invalid emergency location")
	}

	// Convert to internal emergency request
	req := models.CreateEmergencyRequest{
		Type:        emergencyReq.Type,
		Description: emergencyReq.Description,
		Location:    emergencyReq.Location,
	}

	// Process through emergency service
	if ep.hub.emergencyService != nil {
		go func() {
			emergency, err := ep.hub.emergencyService.CreateEmergency(context.Background(), userID, req)
			if err != nil {
				logrus.Errorf("Failed to process emergency alert for user %s: %v", userID, err)
				return
			}

			// Get user's circles for broadcasting
			circles, err := ep.hub.circleService.GetUserCircles(context.Background(), userID)
			if err != nil {
				logrus.Errorf("Failed to get circles for emergency broadcast: %v", err)
				return
			}

			// Broadcast emergency alert to all circles
			wsAlert := models.WSEmergencyAlert{
				UserID:      userID,
				EmergencyID: emergency.ID.Hex(),
				Type:        emergency.Type,
				Title:       emergency.Title,
				Message:     emergency.Description,
				Location:    emergency.Location,
				Priority:    emergency.Priority,
				Timestamp:   emergency.CreatedAt,
			}

			var circleIDs []string
			for _, circle := range circles {
				circleIDs = append(circleIDs, circle.ID.Hex())
			}

			ep.hub.BroadcastEmergencyAlert(circleIDs, wsAlert)
		}()
	}

	return nil
}

// Message routing and validation
type MessageRouter struct {
	handlers map[string]MessageProcessor
}

func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		handlers: make(map[string]MessageProcessor),
	}
}

func (mr *MessageRouter) RegisterHandler(messageType string, processor MessageProcessor) {
	mr.handlers[messageType] = processor
}

func (mr *MessageRouter) RouteMessage(messageType, userID string, data map[string]interface{}) error {
	processor, exists := mr.handlers[messageType]
	if !exists {
		return utils.NewValidationError("Unknown message type: " + messageType)
	}

	return processor.ProcessMessage(userID, data)
}

// Message validation and transformation
type MessageValidator struct {
	rules map[string]ValidationRule
}

type ValidationRule interface {
	Validate(data map[string]interface{}) error
}

type LocationValidationRule struct{}

func (lvr *LocationValidationRule) Validate(data map[string]interface{}) error {
	lat, latOk := data["latitude"].(float64)
	lon, lonOk := data["longitude"].(float64)

	if !latOk || !lonOk {
		return utils.NewValidationError("Latitude and longitude are required")
	}

	if !utils.IsValidCoordinate(lat, lon) {
		return utils.NewValidationError("Invalid coordinates")
	}

	return nil
}

type MessageValidationRule struct{}

func (mvr *MessageValidationRule) Validate(data map[string]interface{}) error {
	circleID, ok := data["circleId"].(string)
	if !ok || circleID == "" {
		return utils.NewValidationError("Circle ID is required")
	}

	messageType, ok := data["type"].(string)
	if !ok || messageType == "" {
		return utils.NewValidationError("Message type is required")
	}

	// Validate message type
	validTypes := []string{"text", "photo", "location", "voice", "sticker", "file"}
	isValidType := false
	for _, validType := range validTypes {
		if messageType == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		return utils.NewValidationError("Invalid message type")
	}

	return nil
}

func NewMessageValidator() *MessageValidator {
	validator := &MessageValidator{
		rules: make(map[string]ValidationRule),
	}

	// Register validation rules
	validator.rules[models.WSRequestLocationUpdate] = &LocationValidationRule{}
	validator.rules[models.WSRequestSendMessage] = &MessageValidationRule{}

	return validator
}

func (mv *MessageValidator) ValidateMessage(messageType string, data map[string]interface{}) error {
	rule, exists := mv.rules[messageType]
	if !exists {
		return nil // No validation rule defined
	}

	return rule.Validate(data)
}

// Message transformation and enrichment
type MessageTransformer struct {
	enrichers map[string]MessageEnricher
}

type MessageEnricher interface {
	EnrichMessage(userID string, data map[string]interface{}) error
}

type LocationEnricher struct{}

func (le *LocationEnricher) EnrichMessage(userID string, data map[string]interface{}) error {
	// Add server timestamp
	data["serverTime"] = time.Now().Unix()

	// Add user timezone if not provided
	if _, exists := data["timezone"]; !exists {
		data["timezone"] = "UTC"
	}

	// Add location source if not provided
	if _, exists := data["source"]; !exists {
		data["source"] = models.LocationSourceGPS
	}

	return nil
}

type MessageEnricherImpl struct{}

func (me *MessageEnricherImpl) EnrichMessage(userID string, data map[string]interface{}) error {
	// Add sender ID
	data["senderId"] = userID

	// Add timestamp if not provided
	if _, exists := data["timestamp"]; !exists {
		data["timestamp"] = time.Now().Unix()
	}

	return nil
}

func NewMessageTransformer() *MessageTransformer {
	transformer := &MessageTransformer{
		enrichers: make(map[string]MessageEnricher),
	}

	// Register enrichers
	transformer.enrichers[models.WSRequestLocationUpdate] = &LocationEnricher{}
	transformer.enrichers[models.WSRequestSendMessage] = &MessageEnricherImpl{}

	return transformer
}

func (mt *MessageTransformer) TransformMessage(messageType, userID string, data map[string]interface{}) error {
	enricher, exists := mt.enrichers[messageType]
	if !exists {
		return nil // No enricher defined
	}

	return enricher.EnrichMessage(userID, data)
}

// Utility functions
func unmarshalMapToStruct(data map[string]interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}
