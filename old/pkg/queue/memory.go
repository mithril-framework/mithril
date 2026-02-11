package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryQueue implements Queue using in-memory channels (for testing/dev)
type MemoryQueue struct {
	queues map[string]chan *Job
	failed []*FailedJob
	mu     sync.RWMutex
	closed bool
}

type FailedJob struct {
	Job   *Job
	Error string
	Time  time.Time
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		queues: make(map[string]chan *Job),
		failed: make([]*FailedJob, 0),
	}
}

func (m *MemoryQueue) getQueue(name string) chan *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.queues[name] == nil {
		m.queues[name] = make(chan *Job, 100)
	}
	return m.queues[name]
}

func (m *MemoryQueue) Push(ctx context.Context, job *Job) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return fmt.Errorf("queue is closed")
	}
	m.mu.RUnlock()

	q := m.getQueue(job.Queue)
	select {
	case q <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *MemoryQueue) Pop(ctx context.Context, queue string) (*Job, error) {
	q := m.getQueue(queue)
	select {
	case job := <-q:
		return job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Second):
		return nil, nil
	}
}

func (m *MemoryQueue) Ack(ctx context.Context, job *Job) error {
	// No-op for memory queue
	return nil
}

func (m *MemoryQueue) Fail(ctx context.Context, job *Job, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failed = append(m.failed, &FailedJob{
		Job:   job,
		Error: err.Error(),
		Time:  time.Now(),
	})
	return nil
}

func (m *MemoryQueue) Retry(ctx context.Context, jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, fj := range m.failed {
		if fj.Job.ID == jobID {
			fj.Job.Attempts = 0
			m.failed = append(m.failed[:i], m.failed[i+1:]...)
			m.mu.Unlock()
			return m.Push(ctx, fj.Job)
		}
	}
	return fmt.Errorf("job %s not found in failed queue", jobID)
}

func (m *MemoryQueue) Failed(ctx context.Context, limit int) ([]*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	jobs := make([]*Job, 0, len(m.failed))
	for i, fj := range m.failed {
		if limit > 0 && i >= limit {
			break
		}
		jobs = append(jobs, fj.Job)
	}
	return jobs, nil
}

func (m *MemoryQueue) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	for _, q := range m.queues {
		close(q)
	}
	return nil
}
