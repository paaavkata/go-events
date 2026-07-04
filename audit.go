package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// AuditTopic is the single canonical Kafka topic for the platform audit log.
// Every service that performs an audited action publishes an AuditEvent here,
// partition-keyed by app_id, and event-service is the sole consumer/sink.
const AuditTopic = "audit-events"

// AuditConsumerGroup is the canonical consumer group used by event-service.
const AuditConsumerGroup = "event-service-group"

// Actor type constants. Actors reuse the generic (type, uid) Subject identity
// shape used across the platform rather than a domain-specific identity tuple.
const (
	ActorTypeUser      = "user"
	ActorTypeAnonymous = "anonymous"
	ActorTypeService   = "service"
	ActorTypeSystem    = "system"
)

// AuditActor identifies who performed an audited action. It deliberately mirrors
// the platform Subject (type, uid) shape and carries no per-app meaning.
type AuditActor struct {
	Type string `json:"type"`          // "user" | "anonymous" | "service" | "system"
	UID  string `json:"uid,omitempty"` // user id / subject external id / service key
	IP   string `json:"ip,omitempty"`  // source IP, when known
}

// AuditTarget optionally identifies the object an audited action affected.
type AuditTarget struct {
	Type string `json:"type,omitempty"` // "file" | "job" | "customer" | ...
	UID  string `json:"uid,omitempty"`
}

// AuditEvent is the canonical envelope for the platform audit log. Every audited
// action across every service is published to the Kafka topic AuditTopic
// ("audit-events"), partition-keyed by AppID, and consumed and stored by
// event-service.
//
// It is intentionally generic: event-service validates and stores it but never
// interprets Type, Service or Metadata. AppID is REQUIRED and is never defaulted
// by a platform service.
type AuditEvent struct {
	Version   int             `json:"version"`            // envelope schema version (start at 1)
	UID       string          `json:"uid"`                // producer-generated event UUID; dedupe key
	AppID     string          `json:"app_id"`             // REQUIRED — tenant partition
	Type      string          `json:"type"`               // the audited action, e.g. "file.deleted"
	Service   string          `json:"service"`            // emitting service key, e.g. "file-service"
	Timestamp time.Time       `json:"timestamp"`          // when the action happened
	Trace     string          `json:"trace,omitempty"`    // request/trace correlation id
	Actor     AuditActor      `json:"actor"`              // who did it
	Severity  string          `json:"severity,omitempty"` // INFO/WARNING/ERROR/... (maps to severities)
	Target    *AuditTarget    `json:"target,omitempty"`   // optional affected object
	Message   string          `json:"message,omitempty"`  // human-readable summary
	Metadata  json.RawMessage `json:"metadata,omitempty"` // free-form, opaque action payload
}

// Validate enforces the required envelope fields. A platform audit sink must
// reject (not default) events missing app_id, type or uid.
func (e *AuditEvent) Validate() error {
	if e == nil {
		return errors.New("audit event is nil")
	}
	if e.AppID == "" {
		return errors.New("audit event: app_id is required")
	}
	if e.Type == "" {
		return errors.New("audit event: type is required")
	}
	if e.UID == "" {
		return errors.New("audit event: uid is required")
	}
	return nil
}

// DecodeMetadata unmarshals the event's opaque Metadata into dst.
func (e *AuditEvent) DecodeMetadata(dst interface{}) error {
	if len(e.Metadata) == 0 {
		return nil
	}
	if err := json.Unmarshal(e.Metadata, dst); err != nil {
		return fmt.Errorf("events.DecodeMetadata %s: %w", e.Type, err)
	}
	return nil
}
