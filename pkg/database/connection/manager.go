package connection

import (
	"fmt"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Manager manages multiple database connections
type Manager struct {
	connections map[string]*gorm.DB
	mutex       sync.RWMutex
}

// NewManager creates a new database connection manager
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*gorm.DB),
	}
}

// Connect establishes a database connection
func (m *Manager) Connect(name, driver, dsn string) (*gorm.DB, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if connection already exists
	if conn, exists := m.connections[name]; exists {
		return conn, nil
	}

	// Create new connection based on driver
	var dialector gorm.Dialector
	switch driver {
	case "postgres":
		dialector = postgres.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	case "sqlite3":
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	// Configure GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Connect to database
	db, err := gorm.Open(dialector, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database %s: %w", name, err)
	}

	// Store connection
	m.connections[name] = db

	return db, nil
}

// GetConnection retrieves an existing database connection
func (m *Manager) GetConnection(name string) (*gorm.DB, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	conn, exists := m.connections[name]
	if !exists {
		return nil, fmt.Errorf("database connection %s not found", name)
	}

	return conn, nil
}

// CloseConnection closes a database connection
func (m *Manager) CloseConnection(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	conn, exists := m.connections[name]
	if !exists {
		return fmt.Errorf("database connection %s not found", name)
	}

	sqlDB, err := conn.DB()
	if err != nil {
		return err
	}

	if err := sqlDB.Close(); err != nil {
		return err
	}

	delete(m.connections, name)
	return nil
}

// CloseAllConnections closes all database connections
func (m *Manager) CloseAllConnections() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var lastErr error
	for name, conn := range m.connections {
		if sqlDB, err := conn.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				lastErr = err
			}
		}
		delete(m.connections, name)
	}

	return lastErr
}

// ListConnections returns a list of all connection names
func (m *Manager) ListConnections() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	names := make([]string, 0, len(m.connections))
	for name := range m.connections {
		names = append(names, name)
	}

	return names
}

// Ping tests a database connection
func (m *Manager) Ping(name string) error {
	conn, err := m.GetConnection(name)
	if err != nil {
		return err
	}

	sqlDB, err := conn.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}

// PingAll tests all database connections
func (m *Manager) PingAll() map[string]error {
	results := make(map[string]error)

	for name := range m.connections {
		results[name] = m.Ping(name)
	}

	return results
}
