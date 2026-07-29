package capability_test

import (
	"strings"
	"testing"

	"github.com/mywebsite/construction-ai-gateway/internal/capability"
)

func testRegistry() *capability.Registry {
	return capability.NewRegistry(
		capability.ModelBinding{BaseURL: "http://llm-model:11434", Model: "qwen3:1.7b-q4_K_M", KeepAlive: "5m"},
		capability.ModelBinding{BaseURL: "http://llm-model:11434", Model: "qwen3:4b-q4_K_M", KeepAlive: "5m"},
		capability.ModelBinding{BaseURL: "http://llm-model:11434", Model: "qwen3:14b-q4_K_M", KeepAlive: "2m"},
	)
}

func TestRegistryGet(t *testing.T) {
	reg := testRegistry()

	routing, err := reg.Get(capability.Routing)
	if err != nil {
		t.Fatalf("Get(routing) error = %v", err)
	}
	if routing.Model != "qwen3:1.7b-q4_K_M" {
		t.Fatalf("routing model = %q", routing.Model)
	}
	if routing.BaseURL != "http://llm-model:11434" {
		t.Fatalf("routing url = %q", routing.BaseURL)
	}
	if routing.KeepAlive != "5m" {
		t.Fatalf("routing keep_alive = %q", routing.KeepAlive)
	}

	intent, err := reg.Get(capability.IntentClassification)
	if err != nil {
		t.Fatalf("Get(intent) error = %v", err)
	}
	if intent.KeepAlive != "5m" {
		t.Fatalf("intent keep_alive = %q", intent.KeepAlive)
	}

	translate, err := reg.Get(capability.Translate)
	if err != nil {
		t.Fatalf("Get(translate) error = %v", err)
	}
	if translate.Model != "qwen3:14b-q4_K_M" {
		t.Fatalf("translate model = %q", translate.Model)
	}
	if translate.KeepAlive != "2m" {
		t.Fatalf("translate keep_alive = %q", translate.KeepAlive)
	}
}

func TestRegistryUnknownCapability(t *testing.T) {
	reg := capability.NewRegistry(
		capability.ModelBinding{BaseURL: "http://llm-model:11434", Model: "a", KeepAlive: "1m"},
		capability.ModelBinding{BaseURL: "http://llm-model:11434", Model: "b", KeepAlive: "1m"},
		capability.ModelBinding{BaseURL: "http://llm-model:11434", Model: "c", KeepAlive: "1m"},
	)
	_, err := reg.Get("unknown")
	if err == nil {
		t.Fatalf("expected error for unknown capability")
	}
}

func TestRegistryAllOrder(t *testing.T) {
	reg := testRegistry()
	all := reg.All()
	if len(all) != 3 {
		t.Fatalf("All() len = %d", len(all))
	}
	if all[0].Name != capability.Routing ||
		all[1].Name != capability.IntentClassification ||
		all[2].Name != capability.Translate {
		t.Fatalf("unexpected order: %+v", all)
	}
}

func TestBuildPromptsTranslate(t *testing.T) {
	def, err := testRegistry().Get(capability.Translate)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	systemPrompt, userPrompt, err := capability.BuildPrompts(def, map[string]any{
		"text":          "Hallo wereld",
		"source_locale": "nl",
		"target_locale": "en",
	})
	if err != nil {
		t.Fatalf("BuildPrompts() error = %v", err)
	}
	if userPrompt != "Hallo wereld" {
		t.Fatalf("userPrompt = %q", userPrompt)
	}
	if !strings.Contains(systemPrompt, "Dutch") || !strings.Contains(systemPrompt, "English") {
		t.Fatalf("systemPrompt = %q", systemPrompt)
	}
}

func TestBuildPromptsUsesSystemPromptOverride(t *testing.T) {
	def, err := testRegistry().Get(capability.Routing)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	custom := "Custom routing system prompt for this organisation."
	systemPrompt, userPrompt, err := capability.BuildPrompts(def, map[string]any{
		"message":       `{"customer_request":"muren verven"}`,
		"system_prompt": custom,
	})
	if err != nil {
		t.Fatalf("BuildPrompts() error = %v", err)
	}
	if systemPrompt != custom {
		t.Fatalf("systemPrompt = %q, want override", systemPrompt)
	}
	if userPrompt != `{"customer_request":"muren verven"}` {
		t.Fatalf("userPrompt = %q", userPrompt)
	}
}

func TestBuildPromptsTranslateUsesSystemPromptOverride(t *testing.T) {
	def, err := testRegistry().Get(capability.Translate)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	custom := "Translate this service description carefully."
	systemPrompt, userPrompt, err := capability.BuildPrompts(def, map[string]any{
		"text":          "Hallo",
		"system_prompt": custom,
		"source_locale": "nl",
		"target_locale": "en",
	})
	if err != nil {
		t.Fatalf("BuildPrompts() error = %v", err)
	}
	if systemPrompt != custom {
		t.Fatalf("systemPrompt = %q, want override", systemPrompt)
	}
	if userPrompt != "Hallo" {
		t.Fatalf("userPrompt = %q", userPrompt)
	}
}

func TestParseRawJSON(t *testing.T) {
	result, err := capability.ParseRawJSON(`{"jobs":[{"job_id":1,"confidence":0.9}],"route":"wizard"}`)
	if err != nil {
		t.Fatalf("ParseRawJSON() error = %v", err)
	}
	if result["route"] != "wizard" {
		t.Fatalf("route = %v", result["route"])
	}
}

func TestParseRoutingResult(t *testing.T) {
	result, err := capability.ParseResult(capability.Routing, `{"capability":"intent-classification"}`)
	if err != nil {
		t.Fatalf("ParseResult() error = %v", err)
	}
	if result["capability"] != "intent-classification" {
		t.Fatalf("capability = %v", result["capability"])
	}
}

func TestParseIntentResult(t *testing.T) {
	result, err := capability.ParseResult(capability.IntentClassification, "```json\n{\"intent\":\"wall-painting\",\"confidence\":0.95}\n```")
	if err != nil {
		t.Fatalf("ParseResult() error = %v", err)
	}
	if result["intent"] != "wall-painting" {
		t.Fatalf("intent = %v", result["intent"])
	}
}

func TestParseTranslateResult(t *testing.T) {
	result, err := capability.ParseResult(capability.Translate, `{"text":"Hello world"}`)
	if err != nil {
		t.Fatalf("ParseResult() error = %v", err)
	}
	if result["text"] != "Hello world" {
		t.Fatalf("text = %v", result["text"])
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := capability.ParseResult(capability.Routing, "not-json")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseMissingFields(t *testing.T) {
	_, err := capability.ParseResult(capability.Routing, `{"capability":""}`)
	if err == nil {
		t.Fatalf("expected error for empty capability")
	}
	_, err = capability.ParseResult(capability.IntentClassification, `{"intent":"x"}`)
	if err == nil {
		t.Fatalf("expected error for missing confidence")
	}
	_, err = capability.ParseResult(capability.Translate, `{"text":""}`)
	if err == nil {
		t.Fatalf("expected error for empty text")
	}
}
