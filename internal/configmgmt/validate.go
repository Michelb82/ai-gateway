package configmgmt

import (
	"fmt"
	"strings"
)

var requiredCapabilities = []string{
	"routing",
	"intent-classification",
	"translate",
}

// Validate checks structural and referential integrity of a manifest.
func Validate(m Manifest) error {
	if len(m.Capabilities) == 0 {
		return fmt.Errorf("manifest.capabilities must not be empty")
	}

	capSet := make(map[string]struct{}, len(m.Capabilities))
	for _, name := range m.Capabilities {
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("manifest.capabilities contains a blank entry")
		}
		if _, dup := capSet[name]; dup {
			return fmt.Errorf("manifest.capabilities duplicates %q", name)
		}
		capSet[name] = struct{}{}
	}
	for _, required := range requiredCapabilities {
		if _, ok := capSet[required]; !ok {
			return fmt.Errorf("manifest.capabilities missing required %q", required)
		}
	}

	if len(m.Models) == 0 {
		return fmt.Errorf("manifest.models must not be empty")
	}
	modelsByID := make(map[string]Model, len(m.Models))
	for i, model := range m.Models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			return fmt.Errorf("manifest.models[%d].id must not be blank", i)
		}
		if strings.TrimSpace(model.URL) == "" {
			return fmt.Errorf("manifest.models[%d].url must not be blank", i)
		}
		if strings.TrimSpace(model.Model) == "" {
			return fmt.Errorf("manifest.models[%d].model must not be blank", i)
		}
		if model.KeepAliveSeconds < 1 {
			return fmt.Errorf("manifest.models[%d].keep_alive_seconds must be >= 1", i)
		}
		if _, dup := modelsByID[id]; dup {
			return fmt.Errorf("manifest.models duplicates id %q", id)
		}
		modelsByID[id] = model
	}

	if m.CapabilityModels == nil {
		return fmt.Errorf("manifest.capability_models must not be empty")
	}
	for _, required := range requiredCapabilities {
		refs, ok := m.CapabilityModels[required]
		if !ok || len(refs) == 0 {
			return fmt.Errorf("manifest.capability_models missing entries for %q", required)
		}
		seenRanks := make(map[int]struct{}, len(refs))
		hasRank0 := false
		for j, ref := range refs {
			if _, dup := seenRanks[ref.Rank]; dup {
				return fmt.Errorf("manifest.capability_models[%q] duplicates rank %d", required, ref.Rank)
			}
			seenRanks[ref.Rank] = struct{}{}
			if ref.Rank == 0 {
				hasRank0 = true
			}
			modelID := strings.TrimSpace(ref.Model)
			if modelID == "" {
				return fmt.Errorf("manifest.capability_models[%q][%d].model must not be blank", required, j)
			}
			if _, ok := modelsByID[modelID]; !ok {
				return fmt.Errorf("manifest.capability_models[%q] references unknown model id %q", required, modelID)
			}
			if ref.MaxInputChars < 0 {
				return fmt.Errorf("manifest.capability_models[%q][%d].max_input_chars must be >= 0", required, j)
			}
		}
		if !hasRank0 {
			return fmt.Errorf("manifest.capability_models[%q] must include rank 0", required)
		}
	}

	if strings.TrimSpace(m.Ingress.Adapter) == "" {
		return fmt.Errorf("manifest.ingress.adapter must not be blank")
	}
	if strings.ToLower(strings.TrimSpace(m.Ingress.Adapter)) != "redis" {
		return fmt.Errorf("manifest.ingress.adapter %q is not supported", m.Ingress.Adapter)
	}
	if strings.TrimSpace(m.Ingress.Address) == "" {
		return fmt.Errorf("manifest.ingress.address must not be blank")
	}
	if strings.TrimSpace(m.Ingress.IngressChannel) == "" {
		return fmt.Errorf("manifest.ingress.ingress_channel must not be blank")
	}
	if strings.TrimSpace(m.Ingress.EgressChannel) == "" {
		return fmt.Errorf("manifest.ingress.egress_channel must not be blank")
	}
	if m.Ingress.BRPopTimeoutSeconds < 1 {
		return fmt.Errorf("manifest.ingress.brpop_timeout_seconds must be >= 1")
	}

	if strings.TrimSpace(m.Config.MessagePrefix) == "" {
		return fmt.Errorf("manifest.config.message_prefix must not be blank")
	}
	if strings.TrimSpace(m.Config.HTTPAddress) == "" {
		return fmt.Errorf("manifest.config.http_address must not be blank")
	}
	if m.Config.PriorityCountHigh < 1 {
		return fmt.Errorf("manifest.config.priority_count_high must be >= 1")
	}
	if m.Config.PriorityCountMedium < 1 {
		return fmt.Errorf("manifest.config.priority_count_medium must be >= 1")
	}

	return nil
}
