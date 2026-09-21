# go-events

A data/transport-agnostic event schema package. It defines the platform's canonical event structs plus small decode/validate helpers — it does not itself publish or consume anything. Services publish and consume these events over NATS JetStream via [`github.com/paaavkata/go-nats`](../go-nats).

## Import

```go
import "github.com/paaavkata/go-events"
```

Package name: `events`.

## Main types

- **`PaymentEvent`** (`events.go`) — canonical envelope for payment/subscription events, published by payment-service and consumed by usage-service and service-service. Fields: `Version`, `MessageType`, `Producer`, `Timestamp`, `CorrelationID`, `EventID`, `AppID`, `CustomerUID`, `UserID`, `OrgID`, `Data` (raw JSON payload).
  - Event type constants: `EventCustomerCreated`, `EventSubscriptionCreated`, `EventSubscriptionRenewed`, `EventSubscriptionCanceled`, `EventSubscriptionPlanChanged`, `EventPaymentSucceeded`, `EventPaymentFailed`, `EventPaymentRefunded`, `EventCreditsPurchased`.
  - Typed payloads: `CustomerCreatedData`, `SubscriptionData`, `SubscriptionCanceledData`, `PaymentSucceededData`, `PaymentFailedData`, `PaymentRefundedData`, `CreditsPurchasedData`.
  - `(*PaymentEvent) Decode(dst interface{}) error` — unmarshals `Data` into a typed payload struct.
- **`AuditEvent`** (`audit.go`) — canonical envelope for the platform audit log; every audited action from every service is published to the `AuditTopic` topic, partition-keyed by `AppID`, and consumed/stored solely by event-service. Fields: `Version`, `UID`, `AppID` (required), `Type`, `Service`, `Timestamp`, `Trace`, `Actor` (`AuditActor`), `Severity`, `Target` (`*AuditTarget`), `Message`, `Metadata` (raw JSON).
  - `AuditTopic = "audit-events"`, `AuditConsumerGroup = "event-service-group"`.
  - Actor type constants: `ActorTypeUser`, `ActorTypeAnonymous`, `ActorTypeService`, `ActorTypeSystem`.
  - `(*AuditEvent) Validate() error` — rejects an event missing `app_id`, `type`, or `uid`.
  - `(*AuditEvent) DecodeMetadata(dst interface{}) error` — unmarshals `Metadata` into a typed struct.

## Usage example

```go
import (
    "github.com/paaavkata/go-events"
    gonats "github.com/paaavkata/go-nats"
)

// Publishing (via go-nats producer, not shown): marshal an events.AuditEvent to JSON
// and publish it on the events.AuditTopic subject.

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

_Last verified against code: 2026-09-21_
