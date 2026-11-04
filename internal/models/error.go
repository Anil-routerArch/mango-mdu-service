package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ApiError represents the normalized OpenWiFi/MDU API error response payload.
type ApiError struct {
	ErrorCode        int    `json:"ErrorCode"`
	ErrorDescription string `json:"ErrorDescription"`
	ErrorDetails     string `json:"ErrorDetails,omitempty"`
}

// Error implements the standard Go error interface.
func (e ApiError) Error() string {
	if e.ErrorDetails != "" {
		return fmt.Sprintf("%d %s: %s", e.ErrorCode, e.ErrorDescription, e.ErrorDetails)
	}
	return fmt.Sprintf("%d %s", e.ErrorCode, e.ErrorDescription)
}

// NewApiError creates a new ApiError envelope.
func NewApiError(code int, description string, details ...string) ApiError {
	err := ApiError{
		ErrorCode:        code,
		ErrorDescription: description,
	}
	if len(details) > 0 && details[0] != "" {
		err.ErrorDetails = details[0]
	}
	return err
}

// ParseHttpError parses an HTTP response status code and body into an ApiError,
// extracting any error message or description directly returned by downstream services.
func ParseHttpError(statusCode int, body []byte) ApiError {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 {
		var raw map[string]interface{}
		if err := json.Unmarshal(trimmed, &raw); err == nil {
			var desc string
			var details string

			// 1. Primary error message / description
			for _, k := range []string{"ErrorDescription", "errorDescription", "ErrorText", "errorText", "error", "message", "description"} {
				if v, ok := raw[k]; ok {
					if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
						desc = strings.TrimSpace(s)
						break
					}
				}
			}

			// 2. Error details
			for _, k := range []string{"ErrorDetails", "errorDetails", "details", "Details"} {
				if v, ok := raw[k]; ok {
					if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
						details = strings.TrimSpace(s)
						break
					}
				}
			}

			if desc != "" && details != "" && desc != details {
				return NewApiError(statusCode, desc, details)
			}
			if desc != "" {
				return NewApiError(statusCode, desc)
			}
			if details != "" {
				return NewApiError(statusCode, details)
			}
		}

		// Not JSON, check if it's plain text (not HTML)
		str := string(trimmed)
		if !strings.HasPrefix(str, "<") && len(str) < 500 {
			return NewApiError(statusCode, str)
		}
	}

	// Fallback to standard HTTP status text
	statusText := http.StatusText(statusCode)
	if statusText == "" {
		statusText = fmt.Sprintf("HTTP %d", statusCode)
	}
	return NewApiError(statusCode, statusText)
}

// ParseResponseError reads the response body up to maxBytes and converts it to ApiError.
func ParseResponseError(resp *http.Response, maxBytes int64) ApiError {
	if resp == nil {
		return NewApiError(http.StatusInternalServerError, "Internal Server Error")
	}
	if maxBytes <= 0 {
		maxBytes = 2048
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	return ParseHttpError(resp.StatusCode, body)
}
