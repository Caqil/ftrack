package workers

import (
	"context"
	"ftrack/models"
	"ftrack/repositories"
	"ftrack/services"
	"ftrack/utils"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

type NotificationWorker struct {
	// Dependencies
	db    *mongo.Database
	redis *redis.Client

	// Services
	notificationService *services.NotificationService
	pushService         *services.PushService

	// Repositories
	notificationRepo *repositories.NotificationRepository
	userRepo         *repositories.UserRepository

	// Worker configuration
	config NotificationWorkerConfig

	// Processing channels
	notificationQueue chan NotificationJob

	// Worker state
	isRunning bool
	workers   int
	mutex     sync.RWMutex

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	stats      NotificationWorkerStats
	statsMutex sync.RWMutex
}

type NotificationWorkerConfig struct {
	WorkerCount       int           `json:"workerCount"`
	QueueSize         int           `json:"queueSize"`
	ProcessingTimeout time.Duration `json:"processingTimeout"`
	RetryAttempts     int           `json:"retryAttempts"`
	RetryDelay        time.Duration `json:"retryDelay"`
	BatchSize         int           `json:"batchSize"`
	PollInterval      time.Duration `json:"pollInterval"`
}

type NotificationJob struct {
	ID           string                 `json:"id"`
	Notification models.Notification    `json:"notification"`
	User         models.User            `json:"user"`
	Priority     int                    `json:"priority"`
	RetryCount   int                    `json:"retryCount"`
	CreatedAt    time.Time              `json:"createdAt"`
	Context      map[string]interface{} `json:"context"`
}

type NotificationWorkerStats struct {
	JobsProcessed      int64     `json:"jobsProcessed"`
	JobsFailed         int64     `json:"jobsFailed"`
	JobsRetried        int64     `json:"jobsRetried"`
	PushSent           int64     `json:"pushSent"`
	SMSSent            int64     `json:"smsSent"`
	EmailSent          int64     `json:"emailSent"`
	AverageProcessTime float64   `json:"averageProcessTime"` // ms
	LastProcessedAt    time.Time `json:"lastProcessedAt"`
	QueueLength        int       `json:"queueLength"`
	StartTime          time.Time `json:"startTime"`
}

func NewNotificationWorker(
	db *mongo.Database,
	redis *redis.Client,
	notificationService *services.NotificationService,
	pushService *services.PushService,
) *NotificationWorker {
	ctx, cancel := context.WithCancel(context.Background())

	config := NotificationWorkerConfig{
		WorkerCount:       3,
		QueueSize:         500,
		ProcessingTimeout: 30 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        2 * time.Second,
		BatchSize:         50,
		PollInterval:      10 * time.Second,
	}

	return &NotificationWorker{
		db:                  db,
		redis:               redis,
		notificationService: notificationService,
		pushService:         pushService,
		notificationRepo:    repositories.NewNotificationRepository(db),
		userRepo:            repositories.NewUserRepository(db),
		config:              config,
		notificationQueue:   make(chan NotificationJob, config.QueueSize),
		ctx:                 ctx,
		cancel:              cancel,
		stats: NotificationWorkerStats{
			StartTime: time.Now(),
		},
	}
}

func (nw *NotificationWorker) Start() error {
	nw.mutex.Lock()
	defer nw.mutex.Unlock()

	if nw.isRunning {
		return nil
	}

	nw.isRunning = true
	nw.workers = nw.config.WorkerCount

	logrus.Infof("Starting Notification Worker with %d workers", nw.workers)

	// Start worker goroutines
	for i := 0; i < nw.workers; i++ {
		nw.wg.Add(1)
		go nw.worker(i)
	}

	// Start pending notification poller
	nw.wg.Add(1)
	go nw.pendingNotificationPoller()

	// Start metrics collector
	nw.wg.Add(1)
	go nw.metricsCollector()

	logrus.Info("Notification Worker started successfully")
	return nil
}

func (nw *NotificationWorker) Stop() error {
	nw.mutex.Lock()
	defer nw.mutex.Unlock()

	if !nw.isRunning {
		return nil
	}

	logrus.Info("Stopping Notification Worker...")

	nw.cancel()
	nw.isRunning = false

	close(nw.notificationQueue)
	nw.wg.Wait()

	logrus.Info("Notification Worker stopped successfully")
	return nil
}

func (nw *NotificationWorker) SubmitNotification(notification models.Notification, user models.User) error {
	if !nw.isRunning {
		return utils.NewServiceError("Notification worker is not running")
	}

	job := NotificationJob{
		ID:           utils.GenerateUUID(),
		Notification: notification,
		User:         user,
		Priority:     nw.getPriority(notification.Priority),
		CreatedAt:    time.Now(),
		Context:      make(map[string]interface{}),
	}

	select {
	case nw.notificationQueue <- job:
		return nil
	default:
		return utils.NewServiceError("Notification queue is full")
	}
}

func (nw *NotificationWorker) worker(workerID int) {
	defer nw.wg.Done()

	logrus.Infof("Notification worker %d started", workerID)

	for {
		select {
		case job, ok := <-nw.notificationQueue:
			if !ok {
				logrus.Infof("Notification worker %d stopping", workerID)
				return
			}

			nw.processNotification(job, workerID)

		case <-nw.ctx.Done():
			logrus.Infof("Notification worker %d stopping due to context cancellation", workerID)
			return
		}
	}
}

func (nw *NotificationWorker) processNotification(job NotificationJob, workerID int) {
	startTime := time.Now()

	defer func() {
		duration := time.Since(startTime)
		nw.updateStats(duration, true)
	}()

	ctx, cancel := context.WithTimeout(nw.ctx, nw.config.ProcessingTimeout)
	defer cancel()

	logrus.Debugf("Worker %d processing notification %s for user %s",
		workerID, job.Notification.Type, job.User.ID.Hex())

	// Check user preferences
	prefs, err := nw.notificationRepo.GetUserPreferences(ctx, job.User.ID.Hex())
	if err != nil {
		logrus.Errorf("Failed to get user preferences: %v", err)
		nw.retryJob(job)
		return
	}

	if !prefs.GlobalEnabled {
		logrus.Debugf("Notifications disabled for user %s", job.User.ID.Hex())
		return
	}

	// Check quiet hours
	if nw.isQuietHours(prefs.QuietHours) {
		logrus.Debugf("In quiet hours for user %s, skipping notification", job.User.ID.Hex())
		return
	}

	var success bool

	// Send push notification
	if job.Notification.Channels.Push && job.User.DeviceToken != "" {
		if nw.sendPushNotification(ctx, job) {
			success = true
			nw.incrementPushSent()
		}
	}

	// Send SMS notification
	if job.Notification.Channels.SMS && job.User.Phone != "" {
		if nw.sendSMSNotification(ctx, job) {
			success = true
			nw.incrementSMSSent()
		}
	}

	// Send email notification
	if job.Notification.Channels.Email && job.User.Email != "" {
		if nw.sendEmailNotification(ctx, job) {
			success = true
			nw.incrementEmailSent()
		}
	}

	// Update notification status
	status := "failed"
	if success {
		status = "sent"
	}

	err = nw.notificationRepo.Update(ctx, job.Notification.ID.Hex(), map[string]interface{}{
		"status": status,
		"sentAt": time.Now(),
	})

	if err != nil {
		logrus.Errorf("Failed to update notification status: %v", err)
	}

	if !success && job.RetryCount < nw.config.RetryAttempts {
		nw.retryJob(job)
	}

	logrus.Debugf("Worker %d completed notification processing", workerID)
}

func (nw *NotificationWorker) sendPushNotification(ctx context.Context, job NotificationJob) bool {
	if nw.pushService == nil {
		return false
	}

	pushNotif := utils.PushNotification{
		Title: job.Notification.Title,
		Body:  job.Notification.Body,
		Data:  make(map[string]string),
		Sound: "default",
	}

	// Convert data map to string map
	for k, v := range job.Notification.Data {
		if str, ok := v.(string); ok {
			pushNotif.Data[k] = str
		}
	}

	// Set priority
	switch job.Notification.Priority {
	case "urgent":
		pushNotif.Sound = "emergency"
	case "high":
		pushNotif.Sound = "high_priority"
	}

	result, err := nw.pushService.SendPushNotification(ctx, job.User.DeviceToken, pushNotif)
	if err != nil {
		logrus.Errorf("Failed to send push notification: %v", err)
		return false
	}

	return result.Success
}

func (nw *NotificationWorker) sendSMSNotification(ctx context.Context, job NotificationJob) bool {
	if nw.pushService == nil {
		return false
	}

	sms := utils.SMSMessage{
		To:      job.User.Phone,
		Message: job.Notification.Title + ": " + job.Notification.Body,
	}

	result, err := nw.pushService.SendSMS(ctx, sms)
	if err != nil {
		logrus.Errorf("Failed to send SMS notification: %v", err)
		return false
	}

	return result.Success
}

func (nw *NotificationWorker) sendEmailNotification(ctx context.Context, job NotificationJob) bool {
	if nw.pushService == nil {
		return false
	}

	email := utils.EmailMessage{
		To:      job.User.Email,
		Subject: job.Notification.Title,
		Body:    job.Notification.Body,
		IsHTML:  false,
	}

	result, err := nw.pushService.SendEmail(ctx, email)
	if err != nil {
		logrus.Errorf("Failed to send email notification: %v", err)
		return false
	}

	return result.Success
}

func (nw *NotificationWorker) retryJob(job NotificationJob) {
	if job.RetryCount >= nw.config.RetryAttempts {
		logrus.Errorf("Notification job %s failed after %d attempts", job.ID, job.RetryCount)
		nw.incrementFailedJobs()
		return
	}

	job.RetryCount++
	nw.incrementRetriedJobs()

	// Exponential backoff
	delay := time.Duration(job.RetryCount) * nw.config.RetryDelay

	go func() {
		time.Sleep(delay)
		select {
		case nw.notificationQueue <- job:
		default:
			logrus.Errorf("Failed to requeue notification job %s", job.ID)
		}
	}()
}

func (nw *NotificationWorker) pendingNotificationPoller() {
	defer nw.wg.Done()

	ticker := time.NewTicker(nw.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nw.processPendingNotifications()

		case <-nw.ctx.Done():
			return
		}
	}
}

func (nw *NotificationWorker) processPendingNotifications() {
	ctx, cancel := context.WithTimeout(nw.ctx, nw.config.ProcessingTimeout)
	defer cancel()

	notifications, err := nw.notificationRepo.GetPendingNotifications(ctx, nw.config.BatchSize)
	if err != nil {
		logrus.Errorf("Failed to get pending notifications: %v", err)
		return
	}

	if len(notifications) == 0 {
		return
	}

	logrus.Debugf("Processing %d pending notifications", len(notifications))

	for _, notification := range notifications {
		// Get user info
		user, err := nw.userRepo.GetByID(ctx, notification.UserID.Hex())
		if err != nil {
			logrus.Errorf("Failed to get user for notification: %v", err)
			continue
		}

		// Submit to processing queue
		err = nw.SubmitNotification(notification, *user)
		if err != nil {
			logrus.Errorf("Failed to submit notification to queue: %v", err)
		}
	}
}

func (nw *NotificationWorker) metricsCollector() {
	defer nw.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nw.collectMetrics()

		case <-nw.ctx.Done():
			return
		}
	}
}

func (nw *NotificationWorker) collectMetrics() {
	nw.statsMutex.Lock()
	defer nw.statsMutex.Unlock()

	nw.stats.QueueLength = len(nw.notificationQueue)
}

func (nw *NotificationWorker) isQuietHours(quietHours models.NotificationQuietHours) bool {
	if !quietHours.Enabled {
		return false
	}

	now := time.Now()

	// Parse start and end times
	startTime, err := time.Parse("15:04", quietHours.StartTime)
	if err != nil {
		return false
	}

	endTime, err := time.Parse("15:04", quietHours.EndTime)
	if err != nil {
		return false
	}

	// Check if current day is in weekdays
	if len(quietHours.Weekdays) > 0 {
		currentWeekday := int(now.Weekday())
		found := false
		for _, weekday := range quietHours.Weekdays {
			if weekday == currentWeekday {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check if current time is within quiet hours
	currentTime := now.Hour()*60 + now.Minute()
	startMinutes := startTime.Hour()*60 + startTime.Minute()
	endMinutes := endTime.Hour()*60 + endTime.Minute()

	if startMinutes <= endMinutes {
		return currentTime >= startMinutes && currentTime <= endMinutes
	} else {
		// Overnight quiet hours
		return currentTime >= startMinutes || currentTime <= endMinutes
	}
}

func (nw *NotificationWorker) getPriority(priority string) int {
	switch priority {
	case "urgent":
		return 4
	case "high":
		return 3
	case "normal":
		return 2
	case "low":
		return 1
	default:
		return 2
	}
}

func (nw *NotificationWorker) updateStats(duration time.Duration, success bool) {
	nw.statsMutex.Lock()
	defer nw.statsMutex.Unlock()

	if success {
		nw.stats.JobsProcessed++

		// Update average processing time
		if nw.stats.JobsProcessed == 1 {
			nw.stats.AverageProcessTime = float64(duration.Milliseconds())
		} else {
			nw.stats.AverageProcessTime = (nw.stats.AverageProcessTime + float64(duration.Milliseconds())) / 2
		}

		nw.stats.LastProcessedAt = time.Now()
	}
}

func (nw *NotificationWorker) incrementRetriedJobs() {
	nw.statsMutex.Lock()
	nw.stats.JobsRetried++
	nw.statsMutex.Unlock()
}

func (nw *NotificationWorker) incrementFailedJobs() {
	nw.statsMutex.Lock()
	nw.stats.JobsFailed++
	nw.statsMutex.Unlock()
}

func (nw *NotificationWorker) incrementPushSent() {
	nw.statsMutex.Lock()
	nw.stats.PushSent++
	nw.statsMutex.Unlock()
}

func (nw *NotificationWorker) incrementSMSSent() {
	nw.statsMutex.Lock()
	nw.stats.SMSSent++
	nw.statsMutex.Unlock()
}

func (nw *NotificationWorker) incrementEmailSent() {
	nw.statsMutex.Lock()
	nw.stats.EmailSent++
	nw.statsMutex.Unlock()
}

func (nw *NotificationWorker) GetStats() NotificationWorkerStats {
	nw.statsMutex.RLock()
	defer nw.statsMutex.RUnlock()
	return nw.stats
}

// Public function to start notification worker
func StartNotificationWorker(db *mongo.Database, redis *redis.Client) *NotificationWorker {
	// Initialize services
	notificationRepo := repositories.NewNotificationRepository(db)
	userRepo := repositories.NewUserRepository(db)

	// Initialize push service (would need real credentials)
	pushService, err := services.NewPushService("", "", "", "")
	if err != nil {
		logrus.Errorf("Failed to initialize push service: %v", err)
	}

	notificationService := services.NewNotificationService(notificationRepo, userRepo, pushService)

	worker := NewNotificationWorker(db, redis, notificationService, pushService)

	if err := worker.Start(); err != nil {
		logrus.Fatalf("Failed to start notification worker: %v", err)
	}

	return worker
}
