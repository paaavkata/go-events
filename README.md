# go-events

A transport-agnostic event schema package. It defines the platform's canonical event structs plus small decode/validate helpers and `PublishAudit`, which publishes through any `AuditPublisher` (e.g. a go-nats producer) without importing a transport. Services publish and consume these events over NATS JetStream via [`github.com/paaavkata/go-nats`](../go-nats).

## Import

```go
import "github.com/paaavkata/go-events"
```

Package name: `events`.

## Main types

- **`PaymentEvent`** (`events.go`) — canonical envelope for payment/subscription events, published by payment-service and consumed by usage-service and service-service. Fields: `Version`, `MessageType`, `Producer`, `Timestamp`, `CorrelationID`, `EventID`, `AppID`, `CustomerUID`, `UserID`, `OrgID`, `Data` (raw JSON payload).
  - Event type constants: `EventCustomerCreated`, `EventSubscriptionCreated`, `EventSubscriptionRenewed`, `EventSubscriptionCanceled`, `EventSubscriptionPlanChanged`, `EventPaymentSucceeded`, `EventPaymentFailed`, `EventPaymentRefunded`, `EventCreditsPurchased`.
  - Typed payloads: `CustomerCreatedData`, `SubscriptionData`, `SubscriptionCanceledData`, `PaymentSucceededData`, `PaymentFailedData`, `PaymentRefundedData`, `CreditsPurchasedData`.
  - Refund reasons: `RefundReasonRefund`, `RefundReasonDispute` (a dispute withholds the money immediately, so consumers reverse it like a refund).
  - `(*PaymentEvent) Decode(dst interface{}) error` — unmarshals `Data` into a typed payload struct.
- **`AuditEvent`** (`audit.go`) — canonical envelope for the platform audit log; every audited action from every service is published to the `AuditTopic` topic, partition-keyed by `AppID`, and consumed/stored solely by event-service. Fields: `Version`, `UID`, `AppID` (required), `Type`, `Service`, `Timestamp`, `Trace`, `Actor` (`AuditActor`), `Severity`, `Target` (`*AuditTarget`), `Message`, `Metadata` (raw JSON).
  - `AuditTopic = "audit-events"` (stream / base topic), `AuditConsumerGroup = "event-service-group"`.
  - `AuditSubject(appID) string` — `"audit-events.<appID>"`, the subject events are published on.
  - `PublishAudit(ctx, p AuditPublisher, e *AuditEvent) error` — validates, then publishes on `AuditSubject(e.AppID)` keyed by `AppID` with `UID` as `Nats-Msg-Id`. `AuditPublisher` is satisfied by a `*gonats.Producer` created with `Topic: events.AuditTopic` (no go-nats import here).
  - Actor type constants: `ActorTypeUser`, `ActorTypeAnonymous`, `ActorTypeService`, `ActorTypeSystem`.
  - `(*AuditEvent) Validate() error` — rejects an event missing `app_id`, `type`, or `uid`.
  - `(*AuditEvent) DecodeMetadata(dst interface{}) error` — unmarshals `Metadata` into a typed struct.
  - Which actions MUST be audited, actor/target/metadata conventions and severity mapping: [`AUDIT.md`](AUDIT.md).

## Usage example

```go
import (
    "github.com/paaavkata/go-events"
    gonats "github.com/paaavkata/go-nats"
)

// Publishing: one go-nats producer on the audit stream, scoped per event.
p, _ := gonats.NewProducer(&gonats.ProducerConfig{URLs: urls, ClientID: "file-service", Topic: events.AuditTopic})
err := events.PublishAudit(ctx, p, &events.AuditEvent{Version: 1, UID: uuid, AppID: appID, Type: "file.deleted"})

// Consuming and validating:
func handle(raw []byte) error {
    var evt events.AuditEvent
    if err := json.Unmarshal(raw, &evt); err != nil {
        return err
    }
    if err := evt.Validate(); err != nil {
        return err
    }
    // ... store evt
    return nil
}
```

_Last verified against code: 2026-09-26_
