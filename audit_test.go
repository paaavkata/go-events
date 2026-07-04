package events

import "testing"

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
