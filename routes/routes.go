// routes/routes.go
package routes

import (
	"ftrack/controllers"
	"ftrack/middleware"
	"ftrack/repositories"
	"ftrack/services"
	"ftrack/websocket"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
)

// SetupRoutes initializes all application routes
func SetupRoutes(db *mongo.Database, redis *redis.Client, hub *websocket.Hub) *gin.Engine {
	router := gin.New()

	// Initialize repositories
	repos := initializeRepositories(db)

	// Initialize services
	services := initializeServices(repos, redis, hub)

	// Initialize controllers
	controllers := initializeControllers(services, hub)

	// Global middleware
	setupGlobalMiddleware(router, redis)

	// Setup route groups
	setupPublicRoutes(router, controllers)
	setupAuthenticatedRoutes(router, controllers, redis)
	setupAdminRoutes(router, controllers, redis)
	setupWebSocketRoutes(router, controllers)

	return router
}

// Repositories initialization
type Repositories struct {
	User         *repositories.UserRepository
	Circle       *repositories.CircleRepository
	Message      *repositories.MessageRepository
	Emergency    *repositories.EmergencyRepository
	Location     *repositories.LocationRepository
	Notification *repositories.NotificationRepository
	Place        *repositories.PlaceRepository
}

func initializeRepositories(db *mongo.Database) *Repositories {
	return &Repositories{
		User:         repositories.NewUserRepository(db),
		Circle:       repositories.NewCircleRepository(db),
		Message:      repositories.NewMessageRepository(db),
		Emergency:    repositories.NewEmergencyRepository(db),
		Location:     repositories.NewLocationRepository(db),
		Notification: repositories.NewNotificationRepository(db),
		Place:        repositories.NewPlaceRepository(db),
	}
}

// Services initialization
type Services struct {
	Auth         *services.AuthService
	User         *services.UserService
	Circle       *services.CircleService
	Message      *services.MessageService
	Emergency    *services.EmergencyService
	Location     *services.LocationService
	Notification *services.NotificationService
	Place        *services.PlaceService
}

func initializeServices(repos *Repositories, redis *redis.Client, hub *websocket.Hub) *Services {
	authService := services.NewAuthService(repos.User, redis)
	notificationService := services.NewNotificationService(repos.Notification, redis)

	return &Services{
		Auth:         authService,
		User:         services.NewUserService(repos.User),
		Circle:       services.NewCircleService(repos.Circle, repos.User),
		Message:      services.NewMessageService(repos.Message, repos.Circle, repos.User, hub),
		Emergency:    services.NewEmergencyService(repos.Emergency, repos.Circle, repos.User, notificationService, hub),
		Location:     services.NewLocationService(repos.Location, repos.Place, repos.Circle, hub),
		Notification: notificationService,
		Place:        services.NewPlaceService(repos.Place, repos.Circle),
	}
}

// Controllers initialization
type Controllers struct {
	Auth         *controllers.AuthController
	User         *controllers.UserController
	Circle       *controllers.CircleController
	Message      *controllers.MessageController
	Emergency    *controllers.EmergencyController
	Location     *controllers.LocationController
	Notification *controllers.NotificationController
	Place        *controllers.PlaceController
	WebSocket    *controllers.WebSocketController
	Health       *controllers.HealthController
}

func initializeControllers(services *Services, hub *websocket.Hub) *Controllers {
	return &Controllers{
		Auth:         controllers.NewAuthController(services.Auth),
		User:         controllers.NewUserController(services.User),
		Circle:       controllers.NewCircleController(services.Circle),
		Message:      controllers.NewMessageController(services.Message),
		Emergency:    controllers.NewEmergencyController(services.Emergency),
		Location:     controllers.NewLocationController(services.Location),
		Notification: controllers.NewNotificationController(services.Notification),
		Place:        controllers.NewPlaceController(services.Place),
		WebSocket:    controllers.NewWebSocketController(hub, services.Auth),
		Health:       controllers.NewHealthController(),
	}
}

// Global middleware setup
func setupGlobalMiddleware(router *gin.Engine, redis *redis.Client) {
	// Basic middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.DefaultLoggerMiddleware())

	// Security middleware
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	router.Use(middleware.SecurityHSeaders())
	router.Use(middleware.GlobalRateLimit(redis))

	// Monitoring middleware
	router.Use(middleware.Metrics())
}

// Public routes (no authentication required)
func setupPublicRoutes(router *gin.Engine, controllers *Controllers) {
	// Health check
	router.GET("/health", controllers.Health.HealthCheck)
	router.GET("/health/detailed", controllers.Health.DetailedHealthCheck)

	// API info
	router.GET("/", controllers.Health.APIInfo)
	router.GET("/version", controllers.Health.Version)

	// Documentation
	router.GET("/docs/*any", controllers.Health.SwaggerDocs)

	// Public API group
	public := router.Group("/api/v1")
	{
		// Authentication routes
		SetupAuthRoutes(public, controllers.Auth)
	}
}

// Authenticated routes (requires valid JWT token)
func setupAuthenticatedRoutes(router *gin.Engine, controllers *Controllers, redis *redis.Client) {
	api := router.Group("/api/v1")
	api.Use(authMiddleware.RequireAuth())
	api.Use(middleware.APIRateLimit(redis))

	// Setup all authenticated route groups
	SetupUserRoutes(api, controllers.User, redis)
	SetupCircleRoutes(api, controllers.Circle, redis)
	SetupMessageRoutes(api, controllers.Message, redis)
	SetupEmergencyRoutes(api, controllers.Emergency, redis)
	SetupLocationRoutes(api, controllers.Location, redis)
	SetupNotificationRoutes(api, controllers.Notification, redis)
	SetupPlaceRoutes(api, controllers.Place, redis)
}

// Admin routes (requires admin privileges)
func setupAdminRoutes(router *gin.Engine, controllers *Controllers, redis *redis.Client) {
	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.AdminMiddleware())
	admin.Use(middleware.AdminRateLimit(redis))

	// Admin-only endpoints
	admin.GET("/users", controllers.User.GetAllUsers)
	admin.GET("/users/:id", controllers.User.GetUserByID)
	admin.PUT("/users/:id/status", controllers.User.UpdateUserStatus)
	admin.DELETE("/users/:id", controllers.User.DeleteUser)

	admin.GET("/circles", controllers.Circle.GetAllCircles)
	admin.GET("/circles/:id", controllers.Circle.GetCircleByID)
	admin.DELETE("/circles/:id", controllers.Circle.DeleteCircle)

	admin.GET("/emergencies", controllers.Emergency.GetAllEmergencies)
	admin.GET("/emergencies/active", controllers.Emergency.GetActiveEmergencies)

	admin.GET("/metrics", controllers.Health.Metrics)
	admin.GET("/stats", controllers.Health.SystemStats)
}

// WebSocket routes
func setupWebSocketRoutes(router *gin.Engine, controllers *Controllers) {
	SetupWebSocketRoutes(router, controllers.WebSocket)
}
