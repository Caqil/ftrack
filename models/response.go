package models

import "time"

// Standard API Response wrapper
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Meta      *MetaData   `json:"meta,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Field   string      `json:"field,omitempty"`
}

type MetaData struct {
	Page       int   `json:"page,omitempty"`
	PageSize   int   `json:"pageSize,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"totalPages,omitempty"`
}

// Pagination request
type PaginationRequest struct {
	Page      int    `json:"page" form:"page" validate:"min=1"`
	PageSize  int    `json:"pageSize" form:"pageSize" validate:"min=1,max=100"`
	SortBy    string `json:"sortBy" form:"sortBy"`
	SortOrder string `json:"sortOrder" form:"sortOrder" validate:"oneof=asc desc"`
}

// Health Check Response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
}

// Statistics Response
type StatsResponse struct {
	Users     UserStats      `json:"users"`
	Circles   CircleStats    `json:"circles"`
	Messages  MessageStats   `json:"messages"`
	Emergency EmergencyStats `json:"emergency"`
}

type UserStats struct {
	Total       int64 `json:"total"`
	Active      int64 `json:"active"`
	Online      int64 `json:"online"`
	NewToday    int64 `json:"newToday"`
	NewThisWeek int64 `json:"newThisWeek"`
}

type MessageStats struct {
	Total       int64 `json:"total"`
	Today       int64 `json:"today"`
	ThisWeek    int64 `json:"thisWeek"`
	MediaShared int64 `json:"mediaShared"`
}

type EmergencyStats struct {
	Total           int64   `json:"total"`
	Active          int64   `json:"active"`
	ResolvedToday   int64   `json:"resolvedToday"`
	FalseAlarms     int64   `json:"falseAlarms"`
	AvgResponseTime float64 `json:"avgResponseTime"`
}

// Error Response Codes
const (
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeAuthentication = "AUTHENTICATION_ERROR"
	ErrCodeAuthorization  = "AUTHORIZATION_ERROR"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeConflict       = "CONFLICT"
	ErrCodeRateLimit      = "RATE_LIMIT_EXCEEDED"
	ErrCodeInternal       = "INTERNAL_ERROR"
	ErrCodeExternal       = "EXTERNAL_SERVICE_ERROR"
)
