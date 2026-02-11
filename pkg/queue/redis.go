package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// RedisQueue implements Queue using Redis lists
type RedisQueue struct {
	client RedisClient
	prefix string
}

// RedisClient interface for Redis operations
type RedisClient interface {
	RPush(ctx context.Context, key string, values ...interface{}) error
	BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error)
	LPush(ctx context.Context, key string, values ...interface{}) error
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	Del(ctx context.Context, keys ...string) error
	Close() error
}

// NewRedisQueue creates a new Redis queue
func NewRedisQueue(client RedisClient, prefix string) *RedisQueue {
	if prefix == "" {
		prefix = "mithril:queue"
	}
	return &RedisQueue{client: client, prefix: prefix}
}

func (r *RedisQueue) Push(ctx context.Context, job *Job) error {
	data, err := SerializeJob(job)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s:%s", r.prefix, job.Queue)
	return r.client.RPush(ctx, key, data)
}

func (r *RedisQueue) Pop(ctx context.Context, queue string) (*Job, error) {
	key := fmt.Sprintf("%s:%s", r.prefix, queue)
	result, err := r.client.BLPop(ctx, time.Second, key)
	if err != nil || len(result) < 2 {
		return nil, err
	}
	return DeserializeJob([]byte(result[1]))
}

func (r *RedisQueue) Ack(ctx context.Context, job *Job) error {
	// Redis doesn't require explicit ack, job is removed on pop
	return nil
}

func (r *RedisQueue) Fail(ctx context.Context, job *Job, err error) error {
	data, marshalErr := json.Marshal(map[string]interface{}{
		"job":   job,
		"error": err.Error(),
		"time":  time.Now(),
	})
	if marshalErr != nil {
		return marshalErr
	}
	key := fmt.Sprintf("%s:failed", r.prefix)
	return r.client.LPush(ctx, key, data)
}

func (r *RedisQueue) Retry(ctx context.Context, jobID string) error {
	// Find job in failed queue and requeue
	key := fmt.Sprintf("%s:failed", r.prefix)
	failed, err := r.client.LRange(ctx, key, 0, -1)
	if err != nil {
		return err
	}
	for _, data := range failed {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			continue
		}
		jobData, _ := json.Marshal(entry["job"])
		var job Job
		if err := json.Unmarshal(jobData, &job); err != nil {
			continue
		}
		if job.ID == jobID {
			job.Attempts = 0
			return r.Push(context.Background(), &job)
		}
	}
	return fmt.Errorf("job %s not found in failed queue", jobID)
}

func (r *RedisQueue) Failed(ctx context.Context, limit int) ([]*Job, error) {
	key := fmt.Sprintf("%s:failed", r.prefix)
	failed, err := r.client.LRange(ctx, key, 0, int64(limit-1))
	if err != nil {
		return nil, err
	}
	jobs := make([]*Job, 0, len(failed))
	for _, data := range failed {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			continue
		}
		jobData, _ := json.Marshal(entry["job"])
		var job Job
		if err := json.Unmarshal(jobData, &job); err != nil {
			continue
		}
		jobs = append(jobs, &job)
	}
	return jobs, nil
}

func (r *RedisQueue) Close() error {
	return r.client.Close()
}
