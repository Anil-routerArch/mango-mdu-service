package models

// ExtendedEntity represents the nested entity metadata in an inventory tag.
type ExtendedEntity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ExtendedVenue represents the nested venue metadata in an inventory tag.
type ExtendedVenue struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ExtendedSubscriber represents the nested subscriber metadata in an inventory tag.
type ExtendedSubscriber struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	Description string `json:"description,omitempty"`
}

// ExtendedDeviceConfiguration represents the nested config metadata in an inventory tag.
type ExtendedDeviceConfiguration struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// InventoryExtendedInfo holds the extended metadata returned when withExtendedInfo=true.
type InventoryExtendedInfo struct {
	Entity              *ExtendedEntity              `json:"entity,omitempty"`
	Venue               *ExtendedVenue               `json:"venue,omitempty"`
	Subscriber          *ExtendedSubscriber          `json:"subscriber,omitempty"`
	DeviceConfiguration *ExtendedDeviceConfiguration `json:"deviceConfiguration,omitempty"`
}

// InventoryTag represents a device inventory tag in OWPROV.
type InventoryTag struct {
	SerialNumber        string                 `json:"serialNumber"`
	Name                string                 `json:"name,omitempty"`
	Description         string                 `json:"description,omitempty"`
	DeviceType          string                 `json:"deviceType,omitempty"`
	DevClass            string                 `json:"devClass,omitempty"`
	Entity              string                 `json:"entity,omitempty"`
	Venue               string                 `json:"venue,omitempty"`
	Subscriber          string                 `json:"subscriber,omitempty"`
	DeviceConfiguration string                 `json:"deviceConfiguration,omitempty"`
	RealMacAddress      string                 `json:"realMacAddress,omitempty"`
	ExtendedInfo        *InventoryExtendedInfo `json:"extendedInfo,omitempty"`
}

// InventoryTagListResponse represents the list envelope from OWPROV (supports both taglist and tagList).
type InventoryTagListResponse struct {
	TagList  []InventoryTag `json:"taglist"`
	TagList2 []InventoryTag `json:"tagList"`
}

// Tags returns whichever non-empty tag slice is populated.
func (r *InventoryTagListResponse) Tags() []InventoryTag {
	if len(r.TagList) > 0 {
		return r.TagList
	}
	return r.TagList2
}

// CountOnlyResponse represents the count envelope from OWPROV when countOnly=true.
type CountOnlyResponse struct {
	Count int `json:"count"`
}

// GatewayDeviceStatus represents real-time telemetry and status from OWGW.
type GatewayDeviceStatus struct {
	SerialNumber          string   `json:"serialNumber"`
	Compatible            string   `json:"compatible,omitempty"`
	DeviceType            string   `json:"deviceType,omitempty"`
	Connected             bool     `json:"connected"`
	Sanity                *float64 `json:"sanity,omitempty"`
	IPAddress             string   `json:"ipAddress,omitempty"`
	MacAddress            string   `json:"macAddress,omitempty"`
	MemoryUsed            *float64 `json:"memoryUsed,omitempty"`
	Load                  *float64 `json:"load,omitempty"`
	Temperature           *float64 `json:"temperature,omitempty"`
	Firmware              string   `json:"firmware,omitempty"`
	Started               *int64   `json:"started,omitempty"`
	Locale                string   `json:"locale,omitempty"`
	LastContact           any      `json:"lastContact,omitempty"`
	LastRecordedContact   *int64   `json:"lastRecordedContact,omitempty"`
	LastFWUpdate          *int64   `json:"lastFWUpdate,omitempty"`
	ConnectReason         string   `json:"connectReason,omitempty"`
	CertificateExpiryDate *int64   `json:"certificateExpiryDate,omitempty"`
	HasRADIUSSessions     any      `json:"hasRADIUSSessions,omitempty"`
	HasGPS                bool     `json:"hasGPS"`
	BlackListed           bool     `json:"blackListed"`
	Simulated             bool     `json:"simulated"`
	RestrictedDevice      bool     `json:"restrictedDevice"`
	Associations2G        int      `json:"associations_2G"`
	Associations5G        int      `json:"associations_5G"`
	Associations6G        int      `json:"associations_6G"`
	RxBytes               uint64   `json:"rxBytes"`
	TxBytes               uint64   `json:"txBytes"`
}

// GatewayDevicesWithStatusResponse represents the envelope returned by OWGW for ?deviceWithStatus=true.
type GatewayDevicesWithStatusResponse struct {
	DevicesWithStatus []GatewayDeviceStatus `json:"devicesWithStatus"`
}

// DeviceFleetItem is the unified device model combining inventory metadata with real-time gateway status.
type DeviceFleetItem struct {
	SerialNumber          string   `json:"serialNumber"`
	Name                  string   `json:"name"`
	Description           string   `json:"description,omitempty"`
	Model                 string   `json:"model"`
	DeviceGroup           string   `json:"deviceGroup"`
	DeviceType            string   `json:"deviceType"`
	Connected             bool     `json:"connected"`
	Sanity                float64  `json:"sanity"`
	IPAddress             string   `json:"ipAddress"`
	MacAddress            string   `json:"macAddress,omitempty"`
	Locale                string   `json:"locale,omitempty"`
	MemoryUsedPct         *float64 `json:"memoryUsedPct,omitempty"`
	Load                  *float64 `json:"load,omitempty"`
	Temperature           *float64 `json:"temperature,omitempty"`
	Firmware              string   `json:"firmware"`
	UptimeSeconds         *int64   `json:"uptimeSeconds,omitempty"`
	Started               *int64   `json:"started,omitempty"`
	LastContact           any      `json:"lastContact,omitempty"`
	LastContactTimestamp   *int64   `json:"lastContactTimestamp,omitempty"`
	LastContactText       string   `json:"lastContactText,omitempty"`
	LastRecordedContact   *int64   `json:"lastRecordedContact,omitempty"`
	LastFWUpdate          *int64   `json:"lastFWUpdate,omitempty"`
	ConnectReason         string   `json:"connectReason,omitempty"`
	CertificateExpiryDate *int64   `json:"certificateExpiryDate,omitempty"`
	HasRADIUSSessions     bool     `json:"hasRADIUSSessions"`
	HasGPS                bool     `json:"hasGPS"`
	Blacklisted           bool     `json:"blacklisted"`
	Simulated             bool     `json:"simulated"`
	Restricted            bool     `json:"restricted"`
	AssociationsTotal     int      `json:"associationsTotal"`
	Associations2G        int      `json:"associations_2G"`
	Associations5G        int      `json:"associations_5G"`
	Associations6G        int      `json:"associations_6G"`
	RxBytes               uint64   `json:"rxBytes"`
	TxBytes               uint64   `json:"txBytes"`
	EntityID              string   `json:"entityId,omitempty"`
	EntityName            string   `json:"entityName,omitempty"`
	VenueID               string   `json:"venueId,omitempty"`
	VenueName             string   `json:"venueName,omitempty"`
	SubscriberID          string   `json:"subscriberId,omitempty"`
	SubscriberEmail       string   `json:"subscriberEmail,omitempty"`
	ConfigID              string   `json:"configId,omitempty"`
	ConfigName            string   `json:"configName,omitempty"`
}

// PaginatedDevicesResponse is the response structure for GET /api/v1/devices.
type PaginatedDevicesResponse struct {
	Devices    []DeviceFleetItem `json:"devices"`
	TotalCount int               `json:"totalCount"`
	PageIndex  int               `json:"pageIndex"`
	PageSize   int               `json:"pageSize"`
	TotalPages int               `json:"totalPages"`
}

// InventorySerialsResponse is the response structure returned by OWPROV when serialOnly=true.
type InventorySerialsResponse struct {
	SerialNumbers []string `json:"serialNumbers"`
}


