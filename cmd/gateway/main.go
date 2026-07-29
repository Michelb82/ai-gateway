package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mywebsite/construction-ai-gateway/internal/capability"
	"github.com/mywebsite/construction-ai-gateway/internal/cloudevent"
	"github.com/mywebsite/construction-ai-gateway/internal/config"
	"github.com/mywebsite/construction-ai-gateway/internal/health"
	"github.com/mywebsite/construction-ai-gateway/internal/ollama"
	"github.com/mywebsite/construction-ai-gateway/internal/queue"
	"github.com/mywebsite/construction-ai-gateway/internal/worker"
	"github.com/redis/go-redis/v9"
)

const debugLogPath = "debug.log"

func main() {
	bootstrap := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		bootstrap.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	cloudevent.ConfigureTypes(cfg.CloudEventTypePrefix)

	logger, debugFile, err := newLogger(cfg.Debug)
	if err != nil {
		bootstrap.Error("failed to set up logger", "error", err)
		os.Exit(1)
	}
	if debugFile != nil {
		defer debugFile.Close()
	}
	slog.SetDefault(logger)

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer redisClient.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to redis", "addr", cfg.RedisAddr, "error", err)
		os.Exit(1)
	}

	registry := capability.NewRegistry(
		capability.ModelBinding{BaseURL: cfg.LLMURLRouting, Model: cfg.LLMModelRouting, KeepAlive: cfg.LLMModelRoutingTTL},
		capability.ModelBinding{BaseURL: cfg.LLMURLIntent, Model: cfg.LLMModelIntent, KeepAlive: cfg.LLMModelIntentTTL},
		capability.ModelBinding{BaseURL: cfg.LLMURLTranslate, Model: cfg.LLMModelTranslate, KeepAlive: cfg.LLMModelTranslateTTL},
	)
	eventQueue := queue.NewRedisQueue(
		redisClient,
		cfg.InputQueue,
		cfg.OutputQueue,
		cfg.BRPopTimeout,
		cfg.PriorityHighCount,
		cfg.PriorityMediumCount,
	)
	llmPool := ollama.NewPool()

	targets := make([]ollama.ModelTarget, 0, len(registry.All()))
	for _, def := range registry.All() {
		targets = append(targets, ollama.ModelTarget{BaseURL: def.BaseURL, Name: def.Model})
	}
	logger.Info("ensuring llm models", "targets", targets)
	availableModels, unavailableModels, err := llmPool.EnsureModels(ctx, targets)
	if err != nil {
		logger.Error("llm model readiness failed", "error", err, "unavailable", unavailableModels)
		os.Exit(1)
	}
	if len(unavailableModels) > 0 {
		logger.Warn("some llm models unavailable; continuing with remaining models",
			"available", availableModels,
			"unavailable", unavailableModels,
		)
	} else {
		logger.Info("llm models ready", "available", availableModels)
	}

	appWorker := worker.New(eventQueue, eventQueue, llmPool, llmPool, registry, logger)

	healthHandler := health.NewHandler(registry, llmPool)
	httpServer := health.NewServer(cfg.HTTPAddr, healthHandler)

	go func() {
		logger.Info("health server listening", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health server stopped with error", "error", err)
			stop()
		}
	}()

	logger.Info("ai gateway started",
		"redis_addr", cfg.RedisAddr,
		"input_queue", cfg.InputQueue,
		"output_queue", cfg.OutputQueue,
		"llm_url_routing", cfg.LLMURLRouting,
		"llm_model_routing", cfg.LLMModelRouting,
		"llm_model_routing_ttl", cfg.LLMModelRoutingTTL,
		"llm_url_intent", cfg.LLMURLIntent,
		"llm_model_intent", cfg.LLMModelIntent,
		"llm_model_intent_ttl", cfg.LLMModelIntentTTL,
		"llm_url_translate", cfg.LLMURLTranslate,
		"llm_model_translate", cfg.LLMModelTranslate,
		"llm_model_translate_ttl", cfg.LLMModelTranslateTTL,
		"cloudevent_type_prefix", cfg.CloudEventTypePrefix,
		"priority_high_count", cfg.PriorityHighCount,
		"priority_medium_count", cfg.PriorityMediumCount,
		"http_addr", cfg.HTTPAddr,
		"debug", cfg.Debug,
	)

	workerErr := appWorker.Run(ctx)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("health server shutdown failed", "error", err)
	}

	if workerErr != nil {
		logger.Error("worker stopped with error", "error", workerErr)
		os.Exit(1)
	}

	logger.Info("ai gateway stopped")
}

func newLogger(debug bool) (*slog.Logger, *os.File, error) {
	writer := io.Writer(os.Stdout)
	var debugFile *os.File

	if debug {
		file, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		debugFile = file
		writer = io.MultiWriter(os.Stdout, debugFile)
	}

	return slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})), debugFile, nil
}
