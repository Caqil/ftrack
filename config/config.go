// config/config.go - Updated with baseURL for email links
package config

import (
	"ftrack/services"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Environment string
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	BaseURL     string // Added for email links

	// Firebase Config
	FirebaseCredentials string

	// Twilio Config
	TwilioAccountSID  string
	TwilioAuthToken   string
	TwilioPhoneNumber string

	// App Settings
	MaxCircleMembers  int
	LocationRetention int // days
	RateLimitRequest  int
	RateLimitWindow   int // minutes

	EmailProvider string `env:"EMAIL_PROVIDER" envDefault:"smtp"`

	// SMTP Settings
	SMTPHost     string `env:"SMTP_HOST" envDefault:"smtp.gmail.com"`
	SMTPPort     string `env:"SMTP_PORT" envDefault:"587"`
	SMTPUsername string `env:"SMTP_USERNAME"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPFrom     string `env:"SMTP_FROM"`
}

func Load() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "mongodb://localhost:27017/life360"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:   getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
		BaseURL:     getEnv("BASE_URL", "http://localhost:8080"), // Added BaseURL

		// Firebase
		FirebaseCredentials: getEnv("FIREBASE_CREDENTIALS", ""),

		// Twilio
		TwilioAccountSID:  getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:   getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioPhoneNumber: getEnv("TWILIO_PHONE_NUMBER", ""),

		// App Settings
		MaxCircleMembers:  getEnvAsInt("MAX_CIRCLE_MEMBERS", 20),
		LocationRetention: getEnvAsInt("LOCATION_RETENTION_DAYS", 30),
		RateLimitRequest:  getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getEnvAsInt("RATE_LIMIT_WINDOW_MINUTES", 1),

		// Email settings
		EmailProvider: getEnv("EMAIL_PROVIDER", "smtp"),
		SMTPHost:      getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:      getEnv("SMTP_PORT", "587"),
		SMTPUsername:  getEnv("SMTP_USERNAME", ""),
		SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:      getEnv("SMTP_FROM", "noreply@ftrack.app"),
	}
}

func InitRedis(cfg *Config) *redis.Client {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		// Fallback to default config
		opt = &redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}
	}

	client := redis.NewClient(opt)
	return client
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// InitEmailService initializes the email service based on configuration
func (c *Config) InitEmailService() services.EmailService {
	switch c.EmailProvider {
	case "smtp":
		if c.SMTPUsername == "" || c.SMTPPassword == "" {
			logrus.Warn("SMTP credentials not configured, using mock email service")
			return services.NewMockEmailService()
		}
		// Pass BaseURL to the email service for auth links
		return services.NewSMTPEmailService(
			c.SMTPHost,
			c.SMTPPort,
			c.SMTPUsername,
			c.SMTPPassword,
			c.SMTPFrom,
			c.BaseURL, // Added BaseURL parameter
		)
	case "mock":
		return services.NewMockEmailService()
	default:
		logrus.Warn("Unknown email provider, using mock email service")
		return services.NewMockEmailService()
	}
}
