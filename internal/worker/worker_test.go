package worker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildright/construction-ai-gateway/internal/cloudevent"
)

func TestHandleSuccessPublishesCompletedEvent(t *testing.T) {
	request := mustEvent(t, "request_chat.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: "Hello world"}

	w := New(nil, publisher, ollama, "default-model", nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.events))
	}

	response := publisher.events[0]
	if response.Type != cloudevent.EventTypeChatCompleted {
		t.Fatalf("Type = %q", response.Type)
	}
	if response.Subject == nil || *response.Subject != request.ID {
		t.Fatalf("Subject = %v, want %q", response.Subject, request.ID)
	}
	if response.Data["status"] != "ok" {
		t.Fatalf("status = %v", response.Data["status"])
	}
	if response.Data["content"] != "Hello world" {
		t.Fatalf("content = %v", response.Data["content"])
	}
	if response.Data["result"] != "Hello world" {
		t.Fatalf("result = %v", response.Data["result"])
	}
}

func TestHandleSuccessEchoesCallback(t *testing.T) {
	request := mustEvent(t, "request_chat.json")
	request.Data["callback"] = map[string]any{
		"handler": "website.mainpage.translate",
		"context": map[string]any{
			"service_id": float64(42),
		},
	}
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: "translated"}

	w := New(nil, publisher, ollama, "default-model", nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	callback, ok := publisher.events[0].Data["callback"].(map[string]any)
	if !ok {
		t.Fatalf("callback missing or wrong type: %v", publisher.events[0].Data["callback"])
	}
	if callback["handler"] != "website.mainpage.translate" {
		t.Fatalf("callback.handler = %v", callback["handler"])
	}
}

func TestHandleOllamaErrorPublishesFailedEvent(t *testing.T) {
	request := mustEvent(t, "request_chat.json")
	request.Data["callback"] = map[string]any{
		"handler": "website.mainpage.translate",
		"context": map[string]any{},
	}
	publisher := &fakePublisher{}
	ollama := &fakeOllama{err: errors.New("model unavailable")}

	w := New(nil, publisher, ollama, "default-model", nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	response := publisher.events[0]
	if response.Type != cloudevent.EventTypeChatFailed {
		t.Fatalf("Type = %q", response.Type)
	}
	if response.Data["status"] != "error" {
		t.Fatalf("status = %v", response.Data["status"])
	}
	if response.Data["error"] != "model unavailable" {
		t.Fatalf("error = %v", response.Data["error"])
	}
	callback, ok := response.Data["callback"].(map[string]any)
	if !ok || callback["handler"] != "website.mainpage.translate" {
		t.Fatalf("callback = %v", response.Data["callback"])
	}
}

func TestHandleTranslationPayloadMapping(t *testing.T) {
	request := mustEvent(t, "request_translation.json")
	publisher := &fakePublisher{}
	ollama := &fakeOllama{result: "Electrical installations for homes"}

	w := New(nil, publisher, ollama, "default-model", nil)
	if err := w.handle(context.Background(), request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	if ollama.prompt != "Elektrische installaties voor woningen" {
		t.Fatalf("prompt = %q", ollama.prompt)
	}
}

func TestResolvePromptsChatPayload(t *testing.T) {
	systemPrompt, prompt, model, err := resolvePrompts(map[string]any{
		"system_prompt": "system",
		"prompt":        "hello",
		"model":         "custom",
	}, "default")
	if err != nil {
		t.Fatalf("resolvePrompts() error = %v", err)
	}
	if systemPrompt != "system" || prompt != "hello" || model != "custom" {
		t.Fatalf("got %q %q %q", systemPrompt, prompt, model)
	}
}

func TestResolvePromptsMissingPayload(t *testing.T) {
	_, _, _, err := resolvePrompts(map[string]any{}, "default")
	if err == nil {
		t.Fatalf("resolvePrompts() expected error")
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
	result       string
	err          error
}

func (f *fakeOllama) Complete(ctx context.Context, systemPrompt, prompt, model string) (string, error) {
	f.systemPrompt = systemPrompt
	f.prompt = prompt
	if f.err != nil {
		return "", f.err
	}
	return f.result, nil
}
