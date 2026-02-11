package schedule

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Task represents a scheduled task
type Task struct {
	Name     string
	Schedule string // Cron expression or interval
	Handler  TaskHandler
	LastRun  time.Time
	NextRun  time.Time
	Enabled  bool
}

// TaskHandler is a function that executes a scheduled task
type TaskHandler func(ctx context.Context) error

// Scheduler manages scheduled tasks
type Scheduler struct {
	tasks   []*Task
	running bool
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:   make([]*Task, 0),
		running: false,
	}
}

// Add adds a task to the scheduler
func (s *Scheduler) Add(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task.Enabled = true
	task.NextRun = s.calculateNextRun(task)
	s.tasks = append(s.tasks, task)
}

// Every schedules a task to run at fixed intervals
func (s *Scheduler) Every(duration time.Duration, name string, handler TaskHandler) {
	task := &Task{
		Name:     name,
		Schedule: fmt.Sprintf("every:%s", duration.String()),
		Handler:  handler,
		Enabled:  true,
	}
	s.Add(task)
}

// Cron schedules a task using cron expression
func (s *Scheduler) Cron(expression string, name string, handler TaskHandler) {
	task := &Task{
		Name:     name,
		Schedule: fmt.Sprintf("cron:%s", expression),
		Handler:  handler,
		Enabled:  true,
	}
	s.Add(task)
}

// Daily schedules a task to run daily at specific time (HH:MM format)
func (s *Scheduler) Daily(timeStr string, name string, handler TaskHandler) {
	task := &Task{
		Name:     name,
		Schedule: fmt.Sprintf("daily:%s", timeStr),
		Handler:  handler,
		Enabled:  true,
	}
	s.Add(task)
}

// Hourly schedules a task to run every hour at specific minute
func (s *Scheduler) Hourly(minute int, name string, handler TaskHandler) {
	task := &Task{
		Name:     name,
		Schedule: fmt.Sprintf("hourly:%d", minute),
		Handler:  handler,
		Enabled:  true,
	}
	s.Add(task)
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()

	go s.run()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
}

// List returns all scheduled tasks
func (s *Scheduler) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := make([]*Task, len(s.tasks))
	copy(tasks, s.tasks)
	return tasks
}

func (s *Scheduler) run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case now := <-ticker.C:
			s.checkAndRunTasks(now)
		}
	}
}

func (s *Scheduler) checkAndRunTasks(now time.Time) {
	s.mu.RLock()
	tasksToRun := make([]*Task, 0)
	for _, task := range s.tasks {
		if task.Enabled && !task.NextRun.IsZero() && now.After(task.NextRun) {
			tasksToRun = append(tasksToRun, task)
		}
	}
	s.mu.RUnlock()

	for _, task := range tasksToRun {
		go s.executeTask(task)
	}
}

func (s *Scheduler) executeTask(task *Task) {
	log.Printf("[Scheduler] Running task: %s", task.Name)

	task.LastRun = time.Now()

	if err := task.Handler(s.ctx); err != nil {
		log.Printf("[Scheduler] Task %s failed: %v", task.Name, err)
	} else {
		log.Printf("[Scheduler] Task %s completed successfully", task.Name)
	}

	s.mu.Lock()
	task.NextRun = s.calculateNextRun(task)
	s.mu.Unlock()
}

func (s *Scheduler) calculateNextRun(task *Task) time.Time {
	now := time.Now()

	// Parse schedule string
	var schedType, schedValue string
	if len(task.Schedule) > 0 {
		parts := splitSchedule(task.Schedule)
		if len(parts) == 2 {
			schedType = parts[0]
			schedValue = parts[1]
		}
	}

	switch schedType {
	case "every":
		duration, err := time.ParseDuration(schedValue)
		if err != nil {
			log.Printf("[Scheduler] Invalid duration for task %s: %v", task.Name, err)
			return time.Time{}
		}
		if task.LastRun.IsZero() {
			return now.Add(duration)
		}
		return task.LastRun.Add(duration)

	case "daily":
		// Parse HH:MM format
		var hour, minute int
		_, err := fmt.Sscanf(schedValue, "%d:%d", &hour, &minute)
		if err != nil {
			log.Printf("[Scheduler] Invalid daily time for task %s: %v", task.Name, err)
			return time.Time{}
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		return next

	case "hourly":
		var minute int
		_, err := fmt.Sscanf(schedValue, "%d", &minute)
		if err != nil {
			log.Printf("[Scheduler] Invalid hourly minute for task %s: %v", task.Name, err)
			return time.Time{}
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), minute, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(time.Hour)
		}
		return next

	case "cron":
		// Basic cron parsing (simplified - full cron would need a library)
		// For now, just return next hour as fallback
		log.Printf("[Scheduler] Cron expressions not fully implemented yet for task %s", task.Name)
		return now.Add(time.Hour)

	default:
		log.Printf("[Scheduler] Unknown schedule type for task %s: %s", task.Name, schedType)
		return time.Time{}
	}
}

func splitSchedule(schedule string) []string {
	for i, c := range schedule {
		if c == ':' {
			return []string{schedule[:i], schedule[i+1:]}
		}
	}
	return []string{schedule}
}
