package models_test

import (
	"encoding/json"
	"testing"

	"github.com/routerarchitects/mango-mdu-service/internal/models"
)

func TestPolicyOverviewResponseSerialization(t *testing.T) {
	resp := models.PolicyOverviewResponse{
		Policy: models.PolicyMetadata{
			ID:          "523e4567-e89b-12d3-a456-426614174000",
			Name:        "Network Operator",
			Description: "Monitor devices and manage network configuration.",
			Entity:      "",
			Venue:       "",
			Created:     1725000000,
			Modified:    1725500000,
		},
		TotalUsers:             2,
		TotalScopedAssignments: 3,
		TotalProperties:        1,
		TotalVenues:            2,
		UsersWithPolicy: []models.UserWithPolicySummary{
			{
				ID:                     "4b96f542-2304-4c15-b19e-f43d21d93c50",
				Name:                   "Anita Sharma",
				Email:                  "anita@ipnx.example",
				UserRole:               "noc",
				Avatar:                 "1",
				ScopedAssignmentsCount: 2,
				Scopes: []models.UserPolicyScopeInfo{
					{
						EntityID:   "e290f1ee-6c54-4b01-90e6-d701748f0851",
						EntityName: "Sunrise Apartments",
						VenueID:    "f47ac10b-58cc-4372-a567-0e02b2c3d479",
						VenueName:  "Tower A",
					},
					{
						EntityID:   "e290f1ee-6c54-4b01-90e6-d701748f0851",
						EntityName: "Sunrise Apartments",
						VenueID:    "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
						VenueName:  "Tower B",
					},
				},
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal PolicyOverviewResponse: %v", err)
	}

	var parsed models.PolicyOverviewResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal PolicyOverviewResponse: %v", err)
	}

	if parsed.TotalUsers != 2 {
		t.Errorf("expected TotalUsers=2, got %d", parsed.TotalUsers)
	}
	if parsed.TotalScopedAssignments != 3 {
		t.Errorf("expected TotalScopedAssignments=3, got %d", parsed.TotalScopedAssignments)
	}
	if parsed.Policy.Name != "Network Operator" {
		t.Errorf("expected policy name 'Network Operator', got %q", parsed.Policy.Name)
	}
	if len(parsed.UsersWithPolicy) != 1 || len(parsed.UsersWithPolicy[0].Scopes) != 2 {
		t.Errorf("unexpected users or scopes count: %+v", parsed.UsersWithPolicy)
	}
}

func TestApiError(t *testing.T) {
	errWithDetails := models.NewApiError(400, "Bad Request", "Malformed UUID")
	if errWithDetails.Error() != "400 Bad Request: Malformed UUID" {
		t.Errorf("unexpected error string: %s", errWithDetails.Error())
	}

	errWithoutDetails := models.NewApiError(404, "Not Found")
	if errWithoutDetails.Error() != "404 Not Found" {
		t.Errorf("unexpected error string: %s", errWithoutDetails.Error())
	}

	data, err := json.Marshal(errWithDetails)
	if err != nil {
		t.Fatalf("failed to marshal ApiError: %v", err)
	}

	var parsed models.ApiError
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal ApiError: %v", err)
	}

	if parsed.ErrorCode != 400 || parsed.ErrorDescription != "Bad Request" || parsed.ErrorDetails != "Malformed UUID" {
		t.Errorf("unexpected parsed ApiError: %+v", parsed)
	}
}

func TestParseHttpError(t *testing.T) {
	tests := []struct {
		name         string
		code         int
		body         string
		expectedDesc string
		expectedDet  string
	}{
		{
			name:         "JSON with ErrorDescription",
			code:         403,
			body:         `{"ErrorCode": 403, "ErrorDescription": "Access denied"}`,
			expectedDesc: "Access denied",
		},
		{
			name:         "JSON with ErrorDescription and ErrorDetails",
			code:         400,
			body:         `{"ErrorDescription": "Device cannot be deleted", "ErrorDetails": "Device is assigned to venue"}`,
			expectedDesc: "Device cannot be deleted",
			expectedDet:  "Device is assigned to venue",
		},
		{
			name:         "JSON with ErrorText",
			code:         403,
			body:         `{"ErrorText": "User has insufficient permissions"}`,
			expectedDesc: "User has insufficient permissions",
		},
		{
			name:         "JSON with error and message",
			code:         500,
			body:         `{"error": "Database error", "message": "Connection refused"}`,
			expectedDesc: "Database error",
		},
		{
			name:         "Plain text error message",
			code:         403,
			body:         `Access denied: invalid role`,
			expectedDesc: "Access denied: invalid role",
		},
		{
			name:         "Empty body fallback to status text",
			code:         403,
			body:         ``,
			expectedDesc: "Forbidden",
		},
		{
			name:         "HTML body fallback to status text",
			code:         502,
			body:         `<html><body>502 Bad Gateway nginx</body></html>`,
			expectedDesc: "Bad Gateway",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := models.ParseHttpError(tc.code, []byte(tc.body))
			if apiErr.ErrorCode != tc.code {
				t.Errorf("expected code %d, got %d", tc.code, apiErr.ErrorCode)
			}
			if apiErr.ErrorDescription != tc.expectedDesc {
				t.Errorf("expected desc %q, got %q", tc.expectedDesc, apiErr.ErrorDescription)
			}
			if apiErr.ErrorDetails != tc.expectedDet {
				t.Errorf("expected details %q, got %q", tc.expectedDet, apiErr.ErrorDetails)
			}
		})
	}
}
