package cursorpool

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
	cleanupStop chan struct{}
}

// NewCursorPool creates and initializes a new CursorPool
func NewCursorPool(connStr string, maxConns int, idleTimeout, absTimeout time.Duration) (*CursorPool, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(maxConns)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pool := &CursorPool{
		db:          db,
		cursors:     make(map[string]*CursorState),
		idleTimeout: idleTimeout,
		absTimeout:  absTimeout,
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
			log.Printf("Cleaning up expired cursor: %s", state.CursorName)
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
func (p *CursorPool) InitializeCursor(ctx context.Context, sid, query string) (*CursorState, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if state, exists := p.cursors[sid]; exists {
		state.Lock()
		defer state.Unlock()
		state.LastUsed = time.Now()
		return state, nil
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

	if _, err := tx.ExecContext(ctx, declareSQL); err != nil {
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

// FetchNext fetches rows from an active cursor
func (p *CursorPool) FetchNext(ctx context.Context, sid string, count int) ([]map[string]interface{}, error) {
	p.mu.Lock()
	state, ok := p.cursors[sid]
	p.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no active cursor for session %s", sid)
	}

	state.Lock()
	defer state.Unlock()
	state.LastUsed = time.Now()

	fetchSQL := fmt.Sprintf("FETCH FORWARD %d FROM %s", count, state.CursorName)
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
