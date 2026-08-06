package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mywebsite/construction-ai-gateway/internal/capability"
	"github.com/mywebsite/construction-ai-gateway/internal/cloudevent"
)

type RequestConsumer interface {
	Consume(ctx context.Context) (*cloudevent.Event, error)
}

type ResponsePublisher interface {
	Publish(ctx context.Context, event *cloudevent.Event) error
}

type ChatCompleter interface {
	Complete(ctx context.Context, baseURL, systemPrompt, prompt, model, keepAlive string) (string, error)
}

type ModelChecker interface {
	ModelAvailable(ctx context.Context, baseURL, name string) (bool, error)
}

type CapabilityResolver interface {
	Get(name string) (capability.Definition, error)
}

type Worker struct {
	consumer  RequestConsumer
	publisher ResponsePublisher
	ollama    ChatCompleter
	models    ModelChecker
	registry  CapabilityResolver
	logger    *slog.Logger
}

func New(
	consumer RequestConsumer,
	publisher ResponsePublisher,
	ollama ChatCompleter,
	models ModelChecker,
	registry CapabilityResolver,
	logger *slog.Logger,
) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		consumer:  consumer,
		publisher: publisher,
		ollama:    ollama,
		models:    models,
		registry:  registry,
		logger:    logger,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		event, err := w.consumer.Consume(ctx)
		if err != nil {
			return err
		}
		if event == nil {
			continue
		}

		w.logger.Info("incoming traffic",
			"request_id", event.ID,
			"sender", event.Source,
		)
		w.logger.Debug("incoming traffic payload",
			"request_id", event.ID,
			"sender", event.Source,
			"payload", event.ToMap(),
		)

		if err := w.handle(ctx, event); err != nil {
			w.logger.Error("failed to handle event", "event_id", event.ID, "error", err)
		}
	}
}

func (w *Worker) handle(ctx context.Context, event *cloudevent.Event) error {
	capabilityName, input, err := parseRequest(event)
	if err != nil {
		return w.publishFailure(ctx, event, capabilityName, err)
	}

	if priority := stringValue(event.Data["priority"]); priority != "" {
		w.logger.Info("request priority",
			"request_id", event.ID,
			"capability", capabilityName,
			"priority", priority,
		)
	}

	def, err := w.registry.Get(capabilityName)
	if err != nil {
		return w.publishFailure(ctx, event, capabilityName, err)
	}

	if err := capability.ValidateInputBounds(def, input); err != nil {
		return w.publishFailure(ctx, event, capabilityName, err)
	}

	available, err := w.models.ModelAvailable(ctx, def.BaseURL, def.Model)
	if err != nil {
		w.logger.Error("model availability check failed",
			"capability", capabilityName,
			"llm_url", def.BaseURL,
			"model", def.Model,
			"error", err,
		)
		return w.publishFailure(ctx, event, capabilityName, fmt.Errorf("model unavailable: %v", err))
	}
	if !available {
		w.logger.Error("model unavailable",
			"capability", capabilityName,
			"llm_url", def.BaseURL,
			"model", def.Model,
		)
		return w.publishFailure(ctx, event, capabilityName, fmt.Errorf("model unavailable: %s is not present on %s", def.Model, def.BaseURL))
	}

	systemPrompt, userPrompt, err := capability.BuildPrompts(def, input)
	if err != nil {
		return w.publishFailure(ctx, event, capabilityName, err)
	}

	raw, err := w.ollama.Complete(ctx, def.BaseURL, systemPrompt, userPrompt, def.Model, def.KeepAlive)
	if err != nil {
		return w.publishFailure(ctx, event, capabilityName, err)
	}

	var result map[string]any
	if stringValue(input["system_prompt"]) != "" {
		// Caller-owned prompt ⇒ caller-owned response schema.
		result, err = capability.ParseRawJSON(raw)
	} else {
		result, err = capability.ParseResult(capabilityName, raw)
	}
	if err != nil {
		return w.publishFailure(ctx, event, capabilityName, err)
	}

	response := cloudevent.NewResponse(event, cloudevent.EventTypeRequestCompleted, map[string]any{
		"capability": capabilityName,
		"result":     result,
	})
	return w.publish(ctx, event, response)
}

func (w *Worker) publishFailure(ctx context.Context, event *cloudevent.Event, capabilityName string, cause error) error {
	var errorField any
	if boundsErr, ok := cause.(capability.PromptBoundsError); ok {
		errorField = boundsErr.ToMap()
	} else {
		errorField = cause.Error()
	}
	data := map[string]any{
		"error": errorField,
	}
	if strings.TrimSpace(capabilityName) != "" {
		data["capability"] = capabilityName
	}
	response := cloudevent.NewResponse(event, cloudevent.EventTypeRequestFailed, data)
	return w.publish(ctx, event, response)
}

// responseData builds response payload fields expected by construction PHP consumers,
// including echoed callback context used for handler dispatch.
func responseData(request *cloudevent.Event, data map[string]any) map[string]any {
	if callback, ok := request.Data["callback"]; ok {
		data["callback"] = callback
	}
	return data
}

func (w *Worker) publish(ctx context.Context, request *cloudevent.Event, response *cloudevent.Event) error {
	if err := w.publisher.Publish(ctx, response); err != nil {
		return err
	}
	w.logger.Info("outgoing traffic",
		"request_id", request.ID,
		"sender", request.Source,
	)
	w.logger.Debug("outgoing traffic payload",
		"request_id", request.ID,
		"sender", request.Source,
		"payload", response.ToMap(),
	)
	return nil
}

func parseRequest(event *cloudevent.Event) (capabilityName string, input map[string]any, err error) {
	if event.Type != cloudevent.EventTypeRequest {
		return "", nil, fmt.Errorf("unsupported event type: %s", event.Type)
	}
	if _, hasModel := event.Data["model"]; hasModel {
		return stringValue(event.Data["capability"]), nil, fmt.Errorf("data.model is not allowed; models are selected by the AI gateway")
	}

	capabilityName = stringValue(event.Data["capability"])
	if capabilityName == "" {
		return "", nil, fmt.Errorf("data.capability is required")
	}

	input, _ = event.Data["input"].(map[string]any)
	if input == nil {
		return capabilityName, nil, fmt.Errorf("data.input is required")
	}
	return capabilityName, input, nil
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
