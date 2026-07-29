package worker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mywebsite/construction-ai-gateway/internal/capability"
	"github.com/mywebsite/construction-ai-gateway/internal/cloudevent"
)

func testRegistry() *capability.Registry {
	return capability.NewRegistry(
		capability.ModelBinding{Model: "qwen3:1.7b-q4_K_M", KeepAlive: "5m"},
		capability.ModelBinding{Model: "qwen3:4b-q4_K_M", KeepAlive: "5m"},
		capability.ModelBinding{Model: "qwen3:14b-q4_K_M", KeepAlive: "2m"},
	)
}

func TestHandleIntentSuccess(t *testing.T) {
	request := mustEvent(t, "request_intent.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: `{"intent":"wall-painting","confidence":0.95}`}
	models := &fakeModels{available: true}

	w := New(nil, publisher, ollama, models, testRegistry(), nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	response := publisher.events[0]
	if response.Type != cloudevent.EventTypeRequestCompleted {
		t.Fatalf("Type = %q", response.Type)
	}
	if _, ok := response.Data["model"]; ok {
		t.Fatalf("success payload must not include model")
	}
	input, ok := response.Data["input"].(map[string]any)
	if !ok || input["message"] != "I need my living room painted" {
		t.Fatalf("expected request input echoed, got %v", response.Data["input"])
	}
	if response.Data["capability"] != "intent-classification" {
		t.Fatalf("capability = %v", response.Data["capability"])
	}
	if ollama.model != "qwen3:4b-q4_K_M" {
		t.Fatalf("model used = %q", ollama.model)
	}
	if ollama.keepAlive != "5m" {
		t.Fatalf("intent keepAlive = %q, want 5m", ollama.keepAlive)
	}
}

func TestHandleRoutingSuccess(t *testing.T) {
	request := mustEvent(t, "request_routing.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: `{"capability":"intent-classification"}`}
	models := &fakeModels{available: true}

	w := New(nil, publisher, ollama, models, testRegistry(), nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	result := publisher.events[0].Data["result"].(map[string]any)
	if result["capability"] != "intent-classification" {
		t.Fatalf("result = %v", result)
	}
}

func TestHandleTranslateSuccess(t *testing.T) {
	request := mustEvent(t, "request_translate.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: `{"text":"Electrical installations for homes"}`}
	models := &fakeModels{available: true}

	w := New(nil, publisher, ollama, models, testRegistry(), nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	if ollama.prompt != "Elektrische installaties voor woningen" {
		t.Fatalf("prompt = %q", ollama.prompt)
	}
	if ollama.model != "qwen3:14b-q4_K_M" {
		t.Fatalf("model = %q", ollama.model)
	}
	if ollama.keepAlive != "2m" {
		t.Fatalf("keepAlive = %q", ollama.keepAlive)
	}
	result := publisher.events[0].Data["result"].(map[string]any)
	if result["text"] != "Electrical installations for homes" {
		t.Fatalf("result = %v", result)
	}
}

func TestHandleModelUnavailable(t *testing.T) {
	request := mustEvent(t, "request_intent.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: `{"intent":"x","confidence":1}`}
	models := &fakeModels{available: false}

	w := New(nil, publisher, ollama, models, testRegistry(), nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	response := publisher.events[0]
	if response.Type != cloudevent.EventTypeRequestFailed {
		t.Fatalf("Type = %q", response.Type)
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
	w := New(nil, publisher, &fakeOllama{}, &fakeModels{available: true}, testRegistry(), nil)
	_ = w.handle(context.Background(), event)
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
	w := New(nil, publisher, &fakeOllama{}, &fakeModels{available: true}, testRegistry(), nil)
	_ = w.handle(context.Background(), event)
	if publisher.events[0].Type != cloudevent.EventTypeRequestFailed {
		t.Fatalf("Type = %q", publisher.events[0].Type)
	}
}

func TestHandleOllamaError(t *testing.T) {
	request := mustEvent(t, "request_intent.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{err: errors.New("boom")}
	w := New(nil, publisher, ollama, &fakeModels{available: true}, testRegistry(), nil)
	_ = w.handle(context.Background(), request)
	if publisher.events[0].Data["error"] != "boom" {
		t.Fatalf("error = %v", publisher.events[0].Data["error"])
	}
}

func TestHandleUnparseableOutput(t *testing.T) {
	request := mustEvent(t, "request_intent.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: "not-json"}
	w := New(nil, publisher, ollama, &fakeModels{available: true}, testRegistry(), nil)
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
	keepAlive    string
	result       string
	err          error
	called       bool
}

func (f *fakeOllama) Complete(ctx context.Context, systemPrompt, prompt, model, keepAlive string) (string, error) {
	f.called = true
	f.systemPrompt = systemPrompt
	f.prompt = prompt
	f.model = model
	f.keepAlive = keepAlive
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
