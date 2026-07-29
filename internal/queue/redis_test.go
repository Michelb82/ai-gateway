package queue_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/mywebsite/construction-ai-gateway/internal/cloudevent"
	"github.com/mywebsite/construction-ai-gateway/internal/queue"
	"github.com/redis/go-redis/v9"
)

func TestPublishUsesQueuePrefix(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	q := newQueue(client)
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

	q := newQueue(client)
	event, err := q.Consume(context.Background())
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if event != nil {
		t.Fatalf("event = %#v, want nil", event)
	}
}

func TestConsumeSamePriorityPreservesOrder(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	pushRequest(t, mr, "1", "LOW")
	pushRequest(t, mr, "2", "LOW")

	q := newQueue(client)

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

func TestConsumeCriticalJumpsQueue(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	pushRequest(t, mr, "low-1", "LOW")
	pushRequest(t, mr, "high-1", "HIGH")
	pushRequest(t, mr, "crit-1", "CRITICAL")

	q := newQueue(client)
	first, err := q.Consume(context.Background())
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if first.ID != "crit-1" {
		t.Fatalf("first.ID = %q, want crit-1", first.ID)
	}
}

func TestConsumeMissingPriorityDefaultsToLow(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	mr.Lpush("queue:ai.requests", `{"type":"t","source":"/s","id":"no-prio","data":{}}`)
	pushRequest(t, mr, "high-1", "HIGH")

	q := newQueue(client)
	first, err := q.Consume(context.Background())
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if first.ID != "high-1" {
		t.Fatalf("first.ID = %q, want high-1", first.ID)
	}
}

func TestConsumeFairnessHighMediumLow(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	for i := 1; i <= 9; i++ {
		pushRequest(t, mr, fmt.Sprintf("h-%d", i), "HIGH")
	}
	pushRequest(t, mr, "m-1", "MEDIUM")
	pushRequest(t, mr, "m-2", "MEDIUM")
	pushRequest(t, mr, "m-3", "MEDIUM")
	pushRequest(t, mr, "l-1", "LOW")
	pushRequest(t, mr, "l-2", "LOW")

	q := newQueue(client)
	want := []string{
		"h-1", "h-2", "h-3", "m-1",
		"h-4", "h-5", "h-6", "m-2",
		"h-7", "h-8", "h-9", "l-1",
		"m-3", "l-2",
	}

	for i, id := range want {
		event, err := q.Consume(context.Background())
		if err != nil {
			t.Fatalf("Consume() step %d error = %v", i, err)
		}
		if event == nil {
			t.Fatalf("Consume() step %d returned nil", i)
		}
		if event.ID != id {
			t.Fatalf("step %d: ID = %q, want %q", i, event.ID, id)
		}
	}
}

func newQueue(client *redis.Client) *queue.RedisQueue {
	return queue.NewRedisQueue(client, "ai.requests", "ai.responses", 1, 3, 3)
}

func pushRequest(t *testing.T, mr *miniredis.Miniredis, id, prio string) {
	t.Helper()
	payload := fmt.Sprintf(
		`{"type":"t","source":"/s","id":%q,"data":{"priority":%q}}`,
		id, prio,
	)
	mr.Lpush("queue:ai.requests", payload)
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
		"type":"com.mywebsite.ai.chat",
		"source":"/ai-gateway",
		"id":"abc-123",
		"data":{"prompt":"hello"}
	}`)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}
	return event
}
