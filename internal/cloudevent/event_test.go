package cloudevent_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/buildright/construction-ai-gateway/internal/cloudevent"
)

func TestFromJSONValid(t *testing.T) {
	raw := readFixture(t, "request_chat.json")

	event, err := cloudevent.FromJSON(raw)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if event.Type != "com.buildright.ai.chat" {
		t.Fatalf("Type = %q", event.Type)
	}
	if event.Source != "/ai-gateway" {
		t.Fatalf("Source = %q", event.Source)
	}
	if event.ID != "abc-123" {
		t.Fatalf("ID = %q", event.ID)
	}
	if event.OrganisationID == nil || *event.OrganisationID != "7" {
		t.Fatalf("OrganisationID = %v", event.OrganisationID)
	}
	if event.DataContentType != cloudevent.DataContentTypeJSON {
		t.Fatalf("DataContentType = %q", event.DataContentType)
	}
	if event.Data["prompt"] != "Translate to English: Hallo wereld" {
		t.Fatalf("prompt = %v", event.Data["prompt"])
	}
}

func TestRoundTripJSON(t *testing.T) {
	raw := readFixture(t, "request_chat.json")
	event, err := cloudevent.FromJSON(raw)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	encoded, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	roundTrip, err := cloudevent.FromJSON(encoded)
	if err != nil {
		t.Fatalf("FromJSON(roundTrip) error = %v", err)
	}

	if roundTrip.Type != event.Type {
		t.Fatalf("Type = %q, want %q", roundTrip.Type, event.Type)
	}
	if roundTrip.Source != event.Source {
		t.Fatalf("Source = %q, want %q", roundTrip.Source, event.Source)
	}
	if roundTrip.ID != event.ID {
		t.Fatalf("ID = %q, want %q", roundTrip.ID, event.ID)
	}
}

func TestFromJSONInvalid(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "invalid json", raw: "{"},
		{name: "missing type", raw: `{"source":"/x","id":"1","data":{}}`},
		{name: "missing source", raw: `{"type":"t","id":"1","data":{}}`},
		{name: "invalid time", raw: `{"type":"t","source":"/x","id":"1","time":"bad","data":{}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cloudevent.FromJSON(tt.raw)
			if err == nil {
				t.Fatalf("FromJSON() expected error")
			}
		})
	}
}

func TestNewResponse(t *testing.T) {
	request := &cloudevent.Event{
		Type:           "com.buildright.ai.chat",
		Source:         "/test",
		ID:             "req-1",
		OrganisationID: strPtr("7"),
		Time:           time.Now().UTC(),
		Data:           map[string]any{"prompt": "hello"},
	}

	response := cloudevent.NewResponse(request, cloudevent.EventTypeChatCompleted, map[string]any{
		"result": "world",
	})

	if response.Type != cloudevent.EventTypeChatCompleted {
		t.Fatalf("Type = %q", response.Type)
	}
	if response.Subject == nil || *response.Subject != "req-1" {
		t.Fatalf("Subject = %v", response.Subject)
	}
	if response.OrganisationID == nil || *response.OrganisationID != "7" {
		t.Fatalf("OrganisationID = %v", response.OrganisationID)
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(raw)
}

func strPtr(value string) *string {
	return &value
}
