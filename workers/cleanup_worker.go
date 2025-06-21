package workers

import (
	"context"
	"ftrack/repositories"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

type CleanupWorker struct {
	// Dependencies
	db    *mongo.Database
	redis *redis.Client

	// Repositories
	locationRepo     *repositories.LocationRepository
	notificationRepo *repositories.NotificationRepository
	emergencyRepo    *repositories.EmergencyRepository
	messageRepo      *repositories.MessageRepository

	// Worker configuration
	config CleanupWorkerConfig

	// Worker state
	isRunning bool
	mutex     sync.RWMutex

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Cleanup tasks
	tasks []CleanupTask

	// Metrics
	stats      CleanupWorkerStats
	statsMutex sync.RWMutex
}

type CleanupWorkerConfig struct {
	// Retention periods
	LocationRetentionDays     int `json:"locationRetentionDays"`
	NotificationRetentionDays int `json:"notificationRetentionDays"`
	MessageRetentionDays      int `json:"messageRetentionDays"`
	LogRetentionDays          int `json:"logRetentionDays"`

	// Cleanup intervals
	LocationCleanupInterval     time.Duration `json:"locationCleanupInterval"`
	NotificationCleanupInterval time.Duration `json:"notificationCleanupInterval"`
	MessageCleanupInterval      time.Duration `json:"messageCleanupInterval"`
	RedisCleanupInterval        time.Duration `json:"redisCleanupInterval"`
	TempFileCleanupInterval     time.Duration `json:"tempFileCleanupInterval"`

	// Batch sizes
	CleanupBatchSize int `json:"cleanupBatchSize"`

	// Feature flags
	EnableLocationCleanup     bool `json:"enableLocationCleanup"`
	EnableNotificationCleanup bool `json:"enableNotificationCleanup"`
	EnableMessageCleanup      bool `json:"enableMessageCleanup"`
	EnableRedisCleanup        bool `json:"enableRedisCleanup"`
	EnableTempFileCleanup     bool `json:"enableTempFileCleanup"`
}

type CleanupTask struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Interval    time.Duration `json:"interval"`
	LastRun     time.Time     `json:"lastRun"`
	NextRun     time.Time     `json:"nextRun"`
	Enabled     bool          `json:"enabled"`
	Function    func(ctx context.Context) error
}

type CleanupWorkerStats struct {
	TasksExecuted        int64            `json:"tasksExecuted"`
	TasksFailed          int64            `json:"tasksFailed"`
	RecordsDeleted       int64            `json:"recordsDeleted"`
	LocationsCleaned     int64            `json:"locationsCleaned"`
	NotificationsCleaned int64            `json:"notificationsCleaned"`
	MessagesCleaned      int64            `json:"messagesCleaned"`
	RedisKeysCleaned     int64            `json:"redisKeysCleaned"`
	TempFilesCleaned     int64            `json:"tempFilesCleaned"`
	BytesFreed           int64            `json:"bytesFreed"`
	LastCleanupAt        time.Time        `json:"lastCleanupAt"`
	TaskExecutionTimes   map[string]int64 `json:"taskExecutionTimes"` // ms
	StartTime            time.Time        `json:"startTime"`
}

func NewCleanupWorker(db *mongo.Database, redis *redis.Client) *CleanupWorker {
	ctx, cancel := context.WithCancel(context.Background())

	config := CleanupWorkerConfig{
		// Default retention periods
		LocationRetentionDays:     30,
		NotificationRetentionDays: 90,
		MessageRetentionDays:      365,
		LogRetentionDays:          7,

		// Default cleanup intervals
		LocationCleanupInterval:     24 * time.Hour,     // Daily
		NotificationCleanupInterval: 24 * time.Hour,     // Daily
		MessageCleanupInterval:      7 * 24 * time.Hour, // Weekly
		RedisCleanupInterval:        1 * time.Hour,      // Hourly
		TempFileCleanupInterval:     6 * time.Hour,      // Every 6 hours

		// Default batch size
		CleanupBatchSize: 1000,

		// Feature flags (all enabled by default)
		EnableLocationCleanup:     true,
		EnableNotificationCleanup: true,
		EnableMessageCleanup:      true,
		EnableRedisCleanup:        true,
		EnableTempFileCleanup:     true,
	}

	worker := &CleanupWorker{
		db:               db,
		redis:            redis,
		locationRepo:     repositories.NewLocationRepository(db),
		notificationRepo: repositories.NewNotificationRepository(db),
		emergencyRepo:    repositories.NewEmergencyRepository(db),
		messageRepo:      repositories.NewMessageRepository(db),
		config:           config,
		ctx:              ctx,
		cancel:           cancel,
		stats: CleanupWorkerStats{
			StartTime:          time.Now(),
			TaskExecutionTimes: make(map[string]int64),
		},
	}

	// Initialize cleanup tasks
	worker.initializeTasks()

	return worker
}

func (cw *CleanupWorker) Start() error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	if cw.isRunning {
		return nil
	}

	cw.isRunning = true

	logrus.Info("Starting Cleanup Worker...")

	// Start task scheduler
	cw.wg.Add(1)
	go cw.taskScheduler()

	// Start metrics collector
	cw.wg.Add(1)
	go cw.metricsCollector()

	logrus.Infof("Cleanup Worker started with %d tasks", len(cw.tasks))
	return nil
}

func (cw *CleanupWorker) Stop() error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	if !cw.isRunning {
		return nil
	}

	logrus.Info("Stopping Cleanup Worker...")

	cw.cancel()
	cw.isRunning = false
	cw.wg.Wait()

	logrus.Info("Cleanup Worker stopped successfully")
	return nil
}

func (cw *CleanupWorker) initializeTasks() {
	cw.tasks = []CleanupTask{
		{
			Name:        "location_cleanup",
			Description: "Clean up old location records",
			Interval:    cw.config.LocationCleanupInterval,
			Enabled:     cw.config.EnableLocationCleanup,
			Function:    cw.cleanupLocations,
		},
		{
			Name:        "notification_cleanup",
			Description: "Clean up old notification records",
			Interval:    cw.config.NotificationCleanupInterval,
			Enabled:     cw.config.EnableNotificationCleanup,
			Function:    cw.cleanupNotifications,
		},
		{
			Name:        "message_cleanup",
			Description: "Clean up old message records",
			Interval:    cw.config.MessageCleanupInterval,
			Enabled:     cw.config.EnableMessageCleanup,
			Function:    cw.cleanupMessages,
		},
		{
			Name:        "redis_cleanup",
			Description: "Clean up expired Redis keys",
			Interval:    cw.config.RedisCleanupInterval,
			Enabled:     cw.config.EnableRedisCleanup,
			Function:    cw.cleanupRedisKeys,
		},
		{
			Name:        "temp_file_cleanup",
			Description: "Clean up temporary files",
			Interval:    cw.config.TempFileCleanupInterval,
			Enabled:     cw.config.EnableTempFileCleanup,
			Function:    cw.cleanupTempFiles,
		},
	}

	// Set initial next run times
	now := time.Now()
	for i := range cw.tasks {
		cw.tasks[i].NextRun = now.Add(cw.tasks[i].Interval)
	}
}

func (cw *CleanupWorker) taskScheduler() {
	defer cw.wg.Done()

	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cw.executeScheduledTasks()

		case <-cw.ctx.Done():
			return
		}
	}
}

func (cw *CleanupWorker) executeScheduledTasks() {
	now := time.Now()

	for i := range cw.tasks {
		task := &cw.tasks[i]

		if !task.Enabled || now.Before(task.NextRun) {
			continue
		}

		logrus.Infof("Executing cleanup task: %s", task.Name)

		startTime := time.Now()
		err := task.Function(cw.ctx)
		executionTime := time.Since(startTime)

		// Update task stats
		cw.statsMutex.Lock()
		cw.stats.TaskExecutionTimes[task.Name] = executionTime.Milliseconds()
		if err != nil {
			cw.stats.TasksFailed++
			logrus.Errorf("Cleanup task %s failed: %v", task.Name, err)
		} else {
			cw.stats.TasksExecuted++
			logrus.Infof("Cleanup task %s completed in %v", task.Name, executionTime)
		}
		cw.statsMutex.Unlock()

		// Update task schedule
		task.LastRun = now
		task.NextRun = now.Add(task.Interval)
	}
}

func (cw *CleanupWorker) cleanupLocations(ctx context.Context) error {
	cutoffTime := time.Now().AddDate(0, 0, -cw.config.LocationRetentionDays)

	deletedCount, err := cw.locationRepo.DeleteOldLocations(ctx, cutoffTime)
	if err != nil {
		return err
	}

	cw.statsMutex.Lock()
	cw.stats.LocationsCleaned += deletedCount
	cw.stats.RecordsDeleted += deletedCount
	cw.stats.LastCleanupAt = time.Now()
	cw.statsMutex.Unlock()

	logrus.Infof("Cleaned up %d old location records", deletedCount)
	return nil
}

func (cw *CleanupWorker) cleanupNotifications(ctx context.Context) error {
	deletedCount, err := cw.notificationRepo.DeleteExpired(ctx)
	if err != nil {
		return err
	}

	cw.statsMutex.Lock()
	cw.stats.NotificationsCleaned += deletedCount
	cw.stats.RecordsDeleted += deletedCount
	cw.stats.LastCleanupAt = time.Now()
	cw.statsMutex.Unlock()

	logrus.Infof("Cleaned up %d expired notifications", deletedCount)
	return nil
}

func (cw *CleanupWorker) cleanupMessages(ctx context.Context) error {
	// This would implement message cleanup logic
	// For now, just log that it's implemented
	logrus.Debug("Message cleanup task executed (implementation pending)")
	return nil
}

func (cw *CleanupWorker) cleanupRedisKeys(ctx context.Context) error {
	if cw.redis == nil {
		return nil
	}

	// Clean up expired sessions, cache entries, etc.
	patterns := []string{
		"session:*",
		"cache:*",
		"temp:*",
		"ws:*",
	}

	var totalCleaned int64

	for _, pattern := range patterns {
		keys, err := cw.redis.Keys(ctx, pattern).Result()
		if err != nil {
			continue
		}

		for _, key := range keys {
			// Check if key has TTL
			ttl, err := cw.redis.TTL(ctx, key).Result()
			if err != nil {
				continue
			}

			// If TTL is -1 (no expiration) and key is old, delete it
			if ttl == -1 {
				// Check key age (this is simplified)
				err = cw.redis.Del(ctx, key).Err()
				if err == nil {
					totalCleaned++
				}
			}
		}
	}

	cw.statsMutex.Lock()
	cw.stats.RedisKeysCleaned += totalCleaned
	cw.stats.LastCleanupAt = time.Now()
	cw.statsMutex.Unlock()

	logrus.Infof("Cleaned up %d Redis keys", totalCleaned)
	return nil
}

func (cw *CleanupWorker) cleanupTempFiles(ctx context.Context) error {
	// This would implement temporary file cleanup
	// For now, just log that it's implemented
	logrus.Debug("Temp file cleanup task executed (implementation pending)")
	return nil
}

func (cw *CleanupWorker) metricsCollector() {
	defer cw.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cw.collectMetrics()

		case <-cw.ctx.Done():
			return
		}
	}
}

func (cw *CleanupWorker) collectMetrics() {
	// Log current cleanup statistics
	cw.statsMutex.RLock()
	stats := cw.stats
	cw.statsMutex.RUnlock()

	logrus.Infof("Cleanup Worker Stats - Tasks: %d executed, %d failed, Records: %d deleted",
		stats.TasksExecuted, stats.TasksFailed, stats.RecordsDeleted)
}

func (cw *CleanupWorker) GetStats() CleanupWorkerStats {
	cw.statsMutex.RLock()
	defer cw.statsMutex.RUnlock()
	return cw.stats
}

func (cw *CleanupWorker) GetTasks() []CleanupTask {
	// Return copy to prevent modification
	tasks := make([]CleanupTask, len(cw.tasks))
	copy(tasks, cw.tasks)
	return tasks
}

func (cw *CleanupWorker) EnableTask(taskName string) error {
	for i := range cw.tasks {
		if cw.tasks[i].Name == taskName {
			cw.tasks[i].Enabled = true
			logrus.Infof("Enabled cleanup task: %s", taskName)
			return nil
		}
	}
	return utils.NewValidationError("Task not found: " + taskName)
}

func (cw *CleanupWorker) DisableTask(taskName string) error {
	for i := range cw.tasks {
		if cw.tasks[i].Name == taskName {
			cw.tasks[i].Enabled = false
			logrus.Infof("Disabled cleanup task: %s", taskName)
			return nil
		}
	}
	return utils.NewValidationError("Task not found: " + taskName)
}

// Public function to start cleanup worker
func StartCleanupWorker(db *mongo.Database, redis *redis.Client) *CleanupWorker {
	worker := NewCleanupWorker(db, redis)

	if err := worker.Start(); err != nil {
		logrus.Fatalf("Failed to start cleanup worker: %v", err)
	}

	return worker
}
