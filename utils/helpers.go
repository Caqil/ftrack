package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"ftrack/models"
	"io"
	"math"
	mrand "math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var validate = validator.New()

// GetUserID retrieves the user ID from the Gin context, assuming it is stored as "userID" in context.
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("userID"); exists {
		if idStr, ok := userID.(string); ok {
			return idStr
		}
	}
	return ""
}

// UUID Generation
func GenerateUUID() string {
	return uuid.New().String()
}

func GenerateShortID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Invite Code Generation
func GenerateInviteCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[mrand.Intn(len(charset))]
	}
	return string(b)
}

// String Utilities
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

func StringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func RemoveStringFromSlice(slice []string, item string) []string {
	for i, s := range slice {
		if s == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func UniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	var unique []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			unique = append(unique, item)
		}
	}
	return unique
}

// Number Utilities
func RoundToDecimalPlaces(value float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Round(value*shift) / shift
}

func ClampFloat64(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func ClampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Time Utilities
func TimePtr(t time.Time) *time.Time {
	return &t
}

func StringPtr(s string) *string {
	return &s
}

func IntPtr(i int) *int {
	return &i
}

func BoolPtr(b bool) *bool {
	return &b
}

func FormatDuration(duration time.Duration) string {
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func GetTimeRangeForPeriod(period string) (time.Time, time.Time) {
	now := time.Now()

	switch period {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 0, 1)
		return start, end
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
		end := start.AddDate(0, 0, 1)
		return start, end
	case "week":
		weekday := int(now.Weekday())
		start := now.AddDate(0, 0, -weekday)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end := start.AddDate(0, 0, 7)
		return start, end
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0)
		return start, end
	default:
		// Default to last 24 hours
		return now.AddDate(0, 0, -1), now
	}
}

// Phone Number Utilities
func FormatPhoneNumber(phone string) string {
	// Remove all non-digit characters
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	// Add country code if missing
	if len(cleaned) == 10 {
		cleaned = "1" + cleaned // Assume US number
	}

	// Format as +1 (XXX) XXX-XXXX
	if len(cleaned) == 11 && cleaned[0] == '1' {
		return fmt.Sprintf("+1 (%s) %s-%s",
			cleaned[1:4], cleaned[4:7], cleaned[7:11])
	}

	return "+" + cleaned
}

func NormalizePhoneNumber(phone string) string {
	// Remove all non-digit characters
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	// Add country code if missing
	if len(cleaned) == 10 {
		cleaned = "1" + cleaned
	}

	return "+" + cleaned
}

// Share Code Generation - for temporary location sharing
func GenerateShareCode() string {
	// Generate 8-character alphanumeric code (uppercase)
	// Format: ABC12DEF
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 8

	// Seed the random number generator
	mrand.Seed(time.Now().UnixNano())

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[mrand.Intn(len(charset))]
	}
	return string(b)
}

// Emergency Code Generation - for emergency location sharing
func GenerateEmergencyCode() string {
	// Generate 6-character emergency code with prefix
	// Format: EMG123ABC
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6
	const prefix = "EMG"

	// Seed the random number generator
	mrand.Seed(time.Now().UnixNano())

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[mrand.Intn(len(charset))]
	}
	return prefix + string(b)
}

// Trip Share Code Generation - for trip sharing
func GenerateTripShareCode() string {
	// Generate 10-character code for trip sharing
	// Format: TR12AB34CD
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 8
	const prefix = "TR"

	// Seed the random number generator
	mrand.Seed(time.Now().UnixNano())

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[mrand.Intn(len(charset))]
	}
	return prefix + string(b)
}

// File Utilities
func GetFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return strings.ToLower(parts[len(parts)-1])
	}
	return ""
}

func IsImageFile(filename string) bool {
	ext := GetFileExtension(filename)
	imageExts := []string{"jpg", "jpeg", "png", "gif", "webp", "bmp"}
	return StringSliceContains(imageExts, ext)
}

func IsVideoFile(filename string) bool {
	ext := GetFileExtension(filename)
	videoExts := []string{"mp4", "avi", "mov", "mkv", "webm", "flv"}
	return StringSliceContains(videoExts, ext)
}

func IsAudioFile(filename string) bool {
	ext := GetFileExtension(filename)
	audioExts := []string{"mp3", "wav", "ogg", "aac", "m4a", "flac"}
	return StringSliceContains(audioExts, ext)
}

func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// Pagination Utilities
func CalculateOffset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * pageSize
}

func CalculateTotalPages(total int64, pageSize int) int {
	return int(math.Ceil(float64(total) / float64(pageSize)))
}

// Color Utilities
func GenerateRandomColor() string {
	colors := []string{
		"#FF6B35", "#F7931E", "#FFD23F", "#06FFA5",
		"#118AB2", "#073B4C", "#EF476F", "#FFD166",
		"#06D6A0", "#8D5524", "#F72585", "#4361EE",
		"#3A0CA3", "#7209B7", "#F72585", "#4CC9F0",
	}
	return colors[mrand.Intn(len(colors))]
}

// Hash Utilities
func GenerateHash(input string) string {
	// Simple hash for non-cryptographic purposes
	h := uint32(0)
	for _, c := range input {
		h = h*31 + uint32(c)
	}
	return strconv.FormatUint(uint64(h), 16)
}

// Distance and Location Utilities
func FormatDistance(meters float64) string {
	if meters < 1000 {
		return fmt.Sprintf("%.0fm", meters)
	}
	km := meters / 1000
	if km < 10 {
		return fmt.Sprintf("%.1fkm", km)
	}
	return fmt.Sprintf("%.0fkm", km)
}

func FormatSpeed(mps float64, unit string) string {
	switch unit {
	case "kmh":
		kmh := mps * 3.6
		return fmt.Sprintf("%.0f km/h", kmh)
	case "mph":
		mph := mps * 2.237
		return fmt.Sprintf("%.0f mph", mph)
	default:
		return fmt.Sprintf("%.1f m/s", mps)
	}
}

// Retry Utilities
func RetryWithBackoff(fn func() error, maxRetries int, baseDelay time.Duration) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if i < maxRetries-1 {
			delay := time.Duration(math.Pow(2, float64(i))) * baseDelay
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("failed after %d retries: %v", maxRetries, lastErr)
}

// Security Utilities
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	username := parts[0]
	domain := parts[1]

	if len(username) <= 2 {
		return email
	}

	masked := username[:1] + strings.Repeat("*", len(username)-2) + username[len(username)-1:]
	return masked + "@" + domain
}

func MaskPhoneNumber(phone string) string {
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")
	if len(cleaned) < 4 {
		return phone
	}

	visible := cleaned[len(cleaned)-4:]
	masked := strings.Repeat("*", len(cleaned)-4) + visible
	return "+" + masked
}

// HashString creates a SHA-256 hash of the input string
func HashString(input string) string {
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)
}

// GenerateAPIKey generates a secure API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("lf360_%s", base64.URLEncoding.EncodeToString(bytes)[:40]), nil
}

// ValidateAPIKey validates the format of an API key
func ValidateAPIKey(apiKey string) bool {
	if len(apiKey) < 10 {
		return false
	}

	return strings.HasPrefix(apiKey, "lf360_")
}

// CalculateAge calculates age from birth date
func CalculateAge(birthDate time.Time) int {
	now := time.Now()
	age := now.Year() - birthDate.Year()

	// Adjust if birthday hasn't occurred this year
	if now.YearDay() < birthDate.YearDay() {
		age--
	}

	return age
}

// IsBusinessHours checks if current time is within business hours
func IsBusinessHours(startHour, endHour int) bool {
	now := time.Now()
	currentHour := now.Hour()

	if startHour <= endHour {
		return currentHour >= startHour && currentHour < endHour
	} else {
		// Handle overnight business hours
		return currentHour >= startHour || currentHour < endHour
	}
}

// IsWeekend checks if current day is weekend
func IsWeekend() bool {
	weekday := time.Now().Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// GetNextWorkday returns the next workday (Monday-Friday)
func GetNextWorkday() time.Time {
	now := time.Now()

	// Add days until we hit a weekday
	for {
		now = now.AddDate(0, 0, 1)
		if now.Weekday() != time.Saturday && now.Weekday() != time.Sunday {
			break
		}
	}

	return now
}

// Returns a map of field errors if validation fails.
func ValidateStruct(s interface{}) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}
	errors := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		errors[err.Field()] = err.Tag()
	}
	return errors
}

// TimeAgo returns a human-readable relative time string (e.g., "5 minutes ago")
func TimeAgo(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	}
	return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
}

// FormatRelativeTime formats time relative to now (e.g., "2 hours ago")
func FormatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}

// Environment Utilities
func GetEnvWithDefault(key, defaultValue string) string {
	if value := GetEnv(key, ""); value != "" {
		return value
	}
	return defaultValue
}
func ObjectIDFromHex(hex string) primitive.ObjectID {
	objectID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return primitive.NewObjectID() // Return new ObjectID if conversion fails
	}
	return objectID
}

// GetLogger returns a configured logger instance
func GetLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	return logger
}

func GetEnv(key, defaultValue string) string {
	// This should be implemented based on your environment loading strategy
	// For now, returning default
	return defaultValue
}

// Debug Utilities
func PrettyPrint(v interface{}) {
	fmt.Printf("%+v\n", v)
}

func TimeFunctionExecution(name string, fn func()) {
	start := time.Now()
	fn()
	duration := time.Since(start)
	fmt.Printf("⏱️  %s took %v\n", name, duration)
}

// UploadFile saves the uploaded file to the specified directory and returns the file URL or path.
func UploadFile(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
	// Ensure the folder exists
	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create folder: %w", err)
	}

	// Generate a unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	filepath := filepath.Join(folder, filename)

	out, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// Return the relative path or URL as needed
	return filepath, nil
}

// IsValidMediaFile checks if the file extension matches the allowed types for the given media type.
func IsValidMediaFile(filename, mediaType string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch mediaType {
	case "photo":
		return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif"
	case "video":
		return ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv"
	default:
		return false
	}
}

func HandleServiceError(c *gin.Context, err error) {
	switch err.Error() {
	case "access denied":
		ForbiddenResponse(c, "Access denied")
	case "place not found":
		NotFoundResponse(c, "Place not found")
	case "user not found":
		NotFoundResponse(c, "User not found")
	case "invalid user ID":
		BadRequestResponse(c, "Invalid user ID")
	case "invalid place ID":
		BadRequestResponse(c, "Invalid place ID")
	case "invalid coordinates":
		BadRequestResponse(c, "Invalid coordinates")
	case "radius must be between 10 and 5000 meters":
		BadRequestResponse(c, "Radius must be between 10 and 5000 meters")
	default:
		InternalServerErrorResponse(c, "Internal server error")
	}
}

func NotImplementedResponse(c *gin.Context, message string) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"success": false,
		"message": message,
		"status":  "not_implemented",
	})
}

func ValidateCreatePlaceRequest(req models.CreatePlaceRequest) error {
	if req.Name == "" {
		return errors.New("place name is required")
	}
	if len(req.Name) > 100 {
		return errors.New("place name too long")
	}
	if req.Latitude < -90 || req.Latitude > 90 {
		return errors.New("invalid latitude")
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return errors.New("invalid longitude")
	}
	if req.Radius < 10 || req.Radius > 5000 {
		return errors.New("radius must be between 10 and 5000 meters")
	}
	if req.Category == "" {
		return errors.New("category is required")
	}
	if req.Priority < 0 || req.Priority > 10 {
		return errors.New("priority must be between 0 and 10")
	}
	return nil
}

func ValidateUpdatePlaceRequest(req models.UpdatePlaceRequest) error {
	if req.Name != nil && *req.Name == "" {
		return errors.New("place name cannot be empty")
	}
	if req.Name != nil && len(*req.Name) > 100 {
		return errors.New("place name too long")
	}
	if req.Latitude != nil && (*req.Latitude < -90 || *req.Latitude > 90) {
		return errors.New("invalid latitude")
	}
	if req.Longitude != nil && (*req.Longitude < -180 || *req.Longitude > 180) {
		return errors.New("invalid longitude")
	}
	if req.Radius != nil && (*req.Radius < 10 || *req.Radius > 5000) {
		return errors.New("radius must be between 10 and 5000 meters")
	}
	if req.Priority != nil && (*req.Priority < 0 || *req.Priority > 10) {
		return errors.New("priority must be between 0 and 10")
	}
	return nil
}
