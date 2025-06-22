package workers

import (
	"context"
	"fmt"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/services"
	"ftrack/utils"
	"ftrack/websocket"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

type GeofenceWorker struct {
	// Dependencies
	db    *mongo.Database
	redis *redis.Client
	hub   *websocket.Hub

	// Services
	geofenceService     *services.GeofenceService
	placeService        *services.PlaceService
	circleService       *services.CircleService
	notificationService *services.NotificationService

	// Repositories
	placeRepo    *repositories.PlaceRepository
	locationRepo *repositories.LocationRepository
	circleRepo   *repositories.CircleRepository
	userRepo     *repositories.UserRepository

	// Worker configuration
	config GeofenceWorkerConfig

	// Processing channels
	geofenceQueue chan GeofenceJob

	// Cache for places and user locations
	placesCache    map[string][]models.Place  // userID -> places
	locationsCache map[string]models.Location // userID -> last location
	cacheMutex     sync.RWMutex

	// Worker state
	isRunning bool
	workers   int
	mutex     sync.RWMutex

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	stats      GeofenceWorkerStats
	statsMutex sync.RWMutex
}

type GeofenceWorkerConfig struct {
	WorkerCount              int           `json:"workerCount"`
	QueueSize                int           `json:"queueSize"`
	ProcessingTimeout        time.Duration `json:"processingTimeout"`
	CacheRefreshInterval     time.Duration `json:"cacheRefreshInterval"`
	GeofenceRadius           float64       `json:"geofenceRadius"` // Default radius in meters
	EnableNotifications      bool          `json:"enableNotifications"`
	EnableWebSocketBroadcast bool          `json:"enableWebSocketBroadcast"`
	BatchSize                int           `json:"batchSize"`
}

type GeofenceJob struct {
	ID               string                 `json:"id"`
	UserID           string                 `json:"userId"`
	PreviousLocation *models.Location       `json:"previousLocation"`
	CurrentLocation  models.Location        `json:"currentLocation"`
	Timestamp        time.Time              `json:"timestamp"`
	Priority         int                    `json:"priority"`
	Context          map[string]interface{} `json:"context"`
}

type GeofenceEvent struct {
	ID        string          `json:"id"`
	UserID    string          `json:"userId"`
	PlaceID   string          `json:"placeId"`
	Place     models.Place    `json:"place"`
	EventType string          `json:"eventType"` // entry, exit
	Location  models.Location `json:"location"`
	Timestamp time.Time       `json:"timestamp"`
	Distance  float64         `json:"distance"` // Distance from place center
}

type GeofenceWorkerStats struct {
	JobsProcessed      int64     `json:"jobsProcessed"`
	EventsDetected     int64     `json:"eventsDetected"`
	EntriesDetected    int64     `json:"entriesDetected"`
	ExitsDetected      int64     `json:"exitsDetected"`
	NotificationsSent  int64     `json:"notificationsSent"`
	CacheHits          int64     `json:"cacheHits"`
	CacheMisses        int64     `json:"cacheMisses"`
	AverageProcessTime float64   `json:"averageProcessTime"` // ms
	LastProcessedAt    time.Time `json:"lastProcessedAt"`
	QueueLength        int       `json:"queueLength"`
	StartTime          time.Time `json:"startTime"`
}

func NewGeofenceWorker(
	db *mongo.Database,
	redis *redis.Client,
	hub *websocket.Hub,
	geofenceService *services.GeofenceService,
	placeService *services.PlaceService,
	circleService *services.CircleService,
	notificationService *services.NotificationService,
) *GeofenceWorker {
	ctx, cancel := context.WithCancel(context.Background())

	config := GeofenceWorkerConfig{
		WorkerCount:              3,
		QueueSize:                500,
		ProcessingTimeout:        15 * time.Second,
		CacheRefreshInterval:     5 * time.Minute,
		GeofenceRadius:           100, // 100 meters default
		EnableNotifications:      true,
		EnableWebSocketBroadcast: true,
		BatchSize:                20,
	}

	return &GeofenceWorker{
		db:                  db,
		redis:               redis,
		hub:                 hub,
		geofenceService:     geofenceService,
		placeService:        placeService,
		circleService:       circleService,
		notificationService: notificationService,
		placeRepo:           repositories.NewPlaceRepository(db),
		locationRepo:        repositories.NewLocationRepository(db),
		circleRepo:          repositories.NewCircleRepository(db),
		userRepo:            repositories.NewUserRepository(db),
		config:              config,
		geofenceQueue:       make(chan GeofenceJob, config.QueueSize),
		placesCache:         make(map[string][]models.Place),
		locationsCache:      make(map[string]models.Location),
		ctx:                 ctx,
		cancel:              cancel,
		stats: GeofenceWorkerStats{
			StartTime: time.Now(),
		},
	}
}

func (gw *GeofenceWorker) Start() error {
	gw.mutex.Lock()
	defer gw.mutex.Unlock()

	if gw.isRunning {
		return nil
	}

	gw.isRunning = true
	gw.workers = gw.config.WorkerCount

	logrus.Infof("Starting Geofence Worker with %d workers", gw.workers)

	// Start worker goroutines
	for i := 0; i < gw.workers; i++ {
		gw.wg.Add(1)
		go gw.worker(i)
	}

	// Start cache refresher
	gw.wg.Add(1)
	go gw.cacheRefresher()

	// Start metrics collector
	gw.wg.Add(1)
	go gw.metricsCollector()

	logrus.Info("Geofence Worker started successfully")
	return nil
}

func (gw *GeofenceWorker) Stop() error {
	gw.mutex.Lock()
	defer gw.mutex.Unlock()

	if !gw.isRunning {
		return nil
	}

	logrus.Info("Stopping Geofence Worker...")

	gw.cancel()
	gw.isRunning = false

	close(gw.geofenceQueue)
	gw.wg.Wait()

	logrus.Info("Geofence Worker stopped successfully")
	return nil
}

func (gw *GeofenceWorker) SubmitLocationUpdate(userID string, previousLocation *models.Location, currentLocation models.Location) error {
	if !gw.isRunning {
		return utils.NewServiceError("Geofence worker is not running", "GEOFENCE_WORKER_NOT_RUNNING")
	}

	job := GeofenceJob{
		ID:               utils.GenerateUUID(),
		UserID:           userID,
		PreviousLocation: previousLocation,
		CurrentLocation:  currentLocation,
		Timestamp:        time.Now(),
		Priority:         2, // Normal priority
		Context:          make(map[string]interface{}),
	}

	// Higher priority for driving or high-speed movement
	if currentLocation.IsDriving || currentLocation.Speed > 10 { // > 10 m/s (~36 km/h)
		job.Priority = 3
	}

	select {
	case gw.geofenceQueue <- job:
		return nil
	default:
		return utils.NewServiceError("Geofence queue is full", "GEOFENCE_QUEUE_FULL")
	}
}

func (gw *GeofenceWorker) worker(workerID int) {
	defer gw.wg.Done()

	logrus.Infof("Geofence worker %d started", workerID)

	for {
		select {
		case job, ok := <-gw.geofenceQueue:
			if !ok {
				logrus.Infof("Geofence worker %d stopping", workerID)
				return
			}

			gw.processGeofenceJob(job, workerID)

		case <-gw.ctx.Done():
			logrus.Infof("Geofence worker %d stopping due to context cancellation", workerID)
			return
		}
	}
}

func (gw *GeofenceWorker) processGeofenceJob(job GeofenceJob, workerID int) {
	startTime := time.Now()

	defer func() {
		duration := time.Since(startTime)
		gw.updateStats(duration)
	}()

	ctx, cancel := context.WithTimeout(gw.ctx, gw.config.ProcessingTimeout)
	defer cancel()

	logrus.Debugf("Worker %d processing geofence job for user %s", workerID, job.UserID)

	// Get user's places from cache or database
	places, err := gw.getUserPlaces(ctx, job.UserID)
	if err != nil {
		logrus.Errorf("Failed to get places for user %s: %v", job.UserID, err)
		return
	}

	if len(places) == 0 {
		logrus.Debugf("No places found for user %s", job.UserID)
		return
	}

	// Detect geofence events
	events := gw.detectGeofenceEvents(job, places)

	if len(events) == 0 {
		logrus.Debugf("No geofence events detected for user %s", job.UserID)
		return
	}

	// Process each event
	for _, event := range events {
		gw.processGeofenceEvent(ctx, event)
	}

	// Update location cache
	gw.updateLocationCache(job.UserID, job.CurrentLocation)

	logrus.Debugf("Worker %d processed %d geofence events for user %s", workerID, len(events), job.UserID)
}

func (gw *GeofenceWorker) detectGeofenceEvents(job GeofenceJob, places []models.Place) []GeofenceEvent {
	var events []GeofenceEvent

	current := job.CurrentLocation

	// If no previous location, only check for entries
	if job.PreviousLocation == nil {
		for _, place := range places {
			if gw.isInsidePlace(current, place) {
				event := GeofenceEvent{
					ID:        utils.GenerateUUID(),
					UserID:    job.UserID,
					PlaceID:   place.ID.Hex(),
					Place:     place,
					EventType: "entry",
					Location:  current,
					Timestamp: job.Timestamp,
					Distance:  utils.CalculateDistance(current.Latitude, current.Longitude, place.Latitude, place.Longitude),
				}
				events = append(events, event)
			}
		}
		return events
	}

	previous := *job.PreviousLocation

	// Check for entries and exits
	for _, place := range places {
		wasInside := gw.isInsidePlace(previous, place)
		isInside := gw.isInsidePlace(current, place)

		if !wasInside && isInside {
			// Entry event
			event := GeofenceEvent{
				ID:        utils.GenerateUUID(),
				UserID:    job.UserID,
				PlaceID:   place.ID.Hex(),
				Place:     place,
				EventType: "entry",
				Location:  current,
				Timestamp: job.Timestamp,
				Distance:  utils.CalculateDistance(current.Latitude, current.Longitude, place.Latitude, place.Longitude),
			}
			events = append(events, event)
		} else if wasInside && !isInside {
			// Exit event
			event := GeofenceEvent{
				ID:        utils.GenerateUUID(),
				UserID:    job.UserID,
				PlaceID:   place.ID.Hex(),
				Place:     place,
				EventType: "exit",
				Location:  current,
				Timestamp: job.Timestamp,
				Distance:  utils.CalculateDistance(current.Latitude, current.Longitude, place.Latitude, place.Longitude),
			}
			events = append(events, event)
		}
	}

	return events
}

func (gw *GeofenceWorker) isInsidePlace(location models.Location, place models.Place) bool {
	distance := utils.CalculateDistance(location.Latitude, location.Longitude, place.Latitude, place.Longitude)
	radius := float64(place.Radius)

	if radius == 0 {
		radius = gw.config.GeofenceRadius
	}

	return distance <= radius
}

func (gw *GeofenceWorker) processGeofenceEvent(ctx context.Context, event GeofenceEvent) {
	logrus.Infof("Processing geofence %s event for user %s at place %s",
		event.EventType, event.UserID, event.Place.Name)

	// Update event statistics
	gw.incrementEventStats(event.EventType)

	// Handle place visit tracking
	go gw.handlePlaceVisit(ctx, event)

	// Send notifications if enabled
	if gw.config.EnableNotifications {
		go gw.sendNotifications(ctx, event)
	}

	// Broadcast WebSocket event if enabled
	if gw.config.EnableWebSocketBroadcast {
		go gw.broadcastEvent(ctx, event)
	}

	// Update place statistics
	go gw.updatePlaceStats(ctx, event)
}

func (gw *GeofenceWorker) handlePlaceVisit(ctx context.Context, event GeofenceEvent) {
	if event.EventType == "entry" {
		// Start place visit
		visit := models.PlaceVisit{
			PlaceID:     event.Place.ID,
			UserID:      utils.ObjectIDFromHex(event.UserID),
			ArrivalTime: event.Timestamp,
			IsOngoing:   true,
		}

		err := gw.placeRepo.CreateVisit(ctx, &visit)
		if err != nil {
			logrus.Errorf("Failed to create place visit: %v", err)
		}
	} else if event.EventType == "exit" {
		// End place visit
		visit, err := gw.placeRepo.GetActiveVisit(ctx, event.UserID, event.PlaceID)
		if err != nil {
			logrus.Errorf("Failed to get active visit: %v", err)
			return
		}

		if visit != nil {
			duration := int64(event.Timestamp.Sub(visit.ArrivalTime).Seconds())

			update := map[string]interface{}{
				"departureTime": event.Timestamp,
				"duration":      duration,
				"isOngoing":     false,
			}

			err = gw.placeRepo.UpdateVisit(ctx, visit.ID.Hex(), update)
			if err != nil {
				logrus.Errorf("Failed to update place visit: %v", err)
			}
		}
	}
}

func (gw *GeofenceWorker) sendNotifications(ctx context.Context, event GeofenceEvent) {
	if gw.notificationService == nil {
		return
	}

	// Check if notifications are enabled for this place and event type
	shouldNotify := (event.EventType == "entry" && event.Place.Notifications.OnArrival) ||
		(event.EventType == "exit" && event.Place.Notifications.OnDeparture)

	if !shouldNotify {
		return
	}

	// Get user info
	user, err := gw.userRepo.GetByID(ctx, event.UserID)
	if err != nil {
		logrus.Errorf("Failed to get user for notification: %v", err)
		return
	}

	// Get circle members to notify
	var notifyUsers []string
	if event.Place.IsShared && !event.Place.CircleID.IsZero() {
		circle, err := gw.circleRepo.GetByID(ctx, event.Place.CircleID.Hex())
		if err == nil {
			for _, member := range circle.Members {
				if member.UserID.Hex() != event.UserID && member.Status == "active" {
					notifyUsers = append(notifyUsers, member.UserID.Hex())
				}
			}
		}
	}

	// Also notify users specified in place notifications
	for _, userID := range event.Place.Notifications.NotifyMembers {
		if userID != event.UserID {
			notifyUsers = append(notifyUsers, userID)
		}
	}

	if len(notifyUsers) == 0 {
		return
	}

	// Create notification
	var title, body string
	if event.EventType == "entry" {
		title = "📍 Arrival Notification"
		body = fmt.Sprintf("%s %s has arrived at %s", user.FirstName, user.LastName, event.Place.Name)
	} else {
		title = "📍 Departure Notification"
		body = fmt.Sprintf("%s %s has left %s", user.FirstName, user.LastName, event.Place.Name)
	}

	notificationReq := models.SendNotificationRequest{
		UserIDs:  notifyUsers,
		Type:     models.NotificationLocationArrival,
		Title:    title,
		Body:     body,
		Priority: "normal",
		Data: map[string]interface{}{
			"type":      "place_event",
			"userId":    event.UserID,
			"placeId":   event.PlaceID,
			"placeName": event.Place.Name,
			"eventType": event.EventType,
			"latitude":  event.Location.Latitude,
			"longitude": event.Location.Longitude,
		},
		Channels: models.NotificationChannels{
			Push:  true,
			InApp: true,
		},
	}

	err = gw.notificationService.SendNotification(ctx, notificationReq)
	if err != nil {
		logrus.Errorf("Failed to send geofence notification: %v", err)
	} else {
		gw.incrementNotificationsSent()
	}
}

func (gw *GeofenceWorker) broadcastEvent(ctx context.Context, event GeofenceEvent) {
	if gw.hub == nil {
		return
	}

	// Get user's circles for broadcasting
	circles, err := gw.circleService.GetUserCircles(ctx, event.UserID)
	if err != nil {
		logrus.Errorf("Failed to get circles for broadcast: %v", err)
		return
	}

	var circleIDs []string
	for _, circle := range circles {
		if circle.Settings.PlaceNotifications {
			circleIDs = append(circleIDs, circle.ID.Hex())
		}
	}

	if len(circleIDs) > 0 {
		wsEvent := models.WSPlaceEvent{
			UserID:    event.UserID,
			PlaceID:   event.PlaceID,
			PlaceName: event.Place.Name,
			EventType: event.EventType,
			Location:  event.Location,
			Timestamp: event.Timestamp,
		}

		gw.hub.BroadcastPlaceEvent(event.UserID, circleIDs, wsEvent)
	}
}

func (gw *GeofenceWorker) updatePlaceStats(ctx context.Context, event GeofenceEvent) {
	// Update place visit statistics
	stats := event.Place.Stats

	if event.EventType == "entry" {
		stats.VisitCount++
		stats.LastVisit = event.Timestamp

		// Update most visited day
		weekday := event.Timestamp.Weekday().String()
		stats.MostVisitedDay = weekday

		// Update usual arrival time
		arrivalHour := event.Timestamp.Hour()
		stats.UsualArrivalTime = fmt.Sprintf("%02d:00", arrivalHour)
	}

	err := gw.placeRepo.UpdateStats(ctx, event.PlaceID, stats)
	if err != nil {
		logrus.Errorf("Failed to update place stats: %v", err)
	}
}

func (gw *GeofenceWorker) getUserPlaces(ctx context.Context, userID string) ([]models.Place, error) {
	// Check cache first
	gw.cacheMutex.RLock()
	if places, exists := gw.placesCache[userID]; exists {
		gw.cacheMutex.RUnlock()
		gw.incrementCacheHits()
		return places, nil
	}
	gw.cacheMutex.RUnlock()

	gw.incrementCacheMisses()

	// Fetch from database
	places, _, err := gw.placeRepo.GetUserPlaces(ctx, userID, models.GetPlacesRequest{})
	if err != nil {
		return nil, err
	}

	// Update cache
	gw.cacheMutex.Lock()
	gw.placesCache[userID] = places
	gw.cacheMutex.Unlock()

	return places, nil
}

func (gw *GeofenceWorker) updateLocationCache(userID string, location models.Location) {
	gw.cacheMutex.Lock()
	gw.locationsCache[userID] = location
	gw.cacheMutex.Unlock()
}

func (gw *GeofenceWorker) cacheRefresher() {
	defer gw.wg.Done()

	ticker := time.NewTicker(gw.config.CacheRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gw.refreshCache()

		case <-gw.ctx.Done():
			return
		}
	}
}

func (gw *GeofenceWorker) refreshCache() {
	gw.cacheMutex.Lock()
	defer gw.cacheMutex.Unlock()

	// Clear old cache entries
	gw.placesCache = make(map[string][]models.Place)

	logrus.Debug("Geofence cache refreshed")
}

func (gw *GeofenceWorker) metricsCollector() {
	defer gw.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gw.collectMetrics()

		case <-gw.ctx.Done():
			return
		}
	}
}

func (gw *GeofenceWorker) collectMetrics() {
	gw.statsMutex.Lock()
	defer gw.statsMutex.Unlock()

	gw.stats.QueueLength = len(gw.geofenceQueue)
}

func (gw *GeofenceWorker) updateStats(duration time.Duration) {
	gw.statsMutex.Lock()
	defer gw.statsMutex.Unlock()

	gw.stats.JobsProcessed++

	// Update average processing time
	if gw.stats.JobsProcessed == 1 {
		gw.stats.AverageProcessTime = float64(duration.Milliseconds())
	} else {
		gw.stats.AverageProcessTime = (gw.stats.AverageProcessTime + float64(duration.Milliseconds())) / 2
	}

	gw.stats.LastProcessedAt = time.Now()
}

func (gw *GeofenceWorker) incrementEventStats(eventType string) {
	gw.statsMutex.Lock()
	defer gw.statsMutex.Unlock()

	gw.stats.EventsDetected++
	if eventType == "entry" {
		gw.stats.EntriesDetected++
	} else if eventType == "exit" {
		gw.stats.ExitsDetected++
	}
}

func (gw *GeofenceWorker) incrementNotificationsSent() {
	gw.statsMutex.Lock()
	gw.stats.NotificationsSent++
	gw.statsMutex.Unlock()
}

func (gw *GeofenceWorker) incrementCacheHits() {
	gw.statsMutex.Lock()
	gw.stats.CacheHits++
	gw.statsMutex.Unlock()
}

func (gw *GeofenceWorker) incrementCacheMisses() {
	gw.statsMutex.Lock()
	gw.stats.CacheMisses++
	gw.statsMutex.Unlock()
}

func (gw *GeofenceWorker) GetStats() GeofenceWorkerStats {
	gw.statsMutex.RLock()
	defer gw.statsMutex.RUnlock()
	return gw.stats
}

// Public function to start geofence worker
func StartGeofenceWorker(db *mongo.Database, redis *redis.Client, hub *websocket.Hub) *GeofenceWorker {
	// Initialize services
	placeRepo := repositories.NewPlaceRepository(db)
	locationRepo := repositories.NewLocationRepository(db)
	circleRepo := repositories.NewCircleRepository(db)
	userRepo := repositories.NewUserRepository(db)
	notificationRepo := repositories.NewNotificationRepository(db)

	circleService := services.NewCircleService(circleRepo, userRepo)
	placeService := services.NewPlaceService(placeRepo, circleRepo)
	geofenceService := services.NewGeofenceService(placeRepo, locationRepo, hub)

	// Initialize push service for notifications
	pushService := services.NewPushService(nil, notificationRepo)

	// Provide nil for EmailService and SMSService if not available, or initialize as needed
	notificationService := services.NewNotificationService(
		notificationRepo,
		userRepo,
		circleRepo,
		redis,
		hub,
		nil, // EmailService
		nil, // SMSService
		pushService,
	)

	worker := NewGeofenceWorker(db, redis, hub, geofenceService, placeService, circleService, notificationService)

	if err := worker.Start(); err != nil {
		logrus.Fatalf("Failed to start geofence worker: %v", err)
	}

	return worker
}
