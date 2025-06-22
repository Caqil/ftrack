package repositories

import (
	"context"
	"errors"
	"ftrack/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CircleRepository struct {
	collection *mongo.Collection
	database   *mongo.Database
}

func NewCircleRepository(db *mongo.Database) *CircleRepository {
	return &CircleRepository{
		collection: db.Collection("circles"),
		database:   db,
	}
}

// ========================
// Collection Getters
// ========================

func (cr *CircleRepository) GetInvitationCollection() *mongo.Collection {
	return cr.database.Collection("circle_invitations")
}

func (cr *CircleRepository) GetJoinRequestCollection() *mongo.Collection {
	return cr.database.Collection("circle_join_requests")
}

func (cr *CircleRepository) GetAnnouncementCollection() *mongo.Collection {
	return cr.database.Collection("circle_announcements")
}

func (cr *CircleRepository) GetActivityCollection() *mongo.Collection {
	return cr.database.Collection("circle_activities")
}

func (cr *CircleRepository) GetExportJobCollection() *mongo.Collection {
	return cr.database.Collection("circle_export_jobs")
}

// ========================
// Basic Circle CRUD
// ========================

func (cr *CircleRepository) Create(ctx context.Context, circle *models.Circle) error {
	circle.ID = primitive.NewObjectID()
	circle.CreatedAt = time.Now()
	circle.UpdatedAt = time.Now()

	// Initialize default settings
	if circle.Settings.MaxMembers == 0 {
		circle.Settings.MaxMembers = 20
	}

	_, err := cr.collection.InsertOne(ctx, circle)
	return err
}

func (cr *CircleRepository) GetByID(ctx context.Context, id string) (*models.Circle, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	var circle models.Circle
	err = cr.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&circle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("circle not found")
		}
		return nil, err
	}

	return &circle, nil
}

func (cr *CircleRepository) GetByInviteCode(ctx context.Context, inviteCode string) (*models.Circle, error) {
	var circle models.Circle
	err := cr.collection.FindOne(ctx, bson.M{"inviteCode": inviteCode}).Decode(&circle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("circle not found")
		}
		return nil, err
	}

	return &circle, nil
}

func (cr *CircleRepository) GetUserCircles(ctx context.Context, userID string) ([]models.Circle, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	filter := bson.M{
		"members.userId": objectID,
	}

	cursor, err := cr.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var circles []models.Circle
	err = cursor.All(ctx, &circles)
	return circles, err
}

func (cr *CircleRepository) Update(ctx context.Context, id string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	update["updatedAt"] = time.Now()

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle not found")
	}

	return nil
}

func (cr *CircleRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	result, err := cr.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("circle not found")
	}

	return nil
}

// ========================
// Member Management
// ========================

func (cr *CircleRepository) AddMember(ctx context.Context, circleID string, member models.CircleMember) error {
	objectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	member.JoinedAt = time.Now()
	member.LastActivity = time.Now()

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$push": bson.M{"members": member},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle not found")
	}

	return nil
}

func (cr *CircleRepository) RemoveMember(ctx context.Context, circleID, userID string) error {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{"_id": circleObjectID},
		bson.M{
			"$pull": bson.M{"members": bson.M{"userId": userObjectID}},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle not found")
	}

	return nil
}

func (cr *CircleRepository) UpdateMemberPermissions(ctx context.Context, circleID, userID string, permissions models.MemberPermissions) error {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":            circleObjectID,
			"members.userId": userObjectID,
		},
		bson.M{
			"$set": bson.M{
				"members.$.permissions": permissions,
				"updatedAt":             time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle or member not found")
	}

	return nil
}

func (cr *CircleRepository) UpdateMemberRole(ctx context.Context, circleID, userID, role string) error {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":            circleObjectID,
			"members.userId": userObjectID,
		},
		bson.M{
			"$set": bson.M{
				"members.$.role": role,
				"updatedAt":      time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle or member not found")
	}

	return nil
}

func (cr *CircleRepository) UpdateLastActivity(ctx context.Context, circleID, userID string) error {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = cr.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":            circleObjectID,
			"members.userId": userObjectID,
		},
		bson.M{
			"$set": bson.M{
				"members.$.lastActivity": time.Now(),
				"stats.lastActivity":     time.Now(),
				"updatedAt":              time.Now(),
			},
		},
	)

	return err
}

func (cr *CircleRepository) IsMember(ctx context.Context, circleID, userID string) (bool, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return false, errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, errors.New("invalid user ID")
	}

	count, err := cr.collection.CountDocuments(ctx, bson.M{
		"_id":            circleObjectID,
		"members.userId": userObjectID,
	})

	return count > 0, err
}

func (cr *CircleRepository) GetMemberRole(ctx context.Context, circleID, userID string) (string, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return "", errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "", errors.New("invalid user ID")
	}

	// Use aggregation to get specific member info
	pipeline := []bson.M{
		{"$match": bson.M{"_id": circleObjectID}},
		{"$unwind": "$members"},
		{"$match": bson.M{"members.userId": userObjectID}},
		{"$project": bson.M{"_id": 0, "role": "$members.role"}},
	}

	cursor, err := cr.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return "", err
	}
	defer cursor.Close(ctx)

	var result struct {
		Role string `bson:"role"`
	}

	if cursor.Next(ctx) {
		err = cursor.Decode(&result)
		if err != nil {
			return "", err
		}
		return result.Role, nil
	}

	return "", errors.New("member not found")
}

// ========================
// Invitation Management
// ========================

func (cr *CircleRepository) CreateInvitation(ctx context.Context, invitation *models.CircleInvitation) error {
	invitationCollection := cr.GetInvitationCollection()

	invitation.ID = primitive.NewObjectID()
	invitation.CreatedAt = time.Now()
	invitation.UpdatedAt = time.Now()
	invitation.Status = "pending"

	// Set expiration time (7 days from now)
	invitation.ExpiresAt = time.Now().AddDate(0, 0, 7)

	_, err := invitationCollection.InsertOne(ctx, invitation)
	return err
}

func (cr *CircleRepository) GetInvitationByID(ctx context.Context, invitationID string) (*models.CircleInvitation, error) {
	objectID, err := primitive.ObjectIDFromHex(invitationID)
	if err != nil {
		return nil, errors.New("invalid invitation ID")
	}

	invitationCollection := cr.GetInvitationCollection()

	var invitation models.CircleInvitation
	err = invitationCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&invitation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invitation not found")
		}
		return nil, err
	}

	return &invitation, nil
}

func (cr *CircleRepository) UpdateInvitationStatus(ctx context.Context, invitationID, status string) error {
	objectID, err := primitive.ObjectIDFromHex(invitationID)
	if err != nil {
		return errors.New("invalid invitation ID")
	}

	invitationCollection := cr.GetInvitationCollection()

	result, err := invitationCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$set": bson.M{
				"status":    status,
				"updatedAt": time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("invitation not found")
	}

	return nil
}

func (cr *CircleRepository) GetUserInvitations(ctx context.Context, userID primitive.ObjectID, status string) ([]models.CircleInvitation, error) {
	invitationCollection := cr.GetInvitationCollection()

	filter := bson.M{"inviteeId": userID}

	// Add status filter if provided
	if status != "" {
		filter["status"] = status
	}

	cursor, err := invitationCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.CircleInvitation
	err = cursor.All(ctx, &invitations)
	if err != nil {
		return nil, err
	}

	return invitations, nil
}

func (cr *CircleRepository) GetCircleInvitations(ctx context.Context, circleID string) ([]models.CircleInvitation, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	invitationCollection := cr.GetInvitationCollection()

	cursor, err := invitationCollection.Find(ctx, bson.M{"circleId": circleObjectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.CircleInvitation
	err = cursor.All(ctx, &invitations)
	return invitations, err
}

func (cr *CircleRepository) DeleteInvitation(ctx context.Context, invitationID string) error {
	objectID, err := primitive.ObjectIDFromHex(invitationID)
	if err != nil {
		return errors.New("invalid invitation ID")
	}

	invitationCollection := cr.GetInvitationCollection()
	_, err = invitationCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (cr *CircleRepository) UpdateInvitation(ctx context.Context, invitationID string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(invitationID)
	if err != nil {
		return errors.New("invalid invitation ID")
	}

	update["updatedAt"] = time.Now()
	invitationCollection := cr.GetInvitationCollection()

	result, err := invitationCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("invitation not found")
	}

	return nil
}

// ========================
// Join Request Management
// ========================

func (cr *CircleRepository) CreateJoinRequest(ctx context.Context, joinRequest *models.JoinRequest) error {
	joinRequestCollection := cr.GetJoinRequestCollection()

	joinRequest.ID = primitive.NewObjectID()
	joinRequest.CreatedAt = time.Now()
	joinRequest.UpdatedAt = time.Now()
	joinRequest.Status = "pending"

	_, err := joinRequestCollection.InsertOne(ctx, joinRequest)
	return err
}

func (cr *CircleRepository) GetJoinRequestByID(ctx context.Context, requestID string) (*models.JoinRequest, error) {
	objectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return nil, errors.New("invalid request ID")
	}

	joinRequestCollection := cr.GetJoinRequestCollection()

	var joinRequest models.JoinRequest
	err = joinRequestCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&joinRequest)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("join request not found")
		}
		return nil, err
	}

	return &joinRequest, nil
}

func (cr *CircleRepository) GetCircleJoinRequests(ctx context.Context, circleID string) ([]models.JoinRequest, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	joinRequestCollection := cr.GetJoinRequestCollection()

	cursor, err := joinRequestCollection.Find(ctx, bson.M{"circleId": circleObjectID, "status": "pending"})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var joinRequests []models.JoinRequest
	err = cursor.All(ctx, &joinRequests)
	return joinRequests, err
}

func (cr *CircleRepository) UpdateJoinRequestStatus(ctx context.Context, requestID, status string) error {
	objectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return errors.New("invalid request ID")
	}

	joinRequestCollection := cr.GetJoinRequestCollection()

	result, err := joinRequestCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$set": bson.M{
				"status":    status,
				"updatedAt": time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("join request not found")
	}

	return nil
}

func (cr *CircleRepository) DeleteJoinRequest(ctx context.Context, requestID string) error {
	objectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return errors.New("invalid request ID")
	}

	joinRequestCollection := cr.GetJoinRequestCollection()
	_, err = joinRequestCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (cr *CircleRepository) GetUserJoinRequests(ctx context.Context, userID string) ([]models.JoinRequest, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	joinRequestCollection := cr.GetJoinRequestCollection()

	cursor, err := joinRequestCollection.Find(ctx, bson.M{"userId": userObjectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var joinRequests []models.JoinRequest
	err = cursor.All(ctx, &joinRequests)
	return joinRequests, err
}

// ========================
// Announcement Management
// ========================

func (cr *CircleRepository) CreateAnnouncement(ctx context.Context, announcement *models.CircleAnnouncement) error {
	announcementCollection := cr.GetAnnouncementCollection()

	announcement.ID = primitive.NewObjectID()
	announcement.CreatedAt = time.Now()
	announcement.UpdatedAt = time.Now()

	_, err := announcementCollection.InsertOne(ctx, announcement)
	return err
}

func (cr *CircleRepository) GetCircleAnnouncements(ctx context.Context, circleID string, page, pageSize int) ([]models.CircleAnnouncement, error) {
	_, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	announcementCollection := cr.GetAnnouncementCollection()

	skip := (page - 1) * pageSize
	cursor, err := announcementCollection.Find(
		ctx,
		&options.FindOptions{
			Sort:  bson.M{"createdAt": -1},
			Skip:  &[]int64{int64(skip)}[0],
			Limit: &[]int64{int64(pageSize)}[0],
		},
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var announcements []models.CircleAnnouncement
	err = cursor.All(ctx, &announcements)
	return announcements, err
}

func (cr *CircleRepository) GetAnnouncementByID(ctx context.Context, announcementID string) (*models.CircleAnnouncement, error) {
	objectID, err := primitive.ObjectIDFromHex(announcementID)
	if err != nil {
		return nil, errors.New("invalid announcement ID")
	}

	announcementCollection := cr.GetAnnouncementCollection()

	var announcement models.CircleAnnouncement
	err = announcementCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&announcement)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("announcement not found")
		}
		return nil, err
	}

	return &announcement, nil
}

func (cr *CircleRepository) UpdateAnnouncement(ctx context.Context, announcementID string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(announcementID)
	if err != nil {
		return errors.New("invalid announcement ID")
	}

	update["updatedAt"] = time.Now()
	announcementCollection := cr.GetAnnouncementCollection()

	result, err := announcementCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("announcement not found")
	}

	return nil
}

func (cr *CircleRepository) DeleteAnnouncement(ctx context.Context, announcementID string) error {
	objectID, err := primitive.ObjectIDFromHex(announcementID)
	if err != nil {
		return errors.New("invalid announcement ID")
	}

	announcementCollection := cr.GetAnnouncementCollection()
	_, err = announcementCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// ========================
// Activity Management
// ========================

func (cr *CircleRepository) CreateActivity(ctx context.Context, activity *models.CircleActivity) error {
	activityCollection := cr.GetActivityCollection()

	activity.ID = primitive.NewObjectID()
	activity.CreatedAt = time.Now()

	_, err := activityCollection.InsertOne(ctx, activity)
	return err
}

func (cr *CircleRepository) GetCircleActivity(ctx context.Context, circleID string, page, pageSize int, activityType string) ([]models.CircleActivity, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	activityCollection := cr.GetActivityCollection()

	filter := bson.M{"circleId": circleObjectID}
	if activityType != "" {
		filter["type"] = activityType
	}

	skip := (page - 1) * pageSize
	cursor, err := activityCollection.Find(
		ctx,
		filter,
		&options.FindOptions{
			Sort:  bson.M{"createdAt": -1},
			Skip:  &[]int64{int64(skip)}[0],
			Limit: &[]int64{int64(pageSize)}[0],
		},
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var activities []models.CircleActivity
	err = cursor.All(ctx, &activities)
	return activities, err
}

func (cr *CircleRepository) GetMemberActivity(ctx context.Context, circleID, userID string, page, pageSize int) ([]models.CircleActivity, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	activityCollection := cr.GetActivityCollection()

	skip := (page - 1) * pageSize
	cursor, err := activityCollection.Find(
		ctx,
		bson.M{"circleId": circleObjectID, "userId": userObjectID},
		&options.FindOptions{
			Sort:  bson.M{"createdAt": -1},
			Skip:  &[]int64{int64(skip)}[0],
			Limit: &[]int64{int64(pageSize)}[0],
		},
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var activities []models.CircleActivity
	err = cursor.All(ctx, &activities)
	return activities, err
}

// ========================
// Export Job Management
// ========================

func (cr *CircleRepository) CreateExportJob(ctx context.Context, exportJob *models.ExportJob) error {
	exportCollection := cr.GetExportJobCollection()

	exportJob.ID = primitive.NewObjectID()
	exportJob.CreatedAt = time.Now()
	exportJob.UpdatedAt = time.Now()
	exportJob.Status = "pending"
	exportJob.Progress = 0
	exportJob.ExpiresAt = time.Now().AddDate(0, 0, 7) // 7 days from now

	_, err := exportCollection.InsertOne(ctx, exportJob)
	return err
}

func (cr *CircleRepository) GetExportJobByID(ctx context.Context, jobID string) (*models.ExportJob, error) {
	objectID, err := primitive.ObjectIDFromHex(jobID)
	if err != nil {
		return nil, errors.New("invalid job ID")
	}

	exportCollection := cr.GetExportJobCollection()

	var exportJob models.ExportJob
	err = exportCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&exportJob)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("export job not found")
		}
		return nil, err
	}

	return &exportJob, nil
}

func (cr *CircleRepository) UpdateExportJob(ctx context.Context, jobID string, update bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(jobID)
	if err != nil {
		return errors.New("invalid job ID")
	}

	update["updatedAt"] = time.Now()
	exportCollection := cr.GetExportJobCollection()

	result, err := exportCollection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("export job not found")
	}

	return nil
}

func (cr *CircleRepository) GetUserExportJobs(ctx context.Context, userID string) ([]models.ExportJob, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	exportCollection := cr.GetExportJobCollection()

	cursor, err := exportCollection.Find(
		ctx,
		bson.M{"userId": userObjectID},
		&options.FindOptions{
			Sort: bson.M{"createdAt": -1},
		},
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var exportJobs []models.ExportJob
	err = cursor.All(ctx, &exportJobs)
	return exportJobs, err
}

// ========================
// Search and Discovery
// ========================

func (cr *CircleRepository) GetPublicCircles(ctx context.Context, page, pageSize int, category string) ([]models.Circle, error) {
	filter := bson.M{"settings.isPublic": true}
	if category != "" {
		filter["category"] = category
	}

	skip := (page - 1) * pageSize
	cursor, err := cr.collection.Find(
		ctx,
		filter,
		&options.FindOptions{
			Sort:  bson.M{"stats.totalMembers": -1},
			Skip:  &[]int64{int64(skip)}[0],
			Limit: &[]int64{int64(pageSize)}[0],
		},
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var circles []models.Circle
	err = cursor.All(ctx, &circles)
	return circles, err
}

func (cr *CircleRepository) SearchCircles(ctx context.Context, query string, filters []string, page, pageSize int) ([]models.Circle, error) {
	filter := bson.M{
		"settings.isPublic": true,
		"$text":             bson.M{"$search": query},
	}

	// Add additional filters if provided
	if len(filters) > 0 {
		filter["category"] = bson.M{"$in": filters}
	}

	skip := (page - 1) * pageSize
	cursor, err := cr.collection.Find(
		ctx,
		filter,
		&options.FindOptions{
			Sort:  bson.M{"score": bson.M{"$meta": "textScore"}},
			Skip:  &[]int64{int64(skip)}[0],
			Limit: &[]int64{int64(pageSize)}[0],
		},
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var circles []models.Circle
	err = cursor.All(ctx, &circles)
	return circles, err
}

// ========================
// Statistics
// ========================

func (cr *CircleRepository) GetCircleStats(ctx context.Context, circleID string) (*models.CircleStats, error) {
	circle, err := cr.GetByID(ctx, circleID)
	if err != nil {
		return nil, err
	}

	return &circle.Stats, nil
}

func (cr *CircleRepository) UpdateCircleStats(ctx context.Context, circleID string, stats models.CircleStats) error {
	objectID, err := primitive.ObjectIDFromHex(circleID)
	if err != nil {
		return errors.New("invalid circle ID")
	}

	result, err := cr.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$set": bson.M{
				"stats":     stats,
				"updatedAt": time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("circle not found")
	}

	return nil
}
