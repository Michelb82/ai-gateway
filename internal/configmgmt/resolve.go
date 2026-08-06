package configmgmt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/mywebsite/construction-ai-gateway/internal/capability"
)

// Resolve turns a validated manifest into a runtime snapshot using rank-0 bindings.
func Resolve(m Manifest) (Snapshot, error) {
	modelsByID := make(map[string]Model, len(m.Models))
	for _, model := range m.Models {
		modelsByID[strings.TrimSpace(model.ID)] = model
	}

	known := capability.Known()
	bindings := make(map[string]ModelBinding, len(known))
	for _, capName := range known {
		refs := m.CapabilityModels[capName]
		var primary *RankedModelRef
		for i := range refs {
			if refs[i].Rank == 0 {
				primary = &refs[i]
				break
			}
		}
		if primary == nil {
			return Snapshot{}, fmt.Errorf("capability %s has no rank 0 model", capName)
		}
		model, ok := modelsByID[strings.TrimSpace(primary.Model)]
		if !ok {
			return Snapshot{}, fmt.Errorf("capability %s references unknown model %q", capName, primary.Model)
		}
		maxChars := primary.MaxInputChars
		if maxChars <= 0 {
			maxChars = capability.DefaultMaxInputChars(capName)
		}
		if maxChars <= 0 {
			return Snapshot{}, fmt.Errorf("capability %s has no max_input_chars default", capName)
		}
		bindings[capName] = ModelBinding{
			BaseURL:       strings.TrimRight(strings.TrimSpace(model.URL), "/"),
			Model:         strings.TrimSpace(model.Model),
			KeepAlive:     fmt.Sprintf("%ds", model.KeepAliveSeconds),
			MaxInputChars: maxChars,
		}
	}

	snap := Snapshot{
		RedisAddr:            strings.TrimSpace(m.Ingress.Address),
		InputQueue:           strings.TrimSpace(m.Ingress.IngressChannel),
		OutputQueue:          strings.TrimSpace(m.Ingress.EgressChannel),
		BRPopTimeout:         m.Ingress.BRPopTimeoutSeconds,
		CloudEventTypePrefix: strings.TrimSpace(m.Config.MessagePrefix),
		HTTPAddr:             strings.TrimSpace(m.Config.HTTPAddress),
		PriorityHighCount:    m.Config.PriorityCountHigh,
		PriorityMediumCount:  m.Config.PriorityCountMedium,
		Bindings:             bindings,
	}
	snap.Fingerprint = fingerprint(snap)
	return snap, nil
}

func fingerprint(s Snapshot) string {
	parts := []string{
		s.RedisAddr,
		s.InputQueue,
		s.OutputQueue,
		fmt.Sprintf("%d", s.BRPopTimeout),
		s.CloudEventTypePrefix,
		s.HTTPAddr,
		fmt.Sprintf("%d", s.PriorityHighCount),
		fmt.Sprintf("%d", s.PriorityMediumCount),
	}
	keys := make([]string, 0, len(s.Bindings))
	for k := range s.Bindings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b := s.Bindings[k]
		parts = append(parts, k, b.BaseURL, b.Model, b.KeepAlive, fmt.Sprintf("%d", b.MaxInputChars))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}
