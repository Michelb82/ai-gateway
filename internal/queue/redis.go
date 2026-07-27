package queue

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buildright/construction-ai-gateway/internal/cloudevent"
	"github.com/redis/go-redis/v9"
)

const keyPrefix = "queue:"

type RedisQueue struct {
	client       *redis.Client
	inputQueue   string
	outputQueue  string
	brpopTimeout time.Duration
}

func NewRedisQueue(client *redis.Client, inputQueue, outputQueue string, brpopTimeoutSeconds int) *RedisQueue {
	return &RedisQueue{
		client:       client,
		inputQueue:   inputQueue,
		outputQueue:  outputQueue,
		brpopTimeout: time.Duration(brpopTimeoutSeconds) * time.Second,
	}
}

func (q *RedisQueue) Consume(ctx context.Context) (*cloudevent.Event, error) {
	key := q.key(q.inputQueue)
	result, err := q.client.BRPop(ctx, q.brpopTimeout, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("consume from %s: %w", key, err)
	}
	if len(result) < 2 {
		return nil, nil
	}

	return cloudevent.FromJSON(result[1])
}

func (q *RedisQueue) Publish(ctx context.Context, event *cloudevent.Event) error {
	payload, err := event.ToJSON()
	if err != nil {
		return err
	}

	key := q.key(q.outputQueue)
	if err := q.client.LPush(ctx, key, payload).Err(); err != nil {
		return fmt.Errorf("publish to %s: %w", key, err)
	}
	return nil
}

func (q *RedisQueue) key(queueName string) string {
	queueName = strings.TrimSpace(queueName)
	return keyPrefix + queueName
}
