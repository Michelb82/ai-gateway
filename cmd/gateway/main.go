package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/buildright/construction-ai-gateway/internal/config"
	"github.com/buildright/construction-ai-gateway/internal/ollama"
	"github.com/buildright/construction-ai-gateway/internal/queue"
	"github.com/buildright/construction-ai-gateway/internal/worker"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

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

	eventQueue := queue.NewRedisQueue(redisClient, cfg.InputQueue, cfg.OutputQueue, cfg.BRPopTimeout)
	ollamaClient := ollama.NewClient(cfg.OllamaURL, cfg.OllamaModel)
	appWorker := worker.New(eventQueue, eventQueue, ollamaClient, cfg.OllamaModel, logger)

	logger.Info("ai gateway started",
		"redis_addr", cfg.RedisAddr,
		"input_queue", cfg.InputQueue,
		"output_queue", cfg.OutputQueue,
		"ollama_url", cfg.OllamaURL,
		"ollama_model", cfg.OllamaModel,
	)

	if err := appWorker.Run(ctx); err != nil {
		logger.Error("worker stopped with error", "error", err)
		os.Exit(1)
	}

	logger.Info("ai gateway stopped")
}
