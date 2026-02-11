package queue

import (
	"context"
	"encoding/json"
	"time"
)

// Job represents a queued job
type Job struct {
	ID        string                 `json:"id"`
	Queue     string                 `json:"queue"`
	Handler   string                 `json:"handler"`
	Payload   map[string]interface{} `json:"payload"`
	Attempts  int                    `json:"attempts"`
	MaxTries  int                    `json:"max_tries"`
	Timeout   time.Duration          `json:"timeout"`
	CreatedAt time.Time              `json:"created_at"`
	StartedAt *time.Time             `json:"started_at,omitempty"`
}

// Handler processes jobs
type Handler interface {
	Handle(ctx context.Context, job *Job) error
}

// Queue interface for different queue backends
type Queue interface {
	// Push adds a job to the queue
	Push(ctx context.Context, job *Job) error
	// Pop retrieves a job from the queue
	Pop(ctx context.Context, queue string) (*Job, error)
	// Ack acknowledges job completion
	Ack(ctx context.Context, job *Job) error
	// Fail marks a job as failed
	Fail(ctx context.Context, job *Job, err error) error
	// Retry requeues a failed job
	Retry(ctx context.Context, jobID string) error
	// Failed returns failed jobs
	Failed(ctx context.Context, limit int) ([]*Job, error)
	// Close closes the queue connection
	Close() error
}

// Manager manages job handlers and queue operations
type Manager struct {
	queue    Queue
	handlers map[string]Handler
}

// NewManager creates a new queue manager
func NewManager(queue Queue) *Manager {
	return &Manager{
		queue:    queue,
		handlers: make(map[string]Handler),
	}
}

// Register registers a job handler
func (m *Manager) Register(name string, handler Handler) {
	m.handlers[name] = handler
}

// Dispatch pushes a job to the queue
func (m *Manager) Dispatch(ctx context.Context, handlerName string, payload map[string]interface{}) error {
	job := &Job{
		ID:        generateID(),
		Queue:     "default",
		Handler:   handlerName,
		Payload:   payload,
		Attempts:  0,
		MaxTries:  3,
		Timeout:   time.Minute * 5,
		CreatedAt: time.Now(),
	}
	return m.queue.Push(ctx, job)
}

// Work starts processing jobs from the queue
func (m *Manager) Work(ctx context.Context, queueName string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			job, err := m.queue.Pop(ctx, queueName)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			if job == nil {
				time.Sleep(time.Second)
				continue
			}

			if err := m.process(ctx, job); err != nil {
				_ = m.queue.Fail(ctx, job, err)
			} else {
				_ = m.queue.Ack(ctx, job)
			}
		}
	}
}

func (m *Manager) process(ctx context.Context, job *Job) error {
	handler, ok := m.handlers[job.Handler]
	if !ok {
		return &JobError{Message: "handler not found: " + job.Handler}
	}

	jobCtx, cancel := context.WithTimeout(ctx, job.Timeout)
	defer cancel()

	now := time.Now()
	job.StartedAt = &now
	job.Attempts++

	return handler.Handle(jobCtx, job)
}

// JobError represents a job processing error
type JobError struct {
	Message string
}

func (e *JobError) Error() string {
	return e.Message
}

// SerializeJob serializes a job to JSON
func SerializeJob(job *Job) ([]byte, error) {
	return json.Marshal(job)
}

// DeserializeJob deserializes a job from JSON
func DeserializeJob(data []byte) (*Job, error) {
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
