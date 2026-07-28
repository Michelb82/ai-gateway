package capability_test

import (
	"testing"

	"github.com/buildright/construction-ai-gateway/internal/capability"
)

func TestRegistryGet(t *testing.T) {
	reg := capability.NewRegistry("qwen3:1.7b", "qwen3:4b")

	routing, err := reg.Get(capability.Routing)
	if err != nil {
		t.Fatalf("Get(routing) error = %v", err)
	}
	if routing.Model != "qwen3:1.7b" {
		t.Fatalf("routing model = %q", routing.Model)
	}

	intent, err := reg.Get(capability.IntentClassification)
	if err != nil {
		t.Fatalf("Get(intent) error = %v", err)
	}
	if intent.Model != "qwen3:4b" {
		t.Fatalf("intent model = %q", intent.Model)
	}
}

func TestRegistryUnknownCapability(t *testing.T) {
	reg := capability.NewRegistry("a", "b")
	_, err := reg.Get("unknown")
	if err == nil {
		t.Fatalf("expected error for unknown capability")
	}
}

func TestRegistryAllOrder(t *testing.T) {
	reg := capability.NewRegistry("a", "b")
	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("All() len = %d", len(all))
	}
	if all[0].Name != capability.Routing || all[1].Name != capability.IntentClassification {
		t.Fatalf("unexpected order: %+v", all)
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
	if result["confidence"] != 0.95 {
		t.Fatalf("confidence = %v", result["confidence"])
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
}
