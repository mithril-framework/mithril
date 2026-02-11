package monitoring

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SystemStats represents overall system statistics
type SystemStats struct {
	App         AppStats      `json:"app"`
	Database    DatabaseStats `json:"database"`
	Queue       *QueueStats   `json:"queue,omitempty"`
	Cron        *CronStats    `json:"cron,omitempty"`
	Cache       CacheStats    `json:"cache"`
	Storage     StorageStats  `json:"storage"`
	LastUpdated time.Time     `json:"last_updated"`
}

// AppStats represents application statistics
type AppStats struct {
	Name        string  `json:"name"`
	Version     string  `json:"version"`
	Environment string  `json:"environment"`
	Uptime      int64   `json:"uptime"`       // in seconds
	MemoryUsage int64   `json:"memory_usage"` // in bytes
	CPUUsage    float64 `json:"cpu_usage"`    // percentage
	Requests    int64   `json:"requests"`
	Errors      int64   `json:"errors"`
}

// DatabaseStats represents database statistics
type DatabaseStats struct {
	Driver       string `json:"driver"`
	Connected    bool   `json:"connected"`
	ActiveConns  int    `json:"active_connections"`
	IdleConns    int    `json:"idle_connections"`
	MaxConns     int    `json:"max_connections"`
	QueryCount   int64  `json:"query_count"`
	SlowQueries  int64  `json:"slow_queries"`
	AvgQueryTime int    `json:"avg_query_time"` // in milliseconds
}

// CacheStats represents cache statistics
type CacheStats struct {
	Driver      string  `json:"driver"`
	Connected   bool    `json:"connected"`
	HitCount    int64   `json:"hit_count"`
	MissCount   int64   `json:"miss_count"`
	HitRate     float64 `json:"hit_rate"`     // percentage
	MemoryUsage int64   `json:"memory_usage"` // in bytes
}

// StorageStats represents storage statistics
type StorageStats struct {
	Driver     string `json:"driver"`
	Connected  bool   `json:"connected"`
	TotalFiles int64  `json:"total_files"`
	TotalSize  int64  `json:"total_size"` // in bytes
	FreeSpace  int64  `json:"free_space"` // in bytes
}

// SystemMonitor handles overall system monitoring
type SystemMonitor struct {
	stats      *SystemStats
	mu         sync.RWMutex
	startTime  time.Time
	collectors map[string]interface{}
}

// NewSystemMonitor creates a new SystemMonitor
func NewSystemMonitor() *SystemMonitor {
	return &SystemMonitor{
		stats: &SystemStats{
			App:         AppStats{},
			Database:    DatabaseStats{},
			Cache:       CacheStats{},
			Storage:     StorageStats{},
			LastUpdated: time.Now(),
		},
		startTime:  time.Now(),
		collectors: make(map[string]interface{}),
	}
}

// RegisterCollector registers a collector for a specific component
func (sm *SystemMonitor) RegisterCollector(component string, collector interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.collectors[component] = collector
}

// UpdateStats updates the system statistics
func (sm *SystemMonitor) UpdateStats() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Update app stats
	sm.stats.App.Uptime = int64(time.Since(sm.startTime).Seconds())

	// Update from registered collectors
	for component, collector := range sm.collectors {
		switch component {
		case "queue":
			if queueMonitor, ok := collector.(*QueueMonitor); ok {
				queueStats := queueMonitor.GetStats()
				sm.stats.Queue = queueStats
			}
		case "cron":
			if cronMonitor, ok := collector.(*CronMonitor); ok {
				cronStats := cronMonitor.GetStats()
				sm.stats.Cron = cronStats
			}
		}
	}

	sm.stats.LastUpdated = time.Now()
	return nil
}

// GetStats returns the current system statistics
func (sm *SystemMonitor) GetStats() *SystemStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.stats
}

// GetSystemStatsJSON returns system statistics as JSON
func (sm *SystemMonitor) GetSystemStatsJSON() ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return json.MarshalIndent(sm.stats, "", "  ")
}

// RegisterRoutes registers monitoring routes
func (sm *SystemMonitor) RegisterRoutes(app *fiber.App) {
	// Main dashboard
	app.Get("/monitor", sm.serveMainDashboard)

	// System stats endpoint
	app.Get("/monitor/system", func(c *fiber.Ctx) error {
		if err := sm.UpdateStats(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		}

		stats := sm.GetStats()
		return c.JSON(stats)
	})

	// System stats JSON endpoint
	app.Get("/monitor/system.json", func(c *fiber.Ctx) error {
		if err := sm.UpdateStats(); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		}

		jsonData, err := sm.GetSystemStatsJSON()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		}

		c.Set("Content-Type", "application/json")
		return c.Send(jsonData)
	})

	// Health check endpoint
	app.Get("/monitor/health", func(c *fiber.Ctx) error {
		if err := sm.UpdateStats(); err != nil {
			return c.Status(503).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  err.Error(),
				"time":   time.Now().Unix(),
			})
		}

		stats := sm.GetStats()

		// Simple health check logic
		healthy := stats.Database.Connected && stats.Cache.Connected

		if healthy {
			return c.JSON(fiber.Map{
				"status": "healthy",
				"time":   time.Now().Unix(),
				"uptime": stats.App.Uptime,
			})
		}

		return c.Status(503).JSON(fiber.Map{
			"status": "unhealthy",
			"time":   time.Now().Unix(),
			"issues": func() []string {
				var issues []string
				if !stats.Database.Connected {
					issues = append(issues, "Database disconnected")
				}
				if !stats.Cache.Connected {
					issues = append(issues, "Cache disconnected")
				}
				return issues
			}(),
		})
	})
}

// serveMainDashboard serves the main monitoring dashboard
func (sm *SystemMonitor) serveMainDashboard(c *fiber.Ctx) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Mithril Monitor - System Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1400px; margin: 0 auto; }
        .header { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .nav { display: flex; gap: 10px; margin-bottom: 20px; }
        .nav-btn { background: #2563eb; color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; text-decoration: none; }
        .nav-btn:hover { background: #1d4ed8; }
        .nav-btn.active { background: #1d4ed8; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .stat-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .stat-value { font-size: 2em; font-weight: bold; color: #2563eb; }
        .stat-label { color: #6b7280; margin-top: 5px; }
        .section { background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 20px; }
        .section-header { background: #f8fafc; padding: 15px 20px; border-bottom: 1px solid #e5e7eb; font-weight: 600; }
        .section-content { padding: 20px; }
        .status-indicator { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 8px; }
        .status-healthy { background: #10b981; }
        .status-unhealthy { background: #ef4444; }
        .status-warning { background: #f59e0b; }
        .refresh-btn { background: #2563eb; color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; margin-bottom: 20px; }
        .refresh-btn:hover { background: #1d4ed8; }
        .metric-row { display: flex; justify-content: space-between; align-items: center; padding: 10px 0; border-bottom: 1px solid #f3f4f6; }
        .metric-row:last-child { border-bottom: none; }
        .metric-label { font-weight: 500; }
        .metric-value { color: #6b7280; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Mithril System Monitor</h1>
            <p>Real-time system monitoring and health checks</p>
            <div class="nav">
                <a href="/monitor" class="nav-btn active">Overview</a>
                <a href="/monitor/queue/dashboard" class="nav-btn">Queue Monitor</a>
                <a href="/monitor/cron/dashboard" class="nav-btn">Cron Monitor</a>
                <a href="/monitor/system.json" class="nav-btn">JSON API</a>
            </div>
            <button class="refresh-btn" onclick="refreshStats()">Refresh</button>
        </div>

        <div id="stats-container">
            <!-- Stats will be loaded here -->
        </div>
    </div>

    <script>
        function refreshStats() {
            fetch('/monitor/system')
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
            
            // Calculate queue totals
            let queueStats = '';
            if (data.queue) {
                let totalQueues = Object.keys(data.queue.queues).length;
                let totalJobs = 0;
                Object.values(data.queue.queues).forEach(queue => {
                    totalJobs += queue.size + queue.processing + queue.completed;
                });
                queueStats = 
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + totalQueues + '</div>' +
                        '<div class="stat-label">Active Queues</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + totalJobs + '</div>' +
                        '<div class="stat-label">Total Jobs</div>' +
                    '</div>';
            }

            // Calculate cron totals
            let cronStats = '';
            if (data.cron) {
                let totalTasks = data.cron.tasks.length;
                let enabledTasks = data.cron.tasks.filter(t => t.enabled).length;
                cronStats = 
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + totalTasks + '</div>' +
                        '<div class="stat-label">Scheduled Tasks</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + enabledTasks + '</div>' +
                        '<div class="stat-label">Enabled Tasks</div>' +
                    '</div>';
            }

            container.innerHTML = 
                '<div class="stats-grid">' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + data.app.name + '</div>' +
                        '<div class="stat-label">Application</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + data.app.version + '</div>' +
                        '<div class="stat-label">Version</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + data.app.environment + '</div>' +
                        '<div class="stat-label">Environment</div>' +
                    '</div>' +
                    '<div class="stat-card">' +
                        '<div class="stat-value">' + Math.floor(data.app.uptime / 3600) + 'h</div>' +
                        '<div class="stat-label">Uptime</div>' +
                    '</div>' +
                    queueStats +
                    cronStats +
                '</div>' +

                '<div class="section">' +
                    '<div class="section-header">System Health</div>' +
                    '<div class="section-content">' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">' +
                                '<span class="status-indicator status-' + (data.database.connected ? 'healthy' : 'unhealthy') + '"></span>' +
                                'Database' +
                            '</span>' +
                            '<span class="metric-value">' + (data.database.connected ? 'Connected' : 'Disconnected') + '</span>' +
                        '</div>' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">' +
                                '<span class="status-indicator status-' + (data.cache.connected ? 'healthy' : 'unhealthy') + '"></span>' +
                                'Cache' +
                            '</span>' +
                            '<span class="metric-value">' + (data.cache.connected ? 'Connected' : 'Disconnected') + '</span>' +
                        '</div>' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">' +
                                '<span class="status-indicator status-' + (data.storage.connected ? 'healthy' : 'unhealthy') + '"></span>' +
                                'Storage' +
                            '</span>' +
                            '<span class="metric-value">' + (data.storage.connected ? 'Connected' : 'Disconnected') + '</span>' +
                        '</div>' +
                    '</div>' +
                '</div>' +

                '<div class="section">' +
                    '<div class="section-header">Database Statistics</div>' +
                    '<div class="section-content">' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">Driver</span>' +
                            '<span class="metric-value">' + data.database.driver + '</span>' +
                        '</div>' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">Active Connections</span>' +
                            '<span class="metric-value">' + data.database.active_connections + '/' + data.database.max_connections + '</span>' +
                        '</div>' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">Query Count</span>' +
                            '<span class="metric-value">' + data.database.query_count + '</span>' +
                        '</div>' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">Average Query Time</span>' +
                            '<span class="metric-value">' + data.database.avg_query_time + 'ms</span>' +
                        '</div>' +
                    '</div>' +
                '</div>' +

                '<div class="section">' +
                    '<div class="section-header">Cache Statistics</div>' +
                    '<div class="section-content">' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">Driver</span>' +
                            '<span class="metric-value">' + data.cache.driver + '</span>' +
                        '</div>' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">Hit Rate</span>' +
                            '<span class="metric-value">' + data.cache.hit_rate.toFixed(2) + '%</span>' +
                        '</div>' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">Memory Usage</span>' +
                            '<span class="metric-value">' + (data.cache.memory_usage / 1024 / 1024).toFixed(2) + ' MB</span>' +
                        '</div>' +
                    '</div>' +
                '</div>' +

                '<div class="section">' +
                    '<div class="section-header">Application Statistics</div>' +
                    '<div class="section-content">' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">Total Requests</span>' +
                            '<span class="metric-value">' + data.app.requests + '</span>' +
                        '</div>' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">Errors</span>' +
                            '<span class="metric-value">' + data.app.errors + '</span>' +
                        '</div>' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">Memory Usage</span>' +
                            '<span class="metric-value">' + (data.app.memory_usage / 1024 / 1024).toFixed(2) + ' MB</span>' +
                        '</div>' +
                        '<div class="metric-row">' +
                            '<span class="metric-label">CPU Usage</span>' +
                            '<span class="metric-value">' + data.app.cpu_usage.toFixed(2) + '%</span>' +
                        '</div>' +
                    '</div>' +
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
