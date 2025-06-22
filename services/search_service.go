package services

import (
	"context"
	"errors"
	"ftrack/models"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SearchService struct {
	messageCollection *mongo.Collection
	db                *mongo.Database
}

func NewSearchService(db *mongo.Database) *SearchService {
	return &SearchService{
		messageCollection: db.Collection("messages"),
		db:                db,
	}
}

func (ss *SearchService) SearchMessages(ctx context.Context, req models.SearchMessagesRequest, circleIDs []string) (*models.SearchResponse, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	filter := bson.M{
		"circleId":  bson.M{"$in": circleObjectIDs},
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	// Add text search
	if req.Query != "" {
		filter["$text"] = bson.M{"$search": req.Query}
	}

	// Add message type filter
	if req.MessageType != "" {
		filter["type"] = req.MessageType
	}

	// Add date range filters
	if req.DateFrom != "" || req.DateTo != "" {
		dateFilter := bson.M{}

		if req.DateFrom != "" {
			if fromDate, err := time.Parse("2006-01-02", req.DateFrom); err == nil {
				dateFilter["$gte"] = fromDate
			}
		}

		if req.DateTo != "" {
			if toDate, err := time.Parse("2006-01-02", req.DateTo); err == nil {
				dateFilter["$lte"] = toDate.Add(24 * time.Hour) // End of day
			}
		}

		if len(dateFilter) > 0 {
			filter["createdAt"] = dateFilter
		}
	}

	// Get total count
	total, err := ss.messageCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Get messages
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	// Add text search score sorting if searching
	if req.Query != "" {
		opts.SetSort(bson.D{{"score", bson.M{"$meta": "textScore"}}, {"createdAt", -1}})
	}

	cursor, err := ss.messageCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	return &models.SearchResponse{
		Messages:    messages,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		Query:       req.Query,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ss *SearchService) SearchInCircle(ctx context.Context, req models.SearchInCircleRequest) (*models.SearchResponse, error) {
	circleObjectID, err := primitive.ObjectIDFromHex(req.CircleID)
	if err != nil {
		return nil, errors.New("invalid circle ID")
	}

	filter := bson.M{
		"circleId":  circleObjectID,
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	// Add text search
	if req.Query != "" {
		filter["$text"] = bson.M{"$search": req.Query}
	}

	// Get total count
	total, err := ss.messageCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Get messages
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	// Add text search score sorting if searching
	if req.Query != "" {
		opts.SetSort(bson.D{{Key: "score", Value: bson.M{"$meta": "textScore"}}, {Key: "createdAt", Value: -1}})
	}

	cursor, err := ss.messageCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	return &models.SearchResponse{
		Messages:    messages,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		Query:       req.Query,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ss *SearchService) SearchMedia(ctx context.Context, req models.SearchMediaRequest, circleIDs []string) (*models.MediaSearchResponse, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	filter := bson.M{
		"circleId":  bson.M{"$in": circleObjectIDs},
		"media.url": bson.M{"$exists": true, "$ne": ""},
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	if req.MediaType != "" {
		filter["media.type"] = req.MediaType
	}

	if req.CircleID != "" {
		circleObjectID, err := primitive.ObjectIDFromHex(req.CircleID)
		if err == nil {
			filter["circleId"] = circleObjectID
		}
	}

	// Get total count
	total, err := ss.messageCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Get messages with media
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := ss.messageCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	// Convert to media response
	media := make([]models.MessageMediaExtended, len(messages))
	for i, msg := range messages {
		media[i] = models.MessageMediaExtended{
			MessageMedia: msg.Media,
			ID:           primitive.NewObjectID(),
			CreatedAt:    msg.CreatedAt,
			UpdatedAt:    msg.UpdatedAt,
		}
	}

	return &models.MediaSearchResponse{
		Media:       media,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ss *SearchService) SearchMentions(ctx context.Context, userID string, req models.SearchMentionsRequest, circleIDs []string) (*models.SearchResponse, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	// Search for messages that mention the user
	// This could be by @username, user ID, or other mention patterns
	mentionPatterns := []string{
		"@" + userID,
		userID, // Direct user ID mention
	}

	// Get user info for additional mention patterns
	userCollection := ss.db.Collection("users")
	var user models.User
	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	err := userCollection.FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&user)
	if err == nil {
		// Add name-based mentions
		if user.FirstName != "" {
			mentionPatterns = append(mentionPatterns, "@"+user.FirstName)
		}
		if user.FirstName != "" {
			mentionPatterns = append(mentionPatterns, "@"+user.FirstName)
		}
	}

	filter := bson.M{
		"circleId":  bson.M{"$in": circleObjectIDs},
		"senderId":  bson.M{"$ne": userObjectID}, // Exclude own messages
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
		"$or":       []bson.M{},
	}

	// Add mention patterns to filter
	orConditions := make([]bson.M, len(mentionPatterns))
	for i, pattern := range mentionPatterns {
		orConditions[i] = bson.M{"content": bson.M{"$regex": regexp.QuoteMeta(pattern), "$options": "i"}}
	}
	filter["$or"] = orConditions

	if req.CircleID != "" {
		circleObjectID, err := primitive.ObjectIDFromHex(req.CircleID)
		if err == nil {
			filter["circleId"] = circleObjectID
		}
	}

	// Get total count
	total, err := ss.messageCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Get messages
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := ss.messageCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	return &models.SearchResponse{
		Messages:    messages,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		Query:       "mentions",
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ss *SearchService) SearchLinks(ctx context.Context, req models.SearchLinksRequest, circleIDs []string) (*models.LinksSearchResponse, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	// Regex pattern to match URLs
	urlPattern := `https?://[^\s]+`

	filter := bson.M{
		"circleId":  bson.M{"$in": circleObjectIDs},
		"content":   bson.M{"$regex": urlPattern, "$options": "i"},
		"isDeleted": bson.M{"$ne": true},
		"isHidden":  bson.M{"$ne": true},
	}

	if req.CircleID != "" {
		circleObjectID, err := primitive.ObjectIDFromHex(req.CircleID)
		if err == nil {
			filter["circleId"] = circleObjectID
		}
	}

	if req.Domain != "" {
		// Search for specific domain
		domainPattern := strings.Replace(req.Domain, ".", `\.`, -1)
		filter["content"] = bson.M{"$regex": `https?://[^/]*` + domainPattern, "$options": "i"}
	}

	// Get total count
	total, err := ss.messageCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Get messages
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := ss.messageCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	// Extract links from messages
	links := ss.extractLinksFromMessages(messages)

	return &models.LinksSearchResponse{
		Links:       links,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ss *SearchService) SearchFiles(ctx context.Context, req models.SearchFilesRequest, circleIDs []string) (*models.FilesSearchResponse, error) {
	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	filter := bson.M{
		"circleId":       bson.M{"$in": circleObjectIDs},
		"media.url":      bson.M{"$exists": true, "$ne": ""},
		"media.filename": bson.M{"$exists": true, "$ne": ""},
		"isDeleted":      bson.M{"$ne": true},
		"isHidden":       bson.M{"$ne": true},
	}

	// Filter by file type
	if req.FileType != "" {
		switch req.FileType {
		case "document":
			filter["media.mimeType"] = bson.M{"$regex": "application/", "$options": "i"}
		case "image":
			filter["media.mimeType"] = bson.M{"$regex": "image/", "$options": "i"}
		case "video":
			filter["media.mimeType"] = bson.M{"$regex": "video/", "$options": "i"}
		case "audio":
			filter["media.mimeType"] = bson.M{"$regex": "audio/", "$options": "i"}
		default:
			filter["media.type"] = req.FileType
		}
	}

	if req.CircleID != "" {
		circleObjectID, err := primitive.ObjectIDFromHex(req.CircleID)
		if err == nil {
			filter["circleId"] = circleObjectID
		}
	}

	// Get total count
	total, err := ss.messageCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Get messages
	skip := (req.Page - 1) * req.PageSize
	opts := options.Find().
		SetSort(bson.D{{"createdAt", -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize))

	cursor, err := ss.messageCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	// Convert to file info
	files := make([]models.FileInfo, len(messages))
	for i, msg := range messages {
		files[i] = models.FileInfo{
			FileID:    msg.Media.URL, // Using URL as ID for now
			Filename:  msg.Media.Filename,
			Size:      msg.Media.Size,
			Type:      msg.Media.Type,
			URL:       msg.Media.URL,
			MessageID: msg.ID.Hex(),
			SenderID:  msg.SenderID.Hex(),
			CircleID:  msg.CircleID.Hex(),
			CreatedAt: msg.CreatedAt,
		}
	}

	return &models.FilesSearchResponse{
		Files:       files,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		HasNext:     total > int64(req.Page*req.PageSize),
		HasPrevious: req.Page > 1,
	}, nil
}

func (ss *SearchService) extractLinksFromMessages(messages []models.Message) []models.LinkInfo {
	var links []models.LinkInfo
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)

	for _, msg := range messages {
		urls := urlRegex.FindAllString(msg.Content, -1)
		for _, urlStr := range urls {
			// Parse URL to get domain
			parsedURL, err := url.Parse(urlStr)
			if err != nil {
				continue
			}

			link := models.LinkInfo{
				URL:       urlStr,
				Domain:    parsedURL.Host,
				MessageID: msg.ID.Hex(),
				SenderID:  msg.SenderID.Hex(),
				CircleID:  msg.CircleID.Hex(),
				CreatedAt: msg.CreatedAt,
			}

			// Try to extract title and description (this would typically be done
			// by fetching the URL and parsing the HTML, but for now we'll leave it empty)
			link.Title = ss.extractTitleFromURL(urlStr)

			links = append(links, link)
		}
	}

	return links
}

func (ss *SearchService) extractTitleFromURL(urlStr string) string {
	// This is a simplified title extraction
	// In a real implementation, you would fetch the URL and parse the HTML title
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	// Return the path as a simple title
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		return strings.Trim(parsedURL.Path, "/")
	}

	return parsedURL.Host
}

func (ss *SearchService) CreateSearchIndexes(ctx context.Context) error {
	// Create text search index on message content
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{"content", "text"},
			{"media.filename", "text"},
		},
		Options: options.Index().SetName("message_text_search"),
	}

	_, err := ss.messageCollection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		logrus.Errorf("Failed to create text search index: %v", err)
		return err
	}

	// Create other useful indexes
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"circleId", 1}, {"createdAt", -1}},
			Options: options.Index().SetName("circle_created_idx"),
		},
		{
			Keys:    bson.D{{"senderId", 1}, {"createdAt", -1}},
			Options: options.Index().SetName("sender_created_idx"),
		},
		{
			Keys:    bson.D{{"type", 1}, {"circleId", 1}},
			Options: options.Index().SetName("type_circle_idx"),
		},
		{
			Keys:    bson.D{{"media.type", 1}, {"createdAt", -1}},
			Options: options.Index().SetName("media_type_created_idx"),
		},
	}

	_, err = ss.messageCollection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		logrus.Errorf("Failed to create additional indexes: %v", err)
		return err
	}

	logrus.Info("Search indexes created successfully")
	return nil
}

func (ss *SearchService) GetSearchSuggestions(ctx context.Context, query string, circleIDs []string, limit int) ([]string, error) {
	if len(query) < 2 {
		return []string{}, nil
	}

	circleObjectIDs := make([]primitive.ObjectID, len(circleIDs))
	for i, id := range circleIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		circleObjectIDs[i] = objectID
	}

	// Use aggregation to get common terms that start with the query
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"circleId":  bson.M{"$in": circleObjectIDs},
			"content":   bson.M{"$regex": "^" + regexp.QuoteMeta(query), "$options": "i"},
			"isDeleted": bson.M{"$ne": true},
			"isHidden":  bson.M{"$ne": true},
		}}},
		{{"$project", bson.M{
			"words": bson.M{"$split": []interface{}{"$content", " "}},
		}}},
		{{"$unwind", "$words"}},
		{{"$match", bson.M{
			"words": bson.M{"$regex": "^" + regexp.QuoteMeta(query), "$options": "i"},
		}}},
		{{"$group", bson.M{
			"_id":   bson.M{"$toLower": "$words"},
			"count": bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.M{"count": -1}}},
		{{"$limit", limit}},
		{{"$project", bson.M{"_id": 1}}},
	}

	cursor, err := ss.messageCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		ID string `bson:"_id"`
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}

	suggestions := make([]string, len(results))
	for i, result := range results {
		suggestions[i] = result.ID
	}

	return suggestions, nil
}
