package models

// ManagementPolicyEntry represents a single resource access permission in OWPROV.
type ManagementPolicyEntry struct {
	Resources []string `json:"resources"`
	Access    []string `json:"access"`
}

// ManagementPolicy represents a policy record in OWPROV.
type ManagementPolicy struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Entity      string                  `json:"entity,omitempty"`
	Venue       string                  `json:"venue,omitempty"`
	Entries     []ManagementPolicyEntry `json:"entries,omitempty"`
	Created     int64                   `json:"created,omitempty"`
	Modified    int64                   `json:"modified,omitempty"`
}

// ManagementRole represents a management role assignment record in OWPROV.
type ManagementRole struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	ManagementPolicy string   `json:"managementPolicy"`
	Users            []string `json:"users"`
	Entity           string   `json:"entity,omitempty"`
	Venue            string   `json:"venue,omitempty"`
	Created          int64    `json:"created,omitempty"`
	Modified         int64    `json:"modified,omitempty"`
}

// ManagementRoleListResponse represents the JSON response envelope from OWPROV for role queries.
type ManagementRoleListResponse struct {
	Roles []ManagementRole `json:"roles"`
}

// Entity represents a property/entity record in OWPROV.
type Entity struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Parent      string   `json:"parent,omitempty"`
	Venues      []string `json:"venues,omitempty"`
}

// EntityListResponse represents the JSON response envelope from OWPROV for entity queries.
type EntityListResponse struct {
	Entities []Entity `json:"entities"`
}

// Venue represents a venue record in OWPROV.
type Venue struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Entity      string `json:"entity,omitempty"`
}

// VenueListResponse represents the JSON response envelope from OWPROV for venue queries.
type VenueListResponse struct {
	Venues []Venue `json:"venues"`
}

// SecUser represents a user profile record in OWSEC.
type SecUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	UserRole    string `json:"userRole"`
	Avatar      string `json:"avatar,omitempty"`
	Description string `json:"description,omitempty"`
}

// SecUserListResponse represents the JSON response envelope from OWSEC for user queries.
type SecUserListResponse struct {
	Users []SecUser `json:"users"`
}
