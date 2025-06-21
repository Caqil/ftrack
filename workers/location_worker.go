package workers

import (
	"context"
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

type LocationWorker struct {
	// Dependencies
	db    *mongo.Database
	redis *redis.Client
	hub   *websocket.Hub

	// Services
	locationService *services.LocationService
	geofenceService *services.GeofenceService
	circleService   *services.CircleService
	userService     *services.UserService

	// Repositories
	locationRepo *repositories.LocationRepository
	placeRepo    *repositories.PlaceRepository

	// Worker configuration
	config LocationWorkerConfig

	// Processing channels
	locationQueue chan LocationJob
	batchQueue    chan []LocationJob

	// Worker state
	isRunning bool
	workers   int
	mutex     sync.RWMutex

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	stats      LocationWorkerStats
	statsMutex sync.RWMutex
}

type LocationWorkerConfig struct {
	WorkerCount       int           `json:"workerCount"`
	QueueSize         int           `json:"queueSize"`
	BatchSize         int           `json:"batchSize"`
	BatchTimeout      time.Duration `json:"batchTimeout"`
	ProcessingTimeout time.Duration `json:"processingTimeout"`
	RetryAttempts     int           `json:"retryAttempts"`
	RetryDelay        time.Duration `json:"retryDelay"`
	EnableBatching    bool          `json:"enableBatching"`
	EnableGeofencing  bool          `json:"enableGeofencing"`
	EnableBroadcast   bool          `json:"enableBroadcast"`
}

type LocationJob struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"userId"`
	Location   models.Location        `json:"location"`
	Timestamp  time.Time              `json:"timestamp"`
	Priority   int                    `json:"priority"` // 1=low, 2=normal, 3=high, 4=urgent
	RetryCount int                    `json:"retryCount"`
	Context    map[string]interface{} `json:"context"`
}

type LocationWorkerStats struct {
	JobsProcessed      int64         `json:"jobsProcessed"`
	JobsFailed         int64         `json:"jobsFailed"`
	JobsRetried        int64         `json:"jobsRetried"`
	AverageProcessTime float64       `json:"averageProcessTime"` // ms
	LastProcessedAt    time.Time     `json:"lastProcessedAt"`
	QueueLength        int           `json:"queueLength"`
	ActiveWorkers      int           `json:"activeWorkers"`
	Uptime             time.Duration `json:"uptime"`
	StartTime          time.Time     `json:"startTime"`
}

func NewLocationWorker(
	db *mongo.Database,
	redis *redis.Client,
	hub *websocket.Hub,
	locationService *services.LocationService,
	geofenceService *services.GeofenceService,
	circleService *services.CircleService,
	userService *services.UserService,
) *LocationWorker {
	ctx, cancel := context.WithCancel(context.Background())

	config := LocationWorkerConfig{
		WorkerCount:       5,
		QueueSize:         1000,
		BatchSize:         10,
		BatchTimeout:      5 * time.Second,
		ProcessingTimeout: 30 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        1 * time.Second,
		EnableBatching:    true,
		EnableGeofencing:  true,
		EnableBroadcast:   true,
	}

	return &LocationWorker{
		db:              db,
		redis:           redis,
		hub:             hub,
		locationService: locationService,
		geofenceService: geofenceService,
		circleService:   circleService,
		userService:     userService,
		locationRepo:    repositories.NewLocationRepository(db),
		placeRepo:       repositories.NewPlaceRepository(db),
		config:          config,
		locationQueue:   make(chan LocationJob, config.QueueSize),
		batchQueue:      make(chan []LocationJob, 100),
		ctx:             ctx,
		cancel:          cancel,
		stats: LocationWorkerStats{
			StartTime: time.Now(),
		},
	}
}

func (lw *LocationWorker) Start() error {
	lw.mutex.Lock()
	defer lw.mutex.Unlock()

	if lw.isRunning {
		return nil
	}

	lw.isRunning = true
	lw.workers = lw.config.WorkerCount

	logrus.Infof("Starting Location Worker with %d workers", lw.workers)

	// Start worker goroutines
	for i := 0; i < lw.workers; i++ {
		lw.wg.Add(1)
		go lw.worker(i)
	}

	// Start batch processor if enabled
	if lw.config.EnableBatching {
		lw.wg.Add(1)
		go lw.batchProcessor()
	}

	// Start metrics collector
	lw.wg.Add(1)
	go lw.metricsCollector()

	// Start queue monitor
	lw.wg.Add(1)
	go lw.queueMonitor()

	logrus.Info("Location Worker started successfully")
	return nil
}

func (lw *LocationWorker) Stop() error {
	lw.mutex.Lock()
	defer lw.mutex.Unlock()

	if !lw.isRunning {
		return nil
	}

	logrus.Info("Stopping Location Worker...")

	lw.cancel()
	lw.isRunning = false

	// Close channels
	close(lw.locationQueue)
	if lw.config.EnableBatching {
		close(lw.batchQueue)
	}

	// Wait for workers to finish
	lw.wg.Wait()

	logrus.Info("Location Worker stopped successfully")
	return nil
}

func (lw *LocationWorker) SubmitLocation(userID string, location models.Location) error {
	if !lw.isRunning {
		return utils.NewServiceError("Location worker is not running")
	}

	job := LocationJob{
		ID:        utils.GenerateUUID(),
		UserID:    userID,
		Location:  location,
		Timestamp: time.Now(),
		Priority:  2, // Normal priority
		Context:   make(map[string]interface{}),
	}

	// Set priority based on location characteristics
	if location.IsDriving {
		job.Priority = 3 // High priority for driving
	}

	select {
	case lw.locationQueue <- job:
		return nil
	default:
		return utils.NewServiceError("Location queue is full")
	}
}

func (lw *LocationWorker) worker(workerID int) {
	defer lw.wg.Done()

	logrus.Infof("Location worker %d started", workerID)

	for {
		select {
		case job, ok := <-lw.locationQueue:
			if !ok {
				logrus.Infof("Location worker %d stopping", workerID)
				return
			}

			lw.processLocation(job, workerID)

		case <-lw.ctx.Done():
			logrus.Infof("Location worker %d stopping due to context cancellation", workerID)
			return
		}
	}
}

func (lw *LocationWorker) processLocation(job LocationJob, workerID int) {
	startTime := time.Now()

	defer func() {
		duration := time.Since(startTime)
		lw.updateStats(duration, true)
	}()

	ctx, cancel := context.WithTimeout(lw.ctx, lw.config.ProcessingTimeout)
	defer cancel()

	logrus.Debugf("Worker %d processing location for user %s", workerID, job.UserID)

	// Store location in database
	err := lw.locationRepo.Create(ctx, &job.Location)
	if err != nil {
		logrus.Errorf("Failed to store location for user %s: %v", job.UserID, err)
		lw.retryJob(job)
		return
	}

	// Process geofencing if enabled
	if lw.config.EnableGeofencing {
		go lw.processGeofencing(ctx, job)
	}

	// Broadcast location update if enabled
	if lw.config.EnableBroadcast {
		go lw.broadcastLocationUpdate(ctx, job)
	}

	// Update user's last seen
	go lw.updateUserLastSeen(ctx, job.UserID)

	logrus.Debugf("Worker %d completed location processing for user %s", workerID, job.UserID)
}

func (lw *LocationWorker) processGeofencing(ctx context.Context, job LocationJob) {
	if lw.geofenceService == nil {
		return
	}

	// Check geofence events
	events, err := lw.geofenceService.CheckGeofences(ctx, job.UserID, job.Location.Latitude, job.Location.Longitude)
	if err != nil {
		logrus.Errorf("Failed to check geofences for user %s: %v", job.UserID, err)
		return
	}

	// Process each geofence event
	for _, event := range events {
		// Get user's circles for broadcasting
		circles, err := lw.circleService.GetUserCircles(ctx, job.UserID)
		if err != nil {
			logrus.Errorf("Failed to get circles for geofence event: %v", err)
			continue
		}

		var circleIDs []string
		for _, circle := range circles {
			circleIDs = append(circleIDs, circle.ID.Hex())
		}

		// Broadcast place event
		if lw.hub != nil {
			lw.hub.BroadcastPlaceEvent(job.UserID, circleIDs, event)
		}
	}
}

func (lw *LocationWorker) broadcastLocationUpdate(ctx context.Context, job LocationJob) {
	if lw.hub == nil {
		return
	}

	// Get user's circles
	circles, err := lw.circleService.GetUserCircles(ctx, job.UserID)
	if err != nil {
		logrus.Errorf("Failed to get circles for location broadcast: %v", err)
		return
	}

	var circleIDs []string
	for _, circle := range circles {
		if circle.Settings.LocationSharing {
			circleIDs = append(circleIDs, circle.ID.Hex())
		}
	}

	if len(circleIDs) > 0 {
		lw.hub.BroadcastLocationUpdate(job.UserID, circleIDs, job.Location)
	}
}

func (lw *LocationWorker) updateUserLastSeen(ctx context.Context, userID string) {
	if lw.userService != nil {
		err := lw.userService.UpdateOnlineStatus(ctx, userID, true)
		if err != nil {
			logrus.Errorf("Failed to update user last seen: %v", err)
		}
	}
}

func (lw *LocationWorker) retryJob(job LocationJob) {
	if job.RetryCount >= lw.config.RetryAttempts {
		logrus.Errorf("Job %s failed after %d attempts", job.ID, job.RetryCount)
		lw.incrementFailedJobs()
		return
	}

	job.RetryCount++
	lw.incrementRetriedJobs()

	// Exponential backoff
	delay := time.Duration(job.RetryCount) * lw.config.RetryDelay

	go func() {
		time.Sleep(delay)
		select {
		case lw.locationQueue <- job:
		default:
			logrus.Errorf("Failed to requeue job %s", job.ID)
		}
	}()
}

func (lw *LocationWorker) batchProcessor() {
	defer lw.wg.Done()

	batch := make([]LocationJob, 0, lw.config.BatchSize)
	ticker := time.NewTicker(lw.config.BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case job, ok := <-lw.locationQueue:
			if !ok {
				// Process remaining batch
				if len(batch) > 0 {
					lw.processBatch(batch)
				}
				return
			}

			batch = append(batch, job)

			// Process batch if it's full
			if len(batch) >= lw.config.BatchSize {
				lw.processBatch(batch)
				batch = batch[:0] // Reset batch
			}

		case <-ticker.C:
			// Process batch on timeout
			if len(batch) > 0 {
				lw.processBatch(batch)
				batch = batch[:0] // Reset batch
			}

		case <-lw.ctx.Done():
			return
		}
	}
}

func (lw *LocationWorker) processBatch(batch []LocationJob) {
	logrus.Debugf("Processing batch of %d location updates", len(batch))

	// Process all jobs in the batch
	for _, job := range batch {
		go lw.processLocation(job, -1) // -1 indicates batch processing
	}
}

func (lw *LocationWorker) metricsCollector() {
	defer lw.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lw.collectMetrics()

		case <-lw.ctx.Done():
			return
		}
	}
}

func (lw *LocationWorker) queueMonitor() {
	defer lw.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			queueLength := len(lw.locationQueue)
			lw.updateQueueLength(queueLength)

			// Log warning if queue is getting full
			if queueLength > lw.config.QueueSize*80/100 {
				logrus.Warnf("Location queue is %d%% full (%d/%d)",
					queueLength*100/lw.config.QueueSize, queueLength, lw.config.QueueSize)
			}

		case <-lw.ctx.Done():
			return
		}
	}
}

func (lw *LocationWorker) collectMetrics() {
	lw.statsMutex.Lock()
	defer lw.statsMutex.Unlock()

	lw.stats.QueueLength = len(lw.locationQueue)
	lw.stats.ActiveWorkers = lw.workers
	lw.stats.Uptime = time.Since(lw.stats.StartTime)
}

func (lw *LocationWorker) updateStats(duration time.Duration, success bool) {
	lw.statsMutex.Lock()
	defer lw.statsMutex.Unlock()

	if success {
		lw.stats.JobsProcessed++

		// Update average processing time
		if lw.stats.JobsProcessed == 1 {
			lw.stats.AverageProcessTime = float64(duration.Milliseconds())
		} else {
			lw.stats.AverageProcessTime = (lw.stats.AverageProcessTime + float64(duration.Milliseconds())) / 2
		}

		lw.stats.LastProcessedAt = time.Now()
	} else {
		lw.stats.JobsFailed++
	}
}

func (lw *LocationWorker) incrementRetriedJobs() {
	lw.statsMutex.Lock()
	lw.stats.JobsRetried++
	lw.statsMutex.Unlock()
}

func (lw *LocationWorker) incrementFailedJobs() {
	lw.statsMutex.Lock()
	lw.stats.JobsFailed++
	lw.statsMutex.Unlock()
}

func (lw *LocationWorker) updateQueueLength(length int) {
	lw.statsMutex.Lock()
	lw.stats.QueueLength = length
	lw.statsMutex.Unlock()
}

func (lw *LocationWorker) GetStats() LocationWorkerStats {
	lw.statsMutex.RLock()
	defer lw.statsMutex.RUnlock()
	return lw.stats
}

// Public function to start location worker
func StartLocationWorker(db *mongo.Database, redis *redis.Client, hub *websocket.Hub) *LocationWorker {
	// Initialize services (in a real app, these would be injected)
	locationRepo := repositories.NewLocationRepository(db)
	circleRepo := repositories.NewCircleRepository(db)
	placeRepo := repositories.NewPlaceRepository(db)
	userRepo := repositories.NewUserRepository(db)

	circleService := services.NewCircleService(circleRepo, userRepo)
	userService := services.NewUserService(userRepo)
	geofenceService := services.NewGeofenceService(placeRepo, locationRepo, hub)
	locationService := services.NewLocationService(locationRepo, circleRepo, placeRepo, userRepo, geofenceService, hub)

	worker := NewLocationWorker(db, redis, hub, locationService, geofenceService, circleService, userService)

	if err := worker.Start(); err != nil {
		logrus.Fatalf("Failed to start location worker: %v", err)
	}

	return worker
}
