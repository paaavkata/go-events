// Package events defines the canonical Kafka event schema shared across all services.
// The PaymentEvent envelope is published by payment-service and consumed by
// usage-service and service-service.
package events

import (
	"encoding/json"
	"fmt"
	"time"
)

// Event type constants.
const (
	EventCustomerCreated         = "customer.created"
	EventSubscriptionCreated     = "subscription.created"
	EventSubscriptionRenewed     = "subscription.renewed"
	EventSubscriptionCanceled    = "subscription.canceled"
	EventSubscriptionPlanChanged = "subscription.plan_changed"
	EventPaymentSucceeded        = "payment.succeeded"
	EventPaymentFailed           = "payment.failed"
	EventPaymentRefunded         = "payment.refunded"
	EventCreditsPurchased        = "credits.purchased"
)

// PaymentEvent is the canonical Kafka message envelope for all payment events.
// Data holds the event-specific payload as a raw JSON object; use Decode to
// unmarshal it into the appropriate typed struct.
type PaymentEvent struct {
	Version       int             `json:"version"`
	MessageType   string          `json:"message_type"`
	Producer      string          `json:"producer"`
	Timestamp     time.Time       `json:"timestamp"`
	CorrelationID string          `json:"correlation_id"`
	EventID       string          `json:"event_id,omitempty"`
	AppID         string          `json:"app_id"`
	CustomerUID   string          `json:"customer_uid"`
	UserID        string          `json:"user_id,omitempty"`
	OrgID         string          `json:"org_id,omitempty"`
	Data          json.RawMessage `json:"data"`
}

// Decode unmarshals the event's Data into dst.
func (e *PaymentEvent) Decode(dst interface{}) error {
	if err := json.Unmarshal(e.Data, dst); err != nil {
		return fmt.Errorf("events.Decode %s: %w", e.MessageType, err)
	}
	return nil
}

// CustomerCreatedData is the payload for customer.created.
type CustomerCreatedData struct {
	Email string `json:"email"`
}

// SubscriptionData is the payload for subscription.created, .renewed, and .plan_changed.
type SubscriptionData struct {
	PlanUID         string          `json:"plan_uid"`
	PlanName        string          `json:"plan_name"`
	PlanSlug        string          `json:"plan_slug"`
	IsFree          bool            `json:"is_free"`
	MonthlyCredits  int             `json:"monthly_credits"`
	Features        json.RawMessage `json:"features,omitempty"`
	Quotas          json.RawMessage `json:"quotas,omitempty"`
	SubscriptionUID string          `json:"subscription_uid"`
}

// SubscriptionCanceledData is the payload for subscription.canceled.
type SubscriptionCanceledData struct {
	SubscriptionUID   string `json:"subscription_uid"`
	CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
}

// PaymentSucceededData is the payload for payment.succeeded.
type PaymentSucceededData struct {
	SubscriptionUID string `json:"subscription_uid"`
	ProviderEventID string `json:"provider_event_id"`
	AmountCents     int    `json:"amount_cents"`
	Currency        string `json:"currency"`
}

// PaymentFailedData is the payload for payment.failed.
type PaymentFailedData struct {
	SubscriptionUID string `json:"subscription_uid"`
}

// PaymentRefundedData is the payload for payment.refunded.
type PaymentRefundedData struct {
	PaymentUID  string `json:"payment_uid"`
	AmountCents int    `json:"amount_cents"`
}

// CreditsPurchasedData is the payload for credits.purchased.
type CreditsPurchasedData struct {
	Amount      int    `json:"amount"`
	AmountCents int    `json:"amount_cents"`
	Currency    string `json:"currency"`
	PaymentUID  string `json:"payment_uid,omitempty"`
}
