package repositories

import (
	"context"
	"errors"
	"fmt"
	"ftrack/models"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EmergencyRepository struct {
	database            *mongo.Database
	emergencyCollection *mongo.Collection
	settingsCollection  *mongo.Collection
	contactsCollection  *mongo.Collection
	locationsCollection *mongo.Collection
	responsesCollection *mongo.Collection
	checkinsCollection  *mongo.Collection
	drillsCollection    *mongo.Collection
	medicalCollection   *mongo.Collection
	broadcastCollection *mongo.Collection
	exportsCollection   *mongo.Collection
	eventsCollection    *mongo.Collection
}

func NewEmergencyRepository(database *mongo.Database) *EmergencyRepository {
	return &EmergencyRepository{
		database:            database,
		emergencyCollection: database.Collection("emergencies"),
		settingsCollection:  database.Collection("emergency_settings"),
		contactsCollection:  database.Collection("emergency_contacts"),
		locationsCollection: database.Collection("emergency_locations"),
		responsesCollection: database.Collection("emergency_responses"),
		checkinsCollection:  database.Collection("emergency_checkins"),
		drillsCollection:    database.Collection("emergency_drills"),
		medicalCollection:   database.Collection("emergency_medical"),
		broadcastCollection: database.Collection("emergency_broadcasts"),
		exportsCollection:   database.Collection("emergency_exports"),
		eventsCollection:    database.Collection("emergency_events"),
	}
}

// =================== BASIC CRUD OPERATIONS ===================

func (er *EmergencyRepository) Create(ctx context.Context, emergency *models.Emergency) error {
	emergency.ID = primitive.NewObjectID()
	emergency.CreatedAt = time.Now()
	emergency.UpdatedAt = time.Now()

	if emergency.Status == "" {
		emergency.Status = models.EmergencyStatusActive
	}

	_, err := er.emergencyCollection.InsertOne(ctx, emergency)
	if err != nil {
		logrus.Errorf("Failed to create emergency: %v", err)
		return err
	}

	// Create initial event
	event := models.EmergencyEvent{
		ID:          primitive.NewObjectID(),
		Type:        "created",
		Description: fmt.Sprintf("Emergency created: %s", emergency.Title),
		Actor:       emergency.UserID,
		Timestamp:   time.Now(),
		Data: map[string]interface{}{
			"emergencyId": emergency.ID.Hex(),
			"type":        emergency.Type,
			"priority":    emergency.Priority,
		},
	}

	er.AddEmergencyEvent(ctx, emergency.ID.Hex(), event)

	return nil
}

func (er *EmergencyRepository) GetByID(ctx context.Context, id string) (*models.Emergency, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid emergency ID")
	}

	var emergency models.Emergency
	err = er.emergencyCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&emergency)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("not found")
		}
		logrus.Errorf("Failed to get emergency by ID: %v", err)
		return nil, err
	}

	return &emergency, nil
}

func (er *EmergencyRepository) Update(ctx context.Context, id string, updateFields bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid emergency ID")
	}

	updateFields["updatedAt"] = time.Now()

	result, err := er.emergencyCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		logrus.Errorf("Failed to update emergency: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("emergency not found")
	}

	// Create update event
	event := models.EmergencyEvent{
		ID:          primitive.NewObjectID(),
		Type:        "updated",
		Description: "Emergency updated",
		Timestamp:   time.Now(),
		Data:        updateFields,
	}

	er.AddEmergencyEvent(ctx, id, event)

	return nil
}

func (er *EmergencyRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid emergency ID")
	}

	result, err := er.emergencyCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		logrus.Errorf("Failed to delete emergency: %v", err)
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("emergency not found")
	}

	// Clean up related data
	er.cleanupEmergencyData(ctx, id)

	return nil
}

// =================== USER EMERGENCY OPERATIONS ===================

func (er *EmergencyRepository) GetUserEmergencies(ctx context.Context, userID string) ([]models.Emergency, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := er.emergencyCollection.Find(ctx, bson.M{"userId": userObjectID}, opts)
	if err != nil {
		logrus.Errorf("Failed to get user emergencies: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var emergencies []models.Emergency
	if err = cursor.All(ctx, &emergencies); err != nil {
		logrus.Errorf("Failed to decode user emergencies: %v", err)
		return nil, err
	}

	return emergencies, nil
}

func (er *EmergencyRepository) GetUserActiveEmergencies(ctx context.Context, userID, emergencyType string) ([]models.Emergency, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId": userObjectID,
		"status": models.EmergencyStatusActive,
	}

	if emergencyType != "" {
		filter["type"] = emergencyType
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := er.emergencyCollection.Find(ctx, filter, opts)
	if err != nil {
		logrus.Errorf("Failed to get user active emergencies: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var emergencies []models.Emergency
	if err = cursor.All(ctx, &emergencies); err != nil {
		logrus.Errorf("Failed to decode user active emergencies: %v", err)
		return nil, err
	}

	return emergencies, nil
}

func (er *EmergencyRepository) GetUserEmergenciesByType(ctx context.Context, userID, emergencyType string) ([]models.Emergency, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"userId": userObjectID,
		"type":   emergencyType,
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := er.emergencyCollection.Find(ctx, filter, opts)
	if err != nil {
		logrus.Errorf("Failed to get user emergencies by type: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var emergencies []models.Emergency
	if err = cursor.All(ctx, &emergencies); err != nil {
		logrus.Errorf("Failed to decode user emergencies by type: %v", err)
		return nil, err
	}

	return emergencies, nil
}

func (er *EmergencyRepository) GetCircleEmergencies(ctx context.Context, circleID string) ([]models.Emergency, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := er.emergencyCollection.Find(ctx, bson.M{"circleId": circleObjectID}, opts)
	if err != nil {
		logrus.Errorf("Failed to get circle emergencies: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var emergencies []models.Emergency
	if err = cursor.All(ctx, &emergencies); err != nil {
		logrus.Errorf("Failed to decode circle emergencies: %v", err)
		return nil, err
	}

	return emergencies, nil
}

// =================== EMERGENCY CONTACT OPERATIONS ===================

func (er *EmergencyRepository) GetUserEmergencyContacts(ctx context.Context, userID string) ([]models.EmergencyContact, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	cursor, err := er.contactsCollection.Find(ctx, bson.M{"userId": userObjectID})
	if err != nil {
		logrus.Errorf("Failed to get emergency contacts: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var contacts []models.EmergencyContact
	if err = cursor.All(ctx, &contacts); err != nil {
		logrus.Errorf("Failed to decode emergency contacts: %v", err)
		return nil, err
	}

	return contacts, nil
}

func (er *EmergencyRepository) AddEmergencyContact(ctx context.Context, userID string, contact *models.EmergencyContact) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	contactDoc := bson.M{
		"_id":          primitive.NewObjectID(),
		"userId":       userObjectID,
		"contactId":    contact.ContactID,
		"name":         contact.Name,
		"phone":        contact.Phone,
		"email":        contact.Email,
		"relationship": contact.Relationship,
		"createdAt":    time.Now(),
		"updatedAt":    time.Now(),
	}

	_, err = er.contactsCollection.InsertOne(ctx, contactDoc)
	if err != nil {
		logrus.Errorf("Failed to add emergency contact: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) UpdateEmergencyContact(ctx context.Context, userID, contactID string, contact *models.EmergencyContact) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	contactObjectID, err := primitive.ObjectIDFromHex(contactID)
	if err != nil {
		return errors.New("invalid contact ID")
	}

	updateFields := bson.M{
		"name":         contact.Name,
		"phone":        contact.Phone,
		"email":        contact.Email,
		"relationship": contact.Relationship,
		"updatedAt":    time.Now(),
	}

	result, err := er.contactsCollection.UpdateOne(
		ctx,
		bson.M{"userId": userObjectID, "contactId": contactObjectID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		logrus.Errorf("Failed to update emergency contact: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("contact not found")
	}

	return nil
}

func (er *EmergencyRepository) DeleteEmergencyContact(ctx context.Context, userID, contactID string) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	contactObjectID, err := primitive.ObjectIDFromHex(contactID)
	if err != nil {
		return errors.New("invalid contact ID")
	}

	result, err := er.contactsCollection.DeleteOne(ctx, bson.M{
		"userId":    userObjectID,
		"contactId": contactObjectID,
	})

	if err != nil {
		logrus.Errorf("Failed to delete emergency contact: %v", err)
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("contact not found")
	}

	return nil
}

func (er *EmergencyRepository) GetContactNotificationHistory(ctx context.Context, userID, contactID string) ([]models.EmergencyEvent, error) {
	filter := bson.M{
		"data.userId":    userID,
		"data.contactId": contactID,
		"type":           bson.M{"$in": []string{"contact_notified", "contact_responded"}},
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := er.eventsCollection.Find(ctx, filter, opts)
	if err != nil {
		logrus.Errorf("Failed to get contact notification history: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []models.EmergencyEvent
	if err = cursor.All(ctx, &events); err != nil {
		logrus.Errorf("Failed to decode contact notification history: %v", err)
		return nil, err
	}

	return events, nil
}

// =================== SETTINGS OPERATIONS ===================

func (er *EmergencyRepository) GetUserSettings(ctx context.Context, userID string) (*models.EmergencySettings, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var settings models.EmergencySettings
	err = er.settingsCollection.FindOne(ctx, bson.M{"userId": userObjectID}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default settings
			return &models.EmergencySettings{
				UserID:              userObjectID,
				CrashDetection:      true,
				FallDetection:       false,
				HeartRateAlert:      false,
				AutoCallEmergency:   false,
				AutoNotifyContacts:  true,
				CountdownDuration:   30,
				ShareLocationAlways: true,
				UpdatedAt:           time.Now(),
			}, nil
		}
		logrus.Errorf("Failed to get user settings: %v", err)
		return nil, err
	}

	return &settings, nil
}

func (er *EmergencyRepository) UpdateUserSettings(ctx context.Context, userID string, settings *models.EmergencySettings) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	settings.UserID = userObjectID
	settings.UpdatedAt = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err = er.settingsCollection.ReplaceOne(
		ctx,
		bson.M{"userId": userObjectID},
		settings,
		opts,
	)

	if err != nil {
		logrus.Errorf("Failed to update user settings: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) GetNotificationSettings(ctx context.Context, userID string) (map[string]interface{}, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var result map[string]interface{}
	err = er.database.Collection("notification_settings").FindOne(ctx, bson.M{"userId": userObjectID}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default notification settings
			return map[string]interface{}{
				"userId":             userID,
				"pushNotifications":  true,
				"smsNotifications":   true,
				"emailNotifications": true,
				"callNotifications":  false,
				"quietHours":         []string{},
				"notificationSound":  "default",
				"vibration":          true,
				"createdAt":          time.Now(),
				"updatedAt":          time.Now(),
			}, nil
		}
		logrus.Errorf("Failed to get notification settings: %v", err)
		return nil, err
	}

	return result, nil
}

func (er *EmergencyRepository) UpdateNotificationSettings(ctx context.Context, userID string, settings map[string]interface{}) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	settings["userId"] = userObjectID
	settings["updatedAt"] = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err = er.database.Collection("notification_settings").ReplaceOne(
		ctx,
		bson.M{"userId": userObjectID},
		settings,
		opts,
	)

	if err != nil {
		logrus.Errorf("Failed to update notification settings: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) GetAutomationSettings(ctx context.Context, userID string) (map[string]interface{}, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var result map[string]interface{}
	err = er.database.Collection("automation_settings").FindOne(ctx, bson.M{"userId": userObjectID}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default automation settings
			return map[string]interface{}{
				"userId":                  userID,
				"autoTriggerRules":        []models.AutoTriggerRule{},
				"autoResponseEnabled":     false,
				"autoLocationSharing":     true,
				"autoContactNotification": true,
				"smartDetection":          true,
				"createdAt":               time.Now(),
				"updatedAt":               time.Now(),
			}, nil
		}
		logrus.Errorf("Failed to get automation settings: %v", err)
		return nil, err
	}

	return result, nil
}

func (er *EmergencyRepository) UpdateAutomationSettings(ctx context.Context, userID string, settings map[string]interface{}) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	settings["userId"] = userObjectID
	settings["updatedAt"] = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err = er.database.Collection("automation_settings").ReplaceOne(
		ctx,
		bson.M{"userId": userObjectID},
		settings,
		opts,
	)

	if err != nil {
		logrus.Errorf("Failed to update automation settings: %v", err)
		return err
	}

	return nil
}

// =================== LOCATION SHARING OPERATIONS ===================

func (er *EmergencyRepository) CreateLocationShare(ctx context.Context, share map[string]interface{}) error {
	if share["shareId"] == nil {
		share["shareId"] = primitive.NewObjectID().Hex()
	}
	share["createdAt"] = time.Now()
	share["updatedAt"] = time.Now()

	_, err := er.locationsCollection.InsertOne(ctx, share)
	if err != nil {
		logrus.Errorf("Failed to create location share: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) GetUserLocationShares(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"userId": userID},
			{"recipients": bson.M{"$in": []string{userID}}},
		},
		"active": true,
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := er.locationsCollection.Find(ctx, filter, opts)
	if err != nil {
		logrus.Errorf("Failed to get user location shares: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var shares []map[string]interface{}
	if err = cursor.All(ctx, &shares); err != nil {
		logrus.Errorf("Failed to decode user location shares: %v", err)
		return nil, err
	}

	return shares, nil
}

func (er *EmergencyRepository) GetLocationShare(ctx context.Context, shareID string) (map[string]interface{}, error) {
	var share map[string]interface{}
	err := er.locationsCollection.FindOne(ctx, bson.M{"shareId": shareID}).Decode(&share)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("location share not found")
		}
		logrus.Errorf("Failed to get location share: %v", err)
		return nil, err
	}

	return share, nil
}

func (er *EmergencyRepository) UpdateLocationShare(ctx context.Context, userID, shareID string, updateFields bson.M) error {
	updateFields["updatedAt"] = time.Now()

	result, err := er.locationsCollection.UpdateOne(
		ctx,
		bson.M{"shareId": shareID, "userId": userID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		logrus.Errorf("Failed to update location share: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("location share not found")
	}

	return nil
}

func (er *EmergencyRepository) GetUserCurrentLocation(ctx context.Context, userID string) (map[string]interface{}, error) {
	// This would typically get the latest location from a locations collection
	// For now, return a placeholder
	location := map[string]interface{}{
		"userId":    userID,
		"latitude":  0.0,
		"longitude": 0.0,
		"accuracy":  10.0,
		"timestamp": time.Now(),
	}

	return location, nil
}

// =================== RESPONSE OPERATIONS ===================

func (er *EmergencyRepository) AddEmergencyResponse(ctx context.Context, alertID string, response map[string]interface{}) error {
	response["emergencyId"] = alertID
	response["createdAt"] = time.Now()
	response["updatedAt"] = time.Now()

	_, err := er.responsesCollection.InsertOne(ctx, response)
	if err != nil {
		logrus.Errorf("Failed to add emergency response: %v", err)
		return err
	}

	// Create response event
	event := models.EmergencyEvent{
		ID:          primitive.NewObjectID(),
		Type:        "response_received",
		Description: "Emergency response received",
		Timestamp:   time.Now(),
		Data:        response,
	}

	er.AddEmergencyEvent(ctx, alertID, event)

	return nil
}

func (er *EmergencyRepository) GetEmergencyResponses(ctx context.Context, alertID string) ([]map[string]interface{}, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := er.responsesCollection.Find(ctx, bson.M{"emergencyId": alertID}, opts)
	if err != nil {
		logrus.Errorf("Failed to get emergency responses: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var responses []map[string]interface{}
	if err = cursor.All(ctx, &responses); err != nil {
		logrus.Errorf("Failed to decode emergency responses: %v", err)
		return nil, err
	}

	return responses, nil
}

func (er *EmergencyRepository) GetEmergencyResponse(ctx context.Context, alertID, responseID string) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := er.responsesCollection.FindOne(ctx, bson.M{
		"emergencyId": alertID,
		"responseId":  responseID,
	}).Decode(&response)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("response not found")
		}
		logrus.Errorf("Failed to get emergency response: %v", err)
		return nil, err
	}

	return response, nil
}

func (er *EmergencyRepository) UpdateEmergencyResponse(ctx context.Context, alertID, responseID string, updateFields bson.M) error {
	updateFields["updatedAt"] = time.Now()

	result, err := er.responsesCollection.UpdateOne(
		ctx,
		bson.M{"emergencyId": alertID, "responseId": responseID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		logrus.Errorf("Failed to update emergency response: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("response not found")
	}

	return nil
}

// =================== HELP REQUEST/OFFER OPERATIONS ===================

func (er *EmergencyRepository) CreateHelpRequest(ctx context.Context, request map[string]interface{}) error {
	request["createdAt"] = time.Now()
	request["updatedAt"] = time.Now()

	_, err := er.database.Collection("help_requests").InsertOne(ctx, request)
	if err != nil {
		logrus.Errorf("Failed to create help request: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) CreateHelpOffer(ctx context.Context, offer map[string]interface{}) error {
	offer["createdAt"] = time.Now()
	offer["updatedAt"] = time.Now()

	_, err := er.database.Collection("help_offers").InsertOne(ctx, offer)
	if err != nil {
		logrus.Errorf("Failed to create help offer: %v", err)
		return err
	}

	return nil
}

// =================== CHECK-IN OPERATIONS ===================

func (er *EmergencyRepository) CreateCheckIn(ctx context.Context, checkIn map[string]interface{}) error {
	checkIn["createdAt"] = time.Now()

	_, err := er.checkinsCollection.InsertOne(ctx, checkIn)
	if err != nil {
		logrus.Errorf("Failed to create check-in: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) GetLastCheckIn(ctx context.Context, userID string) (map[string]interface{}, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	var checkIn map[string]interface{}
	err := er.checkinsCollection.FindOne(ctx, bson.M{"userId": userID}, opts).Decode(&checkIn)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("no check-in found")
		}
		logrus.Errorf("Failed to get last check-in: %v", err)
		return nil, err
	}

	return checkIn, nil
}

func (er *EmergencyRepository) GetCheckInSettings(ctx context.Context, userID string) (map[string]interface{}, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var settings map[string]interface{}
	err = er.database.Collection("checkin_settings").FindOne(ctx, bson.M{"userId": userObjectID}).Decode(&settings)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default settings
			return map[string]interface{}{
				"userId":            userID,
				"enabled":           false,
				"frequency":         24, // hours
				"autoReminder":      true,
				"quietHours":        []string{"22:00-06:00"},
				"emergencyContacts": []string{},
				"geoFencing":        false,
				"createdAt":         time.Now(),
				"updatedAt":         time.Now(),
			}, nil
		}
		logrus.Errorf("Failed to get check-in settings: %v", err)
		return nil, err
	}

	return settings, nil
}

func (er *EmergencyRepository) UpdateCheckInSettings(ctx context.Context, userID string, settings map[string]interface{}) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	settings["userId"] = userObjectID
	settings["updatedAt"] = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err = er.database.Collection("checkin_settings").ReplaceOne(
		ctx,
		bson.M{"userId": userObjectID},
		settings,
		opts,
	)

	if err != nil {
		logrus.Errorf("Failed to update check-in settings: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) CreateCheckInRequest(ctx context.Context, request map[string]interface{}) error {
	request["createdAt"] = time.Now()

	_, err := er.database.Collection("checkin_requests").InsertOne(ctx, request)
	if err != nil {
		logrus.Errorf("Failed to create check-in request: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) GetCheckInRequests(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"requesterId": userID},
			{"targetUserId": userID},
		},
		"status": bson.M{"$ne": "completed"},
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := er.database.Collection("checkin_requests").Find(ctx, filter, opts)
	if err != nil {
		logrus.Errorf("Failed to get check-in requests: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var requests []map[string]interface{}
	if err = cursor.All(ctx, &requests); err != nil {
		logrus.Errorf("Failed to decode check-in requests: %v", err)
		return nil, err
	}

	return requests, nil
}

// =================== TIMELINE AND EVENTS ===================

func (er *EmergencyRepository) GetEmergencyTimeline(ctx context.Context, alertID string) ([]models.EmergencyEvent, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"data.emergencyId": alertID},
			{"emergencyId": alertID},
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})
	cursor, err := er.eventsCollection.Find(ctx, filter, opts)
	if err != nil {
		logrus.Errorf("Failed to get emergency timeline: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []models.EmergencyEvent
	if err = cursor.All(ctx, &events); err != nil {
		logrus.Errorf("Failed to decode emergency timeline: %v", err)
		return nil, err
	}

	return events, nil
}

func (er *EmergencyRepository) AddEmergencyEvent(ctx context.Context, alertID string, event models.EmergencyEvent) error {
	if event.Data == nil {
		event.Data = make(map[string]interface{})
	}
	event.Data["emergencyId"] = alertID
	event.Timestamp = time.Now()

	_, err := er.eventsCollection.InsertOne(ctx, event)
	if err != nil {
		logrus.Errorf("Failed to add emergency event: %v", err)
		return err
	}

	return nil
}

// =================== STATISTICS AND ANALYTICS ===================

func (er *EmergencyRepository) GetEmergencyStats(ctx context.Context, startTime, endTime time.Time) (*models.EmergencyStats, error) {
	// Total emergencies
	totalEmergencies, err := er.emergencyCollection.CountDocuments(ctx, bson.M{
		"createdAt": bson.M{"$gte": startTime, "$lte": endTime},
	})
	if err != nil {
		return nil, err
	}

	// Active emergencies
	activeEmergencies, err := er.emergencyCollection.CountDocuments(ctx, bson.M{
		"status": models.EmergencyStatusActive,
	})
	if err != nil {
		return nil, err
	}

	// Resolved emergencies
	resolvedEmergencies, err := er.emergencyCollection.CountDocuments(ctx, bson.M{
		"status":    models.EmergencyStatusResolved,
		"createdAt": bson.M{"$gte": startTime, "$lte": endTime},
	})
	if err != nil {
		return nil, err
	}

	// False alarms
	falseAlarms, err := er.emergencyCollection.CountDocuments(ctx, bson.M{
		"status":    models.EmergencyStatusFalseAlarm,
		"createdAt": bson.M{"$gte": startTime, "$lte": endTime},
	})
	if err != nil {
		return nil, err
	}

	// Type breakdown
	pipeline := []bson.M{
		{"$match": bson.M{"createdAt": bson.M{"$gte": startTime, "$lte": endTime}}},
		{"$group": bson.M{
			"_id":   "$type",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := er.emergencyCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	typeBreakdown := make(map[string]int64)
	var typeStats []models.EmergencyTypeStats

	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		typeBreakdown[result.ID] = result.Count

		percentage := float64(result.Count) / float64(totalEmergencies) * 100
		typeStats = append(typeStats, models.EmergencyTypeStats{
			Type:       result.ID,
			Count:      result.Count,
			Percentage: percentage,
		})
	}

	stats := &models.EmergencyStats{
		TotalEmergencies:    totalEmergencies,
		ActiveEmergencies:   activeEmergencies,
		ResolvedEmergencies: resolvedEmergencies,
		FalseAlarms:         falseAlarms,
		ResponseTime:        map[string]float64{"avg": 0, "min": 0, "max": 0}, // Would calculate from actual data
		TypeBreakdown:       typeBreakdown,
		MostCommonTypes:     typeStats,
		MonthlyTrend:        []models.MonthlyEmergencyStats{}, // Would calculate monthly trends
	}

	return stats, nil
}

// =================== EXPORT OPERATIONS ===================

func (er *EmergencyRepository) GetExport(ctx context.Context, userID, exportID string) (*models.EmergencyFileExport, error) {
	var export models.EmergencyFileExport
	err := er.exportsCollection.FindOne(ctx, bson.M{
		"id":     exportID,
		"userId": userID,
	}).Decode(&export)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("export not found")
		}
		logrus.Errorf("Failed to get export: %v", err)
		return nil, err
	}

	return &export, nil
}

func (er *EmergencyRepository) UpdateExport(ctx context.Context, export *models.EmergencyFileExport) error {
	_, err := er.exportsCollection.UpdateOne(
		ctx,
		bson.M{"id": export.ID},
		bson.M{"$set": export},
	)

	if err != nil {
		logrus.Errorf("Failed to update export: %v", err)
		return err
	}

	return nil
}

// =================== DRILL OPERATIONS ===================

func (er *EmergencyRepository) GetUserDrills(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := er.drillsCollection.Find(ctx, bson.M{"userId": userID}, opts)
	if err != nil {
		logrus.Errorf("Failed to get user drills: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var drills []map[string]interface{}
	if err = cursor.All(ctx, &drills); err != nil {
		logrus.Errorf("Failed to decode user drills: %v", err)
		return nil, err
	}

	return drills, nil
}

func (er *EmergencyRepository) CreateDrill(ctx context.Context, drill map[string]interface{}) error {
	drill["createdAt"] = time.Now()
	drill["updatedAt"] = time.Now()

	_, err := er.drillsCollection.InsertOne(ctx, drill)
	if err != nil {
		logrus.Errorf("Failed to create drill: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) GetDrill(ctx context.Context, drillID string) (map[string]interface{}, error) {
	var drill map[string]interface{}
	err := er.drillsCollection.FindOne(ctx, bson.M{"drillId": drillID}).Decode(&drill)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("drill not found")
		}
		logrus.Errorf("Failed to get drill: %v", err)
		return nil, err
	}

	return drill, nil
}

func (er *EmergencyRepository) UpdateDrill(ctx context.Context, userID, drillID string, updateFields bson.M) error {
	updateFields["updatedAt"] = time.Now()

	result, err := er.drillsCollection.UpdateOne(
		ctx,
		bson.M{"drillId": drillID, "userId": userID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		logrus.Errorf("Failed to update drill: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("drill not found")
	}

	return nil
}

func (er *EmergencyRepository) DeleteDrill(ctx context.Context, userID, drillID string) error {
	result, err := er.drillsCollection.DeleteOne(ctx, bson.M{
		"drillId": drillID,
		"userId":  userID,
	})

	if err != nil {
		logrus.Errorf("Failed to delete drill: %v", err)
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("drill not found")
	}

	return nil
}

func (er *EmergencyRepository) GetDrillResults(ctx context.Context, userID, drillID string) (map[string]interface{}, error) {
	drill, err := er.GetDrill(ctx, drillID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if drill["userId"].(string) != userID {
		return nil, errors.New("access denied")
	}

	// Return drill results
	if results, ok := drill["results"]; ok {
		return results.(map[string]interface{}), nil
	}

	return map[string]interface{}{}, nil
}

// =================== MEDICAL INFORMATION OPERATIONS ===================

func (er *EmergencyRepository) GetMedicalInfo(ctx context.Context, userID string) (map[string]interface{}, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var info map[string]interface{}
	err = er.medicalCollection.FindOne(ctx, bson.M{"userId": userObjectID}).Decode(&info)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return empty medical info
			return map[string]interface{}{
				"userId":           userID,
				"bloodType":        "",
				"allergies":        []string{},
				"medications":      []string{},
				"conditions":       []string{},
				"emergencyContact": "",
				"insuranceInfo":    "",
				"doctorContact":    "",
				"specialNeeds":     "",
				"createdAt":        time.Now(),
				"updatedAt":        time.Now(),
			}, nil
		}
		logrus.Errorf("Failed to get medical info: %v", err)
		return nil, err
	}

	return info, nil
}

func (er *EmergencyRepository) UpdateMedicalInfo(ctx context.Context, userID string, info map[string]interface{}) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	info["userId"] = userObjectID
	info["updatedAt"] = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err = er.medicalCollection.ReplaceOne(
		ctx,
		bson.M{"userId": userObjectID},
		info,
		opts,
	)

	if err != nil {
		logrus.Errorf("Failed to update medical info: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) GetAllergies(ctx context.Context, userID string) ([]models.MedicalAllergy, error) {
	info, err := er.GetMedicalInfo(ctx, userID)
	if err != nil {
		return nil, err
	}

	if allergies, ok := info["allergies"]; ok {
		// Convert interface{} to []models.MedicalAllergy
		if allergiesList, ok := allergies.([]interface{}); ok {
			var result []models.MedicalAllergy
			for _, allergy := range allergiesList {
				if allergyMap, ok := allergy.(map[string]interface{}); ok {
					result = append(result, models.MedicalAllergy{
						Name:      allergyMap["name"].(string),
						Severity:  allergyMap["severity"].(string),
						Reaction:  allergyMap["reaction"].(string),
						Treatment: allergyMap["treatment"].(string),
					})
				}
			}
			return result, nil
		}
	}

	return []models.MedicalAllergy{}, nil
}

func (er *EmergencyRepository) UpdateAllergies(ctx context.Context, userID string, allergies []models.MedicalAllergy) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = er.medicalCollection.UpdateOne(
		ctx,
		bson.M{"userId": userObjectID},
		bson.M{"$set": bson.M{
			"allergies": allergies,
			"updatedAt": time.Now(),
		}},
		options.Update().SetUpsert(true),
	)

	if err != nil {
		logrus.Errorf("Failed to update allergies: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) GetMedications(ctx context.Context, userID string) ([]models.Medication, error) {
	info, err := er.GetMedicalInfo(ctx, userID)
	if err != nil {
		return nil, err
	}

	if medications, ok := info["medications"]; ok {
		if medicationsList, ok := medications.([]interface{}); ok {
			var result []models.Medication
			for _, medication := range medicationsList {
				if medicationMap, ok := medication.(map[string]interface{}); ok {
					result = append(result, models.Medication{
						Name:       medicationMap["name"].(string),
						Dosage:     medicationMap["dosage"].(string),
						Frequency:  medicationMap["frequency"].(string),
						Purpose:    medicationMap["purpose"].(string),
						Prescriber: medicationMap["prescriber"].(string),
					})
				}
			}
			return result, nil
		}
	}

	return []models.Medication{}, nil
}

func (er *EmergencyRepository) UpdateMedications(ctx context.Context, userID string, medications []models.Medication) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = er.medicalCollection.UpdateOne(
		ctx,
		bson.M{"userId": userObjectID},
		bson.M{"$set": bson.M{
			"medications": medications,
			"updatedAt":   time.Now(),
		}},
		options.Update().SetUpsert(true),
	)

	if err != nil {
		logrus.Errorf("Failed to update medications: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) GetMedicalConditions(ctx context.Context, userID string) ([]models.MedicalCondition, error) {
	info, err := er.GetMedicalInfo(ctx, userID)
	if err != nil {
		return nil, err
	}

	if conditions, ok := info["conditions"]; ok {
		if conditionsList, ok := conditions.([]interface{}); ok {
			var result []models.MedicalCondition
			for _, condition := range conditionsList {
				if conditionMap, ok := condition.(map[string]interface{}); ok {
					result = append(result, models.MedicalCondition{
						Name:      conditionMap["name"].(string),
						Severity:  conditionMap["severity"].(string),
						Treatment: conditionMap["treatment"].(string),
						Notes:     conditionMap["notes"].(string),
						Diagnosed: conditionMap["diagnosed"].(string),
					})
				}
			}
			return result, nil
		}
	}

	return []models.MedicalCondition{}, nil
}

func (er *EmergencyRepository) UpdateMedicalConditions(ctx context.Context, userID string, conditions []models.MedicalCondition) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = er.medicalCollection.UpdateOne(
		ctx,
		bson.M{"userId": userObjectID},
		bson.M{"$set": bson.M{
			"conditions": conditions,
			"updatedAt":  time.Now(),
		}},
		options.Update().SetUpsert(true),
	)

	if err != nil {
		logrus.Errorf("Failed to update medical conditions: %v", err)
		return err
	}

	return nil
}

// =================== BROADCAST OPERATIONS ===================

func (er *EmergencyRepository) CreateBroadcast(ctx context.Context, broadcast map[string]interface{}) error {
	broadcast["createdAt"] = time.Now()
	broadcast["updatedAt"] = time.Now()

	_, err := er.broadcastCollection.InsertOne(ctx, broadcast)
	if err != nil {
		logrus.Errorf("Failed to create broadcast: %v", err)
		return err
	}

	return nil
}

func (er *EmergencyRepository) GetUserBroadcasts(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"senderId": userID},
			{"recipients": bson.M{"$in": []string{userID}}},
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := er.broadcastCollection.Find(ctx, filter, opts)
	if err != nil {
		logrus.Errorf("Failed to get user broadcasts: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var broadcasts []map[string]interface{}
	if err = cursor.All(ctx, &broadcasts); err != nil {
		logrus.Errorf("Failed to decode user broadcasts: %v", err)
		return nil, err
	}

	return broadcasts, nil
}

func (er *EmergencyRepository) GetBroadcast(ctx context.Context, broadcastID string) (map[string]interface{}, error) {
	var broadcast map[string]interface{}
	err := er.broadcastCollection.FindOne(ctx, bson.M{"broadcastId": broadcastID}).Decode(&broadcast)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("broadcast not found")
		}
		logrus.Errorf("Failed to get broadcast: %v", err)
		return nil, err
	}

	return broadcast, nil
}

func (er *EmergencyRepository) UpdateBroadcast(ctx context.Context, userID, broadcastID string, updateFields bson.M) error {
	updateFields["updatedAt"] = time.Now()

	result, err := er.broadcastCollection.UpdateOne(
		ctx,
		bson.M{"broadcastId": broadcastID, "senderId": userID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		logrus.Errorf("Failed to update broadcast: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("broadcast not found")
	}

	return nil
}

func (er *EmergencyRepository) DeleteBroadcast(ctx context.Context, userID, broadcastID string) error {
	result, err := er.broadcastCollection.DeleteOne(ctx, bson.M{
		"broadcastId": broadcastID,
		"senderId":    userID,
	})

	if err != nil {
		logrus.Errorf("Failed to delete broadcast: %v", err)
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("broadcast not found")
	}

	return nil
}

func (er *EmergencyRepository) CreateBroadcastAck(ctx context.Context, ack map[string]interface{}) error {
	ack["createdAt"] = time.Now()

	_, err := er.database.Collection("broadcast_acknowledgments").InsertOne(ctx, ack)
	if err != nil {
		logrus.Errorf("Failed to create broadcast acknowledgment: %v", err)
		return err
	}

	return nil
}

// =================== MEDIA OPERATIONS ===================

func (er *EmergencyRepository) AddMedia(ctx context.Context, emergencyID string, media models.EmergencyMedia) error {
	media.ID = primitive.NewObjectID()
	media.UploadedAt = time.Now()

	// Add media to emergency document
	emergencyObjectID, err := primitive.ObjectIDFromHex(emergencyID)
	if err != nil {
		return errors.New("invalid emergency ID")
	}

	_, err = er.emergencyCollection.UpdateOne(
		ctx,
		bson.M{"_id": emergencyObjectID},
		bson.M{"$push": bson.M{"media": media}},
	)

	if err != nil {
		logrus.Errorf("Failed to add media to emergency: %v", err)
		return err
	}

	// Create media event
	event := models.EmergencyEvent{
		ID:          primitive.NewObjectID(),
		Type:        "media_added",
		Description: fmt.Sprintf("Media added: %s", media.FileName),
		Actor:       media.UploadedBy,
		Timestamp:   time.Now(),
		Data: map[string]interface{}{
			"emergencyId": emergencyID,
			"mediaType":   media.Type,
			"fileName":    media.FileName,
		},
	}

	er.AddEmergencyEvent(ctx, emergencyID, event)

	return nil
}

// =================== HELPER METHODS ===================

func (er *EmergencyRepository) cleanupEmergencyData(ctx context.Context, emergencyID string) {
	// Clean up related data when emergency is deleted
	collections := []string{
		"emergency_responses",
		"emergency_events",
		"help_requests",
		"help_offers",
	}

	for _, collectionName := range collections {
		collection := er.database.Collection(collectionName)
		collection.DeleteMany(ctx, bson.M{"emergencyId": emergencyID})
	}
}

// =================== INDEXES AND SETUP ===================

func (er *EmergencyRepository) CreateIndexes(ctx context.Context) error {
	// Emergency collection indexes
	emergencyIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "circleId", Value: 1}, {Key: "createdAt", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "priority", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "location.coordinates", Value: "2dsphere"}},
		},
	}

	_, err := er.emergencyCollection.Indexes().CreateMany(ctx, emergencyIndexes)
	if err != nil {
		logrus.Errorf("Failed to create emergency indexes: %v", err)
		return err
	}

	// Settings collection indexes
	settingsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = er.settingsCollection.Indexes().CreateMany(ctx, settingsIndexes)
	if err != nil {
		logrus.Errorf("Failed to create settings indexes: %v", err)
		return err
	}

	// Contacts collection indexes
	contactsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userId", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "contactId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = er.contactsCollection.Indexes().CreateMany(ctx, contactsIndexes)
	if err != nil {
		logrus.Errorf("Failed to create contacts indexes: %v", err)
		return err
	}

	// Events collection indexes
	eventsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "data.emergencyId", Value: 1}, {Key: "timestamp", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "timestamp", Value: -1}},
		},
	}

	_, err = er.eventsCollection.Indexes().CreateMany(ctx, eventsIndexes)
	if err != nil {
		logrus.Errorf("Failed to create events indexes: %v", err)
		return err
	}

	logrus.Info("Emergency repository indexes created successfully")
	return nil
}
