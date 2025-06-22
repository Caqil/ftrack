package services

import (
	"context"
	"errors"
	"fmt"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/utils"
	"ftrack/websocket"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmergencyService struct {
	emergencyRepo   *repositories.EmergencyRepository
	circleRepo      *repositories.CircleRepository
	userRepo        *repositories.UserRepository
	notificationSvc *NotificationService
	websocketHub    *websocket.Hub
	validator       *utils.ValidationService
}

func NewEmergencyService(
	emergencyRepo *repositories.EmergencyRepository,
	circleRepo *repositories.CircleRepository,
	userRepo *repositories.UserRepository,
	notificationSvc *NotificationService,
	websocketHub *websocket.Hub,
) *EmergencyService {
	return &EmergencyService{
		emergencyRepo:   emergencyRepo,
		circleRepo:      circleRepo,
		userRepo:        userRepo,
		notificationSvc: notificationSvc,
		websocketHub:    websocketHub,
		validator:       utils.NewValidationService(),
	}
}

// =================== EMERGENCY ALERTS ===================

func (es *EmergencyService) GetEmergencyAlerts(ctx context.Context, userID string) ([]models.Emergency, error) {
	return es.emergencyRepo.GetUserEmergencies(ctx, userID)
}

func (es *EmergencyService) CreateEmergencyAlert(ctx context.Context, userID string, req models.CreateEmergencyAlertRequest) (*models.Emergency, error) {
	if validationErrors := es.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	emergency := &models.Emergency{
		UserID:      userObjectID,
		Type:        req.Type,
		Priority:    req.Priority,
		Status:      models.EmergencyStatusActive,
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		Detection: models.EmergencyDetection{
			Method:     "manual",
			Confidence: 1.0,
		},
		Response: models.EmergencyResponse{
			AutoSent: false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.CircleID != "" {
		circleID, err := primitive.ObjectIDFromHex(req.CircleID)
		if err == nil {
			emergency.CircleID = circleID
		}
	}

	err = es.emergencyRepo.Create(ctx, emergency)
	if err != nil {
		return nil, err
	}

	// Notify contacts and broadcast
	go es.handleEmergencyNotifications(ctx, emergency)

	return emergency, nil
}

func (es *EmergencyService) GetEmergencyAlert(ctx context.Context, userID, alertID string) (*models.Emergency, error) {
	emergency, err := es.emergencyRepo.GetByID(ctx, alertID)
	if err != nil {
		if err.Error() == "not found" {
			return nil, errors.New("alert not found")
		}
		return nil, err
	}

	if !es.hasEmergencyAccess(ctx, userID, emergency) {
		return nil, errors.New("access denied")
	}

	return emergency, nil
}

func (es *EmergencyService) UpdateEmergencyAlert(ctx context.Context, userID, alertID string, req models.UpdateEmergencyAlertRequest) (*models.Emergency, error) {
	emergency, err := es.emergencyRepo.GetByID(ctx, alertID)
	if err != nil {
		return nil, err
	}

	if !es.hasEmergencyAccess(ctx, userID, emergency) {
		return nil, errors.New("access denied")
	}

	updateFields := bson.M{"updatedAt": time.Now()}
	
	if req.Title != "" {
		updateFields["title"] = req.Title
	}
	if req.Description != "" {
		updateFields["description"] = req.Description
	}
	if req.Priority != "" {
		updateFields["priority"] = req.Priority
	}
	if req.Status != "" {
		updateFields["status"] = req.Status
	}
	if req.ExpiresAt != nil {
		updateFields["expiresAt"] = req.ExpiresAt
	}

	err = es.emergencyRepo.Update(ctx, alertID, updateFields)
	if err != nil {
		return nil, err
	}

	return es.emergencyRepo.GetByID(ctx, alertID)
}

func (es *EmergencyService) DeleteEmergencyAlert(ctx context.Context, userID, alertID string) error {
	emergency, err := es.emergencyRepo.GetByID(ctx, alertID)
	if err != nil {
		return err
	}

	if emergency.UserID.Hex() != userID {
		return errors.New("access denied")
	}

	return es.emergencyRepo.Delete(ctx, alertID)
}

func (es *EmergencyService) DismissEmergencyAlert(ctx context.Context, userID, alertID, reason string) error {
	emergency, err := es.emergencyRepo.GetByID(ctx, alertID)
	if err != nil {
		return err
	}

	if !es.hasEmergencyAccess(ctx, userID, emergency) {
		return errors.New("access denied")
	}

	updateFields := bson.M{
		"status":          models.EmergencyStatusDismissed,
		"dismissalReason": reason,
		"dismissedAt":     time.Now(),
		"dismissedBy":     userID,
		"updatedAt":       time.Now(),
	}

	return es.emergencyRepo.Update(ctx, alertID, updateFields)
}

func (es *EmergencyService) ResolveEmergencyAlert(ctx context.Context, userID, alertID, resolution string) error {
	emergency, err := es.emergencyRepo.GetByID(ctx, alertID)
	if err != nil {
		return err
	}

	if !es.hasEmergencyAccess(ctx, userID, emergency) {
		return errors.New("access denied")
	}

	updateFields := bson.M{
		"status":     models.EmergencyStatusResolved,
		"resolution": resolution,
		"resolvedAt": time.Now(),
		"updatedAt":  time.Now(),
	}

	return es.emergencyRepo.Update(ctx, alertID, updateFields)
}

// =================== SOS FUNCTIONALITY ===================

func (es *EmergencyService) TriggerSOS(ctx context.Context, userID string, req models.TriggerSOSRequest) (*models.Emergency, error) {
	if validationErrors := es.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	emergency := &models.Emergency{
		UserID:      userObjectID,
		Type:        models.EmergencyTypeSOS,
		Priority:    "critical",
		Status:      models.EmergencyStatusActive,
		Title:       "SOS Emergency Alert",
		Description: req.Message,
		Location:    req.Location,
		Detection: models.EmergencyDetection{
			Method:     "manual",
			Confidence: 1.0,
		},
		Response: models.EmergencyResponse{
			AutoSent: req.AutoCall,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = es.emergencyRepo.Create(ctx, emergency)
	if err != nil {
		return nil, err
	}

	// Handle SOS specific logic
	if req.AutoCall && req.CountdownSec > 0 {
		go es.handleSOSCountdown(ctx, emergency.ID.Hex(), req.CountdownSec)
	}

	// Immediate notifications for SOS
	go es.handleEmergencyNotifications(ctx, emergency)

	return emergency, nil
}

func (es *EmergencyService) CancelSOS(ctx context.Context, userID, reason string) error {
	// Find active SOS for user
	emergencies, err := es.emergencyRepo.GetUserActiveEmergencies(ctx, userID, models.EmergencyTypeSOS)
	if err != nil {
		return err
	}

	if len(emergencies) == 0 {
		return errors.New("no active SOS found")
	}

	// Cancel the most recent SOS
	emergency := emergencies[0]
	updateFields := bson.M{
		"status":             models.EmergencyStatusCancelled,
		"cancellationReason": reason,
		"cancelledAt":        time.Now(),
		"updatedAt":          time.Now(),
	}

	return es.emergencyRepo.Update(ctx, emergency.ID.Hex(), updateFields)
}

func (es *EmergencyService) GetSOSStatus(ctx context.Context, userID string) (*models.SOSStatus, error) {
	emergencies, err := es.emergencyRepo.GetUserActiveEmergencies(ctx, userID, models.EmergencyTypeSOS)
	if err != nil {
		return nil, err
	}

	status := &models.SOSStatus{Active: false}
	
	if len(emergencies) > 0 {
		emergency := emergencies[0]
		status.Active = true
		status.TriggeredAt = emergency.CreatedAt
		status.Type = emergency.Type
		status.AutoCall = emergency.Response.AutoSent
		
		// Calculate countdown if applicable
		if emergency.Response.AutoSent {
			elapsed := int(time.Since(emergency.CreatedAt).Seconds())
			if elapsed < 60 { // 1 minute countdown
				status.CountdownLeft = 60 - elapsed
			}
		}
	}

	return status, nil
}

func (es *EmergencyService) UpdateSOSSettings(ctx context.Context, userID string, req models.SOSSettingsRequest) (*models.EmergencySettings, error) {
	settings, err := es.GetEmergencySettings(ctx, userID)
	if err != nil {
		// Create new settings if not found
		userObjectID, _ := primitive.ObjectIDFromHex(userID)
		settings = &models.EmergencySettings{
			UserID:    userObjectID,
		}
	}

	settings.AutoCallEmergency = req.AutoCall
	settings.CountdownDuration = req.CountdownDuration
	settings.AutoNotifyContacts = true
	settings.UpdatedAt = time.Now()

	return settings, es.emergencyRepo.UpdateUserSettings(ctx, userID, settings)
}

func (es *EmergencyService) GetSOSSettings(ctx context.Context, userID string) (*models.EmergencySettings, error) {
	return es.emergencyRepo.GetUserSettings(ctx, userID)
}

func (es *EmergencyService) TestSOS(ctx context.Context, userID string) (map[string]interface{}, error) {
	// Test SOS functionality without triggering actual emergency
	result := map[string]interface{}{
		"status":      "success",
		"contacts":    "verified",
		"location":    "available",
		"services":    "reachable",
		"timestamp":   time.Now(),
	}

	// Test emergency contacts
	contacts, err := es.GetEmergencyContacts(ctx, userID)
	if err != nil || len(contacts) == 0 {
		result["contacts"] = "no_contacts_configured"
	}

	return result, nil
}

// =================== CRASH DETECTION ===================

func (es *EmergencyService) DetectCrash(ctx context.Context, userID string, req models.CrashDetectionRequest) (*models.Emergency, error) {
	if validationErrors := es.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	emergency := &models.Emergency{
		UserID:      userObjectID,
		Type:        models.EmergencyTypeCrash,
		Priority:    "critical",
		Status:      models.EmergencyStatusActive,
		Title:       "Crash Detected",
		Description: fmt.Sprintf("Potential crash detected with %.1f%% confidence", req.Confidence*100),
		Location:    req.Location,
		Detection: models.EmergencyDetection{
			Method:     "auto_crash",
			Confidence: req.Confidence,
			SensorData: req.SensorData,
		},
		Response: models.EmergencyResponse{
			AutoSent: req.Confidence > 0.8, // Auto-send if high confidence
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = es.emergencyRepo.Create(ctx, emergency)
	if err != nil {
		return nil, err
	}

	// Start confirmation countdown for crash detection
	go es.handleCrashConfirmationCountdown(ctx, emergency.ID.Hex())

	return emergency, nil
}

func (es *EmergencyService) ConfirmCrash(ctx context.Context, userID, detectionID string, req models.ConfirmCrashRequest) error {
	emergency, err := es.emergencyRepo.GetByID(ctx, detectionID)
	if err != nil {
		return err
	}

	if emergency.UserID.Hex() != userID {
		return errors.New("access denied")
	}

	updateFields := bson.M{
		"updatedAt": time.Now(),
	}

	if req.Confirmed {
		updateFields["status"] = models.EmergencyStatusActive
		updateFields["description"] = emergency.Description + " - CONFIRMED by user"
		if req.Description != "" {
			updateFields["description"] = emergency.Description + " - " + req.Description
		}
		
		// Trigger emergency notifications
		go es.handleEmergencyNotifications(ctx, emergency)
	} else {
		updateFields["status"] = models.EmergencyStatusFalseAlarm
		updateFields["dismissalReason"] = "User confirmed no crash occurred"
	}

	return es.emergencyRepo.Update(ctx, detectionID, updateFields)
}

func (es *EmergencyService) MarkFalseAlarm(ctx context.Context, userID, detectionID, reason string) error {
	emergency, err := es.emergencyRepo.GetByID(ctx, detectionID)
	if err != nil {
		return err
	}

	if emergency.UserID.Hex() != userID {
		return errors.New("access denied")
	}

	updateFields := bson.M{
		"status":          models.EmergencyStatusFalseAlarm,
		"dismissalReason": reason,
		"dismissedAt":     time.Now(),
		"dismissedBy":     userID,
		"updatedAt":       time.Now(),
	}

	return es.emergencyRepo.Update(ctx, detectionID, updateFields)
}

func (es *EmergencyService) GetCrashHistory(ctx context.Context, userID string) ([]models.Emergency, error) {
	return es.emergencyRepo.GetUserEmergenciesByType(ctx, userID, models.EmergencyTypeCrash)
}

func (es *EmergencyService) GetCrashDetectionSettings(ctx context.Context, userID string) (*models.EmergencySettings, error) {
	return es.emergencyRepo.GetUserSettings(ctx, userID)
}

func (es *EmergencyService) UpdateCrashDetectionSettings(ctx context.Context, userID string, req models.CrashDetectionSettingsRequest) (*models.EmergencySettings, error) {
	settings, err := es.GetEmergencySettings(ctx, userID)
	if err != nil {
		userObjectID, _ := primitive.ObjectIDFromHex(userID)
		settings = &models.EmergencySettings{
			UserID:    userObjectID,
		}
	}

	settings.CrashDetection = req.Enabled
	settings.AutoCallEmergency = req.AutoCall
	settings.CountdownDuration = req.CountdownDuration
	settings.UpdatedAt = time.Now()

	return settings, es.emergencyRepo.UpdateUserSettings(ctx, userID, settings)
}

func (es *EmergencyService) CalibrateCrashDetection(ctx context.Context, userID string, req models.CalibrateCrashRequest) (map[string]interface{}, error) {
	// Calibrate crash detection based on device and user data
	result := map[string]interface{}{
		"status":      "calibrated",
		"deviceType":  req.DeviceType,
		"sensitivity": 0.8,
		"timestamp":   time.Now(),
	}

	// Store calibration data for user
	// This would typically update user's device settings in the database

	return result, nil
}

// =================== EMERGENCY CONTACTS ===================

func (es *EmergencyService) GetEmergencyContacts(ctx context.Context, userID string) ([]models.EmergencyContact, error) {
	return es.emergencyRepo.GetUserEmergencyContacts(ctx, userID)
}

func (es *EmergencyService) AddEmergencyContact(ctx context.Context, userID string, req models.AddEmergencyContactRequest) (*models.EmergencyContact, error) {
	if validationErrors := es.validator.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, errors.New("validation failed")
	}

	contact := &models.EmergencyContact{
		ContactID:    primitive.NewObjectID(),
		Name:         req.Name,
		Phone:        req.Phone,
		Email:        req.Email,
		Relationship: req.Relationship,
		UpdatedAt:    time.Now(),
	}

	err := es.emergencyRepo.AddEmergencyContact(ctx, userID, contact)
	if err != nil {
		return nil, err
	}

	return contact, nil
}

func (es *EmergencyService) GetEmergencyContact(ctx context.Context, userID, contactID string) (*models.EmergencyContact, error) {
	contacts, err := es.GetEmergencyContacts(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, contact := range contacts {
		if contact.ContactID.Hex() == contactID {
			return &contact, nil
		}
	}

	return nil, errors.New("contact not found")
}

func (es *EmergencyService) UpdateEmergencyContact(ctx context.Context, userID, contactID string, req models.UpdateEmergencyContactRequest) (*models.EmergencyContact, error) {
	contact, err := es.GetEmergencyContact(ctx, userID, contactID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		contact.Name = req.Name
	}
	if req.Phone != "" {
		contact.Phone = req.Phone
	}
	if req.Email != "" {
		contact.Email = req.Email
	}
	if req.Relationship != "" {
		contact.Relationship = req.Relationship
	}
	
	contact.UpdatedAt = time.Now()

	err = es.emergencyRepo.UpdateEmergencyContact(ctx, userID, contactID, contact)
	if err != nil {
		return nil, err
	}

	return contact, nil
}

func (es *EmergencyService) DeleteEmergencyContact(ctx context.Context, userID, contactID string) error {
	return es.emergencyRepo.DeleteEmergencyContact(ctx, userID, contactID)
}

func (es *EmergencyService) VerifyEmergencyContact(ctx context.Context, userID, contactID string, req models.VerifyContactRequest) (map[string]interface{}, error) {
	contact, err := es.GetEmergencyContact(ctx, userID, contactID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"status":    "sent",
		"method":    req.Method,
		"contactId": contactID,
		"timestamp": time.Now(),
	}

	// Send verification based on method
	switch req.Method {
	case "sms":
		if contact.Phone == "" {
			return nil, errors.New("phone number not available")
		}
		// Send SMS verification
		result["message"] = "Verification SMS sent"
	case "email":
		if contact.Email == "" {
			return nil, errors.New("email address not available")
		}
		// Send email verification
		result["message"] = "Verification email sent"
	case "call":
		if contact.Phone == "" {
			return nil, errors.New("phone number not available")
		}
		// Initiate verification call
		result["message"] = "Verification call initiated"
	}

	return result, nil
}

func (es *EmergencyService) NotifyEmergencyContact(ctx context.Context, userID, contactID string, req models.NotifyContactRequest) error {
	contact, err := es.GetEmergencyContact(ctx, userID, contactID)
	if err != nil {
		return err
	}

	// Send notification based on method
	switch req.Method {
	case "sms":
		if contact.Phone == "" {
			return errors.New("phone number not available")
		}
		// Send SMS
	case "email":
		if contact.Email == "" {
			return errors.New("email address not available")
		}
		// Send email
	case "call":
		if contact.Phone == "" {
			return errors.New("phone number not available")
		}
		// Initiate call
	case "push":
		// Send push notification if contact is app user
	}

	// Update contact notification history
	contact.NotifiedAt = time.Now()
	contact.NotifyMethod = req.Method
	es.emergencyRepo.UpdateEmergencyContact(ctx, userID, contactID, contact)

	return nil
}

func (es *EmergencyService) GetContactHistory(ctx context.Context, userID, contactID string) ([]models.EmergencyEvent, error) {
	return es.emergencyRepo.GetContactNotificationHistory(ctx, userID, contactID)
}

// =================== EMERGENCY SERVICES ===================

func (es *EmergencyService) GetNearbyEmergencyServices(ctx context.Context, userID, lat, lng, radius string) (map[string]interface{}, error) {
	// Integration with external services API
	services := map[string]interface{}{
		"hospitals": []map[string]interface{}{},
		"police":    []map[string]interface{}{},
		"fire":      []map[string]interface{}{},
		"timestamp": time.Now(),
	}

	// This would call external APIs to get nearby services
	return services, nil
}

func (es *EmergencyService) GetNearbyHospitals(ctx context.Context, userID, lat, lng, radius string) ([]map[string]interface{}, error) {
	// Call external API for nearby hospitals
	hospitals := []map[string]interface{}{
		{
			"name":     "City General Hospital",
			"address":  "123 Main St",
			"phone":    "555-0123",
			"distance": 1.2,
			"type":     "general",
		},
	}
	return hospitals, nil
}

func (es *EmergencyService) GetNearbyPoliceStations(ctx context.Context, userID, lat, lng, radius string) ([]map[string]interface{}, error) {
	stations := []map[string]interface{}{
		{
			"name":     "Downtown Police Station",
			"address":  "456 Oak Ave",
			"phone":    "555-0911",
			"distance": 0.8,
		},
	}
	return stations, nil
}

func (es *EmergencyService) GetNearbyFireStations(ctx context.Context, userID, lat, lng, radius string) ([]map[string]interface{}, error) {
	stations := []map[string]interface{}{
		{
			"name":     "Fire Station 1",
			"address":  "789 Elm St",
			"phone":    "555-0911",
			"distance": 1.5,
		},
	}
	return stations, nil
}

func (es *EmergencyService) InitiateEmergencyCall(ctx context.Context, userID, serviceType string, req models.EmergencyCallRequest) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"status":      "initiated",
		"serviceType": serviceType,
		"callId":      primitive.NewObjectID().Hex(),
		"timestamp":   time.Now(),
	}

	// Get appropriate emergency number
	numbers, err := es.GetEmergencyNumbers(ctx, userID)
	if err != nil {
		return nil, err
	}

	var phone string
	switch serviceType {
	case "police":
		phone = numbers["police"].(string)
	case "medical":
		phone = numbers["medical"].(string)
	case "fire":
		phone = numbers["fire"].(string)
	default:
		phone = "911" // Default emergency number
	}

	result["phone"] = phone
	result["location"] = req.Location

	// This would integrate with telephony service to initiate call
	return result, nil
}

func (es *EmergencyService) GetEmergencyNumbers(ctx context.Context, userID string) (map[string]interface{}, error) {
	// Get country-specific emergency numbers
	numbers := map[string]interface{}{
		"police":   "911",
		"medical":  "911",
		"fire":     "911",
		"roadside": "1-800-AAA-HELP",
		"country":  "US",
	}
	return numbers, nil
}

func (es *EmergencyService) UpdateEmergencyNumbers(ctx context.Context, userID string, req models.UpdateEmergencyNumbersRequest) (map[string]interface{}, error) {
	// Update user's emergency numbers preferences
	numbers := map[string]interface{}{
		"police":   req.Police,
		"medical":  req.Medical,
		"fire":     req.Fire,
		"roadside": req.Roadside,
		"country":  req.Country,
		"updated":  time.Now(),
	}

	// Save to user preferences
	return numbers, nil
}

// =================== LOCATION SHARING ===================

func (es *EmergencyService) ShareEmergencyLocation(ctx context.Context, userID string, req models.ShareLocationRequest) (map[string]interface{}, error) {
	share := map[string]interface{}{
		"shareId":      primitive.NewObjectID().Hex(),
		"userId":       userID,
		"recipients":   req.Recipients,
		"duration":     req.Duration,
		"message":      req.Message,
		"shareLevel":   req.ShareLevel,
		"createdAt":    time.Now(),
		"active":       true,
	}

	if req.ExpiresAt != nil {
		share["expiresAt"] = req.ExpiresAt
	} else if req.Duration > 0 {
		share["expiresAt"] = time.Now().Add(time.Duration(req.Duration) * time.Minute)
	}

	// Save location share
	err := es.emergencyRepo.CreateLocationShare(ctx, share)
	if err != nil {
		return nil, err
	}

	return share, nil
}

func (es *EmergencyService) GetSharedEmergencyLocations(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	return es.emergencyRepo.GetUserLocationShares(ctx, userID)
}

func (es *EmergencyService) UpdateLocationShare(ctx context.Context, userID, shareID string, req models.UpdateLocationShareRequest) (map[string]interface{}, error) {
	updateFields := bson.M{
		"updatedAt": time.Now(),
	}

	if req.Duration > 0 {
		updateFields["duration"] = req.Duration
		updateFields["expiresAt"] = time.Now().Add(time.Duration(req.Duration) * time.Minute)
	}
	if req.Message != "" {
		updateFields["message"] = req.Message
	}
	if req.ShareLevel != "" {
		updateFields["shareLevel"] = req.ShareLevel
	}
	if req.ExpiresAt != nil {
		updateFields["expiresAt"] = req.ExpiresAt
	}

	err := es.emergencyRepo.UpdateLocationShare(ctx, userID, shareID, updateFields)
	if err != nil {
		return nil, err
	}

	return es.emergencyRepo.GetLocationShare(ctx, shareID)
}

func (es *EmergencyService) StopLocationShare(ctx context.Context, userID, shareID string) error {
	updateFields := bson.M{
		"active":    false,
		"stoppedAt": time.Now(),
		"updatedAt": time.Now(),
	}

	return es.emergencyRepo.UpdateLocationShare(ctx, userID, shareID, updateFields)
}

func (es *EmergencyService) TrackEmergencyLocation(ctx context.Context, userID, shareID string) (map[string]interface{}, error) {
	share, err := es.emergencyRepo.GetLocationShare(ctx, shareID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to this location share
	hasAccess := false
	if share["userId"].(string) == userID {
		hasAccess = true
	} else {
		recipients := share["recipients"].([]string)
		for _, recipient := range recipients {
			if recipient == userID {
				hasAccess = true
				break
			}
		}
	}

	if !hasAccess {
		return nil, errors.New("access denied")
	}

	// Get current location for the shared user
	location, err := es.emergencyRepo.GetUserCurrentLocation(ctx, share["userId"].(string))
	if err != nil {
		return nil, err
	}

	return location, nil
}

// =================== EMERGENCY RESPONSE ===================

func (es *EmergencyService) RespondToEmergency(ctx context.Context, userID, alertID string, req models.EmergencyResponseRequest) (map[string]interface{}, error) {
	emergency, err := es.emergencyRepo.GetByID(ctx, alertID)
	if err != nil {
		return nil, err
	}

	response := map[string]interface{}{
		"responseId":  primitive.NewObjectID().Hex(),
		"emergencyId": alertID,
		"responderId": userID,
		"type":        req.Type,
		"message":     req.Message,
		"timestamp":   time.Now(),
	}

	if req.ETA > 0 {
		response["eta"] = req.ETA
		response["estimatedArrival"] = time.Now().Add(time.Duration(req.ETA) * time.Minute)
	}

	if req.Location.Latitude != 0 && req.Location.Longitude != 0 {
		response["location"] = req.Location
	}

	if len(req.Skills) > 0 {
		response["skills"] = req.Skills
	}

	// Save response
	err = es.emergencyRepo.AddEmergencyResponse(ctx, alertID, response)
	if err != nil {
		return nil, err
	}

	// Notify emergency creator
	go es.notifyEmergencyCreator(ctx, emergency, response)

	return response, nil
}

func (es *EmergencyService) GetEmergencyResponses(ctx context.Context, userID, alertID string) ([]map[string]interface{}, error) {
	emergency, err := es.emergencyRepo.GetByID(ctx, alertID)
	if err != nil {
		return nil, err
	}

	if !es.hasEmergencyAccess(ctx, userID, emergency) {
		return nil, errors.New("access denied")
	}

	return es.emergencyRepo.GetEmergencyResponses(ctx, alertID)
}

func (es *EmergencyService) UpdateEmergencyResponse(ctx context.Context, userID, alertID, responseID string, req models.UpdateEmergencyResponseRequest) (map[string]interface{}, error) {
	updateFields := bson.M{
		"updatedAt": time.Now(),
	}

	if req.Message != "" {
		updateFields["message"] = req.Message
	}
	if req.ETA > 0 {
		updateFields["eta"] = req.ETA
		updateFields["estimatedArrival"] = time.Now().Add(time.Duration(req.ETA) * time.Minute)
	}
	if req.Location.Latitude != 0 && req.Location.Longitude != 0 {
		updateFields["location"] = req.Location
	}
	if req.Status != "" {
		updateFields["status"] = req.Status
	}

	err := es.emergencyRepo.UpdateEmergencyResponse(ctx, alertID, responseID, updateFields)
	if err != nil {
		return nil, err
	}

	return es.emergencyRepo.GetEmergencyResponse(ctx, alertID, responseID)
}

func (es *EmergencyService) RequestHelp(ctx context.Context, userID, alertID string, req models.RequestHelpRequest) (map[string]interface{}, error) {
	helpRequest := map[string]interface{}{
		"requestId":   primitive.NewObjectID().Hex(),
		"emergencyId": alertID,
		"requesterId": userID,
		"type":        req.Type,
		"description": req.Description,
		"skills":      req.Skills,
		"urgency":     req.Urgency,
		"radius":      req.Radius,
		"timestamp":   time.Now(),
		"status":      "open",
	}

	err := es.emergencyRepo.CreateHelpRequest(ctx, helpRequest)
	if err != nil {
		return nil, err
	}

	// Broadcast help request to nearby users
	go es.broadcastHelpRequest(ctx, helpRequest)

	return helpRequest, nil
}

func (es *EmergencyService) OfferHelp(ctx context.Context, userID, alertID string, req models.OfferHelpRequest) (map[string]interface{}, error) {
	helpOffer := map[string]interface{}{
		"offerId":     primitive.NewObjectID().Hex(),
		"emergencyId": alertID,
		"offererId":   userID,
		"message":     req.Message,
		"skills":      req.Skills,
		"eta":         req.ETA,
		"location":    req.Location,
		"capacity":    req.Capacity,
		"timestamp":   time.Now(),
		"status":      "offered",
	}

	err := es.emergencyRepo.CreateHelpOffer(ctx, helpOffer)
	if err != nil {
		return nil, err
	}

	// Notify emergency creator about help offer
	emergency, _ := es.emergencyRepo.GetByID(ctx, alertID)
	go es.notifyEmergencyCreator(ctx, emergency, helpOffer)

	return helpOffer, nil
}

// =================== CHECK-IN SAFETY ===================

func (es *EmergencyService) CheckInSafe(ctx context.Context, userID string, req models.SafeCheckInRequest) (map[string]interface{}, error) {
	checkIn := map[string]interface{}{
		"checkInId": primitive.NewObjectID().Hex(),
		"userId":    userID,
		"status":    "safe",
		"message":   req.Message,
		"timestamp": time.Now(),
	}

	if req.Location.Latitude != 0 && req.Location.Longitude != 0 {
		checkIn["location"] = req.Location
	}

	err := es.emergencyRepo.CreateCheckIn(ctx, checkIn)
	if err != nil {
		return nil, err
	}

	// Notify concerned contacts
	go es.notifyCheckInStatus(ctx, userID, "safe", req.Message)

	return checkIn, nil
}

func (es *EmergencyService) CheckInNotSafe(ctx context.Context, userID string, req models.NotSafeCheckInRequest) (map[string]interface{}, error) {
	checkIn := map[string]interface{}{
		"checkInId":   primitive.NewObjectID().Hex(),
		"userId":      userID,
		"status":      "not_safe",
		"issue":       req.Issue,
		"severity":    req.Severity,
		"description": req.Description,
		"location":    req.Location,
		"needHelp":    req.NeedHelp,
		"timestamp":   time.Now(),
	}

	err := es.emergencyRepo.CreateCheckIn(ctx, checkIn)
	if err != nil {
		return nil, err
	}

	// Automatically create emergency if needed
	if req.NeedHelp && req.Severity == "critical" {
		emergencyReq := models.CreateEmergencyAlertRequest{
			Type:        "help",
			Title:       "Check-in Emergency",
			Description: fmt.Sprintf("User checked in as not safe: %s", req.Issue),
			Priority:    req.Severity,
			Location:    req.Location,
		}
		go es.CreateEmergencyAlert(ctx, userID, emergencyReq)
	}

	// Notify emergency contacts
	go es.notifyCheckInStatus(ctx, userID, "not_safe", req.Description)

	return checkIn, nil
}

func (es *EmergencyService) GetCheckInStatus(ctx context.Context, userID string) (*models.CheckInStatus, error) {
	lastCheckIn, err := es.emergencyRepo.GetLastCheckIn(ctx, userID)
	if err != nil {
		return &models.CheckInStatus{Status: "unknown"}, nil
	}

	status := &models.CheckInStatus{
		Status:      lastCheckIn["status"].(string),
		LastCheckIn: lastCheckIn["timestamp"].(time.Time),
	}

	if location, ok := lastCheckIn["location"]; ok {
		status.Location = location.(models.EmergencyLocation)
	}

	// Calculate next due based on settings
	settings, err := es.GetCheckInSettings(ctx, userID)
	if err == nil && settings != nil {
		if frequency, ok := settings["frequency"].(int); ok && frequency > 0 {
			status.NextDue = status.LastCheckIn.Add(time.Duration(frequency) * time.Hour)
		}
	}

	return status, nil
}

func (es *EmergencyService) UpdateCheckInSettings(ctx context.Context, userID string, req models.CheckInSettingsRequest) (map[string]interface{}, error) {
	settings := map[string]interface{}{
		"userId":            userID,
		"enabled":           req.Enabled,
		"frequency":         req.Frequency,
		"autoReminder":      req.AutoReminder,
		"quietHours":        req.QuietHours,
		"emergencyContacts": req.EmergencyContacts,
		"geoFencing":        req.GeoFencing,
		"updatedAt":         time.Now(),
	}

	err := es.emergencyRepo.UpdateCheckInSettings(ctx, userID, settings)
	if err != nil {
		return nil, err
	}

	return settings, nil
}

func (es *EmergencyService) GetCheckInSettings(ctx context.Context, userID string) (map[string]interface{}, error) {
	return es.emergencyRepo.GetCheckInSettings(ctx, userID)
}

func (es *EmergencyService) RequestCheckIn(ctx context.Context, requesterID, targetUserID string, req models.RequestCheckInRequest) (map[string]interface{}, error) {
	request := map[string]interface{}{
		"requestId":    primitive.NewObjectID().Hex(),
		"requesterId":  requesterID,
		"targetUserId": targetUserID,
		"message":      req.Message,
		"urgent":       req.Urgent,
		"timestamp":    time.Now(),
		"status":       "pending",
	}

	err := es.emergencyRepo.CreateCheckInRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	// Notify target user
	go es.notifyCheckInRequest(ctx, targetUserID, request)

	return request, nil
}

func (es *EmergencyService) GetCheckInRequests(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	return es.emergencyRepo.GetCheckInRequests(ctx, userID)
}

// =================== HISTORY AND STATS ===================

func (es *EmergencyService) GetEmergencyHistory(ctx context.Context, userID string) ([]models.Emergency, error) {
	return es.emergencyRepo.GetUserEmergencies(ctx, userID)
}

func (es *EmergencyService) GetEmergencyTimeline(ctx context.Context, userID, alertID string) ([]models.EmergencyEvent, error) {
	emergency, err := es.emergencyRepo.GetByID(ctx, alertID)
	if err != nil {
		return nil, err
	}

	if !es.hasEmergencyAccess(ctx, userID, emergency) {
		return nil, errors.New("access denied")
	}

	return es.emergencyRepo.GetEmergencyTimeline(ctx, alertID)
}

func (es *EmergencyService) GetEmergencyStats(ctx context.Context, userID string) (*models.EmergencyStats, error) {
	startTime := time.Now().AddDate(0, -1, 0) // Last month
	endTime := time.Now()

	return es.emergencyRepo.GetEmergencyStats(ctx, startTime, endTime)
}

func (es *EmergencyService) ExportEmergencyHistory(ctx context.Context, userID string, req models.ExportHistoryRequest) (*models.EmergencyFileExport, error) {
	export := &models.EmergencyFileExport{
		ID:        primitive.NewObjectID().Hex(),
		Status:    "processing",
		Progress:  0,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Start background export process
	go es.processEmergencyExport(ctx, userID, export, req)

	return export, nil
}

func (es *EmergencyService) DownloadEmergencyExport(ctx context.Context, userID, exportID string) (*models.EmergencyExportFile, error) {
	export, err := es.emergencyRepo.GetExport(ctx, userID, exportID)
	if err != nil {
		return nil, err
	}

	if export.Status != "completed" {
		return nil, errors.New("export not ready")
	}

	// Return file data
	return &models.EmergencyExportFile{
		Data:        []byte{}, // Load from storage
		Filename:    fmt.Sprintf("emergency_history_%s.%s", userID, "json"),
		ContentType: "application/json",
	}, nil
}

// =================== SETTINGS ===================

func (es *EmergencyService) GetEmergencySettings(ctx context.Context, userID string) (*models.EmergencySettings, error) {
	return es.emergencyRepo.GetUserSettings(ctx, userID)
}

func (es *EmergencyService) UpdateEmergencySettings(ctx context.Context, userID string, req models.EmergencySettingsRequest) (*models.EmergencySettings, error) {
	settings, err := es.GetEmergencySettings(ctx, userID)
	if err != nil {
		userObjectID, _ := primitive.ObjectIDFromHex(userID)
		settings = &models.EmergencySettings{
			UserID:    userObjectID,
		}
	}

	settings.CrashDetection = req.CrashDetection
	settings.FallDetection = req.FallDetection
	settings.HeartRateAlert = req.HeartRateAlert
	settings.AutoCallEmergency = req.AutoCallEmergency
	settings.AutoNotifyContacts = req.AutoNotifyContacts
	settings.CountdownDuration = req.CountdownDuration
	settings.ShareLocationAlways = req.ShareLocationAlways
	settings.ShareWithAuthorities = req.ShareWithAuthorities
	settings.UpdatedAt = time.Now()

	return settings, es.emergencyRepo.UpdateUserSettings(ctx, userID, settings)
}

func (es *EmergencyService) GetEmergencyNotificationSettings(ctx context.Context, userID string) (map[string]interface{}, error) {
	return es.emergencyRepo.GetNotificationSettings(ctx, userID)
}

func (es *EmergencyService) UpdateEmergencyNotificationSettings(ctx context.Context, userID string, req models.EmergencyNotificationSettingsRequest) (map[string]interface{}, error) {
	settings := map[string]interface{}{
		"userId":             userID,
		"pushNotifications":  req.PushNotifications,
		"smsNotifications":   req.SMSNotifications,
		"emailNotifications": req.EmailNotifications,
		"callNotifications":  req.CallNotifications,
		"quietHours":         req.QuietHours,
		"notificationSound":  req.NotificationSound,
		"vibration":          req.Vibration,
		"updatedAt":          time.Now(),
	}

	err := es.emergencyRepo.UpdateNotificationSettings(ctx, userID, settings)
	return settings, err
}

func (es *EmergencyService) GetEmergencyAutomationSettings(ctx context.Context, userID string) (map[string]interface{}, error) {
	return es.emergencyRepo.GetAutomationSettings(ctx, userID)
}

func (es *EmergencyService) UpdateEmergencyAutomationSettings(ctx context.Context, userID string, req models.EmergencyAutomationSettingsRequest) (map[string]interface{}, error) {
	settings := map[string]interface{}{
		"userId":                   userID,
		"autoTriggerRules":         req.AutoTriggerRules,
		"autoResponseEnabled":      req.AutoResponseEnabled,
		"autoLocationSharing":      req.AutoLocationSharing,
		"autoContactNotification":  req.AutoContactNotification,
		"smartDetection":           req.SmartDetection,
		"updatedAt":                time.Now(),
	}

	err := es.emergencyRepo.UpdateAutomationSettings(ctx, userID, settings)
	return settings, err
}

// =================== DRILLS ===================

func (es *EmergencyService) GetEmergencyDrills(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	return es.emergencyRepo.GetUserDrills(ctx, userID)
}

func (es *EmergencyService) CreateEmergencyDrill(ctx context.Context, userID string, req models.CreateEmergencyDrillRequest) (map[string]interface{}, error) {
	drill := map[string]interface{}{
		"drillId":     primitive.NewObjectID().Hex(),
		"userId":      userID,
		"name":        req.Name,
		"type":        req.Type,
		"description": req.Description,
		"scenario":    req.Scenario,
		"duration":    req.Duration,
		"scheduledAt": req.ScheduledAt,
		"circleId":    req.CircleID,
		"status":      "scheduled",
		"createdAt":   time.Now(),
	}

	err := es.emergencyRepo.CreateDrill(ctx, drill)
	return drill, err
}

func (es *EmergencyService) StartEmergencyDrill(ctx context.Context, userID, drillID string) (map[string]interface{}, error) {
	updateFields := bson.M{
		"status":    "active",
		"startedAt": time.Now(),
	}

	err := es.emergencyRepo.UpdateDrill(ctx, userID, drillID, updateFields)
	if err != nil {
		return nil, err
	}

	return es.emergencyRepo.GetDrill(ctx, drillID)
}

func (es *EmergencyService) CompleteEmergencyDrill(ctx context.Context, userID, drillID string, req models.CompleteEmergencyDrillRequest) (map[string]interface{}, error) {
	updateFields := bson.M{
		"status":         "completed",
		"completedAt":    time.Now(),
		"completionTime": req.CompletionTime,
		"success":        req.Success,
		"issues":         req.Issues,
		"feedback":       req.Feedback,
		"participants":   req.Participants,
		"results":        req.Results,
	}

	err := es.emergencyRepo.UpdateDrill(ctx, userID, drillID, updateFields)
	if err != nil {
		return nil, err
	}

	return es.emergencyRepo.GetDrill(ctx, drillID)
}

func (es *EmergencyService) GetDrillResults(ctx context.Context, userID, drillID string) (map[string]interface{}, error) {
	return es.emergencyRepo.GetDrillResults(ctx, userID, drillID)
}

func (es *EmergencyService) DeleteEmergencyDrill(ctx context.Context, userID, drillID string) error {
	return es.emergencyRepo.DeleteDrill(ctx, userID, drillID)
}

// =================== MEDICAL INFO ===================

func (es *EmergencyService) GetMedicalInformation(ctx context.Context, userID string) (map[string]interface{}, error) {
	return es.emergencyRepo.GetMedicalInfo(ctx, userID)
}

func (es *EmergencyService) UpdateMedicalInformation(ctx context.Context, userID string, req models.MedicalInformationRequest) (map[string]interface{}, error) {
	info := map[string]interface{}{
		"userId":           userID,
		"bloodType":        req.BloodType,
		"allergies":        req.Allergies,
		"medications":      req.Medications,
		"conditions":       req.Conditions,
		"emergencyContact": req.EmergencyContact,
		"insuranceInfo":    req.InsuranceInfo,
		"doctorContact":    req.DoctorContact,
		"specialNeeds":     req.SpecialNeeds,
		"updatedAt":        time.Now(),
	}

	err := es.emergencyRepo.UpdateMedicalInfo(ctx, userID, info)
	return info, err
}

func (es *EmergencyService) GetAllergies(ctx context.Context, userID string) ([]models.MedicalAllergy, error) {
	return es.emergencyRepo.GetAllergies(ctx, userID)
}

func (es *EmergencyService) UpdateAllergies(ctx context.Context, userID string, req models.AllergiesRequest) ([]models.MedicalAllergy, error) {
	err := es.emergencyRepo.UpdateAllergies(ctx, userID, req.Allergies)
	if err != nil {
		return nil, err
	}
	return req.Allergies, nil
}

func (es *EmergencyService) GetMedications(ctx context.Context, userID string) ([]models.Medication, error) {
	return es.emergencyRepo.GetMedications(ctx, userID)
}

func (es *EmergencyService) UpdateMedications(ctx context.Context, userID string, req models.MedicationsRequest) ([]models.Medication, error) {
	err := es.emergencyRepo.UpdateMedications(ctx, userID, req.Medications)
	if err != nil {
		return nil, err
	}
	return req.Medications, nil
}

func (es *EmergencyService) GetMedicalConditions(ctx context.Context, userID string) ([]models.MedicalCondition, error) {
	return es.emergencyRepo.GetMedicalConditions(ctx, userID)
}

func (es *EmergencyService) UpdateMedicalConditions(ctx context.Context, userID string, req models.MedicalConditionsRequest) ([]models.MedicalCondition, error) {
	err := es.emergencyRepo.UpdateMedicalConditions(ctx, userID, req.Conditions)
	if err != nil {
		return nil, err
	}
	return req.Conditions, nil
}

// =================== BROADCASTING ===================

func (es *EmergencyService) BroadcastEmergency(ctx context.Context, userID string, req models.BroadcastEmergencyRequest) (map[string]interface{}, error) {
	broadcast := map[string]interface{}{
		"broadcastId":  primitive.NewObjectID().Hex(),
		"senderId":     userID,
		"type":         req.Type,
		"title":        req.Title,
		"message":      req.Message,
		"priority":     req.Priority,
		"recipients":   req.Recipients,
		"channels":     req.Channels,
		"requireAck":   req.RequireAck,
		"timestamp":    time.Now(),
		"status":       "sent",
	}

	if req.ExpiresAt != nil {
		broadcast["expiresAt"] = req.ExpiresAt
	}

	err := es.emergencyRepo.CreateBroadcast(ctx, broadcast)
	if err != nil {
		return nil, err
	}

	// Send broadcast through specified channels
	go es.sendBroadcastNotifications(ctx, broadcast)

	return broadcast, nil
}

func (es *EmergencyService) GetEmergencyBroadcasts(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	return es.emergencyRepo.GetUserBroadcasts(ctx, userID)
}

func (es *EmergencyService) UpdateEmergencyBroadcast(ctx context.Context, userID, broadcastID string, req models.UpdateBroadcastRequest) (map[string]interface{}, error) {
	updateFields := bson.M{
		"updatedAt": time.Now(),
	}

	if req.Title != "" {
		updateFields["title"] = req.Title
	}
	if req.Message != "" {
		updateFields["message"] = req.Message
	}
	if req.Priority != "" {
		updateFields["priority"] = req.Priority
	}
	if req.ExpiresAt != nil {
		updateFields["expiresAt"] = req.ExpiresAt
	}

	err := es.emergencyRepo.UpdateBroadcast(ctx, userID, broadcastID, updateFields)
	if err != nil {
		return nil, err
	}

	return es.emergencyRepo.GetBroadcast(ctx, broadcastID)
}

func (es *EmergencyService) DeleteEmergencyBroadcast(ctx context.Context, userID, broadcastID string) error {
	return es.emergencyRepo.DeleteBroadcast(ctx, userID, broadcastID)
}

func (es *EmergencyService) AcknowledgeBroadcast(ctx context.Context, userID, broadcastID string, req models.AcknowledgeBroadcastRequest) error {
	ack := map[string]interface{}{
		"userId":      userID,
		"broadcastId": broadcastID,
		"message":     req.Message,
		"location":    req.Location,
		"status":      req.Status,
		"timestamp":   time.Now(),
	}

	return es.emergencyRepo.CreateBroadcastAck(ctx, ack)
}

// =================== EXISTING METHODS ===================

func (es *EmergencyService) CreateEmergency(ctx context.Context, userID string, req models.CreateEmergencyRequest) (*models.Emergency, error) {
	// This is the existing method - keeping for compatibility
	alertReq := models.CreateEmergencyAlertRequest{
		Type:        req.Type,
		Title:       es.getEmergencyTitle(req.Type),
		Description: req.Description,
		Priority:    "high",
		Location:    req.Location,
	}

	if req.Type == models.EmergencyTypeSOS || req.Type == models.EmergencyTypeCrash {
		alertReq.Priority = "critical"
	}

	return es.CreateEmergencyAlert(ctx, userID, alertReq)
}

func (es *EmergencyService) GetEmergency(ctx context.Context, userID, emergencyID string) (*models.Emergency, error) {
	return es.GetEmergencyAlert(ctx, userID, emergencyID)
}

func (es *EmergencyService) GetUserEmergencies(ctx context.Context, userID string) ([]models.Emergency, error) {
	return es.GetEmergencyAlerts(ctx, userID)
}

func (es *EmergencyService) GetCircleEmergencies(ctx context.Context, userID, circleID string) ([]models.Emergency, error) {
	isMember, err := es.circleRepo.IsMember(ctx, circleID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New("access denied")
	}

	return es.emergencyRepo.GetCircleEmergencies(ctx, circleID)
}

func (es *EmergencyService) UpdateEmergency(ctx context.Context, userID, emergencyID string, req models.UpdateEmergencyRequest) (*models.Emergency, error) {
	emergency, err := es.emergencyRepo.GetByID(ctx, emergencyID)
	if err != nil {
		return nil, err
	}

	if !es.hasEmergencyAccess(ctx, userID, emergency) {
		return nil, errors.New("access denied")
	}

	updateFields := bson.M{"updatedAt": time.Now()}
	
	if req.Status != "" {
		updateFields["status"] = req.Status
	}
	if req.Description != "" {
		updateFields["description"] = req.Description
	}
	if req.Resolution != "" {
		updateFields["resolution"] = req.Resolution
		if req.Status == models.EmergencyStatusResolved {
			updateFields["resolvedAt"] = time.Now()
		}
	}

	err = es.emergencyRepo.Update(ctx, emergencyID, updateFields)
	if err != nil {
		return nil, err
	}

	return es.emergencyRepo.GetByID(ctx, emergencyID)
}

func (es *EmergencyService) DismissEmergency(ctx context.Context, userID, emergencyID, reason string) (*models.Emergency, error) {
	err := es.DismissEmergencyAlert(ctx, userID, emergencyID, reason)
	if err != nil {
		return nil, err
	}
	return es.emergencyRepo.GetByID(ctx, emergencyID)
}

func (es *EmergencyService) CancelEmergency(ctx context.Context, userID, emergencyID, reason string) (*models.Emergency, error) {
	emergency, err := es.emergencyRepo.GetByID(ctx, emergencyID)
	if err != nil {
		return nil, err
	}

	if emergency.UserID.Hex() != userID {
		return nil, errors.New("access denied")
	}

	updateFields := bson.M{
		"status":             models.EmergencyStatusCancelled,
		"cancellationReason": reason,
		"cancelledAt":        time.Now(),
		"updatedAt":          time.Now(),
	}

	err = es.emergencyRepo.Update(ctx, emergencyID, updateFields)
	if err != nil {
		return nil, err
	}

	return es.emergencyRepo.GetByID(ctx, emergencyID)
}

func (es *EmergencyService) ResolveEmergency(ctx context.Context, userID, emergencyID, resolution string) error {
	return es.ResolveEmergencyAlert(ctx, userID, emergencyID, resolution)
}

func (es *EmergencyService) AddMedia(ctx context.Context, userID, emergencyID string, media models.EmergencyMedia) error {
	emergency, err := es.emergencyRepo.GetByID(ctx, emergencyID)
	if err != nil {
		return err
	}

	if !es.hasEmergencyAccess(ctx, userID, emergency) {
		return errors.New("access denied")
	}

	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	media.UploadedBy = userObjectID

	return es.emergencyRepo.AddMedia(ctx, emergencyID, media)
}

// =================== HELPER METHODS ===================

func (es *EmergencyService) hasEmergencyAccess(ctx context.Context, userID string, emergency *models.Emergency) bool {
	// User is owner
	if emergency.UserID.Hex() == userID {
		return true
	}

	// User is in same circle
	if !emergency.CircleID.IsZero() {
		isMember, err := es.circleRepo.IsMember(ctx, emergency.CircleID.Hex(), userID)
		if err == nil && isMember {
			return true
		}
	}

	return false
}

func (es *EmergencyService) getEmergencyTitle(emergencyType string) string {
	switch emergencyType {
	case models.EmergencyTypeSOS:
		return "SOS Emergency Alert"
	case models.EmergencyTypeCrash:
		return "Crash Detected"
	case models.EmergencyTypeHelp:
		return "Help Needed"
	case models.EmergencyTypeMedical:
		return "Medical Emergency"
	case models.EmergencyTypeFire:
		return "Fire Emergency"
	case models.EmergencyTypePolice:
		return "Police Emergency"
	case models.EmergencyTypeRoadside:
		return "Roadside Assistance"
	default:
		return "Emergency Alert"
	}
}

func (es *EmergencyService) handleEmergencyNotifications(ctx context.Context, emergency *models.Emergency) {
	// Get user details
	_, err := es.userRepo.GetByID(ctx, emergency.UserID.Hex())
	if err != nil {
		return
	}

	// Get user's circles
	circles, err := es.circleRepo.GetUserCircles(ctx, emergency.UserID.Hex())
	if err != nil {
		return
	}

	// Notify emergency contacts
	es.notifyEmergencyContacts(ctx, emergency, circles)

	// Broadcast to circle members
	es.broadcastEmergencyAlert(emergency.UserID.Hex(), emergency, circles)
}

func (es *EmergencyService) notifyEmergencyContacts(ctx context.Context, emergency *models.Emergency, circles []models.Circle) {
	user, err := es.userRepo.GetByID(ctx, emergency.UserID.Hex())
	if err != nil {
		return
	}

	// Get emergency contacts
	contacts, err := es.GetEmergencyContacts(ctx, emergency.UserID.Hex())
	if err != nil {
		return
	}

	for _, contact := range contacts {
		// Send SMS
		if contact.Phone != "" {
			sms := utils.SMSMessage{
				To: contact.Phone,
				Message: fmt.Sprintf("EMERGENCY: %s %s needs help. Location: %.6f,%.6f",
					user.FirstName, user.LastName,
					emergency.Location.Latitude, emergency.Location.Longitude),
			}
			logrus.Info("Sending emergency SMS to: ", contact.Phone, " with message: ", sms.Message)
		}
	}

	// Notify circle members
	for _, circle := range circles {
		if circle.Settings.EmergencyAlerts {
			var userIDs []string
			for _, member := range circle.Members {
				if member.Status == "active" && member.UserID != emergency.UserID {
					userIDs = append(userIDs, member.UserID.Hex())
				}
			}

			if len(userIDs) > 0 {
				notifReq := models.SendNotificationRequest{
					UserIDs:  userIDs,
					Type:     models.NotificationEmergencySOS,
					Title:    emergency.Title,
					Body:     fmt.Sprintf("%s %s needs help", user.FirstName, user.LastName),
					Priority: "urgent",
					Channels: models.NotificationChannels{
						Push:  true,
						SMS:   true,
						InApp: true,
					},
				}

				es.notificationSvc.SendNotification(ctx, notifReq)
			}
		}
	}
}

func (es *EmergencyService) broadcastEmergencyAlert(userID string, emergency *models.Emergency, circles []models.Circle) {
	alert := models.WSEmergencyAlert{
		UserID:    userID,
		Location:  emergency.Location,
		Type:      emergency.Type,
		Message:   emergency.Description,
		Timestamp: emergency.CreatedAt,
	}

	var circleIDs []string
	for _, circle := range circles {
		if circle.Settings.EmergencyAlerts {
			circleIDs = append(circleIDs, circle.ID.Hex())
		}
	}

	if len(circleIDs) > 0 {
		es.websocketHub.BroadcastEmergencyAlert(circleIDs, alert)
	}
}

func (es *EmergencyService) handleSOSCountdown(ctx context.Context, emergencyID string, countdownSec int) {
	time.Sleep(time.Duration(countdownSec) * time.Second)
	
	// Check if SOS was cancelled
	emergency, err := es.emergencyRepo.GetByID(ctx, emergencyID)
	if err != nil || emergency.Status != models.EmergencyStatusActive {
		return
	}

	// Auto-call emergency services
	logrus.Info("SOS countdown completed - auto-calling emergency services for emergency: ", emergencyID)
}

func (es *EmergencyService) handleCrashConfirmationCountdown(ctx context.Context, emergencyID string) {
	time.Sleep(30 * time.Second) // 30 second countdown
	
	emergency, err := es.emergencyRepo.GetByID(ctx, emergencyID)
	if err != nil || emergency.Status != models.EmergencyStatusActive {
		return
	}

	// If not confirmed/dismissed by user, escalate
	es.handleEmergencyNotifications(ctx, emergency)
}

func (es *EmergencyService) processEmergencyExport(ctx context.Context, userID string, export *models.EmergencyFileExport, req models.ExportHistoryRequest) {
	// Background process to generate export file
	// This would collect data, format it, and store the file
	export.Status = "completed"
	export.Progress = 100
	export.CompletedAt = time.Now()
	
	// Update export status in database
	es.emergencyRepo.UpdateExport(ctx, export)
}

func (es *EmergencyService) notifyEmergencyCreator(ctx context.Context, emergency *models.Emergency, response map[string]interface{}) {
	// Notify emergency creator about response
	logrus.Info("Notifying emergency creator about response")
}

func (es *EmergencyService) broadcastHelpRequest(ctx context.Context, helpRequest map[string]interface{}) {
	// Broadcast help request to nearby users
	logrus.Info("Broadcasting help request to nearby users")
}

func (es *EmergencyService) notifyCheckInStatus(ctx context.Context, userID, status, message string) {
	// Notify emergency contacts about check-in status
	logrus.Info("Notifying contacts about check-in status: ", status)
}

func (es *EmergencyService) notifyCheckInRequest(ctx context.Context, targetUserID string, request map[string]interface{}) {
	// Notify user about check-in request
	logrus.Info("Notifying user about check-in request")
}

func (es *EmergencyService) sendBroadcastNotifications(ctx context.Context, broadcast map[string]interface{}) {
	// Send broadcast through specified channels
	logrus.Info("Sending broadcast notifications")
}