package cursorpool

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Mode constants
const (
	ModeConnectless = "connectless"
	ModeRefCursor   = "refcursor"
)

// CursorState holds the session state for PostgreSQL cursors
type CursorState struct {
	SessionID  string
	CursorName string
	Conn       *sql.Conn
	Tx         *sql.Tx
	CreatedAt  time.Time
	LastUsed   time.Time
	Query      string
	sync.Mutex
}

// CursorPool manages database connections and cursor sessions
type CursorPool struct {
	db          *sql.DB
	cursors     map[string]*CursorState
	mu          sync.Mutex
	idleTimeout time.Duration
	absTimeout  time.Duration
	maxCursors  int
	cleanupStop chan struct{}
}

// NewCursorPool creates and initializes a new CursorPool with tuning parameters
func NewCursorPool(connStr string, maxConns int, idleTimeout, absTimeout time.Duration) (*CursorPool, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Dynamic tuning of the underlying pool
	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxConns / 2)
	db.SetConnMaxLifetime(absTimeout)
	db.SetConnMaxIdleTime(idleTimeout)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pool := &CursorPool{
		db:          db,
		cursors:     make(map[string]*CursorState),
		idleTimeout: idleTimeout,
		absTimeout:  absTimeout,
		maxCursors:  maxConns, // Ties cursors to connections
		cleanupStop: make(chan struct{}),
	}

	pool.startCleanupRoutine()
	return pool, nil
}

// Close shuts down the pool and cleanup routine
func (p *CursorPool) Close() error {
	close(p.cleanupStop)
	return p.db.Close()
}

func (p *CursorPool) startCleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				p.cleanupTimeouts()
			case <-p.cleanupStop:
				ticker.Stop()
				return
			}
		}
	}()
}

func (p *CursorPool) cleanupTimeouts() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for sid, state := range p.cursors {
		state.Lock()
		if now.Sub(state.CreatedAt) > p.absTimeout || now.Sub(state.LastUsed) > p.idleTimeout {
			slog.Info("Cleaning up expired cursor", "cursorname", state.CursorName)
			p.removeCursor(sid, state)
		}
		state.Unlock()
	}
}

func (p *CursorPool) removeCursor(sid string, state *CursorState) {
	if state.Tx != nil {
		state.Tx.Rollback()
	}
	if state.Conn != nil {
		state.Conn.Close()
	}
	delete(p.cursors, sid)
}

// InitializeCursor sets up a new cursor or returns an existing one
func (p *CursorPool) InitializeCursor(ctx context.Context, sid, query string, args ...interface{}) (*CursorState, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if state, exists := p.cursors[sid]; exists {
		state.Lock()
		defer state.Unlock()
		state.LastUsed = time.Now()
		return state, nil
	}

	// Explicit limit check
	if len(p.cursors) >= p.maxCursors && p.maxCursors > 0 {
		return nil, fmt.Errorf("cursor pool capacity reached (max %d)", p.maxCursors)
	}

	conn, err := p.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	cursorName := "cur_" + uuid.New().String()[:8]
	declareSQL := fmt.Sprintf("DECLARE %s SCROLL CURSOR FOR %s", cursorName, query)

	if _, err := tx.ExecContext(ctx, declareSQL, args...); err != nil {
		tx.Rollback()
		conn.Close()
		return nil, fmt.Errorf("failed to declare cursor: %w", err)
	}

	state := &CursorState{
		SessionID:  sid,
		CursorName: cursorName,
		Conn:       conn,
		Tx:         tx,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
		Query:      query,
	}
	p.cursors[sid] = state
	return state, nil
}

// BuildFetchQuery constructs the SQL command for cursor pagination based on direction and page size.
func (p *CursorPool) BuildFetchQuery(cursorName string, pageSize int, direction string) string {
	q := "\"" + cursorName + "\""
	switch direction {
	case "NEXT":
		return fmt.Sprintf("FETCH FORWARD %d FROM %s;", pageSize, q)
	case "PRIOR":
		return fmt.Sprintf("MOVE RELATIVE -%d FROM %s;\nFETCH FORWARD %d FROM %s;", 2*pageSize, q, pageSize, q)
	case "LAST":
		return fmt.Sprintf("MOVE LAST FROM %s;\nMOVE RELATIVE -%d FROM %s;\nFETCH FORWARD %d FROM %s;\nMOVE LAST FROM %s;", q, pageSize, q, pageSize, q, q)
	case "BACKWARD":
		return fmt.Sprintf("MOVE RELATIVE -%d FROM %s;", pageSize, q)
	case "FIRST":
		return fmt.Sprintf("MOVE ABSOLUTE 0 FROM %s;\nFETCH FORWARD %d FROM %s;", q, pageSize, q)
	default:
		return fmt.Sprintf("FETCH FORWARD %d FROM %s;", pageSize, q)
	}
}

// FetchPage fetches rows from an active cursor in a specific direction
func (p *CursorPool) FetchPage(ctx context.Context, sid, direction string, count int) ([]map[string]interface{}, error) {
	p.mu.Lock()
	state, ok := p.cursors[sid]
	p.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no active cursor for session %s", sid)
	}

	state.Lock()
	defer state.Unlock()
	state.LastUsed = time.Now()

	fetchSQL := p.BuildFetchQuery(state.CursorName, count, direction)
	rows, err := state.Tx.QueryContext(ctx, fetchSQL)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer rows.Close()

	return scanRows(rows)
}

// QueryDirect runs a standard non-cursor query (Connectless Mode)
func (p *CursorPool) QueryDirect(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("direct query failed: %w", err)
	}
	defer rows.Close()

	return scanRows(rows)
}

func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		pointers := make([]interface{}, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	return results, nil
}
