package capability_test

import (
	"testing"

	"github.com/mywebsite/construction-ai-gateway/internal/capability"
)

func TestOverridePolicyAllows(t *testing.T) {
	policy := capability.PolicyFromOrgs([]string{"7", "42"}, 1000)
	if !policy.Allows("7") {
		t.Fatalf("expected org 7 to be allowed")
	}
	if policy.Allows("99") {
		t.Fatalf("expected org 99 to be denied")
	}
	if policy.Allows("") {
		t.Fatalf("expected blank org to be denied")
	}
	if policy.MaxSystemPromptChars != 1000 {
		t.Fatalf("MaxSystemPromptChars = %d", policy.MaxSystemPromptChars)
	}
}

func TestOverridePolicyHolderHotReload(t *testing.T) {
	h := capability.NewOverridePolicyHolder()
	if h.Get().Allows("7") {
		t.Fatalf("default policy should deny")
	}
	h.Store(capability.PolicyFromOrgs([]string{"7"}, 50))
	got := h.Get()
	if !got.Allows("7") {
		t.Fatalf("expected org 7 after Store")
	}
	if got.MaxSystemPromptChars != 50 {
		t.Fatalf("MaxSystemPromptChars = %d", got.MaxSystemPromptChars)
	}
}

func TestPolicyFromOrgsDefaultsMaxChars(t *testing.T) {
	policy := capability.PolicyFromOrgs(nil, 0)
	if policy.MaxSystemPromptChars != capability.DefaultMaxSystemPromptChars {
		t.Fatalf("MaxSystemPromptChars = %d, want %d", policy.MaxSystemPromptChars, capability.DefaultMaxSystemPromptChars)
	}
}

func TestOverridePolicyHolderStoreIsolatesCallerMap(t *testing.T) {
	h := capability.NewOverridePolicyHolder()
	allowed := map[string]struct{}{"7": {}}
	h.Store(capability.OverridePolicy{AllowedOrgs: allowed, MaxSystemPromptChars: 100})
	delete(allowed, "7")
	allowed["99"] = struct{}{}

	got := h.Get()
	if !got.Allows("7") {
		t.Fatalf("stored policy should keep org 7 after caller mutates source map")
	}
	if got.Allows("99") {
		t.Fatalf("caller mutation must not affect stored policy")
	}

	got.AllowedOrgs["42"] = struct{}{}
	if h.Get().Allows("42") {
		t.Fatalf("mutating Get() result must not affect holder")
	}
}
