package monitoring

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CronStats represents cron/scheduler statistics
type CronStats struct {
	Driver      string          `json:"driver"`
	Tasks       []TaskInfo      `json:"tasks"`
	Executions  []ExecutionInfo `json:"executions"`
	LastUpdated time.Time       `json:"last_updated"`
}

// TaskInfo represents information about a scheduled task
type TaskInfo struct {
	Name         string    `json:"name"`
	Schedule     string    `json:"schedule"`
	Description  string    `json:"description"`
	Enabled      bool      `json:"enabled"`
	LastRun      time.Time `json:"last_run,omitempty"`
	NextRun      time.Time `json:"next_run,omitempty"`
	RunCount     int       `json:"run_count"`
	SuccessCount int       `json:"success_count"`
	FailureCount int       `json:"failure_count"`
	AvgDuration  int       `json:"avg_duration"` // in milliseconds
}

// ExecutionInfo represents information about a task execution
type ExecutionInfo struct {
	TaskName   string    `json:"task_name"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	Duration   int       `json:"duration"` // in milliseconds
	Status     string    `json:"status"`   // running, completed, failed
	Error      string    `json:"error,omitempty"`
	Output     string    `json:"output,omitempty"`
}

// CronMonitor handles cron/scheduler monitoring
type CronMonitor struct {
	stats      *CronStats
	mu         sync.RWMutex
	collectors map[string]CronCollector
}

// CronCollector interface for different scheduler drivers
type CronCollector interface {
	GetTasks() ([]TaskInfo, error)
	GetExecutions(limit int) ([]ExecutionInfo, error)
	GetTaskExecutions(taskName string, limit int) ([]ExecutionInfo, error)
}

// NewCronMonitor creates a new CronMonitor
func NewCronMonitor() *CronMonitor {
	return &CronMonitor{
		stats: &CronStats{
			Driver:      "unknown",
			Tasks:       []TaskInfo{},
			Executions:  []ExecutionInfo{},
			LastUpdated: time.Now(),
		},
		collectors: make(map[string]CronCollector),
	}
}

// RegisterCollector registers a cron collector for a specific driver
func (cm *CronMonitor) RegisterCollector(driver string, collector CronCollector) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.collectors[driver] = collector
	cm.stats.Driver = driver
}

// UpdateStats updates the cron statistics
func (cm *CronMonitor) UpdateStats() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Update from all registered collectors
	for driver, collector := range cm.collectors {
		tasks, err := collector.GetTasks()
		if err != nil {
			return fmt.Errorf("failed to get tasks from %s: %w", driver, err)
		}
		cm.stats.Tasks = append(cm.stats.Tasks, tasks...)

		executions, err := collector.GetExecutions(100) // Get last 100 executions
		if err != nil {
			return fmt.Errorf("failed to get executions from %s: %w", driver, err)
		}
		cm.stats.Executions = append(cm.stats.Executions, executions...)
	}

	cm.stats.LastUpdated = time.Now()
	return nil
}

// GetStats returns the current cron statistics
func (cm *CronMonitor) GetStats() *CronStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.stats
}

// GetCronStatsJSON returns cron statistics as JSON
func (cm *CronMonitor) GetCronStatsJSON() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return json.MarshalIndent(cm.stats, "", "  ")
}

// RegisterRoutes registers monitoring routes
func (cm *CronMonitor) RegisterRoutes(app *fiber.App) {
	// Cron stats endpoint
	app.Get("/monitor/cron", func(c *fiber.Ctx) error {
		if err := cm.UpdateStats(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		}

		stats := cm.GetStats()
		return c.JSON(stats)
	})

	// Cron stats JSON endpoint
	app.Get("/monitor/cron.json", func(c *fiber.Ctx) error {
		if err := cm.UpdateStats(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		}

		jsonData, err := cm.GetCronStatsJSON()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		}

		c.Set("Content-Type", "application/json")
		return c.Send(jsonData)
	})

	// Cron dashboard HTML
	app.Get("/monitor/cron/dashboard", cm.serveCronDashboard)

	// Get executions for a specific task
	app.Get("/monitor/cron/task/:name/executions", func(c *fiber.Ctx) error {
		taskName := c.Params("name")
		limit := c.QueryInt("limit", 50)

		cm.mu.RLock()
		collector, exists := cm.collectors[cm.stats.Driver]
		cm.mu.RUnlock()

		if !exists {
			return c.Status(404).JSON(fiber.Map{
				"error":   true,
				"message": "No collector found",
			})
		}

		executions, err := collector.GetTaskExecutions(taskName, limit)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		}

		return c.JSON(executions)
	})
}

// serveCronDashboard serves the cron monitoring dashboard
func (cm *CronMonitor) serveCronDashboard(c *fiber.Ctx) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Cron Monitor - Mithril</title>
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
        .task-list { background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
        .task-header { background: #f8fafc; padding: 15px 20px; border-bottom: 1px solid #e5e7eb; font-weight: 600; }
        .task-item { padding: 15px 20px; border-bottom: 1px solid #f3f4f6; }
        .task-item:last-child { border-bottom: none; }
        .task-name { font-weight: 500; font-size: 1.1em; margin-bottom: 5px; }
        .task-schedule { color: #6b7280; font-family: monospace; margin-bottom: 10px; }
        .task-stats { display: flex; gap: 20px; margin-bottom: 10px; }
        .task-stat { text-align: center; }
        .task-stat-value { font-weight: bold; color: #2563eb; }
        .task-stat-label { font-size: 0.875em; color: #6b7280; }
        .task-times { display: flex; gap: 20px; font-size: 0.875em; color: #6b7280; }
        .refresh-btn { background: #2563eb; color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; margin-bottom: 20px; }
        .refresh-btn:hover { background: #1d4ed8; }
        .status-indicator { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 8px; }
        .status-enabled { background: #10b981; }
        .status-disabled { background: #ef4444; }
        .status-running { background: #f59e0b; }
        .status-completed { background: #10b981; }
        .status-failed { background: #ef4444; }
        .execution-list { background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; margin-top: 20px; }
        .execution-header { background: #f8fafc; padding: 15px 20px; border-bottom: 1px solid #e5e7eb; font-weight: 600; }
        .execution-item { padding: 15px 20px; border-bottom: 1px solid #f3f4f6; display: flex; justify-content: space-between; align-items: center; }
        .execution-item:last-child { border-bottom: none; }
        .execution-info { flex: 1; }
        .execution-duration { color: #6b7280; font-size: 0.875em; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Cron Monitor</h1>
            <p>Real-time scheduled task monitoring and execution history</p>
            <button class="refresh-btn" onclick="refreshStats()">Refresh</button>
        </div>

        <div id="stats-container">
            <!-- Stats will be loaded here -->
        </div>
    </div>

    <script>
        function refreshStats() {
            fetch('/monitor/cron')
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
            let totalTasks = data.tasks.length;
            let enabledTasks = data.tasks.filter(t => t.enabled).length;
            let totalExecutions = data.executions.length;
            let runningExecutions = data.executions.filter(e => e.status === 'running').length;

            container.innerHTML = 
                '<div class="stats-grid">' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + totalTasks + '</div>' +
                        '<div class="stat-label">Total Tasks</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + enabledTasks + '</div>' +
                        '<div class="stat-label">Enabled Tasks</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + totalExecutions + '</div>' +
                        '<div class="stat-label">Total Executions</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + runningExecutions + '</div>' +
                        '<div class="stat-label">Running Now</div>' +
                    '</div>' +
                '</div>' +

                '<div class="task-list">' +
                    '<div class="task-header">Scheduled Tasks</div>' +
                    data.tasks.map(task => 
                        '<div class="task-item">' +
                            '<div class="task-name">' +
                                '<span class="status-indicator status-' + (task.enabled ? 'enabled' : 'disabled') + '"></span>' +
                                task.name +
                            '</div>' +
                            '<div class="task-schedule">' + task.schedule + '</div>' +
                            '<div class="task-stats">' +
                                '<div class="task-stat">' +
                                    '<div class="task-stat-value">' + task.run_count + '</div>' +
                                    '<div class="task-stat-label">Total Runs</div>' +
                                '</div>' +
                                '<div class="task-stat">' +
                                    '<div class="task-stat-value">' + task.success_count + '</div>' +
                                    '<div class="task-stat-label">Successful</div>' +
                                '</div>' +
                                '<div class="task-stat">' +
                                    '<div class="task-stat-value">' + task.failure_count + '</div>' +
                                    '<div class="task-stat-label">Failed</div>' +
                                '</div>' +
                                '<div class="task-stat">' +
                                    '<div class="task-stat-value">' + task.avg_duration + 'ms</div>' +
                                    '<div class="task-stat-label">Avg Duration</div>' +
                                '</div>' +
                            '</div>' +
                            '<div class="task-times">' +
                                '<div>Last Run: ' + (task.last_run ? new Date(task.last_run).toLocaleString() : 'Never') + '</div>' +
                                '<div>Next Run: ' + (task.next_run ? new Date(task.next_run).toLocaleString() : 'N/A') + '</div>' +
                            '</div>' +
                        '</div>'
                    ).join('') +
                '</div>' +

                '<div class="execution-list">' +
                    '<div class="execution-header">Recent Executions</div>' +
                    data.executions.slice(0, 10).map(execution => 
                        '<div class="execution-item">' +
                            '<div class="execution-info">' +
                                '<div>' +
                                    '<span class="status-indicator status-' + execution.status + '"></span>' +
                                    '<strong>' + execution.task_name + '</strong>' +
                                '</div>' +
                                '<div class="execution-duration">' +
                                    new Date(execution.started_at).toLocaleString() +
                                    (execution.finished_at ? ' - ' + execution.duration + 'ms' : ' - Running...') +
                                '</div>' +
                            '</div>' +
                            '<div>' +
                                (execution.error ? '<span style="color: #ef4444;">' + execution.error + '</span>' : '') +
                            '</div>' +
                        '</div>'
                    ).join('') +
                '</div>';
        }

        // Auto-refresh every 10 seconds
        setInterval(refreshStats, 10000);
        
        // Load initial data
        refreshStats();
    </script>
</body>
</html>`

	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}
