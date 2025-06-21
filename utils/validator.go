package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ValidationService struct {
	validator *validator.Validate
}

type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

func NewValidationService() *ValidationService {
	v := validator.New()

	// Register custom validators
	v.RegisterValidation("phone", validatePhone)
	v.RegisterValidation("coordinate", validateCoordinate)
	v.RegisterValidation("invite_code", validateInviteCode)
	v.RegisterValidation("emergency_type", validateEmergencyType)
	v.RegisterValidation("message_type", validateMessageType)
	v.RegisterValidation("notification_priority", validateNotificationPriority)

	return &ValidationService{
		validator: v,
	}
}

func (vs *ValidationService) ValidateStruct(s interface{}) []ValidationError {
	var validationErrors []ValidationError

	err := vs.validator.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, ValidationError{
				Field:   err.Field(),
				Tag:     err.Tag(),
				Value:   fmt.Sprintf("%v", err.Value()),
				Message: vs.getErrorMessage(err),
			})
		}
	}

	return validationErrors
}

func (vs *ValidationService) getErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return "Invalid email format"
	case "phone":
		return "Invalid phone number format"
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", fe.Field(), fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", fe.Field(), fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", fe.Field(), fe.Param())
	case "coordinate":
		return "Invalid coordinate value"
	case "invite_code":
		return "Invalid invite code format"
	case "emergency_type":
		return "Invalid emergency type"
	case "message_type":
		return "Invalid message type"
	case "notification_priority":
		return "Invalid notification priority"
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}

// Custom validation functions
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	// Remove all non-digit characters
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	// Check if it's a valid length (10-15 digits)
	if len(cleaned) < 10 || len(cleaned) > 15 {
		return false
	}

	// Basic phone number pattern
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{9,14}$`)
	return phoneRegex.MatchString(phone)
}

func validateCoordinate(fl validator.FieldLevel) bool {
	coord := fl.Field().Float()
	fieldName := fl.FieldName()

	if strings.Contains(strings.ToLower(fieldName), "lat") {
		return coord >= -90 && coord <= 90
	}
	if strings.Contains(strings.ToLower(fieldName), "lon") || strings.Contains(strings.ToLower(fieldName), "lng") {
		return coord >= -180 && coord <= 180
	}

	return true
}

func validateInviteCode(fl validator.FieldLevel) bool {
	code := fl.Field().String()
	// Invite codes should be 6-8 alphanumeric characters
	codeRegex := regexp.MustCompile(`^[A-Z0-9]{6,8}$`)
	return codeRegex.MatchString(code)
}

func validateEmergencyType(fl validator.FieldLevel) bool {
	emergencyType := fl.Field().String()
	validTypes := []string{"sos", "crash", "help", "medical", "fire", "police", "roadside", "fall"}

	for _, validType := range validTypes {
		if emergencyType == validType {
			return true
		}
	}
	return false
}

func validateMessageType(fl validator.FieldLevel) bool {
	messageType := fl.Field().String()
	validTypes := []string{"text", "photo", "location", "voice", "sticker", "file"}

	for _, validType := range validTypes {
		if messageType == validType {
			return true
		}
	}
	return false
}

func validateNotificationPriority(fl validator.FieldLevel) bool {
	priority := fl.Field().String()
	validPriorities := []string{"low", "normal", "high", "urgent"}

	for _, validPriority := range validPriorities {
		if priority == validPriority {
			return true
		}
	}
	return false
}

// Additional validation helpers
func ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return errors.New("password must contain uppercase, lowercase, number, and special character")
	}

	return nil
}

// ValidateCircleInviteCode validates circle invite codes
func ValidateCircleInviteCode(code string) bool {
	// Invite codes should be 6-8 alphanumeric characters
	if len(code) < 6 || len(code) > 8 {
		return false
	}

	matched, _ := regexp.MatchString("^[A-Z0-9]+$", code)
	return matched
}

// ValidateLocationPrecision validates location precision values
func ValidateLocationPrecision(precision string) bool {
	validPrecisions := []string{"exact", "approximate", "city"}
	for _, valid := range validPrecisions {
		if precision == valid {
			return true
		}
	}
	return false
}

// ValidatePlaceCategory validates place categories
func ValidatePlaceCategory(category string) bool {
	validCategories := []string{
		"home", "work", "school", "gym", "restaurant", "shopping",
		"hospital", "gas", "park", "airport", "hotel", "church",
		"bank", "library", "cinema", "pharmacy", "other",
	}

	for _, valid := range validCategories {
		if category == valid {
			return true
		}
	}
	return false
}

// ValidateHexColor validates hex color codes
func ValidateHexColor(color string) bool {
	if !strings.HasPrefix(color, "#") {
		return false
	}

	if len(color) != 7 {
		return false
	}

	matched, _ := regexp.MatchString("^#[0-9A-Fa-f]{6}$", color)
	return matched
}

// ValidateTimeFormat validates time format (HH:MM)
func ValidateTimeFormat(timeStr string) bool {
	matched, _ := regexp.MatchString("^([01]?[0-9]|2[0-3]):[0-5][0-9]$", timeStr)
	return matched
}
func SanitizeInput(input string) string {
	// Remove any potentially dangerous characters
	input = strings.TrimSpace(input)
	input = regexp.MustCompile(`[<>\"';&]`).ReplaceAllString(input, "")
	return input
}
