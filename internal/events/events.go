package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Supported Kafka topic names produced by OWGW.
const (
	TopicConnection  = "connection"
	TopicState       = "state"
	TopicHealthcheck = "healthcheck"
)

// SystemHeader represents the system info wrapper produced by OWGW KafkaManager.
type SystemHeader struct {
	ID   int64  `json:"id"`
	Host string `json:"host"`
}

// KafkaEnvelope represents the envelope wrapping OWGW Kafka events.
type KafkaEnvelope struct {
	System  SystemHeader    `json:"system"`
	Payload json.RawMessage `json:"payload"`
}

// DeviceEvent represents a normalized real-time device event pushed to UI clients.
type DeviceEvent struct {
	Type         string    `json:"type"` // "connection", "state", "healthcheck"
	SerialNumber string    `json:"serialNumber"`
	Timestamp    int64     `json:"timestamp"`
	Connected    *bool     `json:"connected,omitempty"`
	ConnectionIP string    `json:"connectionIp,omitempty"`
	Firmware     string    `json:"firmware,omitempty"`
	MemoryFree   uint64    `json:"memoryFree,omitempty"`
	MemoryTotal  uint64    `json:"memoryTotal,omitempty"`
	MemoryPct    float64   `json:"memoryPct,omitempty"`
	Load         []float64 `json:"load,omitempty"`
	Sanity       *int      `json:"sanity,omitempty"`
}

type connectionPayload struct {
	SerialNumber  string `json:"serialNumber"`
	Firmware      string `json:"firmware"`
	ConnectionIP  string `json:"connectionIp"`
	Timestamp     int64  `json:"timestamp"`
	Disconnection *struct {
		SerialNumber string `json:"serialNumber"`
		Timestamp    int64  `json:"timestamp"`
	} `json:"disconnection"`
}

type statePayload struct {
	Serial string `json:"serial"`
	State  struct {
		Unit struct {
			Memory struct {
				Free  uint64 `json:"free"`
				Total uint64 `json:"total"`
			} `json:"memory"`
			Load []float64 `json:"load"`
		} `json:"unit"`
		Uptime    uint64 `json:"uptime"`
		Timestamp int64  `json:"timestamp"`
	} `json:"state"`
}

type healthcheckPayload struct {
	Serial   string `json:"serial"`
	Sanity   *int   `json:"sanity"`
	Recorded int64  `json:"recorded"`
}

// NormalizeSerial trims and lowercases a MAC/serial number.
func NormalizeSerial(s string) string {
	clean := strings.ToLower(strings.TrimSpace(s))
	clean = strings.ReplaceAll(clean, ":", "")
	clean = strings.ReplaceAll(clean, "-", "")
	return clean
}

// ParseKafkaMessage parses a raw Kafka message into a normalized DeviceEvent.
func ParseKafkaMessage(topic string, key, value []byte) (*DeviceEvent, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf("empty kafka message payload")
	}

	keySerial := NormalizeSerial(string(key))

	// Extract payload from envelope if wrapped
	payloadBytes := value
	var envelope KafkaEnvelope
	if err := json.Unmarshal(value, &envelope); err == nil && len(envelope.Payload) > 0 {
		payloadBytes = envelope.Payload
	}

	topicLower := strings.ToLower(strings.TrimSpace(topic))

	switch {
	case strings.Contains(topicLower, TopicConnection):
		return parseConnectionEvent(keySerial, payloadBytes)
	case strings.Contains(topicLower, TopicState):
		return parseStateEvent(keySerial, payloadBytes)
	case strings.Contains(topicLower, TopicHealthcheck):
		return parseHealthcheckEvent(keySerial, payloadBytes)
	default:
		return nil, fmt.Errorf("unsupported topic: %s", topic)
	}
}

func parseConnectionEvent(keySerial string, data []byte) (*DeviceEvent, error) {
	var p connectionPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal connection payload: %w", err)
	}

	event := &DeviceEvent{
		Type:      "connection",
		Timestamp: time.Now().Unix(),
	}

	if p.Disconnection != nil {
		dis := false
		event.Connected = &dis
		event.SerialNumber = NormalizeSerial(p.Disconnection.SerialNumber)
		if p.Disconnection.Timestamp > 0 {
			event.Timestamp = p.Disconnection.Timestamp
		}
	} else {
		conn := true
		event.Connected = &conn
		event.SerialNumber = NormalizeSerial(p.SerialNumber)
		event.ConnectionIP = p.ConnectionIP
		event.Firmware = p.Firmware
		if p.Timestamp > 0 {
			event.Timestamp = p.Timestamp
		}
	}

	if event.SerialNumber == "" {
		event.SerialNumber = keySerial
	}

	if event.SerialNumber == "" {
		return nil, fmt.Errorf("missing serialNumber in connection event")
	}

	return event, nil
}

func parseStateEvent(keySerial string, data []byte) (*DeviceEvent, error) {
	var p statePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state payload: %w", err)
	}

	serial := NormalizeSerial(p.Serial)
	if serial == "" {
		serial = keySerial
	}
	if serial == "" {
		return nil, fmt.Errorf("missing serialNumber in state event")
	}

	event := &DeviceEvent{
		Type:         "state",
		SerialNumber: serial,
		Timestamp:    p.State.Timestamp,
		MemoryFree:   p.State.Unit.Memory.Free,
		MemoryTotal:  p.State.Unit.Memory.Total,
		Load:         p.State.Unit.Load,
	}

	if event.Timestamp <= 0 {
		event.Timestamp = time.Now().Unix()
	}

	if event.MemoryTotal > 0 {
		used := event.MemoryTotal - event.MemoryFree
		event.MemoryPct = (float64(used) / float64(event.MemoryTotal)) * 100.0
	}

	return event, nil
}

func parseHealthcheckEvent(keySerial string, data []byte) (*DeviceEvent, error) {
	var p healthcheckPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal healthcheck payload: %w", err)
	}

	serial := NormalizeSerial(p.Serial)
	if serial == "" {
		serial = keySerial
	}
	if serial == "" {
		return nil, fmt.Errorf("missing serialNumber in healthcheck event")
	}

	event := &DeviceEvent{
		Type:         "healthcheck",
		SerialNumber: serial,
		Timestamp:    p.Recorded,
		Sanity:       p.Sanity,
	}

	if event.Timestamp <= 0 {
		event.Timestamp = time.Now().Unix()
	}

	return event, nil
}
