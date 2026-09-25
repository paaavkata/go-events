package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// AuditTopic is the base topic of the platform audit log: the JetStream stream
// "audit-events" and the base of its app-scoped subjects. Every service that
// performs an audited action publishes an AuditEvent on AuditSubject(app_id)
// ("audit-events.<app_id>") via PublishAudit; event-service is the sole
// consumer/sink. The flat subject "audit-events" is legacy (pre-scoping) and
// is still captured by the same stream during cutover.
const AuditTopic = "audit-events"

// AuditConsumerGroup is the canonical consumer group used by event-service.
const AuditConsumerGroup = "event-service-group"

// AuditSubject returns the app-scoped subject an audit event for appID is
// published on: "audit-events.<appID>". It mirrors go-nats ScopedSubject
// without validating; publishing through go-nats validates appID.
func AuditSubject(appID string) string {
	return AuditTopic + "." + appID
}

// AuditPublisher publishes one value on the app-scoped subject of its base
// topic. *gonats.Producer (github.com/paaavkata/go-nats) created with
// Topic: AuditTopic satisfies it; the interface keeps this package free of a
// transport dependency.
type AuditPublisher interface {
	SendScopedWithMsgID(ctx context.Context, appID, key, msgID string, value interface{}) error
}

// PublishAudit validates e and publishes it on AuditSubject(e.AppID), keyed by
// AppID, with the event UID as the Nats-Msg-Id so a retried publish inside the
// stream's duplicate window is stored once.
func PublishAudit(ctx context.Context, p AuditPublisher, e *AuditEvent) error {
	if err := e.Validate(); err != nil {
		return fmt.Errorf("publish audit event: %w", err)
	}
	if p == nil {
		return errors.New("publish audit event: publisher is nil")
	}
	if err := p.SendScopedWithMsgID(ctx, e.AppID, e.AppID, e.UID, e); err != nil {
		return fmt.Errorf("publish audit event %s (%s): %w", e.UID, e.Type, err)
	}
	return nil
}

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
// action across every service is published via NATS JetStream on the
// app-scoped subject AuditSubject(AppID) ("audit-events.<app_id>", stream
// AuditTopic) with PublishAudit, and consumed and stored by event-service.
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
