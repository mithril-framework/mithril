package monitoring

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// QueueStats represents queue statistics
type QueueStats struct {
	Driver      string               `json:"driver"`
	Queues      map[string]QueueInfo `json:"queues"`
	Workers     []WorkerInfo         `json:"workers"`
	FailedJobs  []FailedJobInfo      `json:"failed_jobs"`
	LastUpdated time.Time            `json:"last_updated"`
}

// QueueInfo represents information about a specific queue
type QueueInfo struct {
	Name        string `json:"name"`
	Size        int    `json:"size"`
	Processing  int    `json:"processing"`
	Completed   int    `json:"completed"`
	Failed      int    `json:"failed"`
	Throughput  int    `json:"throughput"`    // jobs per minute
	AvgWaitTime int    `json:"avg_wait_time"` // in seconds
}

// WorkerInfo represents information about a worker
type WorkerInfo struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"` // idle, working, stopped
	CurrentJob string    `json:"current_job,omitempty"`
	Processed  int       `json:"processed"`
	Failed     int       `json:"failed"`
	StartedAt  time.Time `json:"started_at"`
	LastJobAt  time.Time `json:"last_job_at,omitempty"`
}

// FailedJobInfo represents information about a failed job
type FailedJobInfo struct {
	ID         string    `json:"id"`
	Job        string    `json:"job"`
	Queue      string    `json:"queue"`
	Error      string    `json:"error"`
	FailedAt   time.Time `json:"failed_at"`
	Retries    int       `json:"retries"`
	MaxRetries int       `json:"max_retries"`
}

// QueueMonitor handles queue monitoring
type QueueMonitor struct {
	stats      *QueueStats
	mu         sync.RWMutex
	collectors map[string]QueueCollector
}

// QueueCollector interface for different queue drivers
type QueueCollector interface {
	GetQueueStats() (map[string]QueueInfo, error)
	GetWorkers() ([]WorkerInfo, error)
	GetFailedJobs() ([]FailedJobInfo, error)
}

// NewQueueMonitor creates a new QueueMonitor
func NewQueueMonitor() *QueueMonitor {
	return &QueueMonitor{
		stats: &QueueStats{
			Driver:      "unknown",
			Queues:      make(map[string]QueueInfo),
			Workers:     []WorkerInfo{},
			FailedJobs:  []FailedJobInfo{},
			LastUpdated: time.Now(),
		},
		collectors: make(map[string]QueueCollector),
	}
}

// RegisterCollector registers a queue collector for a specific driver
func (qm *QueueMonitor) RegisterCollector(driver string, collector QueueCollector) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.collectors[driver] = collector
	qm.stats.Driver = driver
}

// UpdateStats updates the queue statistics
func (qm *QueueMonitor) UpdateStats() error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Update from all registered collectors
	for driver, collector := range qm.collectors {
		queueStats, err := collector.GetQueueStats()
		if err != nil {
			return fmt.Errorf("failed to get queue stats from %s: %w", driver, err)
		}

		// Merge queue stats
		for name, info := range queueStats {
			qm.stats.Queues[name] = info
		}

		workers, err := collector.GetWorkers()
		if err != nil {
			return fmt.Errorf("failed to get workers from %s: %w", driver, err)
		}
		qm.stats.Workers = append(qm.stats.Workers, workers...)

		failedJobs, err := collector.GetFailedJobs()
		if err != nil {
			return fmt.Errorf("failed to get failed jobs from %s: %w", driver, err)
		}
		qm.stats.FailedJobs = append(qm.stats.FailedJobs, failedJobs...)
	}

	qm.stats.LastUpdated = time.Now()
	return nil
}

// GetStats returns the current queue statistics
func (qm *QueueMonitor) GetStats() *QueueStats {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	return qm.stats
}

// GetQueueStatsJSON returns queue statistics as JSON
func (qm *QueueMonitor) GetQueueStatsJSON() ([]byte, error) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	return json.MarshalIndent(qm.stats, "", "  ")
}

// RegisterRoutes registers monitoring routes
func (qm *QueueMonitor) RegisterRoutes(app *fiber.App) {
	// Queue stats endpoint
	app.Get("/monitor/queue", func(c *fiber.Ctx) error {
		if err := qm.UpdateStats(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		}

		stats := qm.GetStats()
		return c.JSON(stats)
	})

	// Queue stats JSON endpoint
	app.Get("/monitor/queue.json", func(c *fiber.Ctx) error {
		if err := qm.UpdateStats(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		}

		jsonData, err := qm.GetQueueStatsJSON()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		}

		c.Set("Content-Type", "application/json")
		return c.Send(jsonData)
	})

	// Queue dashboard HTML
	app.Get("/monitor/queue/dashboard", qm.serveQueueDashboard)
}

// serveQueueDashboard serves the queue monitoring dashboard
func (qm *QueueMonitor) serveQueueDashboard(c *fiber.Ctx) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Queue Monitor - Mithril</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .stat-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .stat-value { font-size: 2em; font-weight: bold; color: #2563eb; }
        .stat-label { color: #6b7280; margin-top: 5px; }
        .queue-list { background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
        .queue-header { background: #f8fafc; padding: 15px 20px; border-bottom: 1px solid #e5e7eb; font-weight: 600; }
        .queue-item { padding: 15px 20px; border-bottom: 1px solid #f3f4f6; display: flex; justify-content: space-between; align-items: center; }
        .queue-item:last-child { border-bottom: none; }
        .queue-name { font-weight: 500; }
        .queue-stats { display: flex; gap: 20px; }
        .queue-stat { text-align: center; }
        .queue-stat-value { font-weight: bold; color: #2563eb; }
        .queue-stat-label { font-size: 0.875em; color: #6b7280; }
        .refresh-btn { background: #2563eb; color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; margin-bottom: 20px; }
        .refresh-btn:hover { background: #1d4ed8; }
        .status-indicator { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 8px; }
        .status-idle { background: #10b981; }
        .status-working { background: #f59e0b; }
        .status-stopped { background: #ef4444; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Queue Monitor</h1>
            <p>Real-time queue statistics and worker status</p>
            <button class="refresh-btn" onclick="refreshStats()">Refresh</button>
        </div>

        <div id="stats-container">
            <!-- Stats will be loaded here -->
        </div>
    </div>

    <script>
        function refreshStats() {
            fetch('/monitor/queue')
                .then(response => response.json())
                .then(data => {
                    updateStats(data);
                })
                .catch(error => {
                    console.error('Error fetching stats:', error);
                });
        }

        function updateStats(data) {
            const container = document.getElementById('stats-container');
            
            // Calculate totals
            let totalQueues = Object.keys(data.queues).length;
            let totalJobs = 0;
            let totalWorkers = data.workers.length;
            let totalFailed = 0;
            
            Object.values(data.queues).forEach(queue => {
                totalJobs += queue.size + queue.processing + queue.completed;
                totalFailed += queue.failed;
            });

            container.innerHTML = 
                '<div class="stats-grid">' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + totalQueues + '</div>' +
                        '<div class="stat-label">Active Queues</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + totalJobs + '</div>' +
                        '<div class="stat-label">Total Jobs</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + totalWorkers + '</div>' +
                        '<div class="stat-label">Workers</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + totalFailed + '</div>' +
                        '<div class="stat-label">Failed Jobs</div>' +
                    '</div>' +
                '</div>' +

                '<div class="queue-list">' +
                    '<div class="queue-header">Queue Status</div>' +
                    Object.entries(data.queues).map(([name, queue]) => 
                        '<div class="queue-item">' +
                            '<div class="queue-name">' + name + '</div>' +
                            '<div class="queue-stats">' +
                                '<div class="queue-stat">' +
                                    '<div class="queue-stat-value">' + queue.size + '</div>' +
                                    '<div class="queue-stat-label">Pending</div>' +
                                '</div>' +
                                '<div class="queue-stat">' +
                                    '<div class="queue-stat-value">' + queue.processing + '</div>' +
                                    '<div class="queue-stat-label">Processing</div>' +
                                '</div>' +
                                '<div class="queue-stat">' +
                                    '<div class="queue-stat-value">' + queue.completed + '</div>' +
                                    '<div class="queue-stat-label">Completed</div>' +
                                '</div>' +
                                '<div class="queue-stat">' +
                                    '<div class="queue-stat-value">' + queue.failed + '</div>' +
                                    '<div class="queue-stat-label">Failed</div>' +
                                '</div>' +
                            '</div>' +
                        '</div>'
                    ).join('') +
                '</div>' +

                '<div class="queue-list" style="margin-top: 20px;">' +
                    '<div class="queue-header">Workers</div>' +
                    data.workers.map(worker => 
                        '<div class="queue-item">' +
                            '<div>' +
                                '<span class="status-indicator status-' + worker.status + '"></span>' +
                                '<strong>' + worker.id + '</strong>' +
                                (worker.current_job ? ' - ' + worker.current_job : '') +
                            '</div>' +
                            '<div class="queue-stats">' +
                                '<div class="queue-stat">' +
                                    '<div class="queue-stat-value">' + worker.processed + '</div>' +
                                    '<div class="queue-stat-label">Processed</div>' +
                                '</div>' +
                                '<div class="queue-stat">' +
                                    '<div class="queue-stat-value">' + worker.failed + '</div>' +
                                    '<div class="queue-stat-label">Failed</div>' +
                                '</div>' +
                            '</div>' +
                        '</div>'
                    ).join('') +
                '</div>';
        }

        // Auto-refresh every 5 seconds
        setInterval(refreshStats, 5000);
        
        // Load initial data
        refreshStats();
    </script>
</body>
</html>`

	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}
