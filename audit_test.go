package events

import (
	"context"
	"errors"
	"testing"
)

func TestAuditEventValidate(t *testing.T) {
	cases := []struct {
		name    string
		ev      AuditEvent
		wantErr bool
	}{
		{"valid", AuditEvent{AppID: "fileconvert", Type: "file.deleted", UID: "abc"}, false},
		{"missing app_id", AuditEvent{Type: "file.deleted", UID: "abc"}, true},
		{"missing type", AuditEvent{AppID: "fileconvert", UID: "abc"}, true},
		{"missing uid", AuditEvent{AppID: "fileconvert", Type: "file.deleted"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.ev.Validate()
			if (err != nil) != c.wantErr {
				t.Fatalf("Validate() err=%v wantErr=%v", err, c.wantErr)
			}
		})
	}
}

func TestAuditEventDecodeMetadata(t *testing.T) {
	ev := AuditEvent{Metadata: []byte(`{"k":"v"}`)}
	var out map[string]string
	if err := ev.DecodeMetadata(&out); err != nil {
		t.Fatalf("DecodeMetadata: %v", err)
	}
	if out["k"] != "v" {
		t.Fatalf("got %v", out)
	}

	// Empty metadata is a no-op, not an error.
	empty := AuditEvent{}
	if err := empty.DecodeMetadata(&out); err != nil {
		t.Fatalf("empty DecodeMetadata: %v", err)
	}
}

type fakeAuditPublisher struct {
	appID, key, msgID string
	value             interface{}
	err               error
	calls             int
}

func (f *fakeAuditPublisher) SendScopedWithMsgID(_ context.Context, appID, key, msgID string, value interface{}) error {
	f.calls++
	f.appID, f.key, f.msgID, f.value = appID, key, msgID, value
	return f.err
}

func TestAuditSubject(t *testing.T) {
	cases := map[string]string{
		"fileconvert": "audit-events.fileconvert",
		"scantinel":   "audit-events.scantinel",
		"platform":    "audit-events.platform",
	}
	for app, want := range cases {
		if got := AuditSubject(app); got != want {
			t.Errorf("AuditSubject(%q) = %q, want %q", app, got, want)
		}
	}
}

func TestPublishAudit(t *testing.T) {
	ev := &AuditEvent{AppID: "fileconvert", Type: "file.deleted", UID: "uid-1"}

	cases := []struct {
		name      string
		ev        *AuditEvent
		pubErr    error
		wantErr   bool
		wantCalls int
	}{
		{"routes to app scope", ev, nil, false, 1},
		{"invalid event not published", &AuditEvent{Type: "x", UID: "u"}, nil, true, 0},
		{"nil event", nil, nil, true, 0},
		{"publisher error wrapped", ev, errors.New("boom"), true, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := &fakeAuditPublisher{err: c.pubErr}
			err := PublishAudit(context.Background(), p, c.ev)
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
			if c.pubErr != nil && !errors.Is(err, c.pubErr) {
				t.Errorf("publisher error not wrapped: %v", err)
			}
			if p.calls != c.wantCalls {
				t.Fatalf("calls=%d want %d", p.calls, c.wantCalls)
			}
			if c.wantCalls == 1 && (p.appID != "fileconvert" || p.key != "fileconvert" || p.msgID != "uid-1" || p.value != c.ev) {
				t.Errorf("published appID=%q key=%q msgID=%q value=%v", p.appID, p.key, p.msgID, p.value)
			}
		})
	}

	if err := PublishAudit(context.Background(), nil, ev); err == nil {
		t.Error("nil publisher must error")
	}
}
