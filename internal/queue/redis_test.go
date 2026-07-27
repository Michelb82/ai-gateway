package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/buildright/construction-ai-gateway/internal/cloudevent"
	"github.com/buildright/construction-ai-gateway/internal/queue"
	"github.com/redis/go-redis/v9"
)

func TestPublishUsesQueuePrefix(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	q := queue.NewRedisQueue(client, "ai.requests", "ai.responses", 1)
	event := sampleEvent(t)

	if err := q.Publish(context.Background(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	values, err := mr.List("queue:ai.responses")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("len(values) = %d, want 1", len(values))
	}
}

func TestConsumeReturnsNilOnTimeout(t *testing.T) {
	_, client := newTestRedis(t)

	q := queue.NewRedisQueue(client, "ai.requests", "ai.responses", 1)
	event, err := q.Consume(context.Background())
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if event != nil {
		t.Fatalf("event = %#v, want nil", event)
	}
}

func TestConsumeFIFOOrder(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	mr.Lpush("queue:ai.requests", `{"type":"t","source":"/s","id":"1","data":{}}`)
	mr.Lpush("queue:ai.requests", `{"type":"t","source":"/s","id":"2","data":{}}`)

	q := queue.NewRedisQueue(client, "ai.requests", "ai.responses", 1)

	first, err := q.Consume(context.Background())
	if err != nil {
		t.Fatalf("Consume() first error = %v", err)
	}
	second, err := q.Consume(context.Background())
	if err != nil {
		t.Fatalf("Consume() second error = %v", err)
	}

	if first.ID != "1" {
		t.Fatalf("first.ID = %q, want 1", first.ID)
	}
	if second.ID != "2" {
		t.Fatalf("second.ID = %q, want 2", second.ID)
	}
}

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	return mr, client
}

func sampleEvent(t *testing.T) *cloudevent.Event {
	t.Helper()
	event, err := cloudevent.FromJSON(`{
		"type":"com.buildright.ai.chat",
		"source":"/ai-gateway",
		"id":"abc-123",
		"data":{"prompt":"hello"}
	}`)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}
	return event
}
