package dbmonitor

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

type QueryRecord struct {
	Query      string  `json:"query"`
	Duration   string  `json:"duration"`
	DurationMs float64 `json:"duration_ms"`
	Rows       int64   `json:"rows"`
	Time       string  `json:"time"`
	Error      string  `json:"error,omitempty"`
}

type QueryStats struct {
	TotalQueries    int64   `json:"total_queries"`
	SlowQueries     int64   `json:"slow_queries"`
	AvgDurationMs   float64 `json:"avg_duration_ms"`
	MaxDurationMs   float64 `json:"max_duration_ms"`
	TotalDurationMs float64 `json:"total_duration_ms"`
}

type PoolStats struct {
	MaxOpenConns int `json:"max_open_conns"`
	OpenConns    int `json:"open_conns"`
	InUse        int `json:"in_use"`
	Idle         int `json:"idle"`
	WaitCount    int `json:"wait_count"`
}

type DebugResponse struct {
	Stats       QueryStats    `json:"stats"`
	Pool        PoolStats     `json:"pool"`
	SlowQueries []QueryRecord `json:"slow_queries"`
	Recent      []QueryRecord `json:"recent"`
}

type Monitor struct {
	mu            sync.RWMutex
	recent        []QueryRecord
	slowQueries   []QueryRecord
	maxRecent     int
	maxSlow       int
	totalQueries  int64
	totalDuration float64
	maxDuration   float64
	slowThreshold time.Duration
	sqlDB         *sql.DB
}

func NewMonitor(slowThreshold time.Duration) *Monitor {
	return &Monitor{
		recent:        make([]QueryRecord, 0, 100),
		slowQueries:   make([]QueryRecord, 0, 50),
		maxRecent:     100,
		maxSlow:       50,
		slowThreshold: slowThreshold,
	}
}

func (m *Monitor) Name() string {
	return "dbmonitor"
}

func (m *Monitor) Initialize(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err == nil {
		m.sqlDB = sqlDB
	}

	_ = db.Callback().Create().Before("gorm:create").Register("monitor:before_create", m.before)
	_ = db.Callback().Create().After("gorm:create").Register("monitor:after_create", m.after)
	_ = db.Callback().Query().Before("gorm:query").Register("monitor:before_query", m.before)
	_ = db.Callback().Query().After("gorm:query").Register("monitor:after_query", m.after)
	_ = db.Callback().Update().Before("gorm:update").Register("monitor:before_update", m.before)
	_ = db.Callback().Update().After("gorm:update").Register("monitor:after_update", m.after)
	_ = db.Callback().Delete().Before("gorm:delete").Register("monitor:before_delete", m.before)
	_ = db.Callback().Delete().After("gorm:delete").Register("monitor:after_delete", m.after)
	_ = db.Callback().Raw().Before("gorm:raw").Register("monitor:before_raw", m.before)
	_ = db.Callback().Raw().After("gorm:raw").Register("monitor:after_raw", m.after)

	return nil
}

func (m *Monitor) before(db *gorm.DB) {
	db.InstanceSet("monitor:start", time.Now())
}

func (m *Monitor) after(db *gorm.DB) {
	startVal, ok := db.InstanceGet("monitor:start")
	if !ok {
		return
	}
	start := startVal.(time.Time)
	duration := time.Since(start)
	durationMs := float64(duration.Microseconds()) / 1000.0

	queryStr := db.Statement.SQL.String()
	if queryStr == "" {
		return
	}

	errStr := ""
	if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
		errStr = db.Error.Error()
	}

	record := QueryRecord{
		Query:      queryStr,
		Duration:   fmt.Sprintf("%.2fms", durationMs),
		DurationMs: durationMs,
		Rows:       db.Statement.RowsAffected,
		Time:       time.Now().Format("15:04:05.000"),
		Error:      errStr,
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalQueries++
	m.totalDuration += durationMs
	if durationMs > m.maxDuration {
		m.maxDuration = durationMs
	}

	m.recent = append(m.recent, record)
	if len(m.recent) > m.maxRecent {
		m.recent = m.recent[len(m.recent)-m.maxRecent:]
	}

	if duration >= m.slowThreshold {
		m.slowQueries = append(m.slowQueries, record)
		if len(m.slowQueries) > m.maxSlow {
			m.slowQueries = m.slowQueries[len(m.slowQueries)-m.maxSlow:]
		}
	}
}

func (m *Monitor) GetDebugInfo() DebugResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := QueryStats{
		TotalQueries:    m.totalQueries,
		SlowQueries:     int64(len(m.slowQueries)),
		MaxDurationMs:   m.maxDuration,
		TotalDurationMs: m.totalDuration,
	}
	if m.totalQueries > 0 {
		stats.AvgDurationMs = m.totalDuration / float64(m.totalQueries)
	}

	pool := PoolStats{}
	if m.sqlDB != nil {
		dbStats := m.sqlDB.Stats()
		pool = PoolStats{
			MaxOpenConns: dbStats.MaxOpenConnections,
			OpenConns:    dbStats.OpenConnections,
			InUse:        dbStats.InUse,
			Idle:         dbStats.Idle,
			WaitCount:    int(dbStats.WaitCount),
		}
	}

	recentCopy := make([]QueryRecord, len(m.recent))
	copy(recentCopy, m.recent)
	slowCopy := make([]QueryRecord, len(m.slowQueries))
	copy(slowCopy, m.slowQueries)

	// reverse recent to show newest first
	for i, j := 0, len(recentCopy)-1; i < j; i, j = i+1, j-1 {
		recentCopy[i], recentCopy[j] = recentCopy[j], recentCopy[i]
	}

	return DebugResponse{
		Stats:       stats,
		Pool:        pool,
		SlowQueries: slowCopy,
		Recent:      recentCopy,
	}
}
