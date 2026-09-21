// Package events defines the canonical event schema (published via NATS JetStream) shared across all services.
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

// PaymentEvent is the canonical message envelope (published via NATS JetStream) for all payment events.
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

// Reasons a payment.refunded event was emitted.
const (
	// RefundReasonRefund — the merchant or the customer refunded the charge.
	RefundReasonRefund = "refund"
	// RefundReasonDispute — the customer disputed (charged back) the payment.
	// The money is withheld immediately, before the dispute is decided, so
	// consumers should treat it exactly like a refund rather than waiting.
	RefundReasonDispute = "dispute"
)

// PaymentRefundedData is the payload for payment.refunded.
//
// It is what lets a consumer reverse whatever the payment bought — above all
// usage-service, which claws back purchased credits. The envelope carries the
// subject (app_id + customer_uid/user_id/org_id); this payload carries the
// money and what it was for.
//
// All fields after AmountCents are additive (added for the refund clawback
// path) and omitempty, so a producer or consumer on an older build still reads
// and writes a valid document.
type PaymentRefundedData struct {
	// PaymentUID identifies the local payments row being reversed.
	PaymentUID string `json:"payment_uid"`
	// AmountCents is the amount refunded — NOT necessarily the original charge:
	// a partial refund reports only the part returned.
	AmountCents int `json:"amount_cents"`
	// Credits is how many credits the refunded amount bought, prorated for a
	// partial refund. Zero means the payment granted no credits (e.g. a
	// subscription invoice), not "unknown".
	Credits int `json:"credits,omitempty"`
	// Currency of AmountCents (ISO 4217, lowercase as the provider reports it).
	Currency string `json:"currency,omitempty"`
	// ProviderPaymentID is the provider's charge/payment-intent id, for
	// reconciliation against the provider's own records.
	ProviderPaymentID string `json:"provider_payment_id,omitempty"`
	// ProviderEventID is the provider event that triggered this refund. It is
	// ALSO the envelope's EventID, and is the stable key consumers dedupe the
	// clawback on: delivery is at-least-once, and a credit clawback must not
	// apply twice.
	ProviderEventID string `json:"provider_event_id,omitempty"`
	// Reason is RefundReasonRefund or RefundReasonDispute.
	Reason string `json:"reason,omitempty"`
}

// CreditsPurchasedData is the payload for credits.purchased.
type CreditsPurchasedData struct {
	Amount      int    `json:"amount"`
	AmountCents int    `json:"amount_cents"`
	Currency    string `json:"currency"`
	PaymentUID  string `json:"payment_uid,omitempty"`
}
