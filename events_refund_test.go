package events

import (
	"encoding/json"
	"testing"
)

// TestPaymentRefundedData_RoundTrip pins the wire shape consumers depend on.
func TestPaymentRefundedData_RoundTrip(t *testing.T) {
	in := PaymentRefundedData{
		PaymentUID:        "pmt-uid",
		AmountCents:       1999,
		Credits:           500,
		Currency:          "usd",
		ProviderPaymentID: "pi_1",
		ProviderEventID:   "evt_1",
		Reason:            RefundReasonRefund,
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out PaymentRefundedData
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}

// The fields added for the clawback path are additive: a payload written by an
// older producer (payment_uid + amount_cents only) must still decode, and the
// new fields must stay out of the document when unset so an older consumer is
// unaffected.
func TestPaymentRefundedData_BackwardCompatible(t *testing.T) {
	var out PaymentRefundedData
	if err := json.Unmarshal([]byte(`{"payment_uid":"p","amount_cents":100}`), &out); err != nil {
		t.Fatalf("legacy payload must still decode: %v", err)
	}
	if out.PaymentUID != "p" || out.AmountCents != 100 || out.Credits != 0 {
		t.Fatalf("unexpected decode: %+v", out)
	}

	raw, err := json.Marshal(PaymentRefundedData{PaymentUID: "p", AmountCents: 100})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(raw) != `{"payment_uid":"p","amount_cents":100}` {
		t.Fatalf("unset additive fields must be omitted, got %s", raw)
	}
}

// A dispute is reported as a refund with a distinct reason: the money is
// already withheld, so consumers must reverse it immediately either way.
func TestRefundReasons_AreDistinct(t *testing.T) {
	if RefundReasonRefund == RefundReasonDispute {
		t.Fatal("refund and dispute reasons must be distinguishable")
	}
}
