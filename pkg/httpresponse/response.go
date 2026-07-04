package httpresponse

import (
	"encoding/json"
	"net/http"
	"time"
)

// PaginationInfo holds standard pagination metadata.
type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// Response is the standard JSON envelope used by all API endpoints.
type Response struct {
	Success    bool            `json:"success"`
	Message    string          `json:"message,omitempty"`
	Data       any             `json:"data,omitempty"`
	Pagination *PaginationInfo `json:"pagination,omitempty"`
	Error      string          `json:"error,omitempty"`
	ErrorCode  string          `json:"error_code,omitempty"`
	Timestamp  string          `json:"timestamp"`
}

// JSON writes a JSON response with the given status code and payload.
func JSON(w http.ResponseWriter, status int, resp Response) {
	if resp.Timestamp == "" {
		resp.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// Success writes a 200 OK JSON response with optional data.
func Success(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage writes a 200 OK JSON response with a message.
func SuccessWithMessage(w http.ResponseWriter, message string, data any) {
	JSON(w, http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Created writes a 201 Created JSON response.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// Error writes an error JSON response.
func Error(w http.ResponseWriter, status int, errorCode string, message string) {
	JSON(w, status, Response{
		Success:   false,
		Error:     message,
		ErrorCode: errorCode,
	})
}

// InternalError writes a 500 response. The actual error is NOT exposed to clients.
func InternalError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred")
}
