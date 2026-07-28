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

	"github.com/buildright/construction-ai-gateway/internal/capability"
	"github.com/buildright/construction-ai-gateway/internal/config"
	"github.com/buildright/construction-ai-gateway/internal/health"
	"github.com/buildright/construction-ai-gateway/internal/ollama"
	"github.com/buildright/construction-ai-gateway/internal/queue"
	"github.com/buildright/construction-ai-gateway/internal/worker"
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
		capability.ModelBinding{Model: cfg.OllamaModelRouting, KeepAlive: cfg.OllamaModelRoutingTTL},
		capability.ModelBinding{Model: cfg.OllamaModelIntent, KeepAlive: cfg.OllamaModelIntentTTL},
		capability.ModelBinding{Model: cfg.OllamaModelTranslate, KeepAlive: cfg.OllamaModelTranslateTTL},
	)
	eventQueue := queue.NewRedisQueue(redisClient, cfg.InputQueue, cfg.OutputQueue, cfg.BRPopTimeout)
	ollamaClient := ollama.NewClient(cfg.OllamaURL)
	appWorker := worker.New(eventQueue, eventQueue, ollamaClient, ollamaClient, registry, logger)

	healthHandler := health.NewHandler(registry, ollamaClient)
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
		"ollama_url", cfg.OllamaURL,
		"ollama_model_routing", cfg.OllamaModelRouting,
		"ollama_model_routing_ttl", cfg.OllamaModelRoutingTTL,
		"ollama_model_intent", cfg.OllamaModelIntent,
		"ollama_model_intent_ttl", cfg.OllamaModelIntentTTL,
		"ollama_model_translate", cfg.OllamaModelTranslate,
		"ollama_model_translate_ttl", cfg.OllamaModelTranslateTTL,
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
