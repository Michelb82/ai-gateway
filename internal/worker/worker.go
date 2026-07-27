package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/buildright/construction-ai-gateway/internal/cloudevent"
)

type RequestConsumer interface {
	Consume(ctx context.Context) (*cloudevent.Event, error)
}

type ResponsePublisher interface {
	Publish(ctx context.Context, event *cloudevent.Event) error
}

type ChatCompleter interface {
	Complete(ctx context.Context, systemPrompt, prompt, model string) (string, error)
}

type Worker struct {
	consumer  RequestConsumer
	publisher ResponsePublisher
	ollama    ChatCompleter
	logger    *slog.Logger
	model     string
}

func New(consumer RequestConsumer, publisher ResponsePublisher, ollama ChatCompleter, model string, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		consumer:  consumer,
		publisher: publisher,
		ollama:    ollama,
		logger:    logger,
		model:     model,
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

		if err := w.handle(ctx, event); err != nil {
			w.logger.Error("failed to handle event", "event_id", event.ID, "error", err)
		}
	}
}

func (w *Worker) handle(ctx context.Context, event *cloudevent.Event) error {
	systemPrompt, prompt, model, err := resolvePrompts(event.Data, w.model)
	if err != nil {
		return w.publishFailure(ctx, event, err)
	}

	result, err := w.ollama.Complete(ctx, systemPrompt, prompt, model)
	if err != nil {
		return w.publishFailure(ctx, event, err)
	}

	response := cloudevent.NewResponse(event, cloudevent.EventTypeChatCompleted, map[string]any{
		"result":       result,
		"model":        chooseModel(model, w.model),
		"request_type": event.Type,
	})
	return w.publisher.Publish(ctx, response)
}

func (w *Worker) publishFailure(ctx context.Context, event *cloudevent.Event, cause error) error {
	response := cloudevent.NewResponse(event, cloudevent.EventTypeChatFailed, map[string]any{
		"error":        cause.Error(),
		"request_type": event.Type,
	})
	return w.publisher.Publish(ctx, response)
}

func resolvePrompts(data map[string]any, defaultModel string) (string, string, string, error) {
	model := stringValue(data["model"])
	if model == "" {
		model = defaultModel
	}

	if prompt := stringValue(data["prompt"]); prompt != "" {
		systemPrompt := stringValue(data["system_prompt"])
		if systemPrompt == "" {
			systemPrompt = "You are a helpful assistant."
		}
		return systemPrompt, prompt, model, nil
	}

	text := stringValue(data["text"])
	sourceLocale := stringValue(data["source_locale"])
	targetLocale := stringValue(data["target_locale"])
	if text == "" {
		return "", "", "", fmt.Errorf("invalid event payload: prompt or text is required")
	}
	if sourceLocale == "" {
		sourceLocale = "nl"
	}
	if targetLocale == "" {
		targetLocale = "en"
	}

	sourceLabel := localeLabel(sourceLocale)
	targetLabel := localeLabel(targetLocale)
	systemPrompt := fmt.Sprintf(
		"You are a professional translator for a construction services website. Translate the following service description from %s to %s. Return ONLY the translated text with no quotes, markdown, or explanation.",
		sourceLabel,
		targetLabel,
	)

	return systemPrompt, text, model, nil
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

func chooseModel(requestModel, defaultModel string) string {
	if strings.TrimSpace(requestModel) != "" {
		return requestModel
	}
	return defaultModel
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
