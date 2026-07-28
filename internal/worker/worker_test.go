package worker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildright/construction-ai-gateway/internal/capability"
	"github.com/buildright/construction-ai-gateway/internal/cloudevent"
)

func TestHandleIntentSuccess(t *testing.T) {
	request := mustEvent(t, "request_intent.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: `{"intent":"wall-painting","confidence":0.95}`}
	models := &fakeModels{available: true}
	reg := capability.NewRegistry("qwen3:1.7b", "qwen3:4b")

	w := New(nil, publisher, ollama, models, reg, nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.events))
	}
	response := publisher.events[0]
	if response.Type != cloudevent.EventTypeRequestCompleted {
		t.Fatalf("Type = %q", response.Type)
	}
	if response.Subject == nil || *response.Subject != request.ID {
		t.Fatalf("Subject = %v", response.Subject)
	}
	if response.Data["capability"] != "intent-classification" {
		t.Fatalf("capability = %v", response.Data["capability"])
	}
	if _, ok := response.Data["model"]; ok {
		t.Fatalf("success payload must not include model")
	}
	result, ok := response.Data["result"].(map[string]any)
	if !ok {
		t.Fatalf("result type = %T", response.Data["result"])
	}
	if result["intent"] != "wall-painting" {
		t.Fatalf("intent = %v", result["intent"])
	}
	if ollama.model != "qwen3:4b" {
		t.Fatalf("model used = %q", ollama.model)
	}
}

func TestHandleRoutingSuccess(t *testing.T) {
	request := mustEvent(t, "request_routing.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: `{"capability":"intent-classification"}`}
	models := &fakeModels{available: true}
	reg := capability.NewRegistry("qwen3:1.7b", "qwen3:4b")

	w := New(nil, publisher, ollama, models, reg, nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	response := publisher.events[0]
	if response.Type != cloudevent.EventTypeRequestCompleted {
		t.Fatalf("Type = %q", response.Type)
	}
	result := response.Data["result"].(map[string]any)
	if result["capability"] != "intent-classification" {
		t.Fatalf("result = %v", result)
	}
}

func TestHandleModelUnavailable(t *testing.T) {
	request := mustEvent(t, "request_intent.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: `{"intent":"x","confidence":1}`}
	models := &fakeModels{available: false}
	reg := capability.NewRegistry("qwen3:1.7b", "qwen3:4b")

	w := New(nil, publisher, ollama, models, reg, nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	response := publisher.events[0]
	if response.Type != cloudevent.EventTypeRequestFailed {
		t.Fatalf("Type = %q", response.Type)
	}
	if response.Data["capability"] != "intent-classification" {
		t.Fatalf("capability = %v", response.Data["capability"])
	}
	errMsg, _ := response.Data["error"].(string)
	if errMsg == "" {
		t.Fatalf("expected data.error")
	}
	if ollama.called {
		t.Fatalf("Complete should not be called when model unavailable")
	}
}

func TestHandleUnknownCapability(t *testing.T) {
	event := &cloudevent.Event{
		Type:   cloudevent.EventTypeRequest,
		Source: "/test",
		ID:     "1",
		Data: map[string]any{
			"capability": "unknown",
			"input":      map[string]any{"message": "hi"},
		},
	}
	publisher := &fakePublisher{}
	w := New(nil, publisher, &fakeOllama{}, &fakeModels{available: true}, capability.NewRegistry("a", "b"), nil)
	if err := w.handle(context.Background(), event); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if publisher.events[0].Type != cloudevent.EventTypeRequestFailed {
		t.Fatalf("Type = %q", publisher.events[0].Type)
	}
}

func TestHandleRejectsCallerModel(t *testing.T) {
	event := &cloudevent.Event{
		Type:   cloudevent.EventTypeRequest,
		Source: "/test",
		ID:     "1",
		Data: map[string]any{
			"capability": "routing",
			"model":      "sneaky",
			"input":      map[string]any{"message": "hi"},
		},
	}
	publisher := &fakePublisher{}
	w := New(nil, publisher, &fakeOllama{}, &fakeModels{available: true}, capability.NewRegistry("a", "b"), nil)
	_ = w.handle(context.Background(), event)
	if publisher.events[0].Type != cloudevent.EventTypeRequestFailed {
		t.Fatalf("Type = %q", publisher.events[0].Type)
	}
}

func TestHandleOllamaError(t *testing.T) {
	request := mustEvent(t, "request_intent.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{err: errors.New("boom")}
	w := New(nil, publisher, ollama, &fakeModels{available: true}, capability.NewRegistry("a", "b"), nil)
	_ = w.handle(context.Background(), request)
	if publisher.events[0].Data["error"] != "boom" {
		t.Fatalf("error = %v", publisher.events[0].Data["error"])
	}
}

func TestHandleUnparseableOutput(t *testing.T) {
	request := mustEvent(t, "request_intent.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: "not-json"}
	w := New(nil, publisher, ollama, &fakeModels{available: true}, capability.NewRegistry("a", "b"), nil)
	_ = w.handle(context.Background(), request)
	if publisher.events[0].Type != cloudevent.EventTypeRequestFailed {
		t.Fatalf("Type = %q", publisher.events[0].Type)
	}
}

func mustEvent(t *testing.T, fixture string) *cloudevent.Event {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", fixture))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	event, err := cloudevent.FromJSON(string(raw))
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}
	return event
}

type fakePublisher struct {
	events []*cloudevent.Event
}

func (f *fakePublisher) Publish(ctx context.Context, event *cloudevent.Event) error {
	f.events = append(f.events, event)
	return nil
}

type fakeOllama struct {
	systemPrompt string
	prompt       string
	model        string
	result       string
	err          error
	called       bool
}

func (f *fakeOllama) Complete(ctx context.Context, systemPrompt, prompt, model string) (string, error) {
	f.called = true
	f.systemPrompt = systemPrompt
	f.prompt = prompt
	f.model = model
	if f.err != nil {
		return "", f.err
	}
	return f.result, nil
}

type fakeModels struct {
	available bool
	err       error
}

func (f *fakeModels) ModelAvailable(ctx context.Context, name string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.available, nil
}
