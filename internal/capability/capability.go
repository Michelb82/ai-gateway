package capability

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	Routing              = "routing"
	IntentClassification = "intent-classification"
	Translate            = "translate"
)

type Definition struct {
	Name         string
	Model        string
	KeepAlive    string
	SystemPrompt string
}

// ModelBinding maps a capability to its Ollama model and keep-alive TTL.
type ModelBinding struct {
	Model     string
	KeepAlive string
}

type Registry struct {
	defs map[string]Definition
}

func NewRegistry(routing, intent, translate ModelBinding) *Registry {
	return &Registry{
		defs: map[string]Definition{
			Routing: {
				Name:         Routing,
				Model:        strings.TrimSpace(routing.Model),
				KeepAlive:    strings.TrimSpace(routing.KeepAlive),
				SystemPrompt: routingSystemPrompt,
			},
			IntentClassification: {
				Name:         IntentClassification,
				Model:        strings.TrimSpace(intent.Model),
				KeepAlive:    strings.TrimSpace(intent.KeepAlive),
				SystemPrompt: intentSystemPrompt,
			},
			Translate: {
				Name:         Translate,
				Model:        strings.TrimSpace(translate.Model),
				KeepAlive:    strings.TrimSpace(translate.KeepAlive),
				SystemPrompt: translateSystemPrompt,
			},
		},
	}
}

func (r *Registry) Get(name string) (Definition, error) {
	def, ok := r.defs[strings.TrimSpace(name)]
	if !ok {
		return Definition{}, fmt.Errorf("unknown capability: %s", name)
	}
	if def.Model == "" {
		return Definition{}, fmt.Errorf("capability %s has no model configured", name)
	}
	return def, nil
}

func (r *Registry) All() []Definition {
	order := []string{Routing, IntentClassification, Translate}
	out := make([]Definition, 0, len(order))
	for _, name := range order {
		if def, ok := r.defs[name]; ok {
			out = append(out, def)
		}
	}
	return out
}

func BuildPrompts(def Definition, input map[string]any) (systemPrompt, userPrompt string, err error) {
	switch def.Name {
	case Routing, IntentClassification:
		message := stringValue(input["message"])
		if message == "" {
			return "", "", fmt.Errorf("data.input.message is required")
		}
		return def.SystemPrompt, message, nil
	case Translate:
		text := stringValue(input["text"])
		if text == "" {
			return "", "", fmt.Errorf("data.input.text is required")
		}
		sourceLocale := stringValue(input["source_locale"])
		if sourceLocale == "" {
			sourceLocale = "nl"
		}
		targetLocale := stringValue(input["target_locale"])
		if targetLocale == "" {
			targetLocale = "en"
		}
		systemPrompt := fmt.Sprintf(def.SystemPrompt, localeLabel(sourceLocale), localeLabel(targetLocale))
		return systemPrompt, text, nil
	default:
		return "", "", fmt.Errorf("unknown capability: %s", def.Name)
	}
}

func ParseResult(capabilityName, raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty model output")
	}

	raw = stripCodeFence(raw)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON model output: %w", err)
	}

	switch capabilityName {
	case Routing:
		capability, _ := parsed["capability"].(string)
		if strings.TrimSpace(capability) == "" {
			return nil, fmt.Errorf("routing result missing capability")
		}
		return map[string]any{"capability": strings.TrimSpace(capability)}, nil
	case IntentClassification:
		intent, _ := parsed["intent"].(string)
		if strings.TrimSpace(intent) == "" {
			return nil, fmt.Errorf("intent-classification result missing intent")
		}
		confidence, ok := asFloat(parsed["confidence"])
		if !ok {
			return nil, fmt.Errorf("intent-classification result missing confidence")
		}
		return map[string]any{
			"intent":     strings.TrimSpace(intent),
			"confidence": confidence,
		}, nil
	case Translate:
		text, _ := parsed["text"].(string)
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("translate result missing text")
		}
		return map[string]any{"text": strings.TrimSpace(text)}, nil
	default:
		return nil, fmt.Errorf("unknown capability: %s", capabilityName)
	}
}

func asFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		f, err := typed.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func stripCodeFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)
	if idx := strings.Index(trimmed, "\n"); idx >= 0 {
		first := strings.ToLower(strings.TrimSpace(trimmed[:idx]))
		if first == "json" {
			trimmed = trimmed[idx+1:]
		}
	}
	if end := strings.LastIndex(trimmed, "```"); end >= 0 {
		trimmed = trimmed[:end]
	}
	return strings.TrimSpace(trimmed)
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", typed))
	default:
		return ""
	}
}

func localeLabel(locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "nl":
		return "Dutch"
	case "en":
		return "English"
	default:
		return locale
	}
}

const routingSystemPrompt = `You are a routing model for a construction platform AI gateway.
Given a user message, choose the single best next AI capability.
Respond with ONLY valid JSON in this exact shape:
{"capability":"<capability-name>"}
Allowed capability values: intent-classification, translate.
Do not include markdown, explanation, or extra fields.`

const intentSystemPrompt = `You are an intent classification model for a construction services platform.
Given a user message, determine the user's intent and a confidence score between 0 and 1.
Respond with ONLY valid JSON in this exact shape:
{"intent":"<intent-slug>","confidence":0.0}
Use lowercase kebab-case for intent (for example wall-painting, bathroom-renovation).
Do not include markdown, explanation, or extra fields.`

const translateSystemPrompt = `You are a professional translator for a construction services website. Translate the following service description from %s to %s.
Respond with ONLY valid JSON in this exact shape:
{"text":"<translated text>"}
Return the translated text with no quotes inside the value beyond normal punctuation, and no markdown or explanation.`
