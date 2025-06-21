package config

import (
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
)

type Config struct {
	Environment string
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string

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
}

func Load() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "mongodb://localhost:27017/life360"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:   getEnv("JWT_SECRET", "your-super-secret-jwt-key"),

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
