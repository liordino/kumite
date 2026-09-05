package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure Go SQLite driver, no CGo — the only permitted driver
)

// DB wraps the SQLite handle. There is no global state: handlers and engine
// functions receive *DB explicitly.
type DB struct {
	*sql.DB
}

// Open opens (creating if needed) the SQLite database at path.
// A single connection is used: Kumite is a single-process tool and every
// transaction is short, so serializing access removes SQLITE_BUSY hazards
// entirely.
func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", path)
	return open(dsn)
}

// OpenInMemory opens an in-memory database for tests.
func OpenInMemory() (*DB, error) {
	return open("file::memory:?_pragma=busy_timeout(10000)")
}

func open(dsn string) (*DB, error) {
	handle, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}
	handle.SetMaxOpenConns(1)
	if err := handle.Ping(); err != nil {
		handle.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}
	return &DB{DB: handle}, nil
}

// NowUTC returns the canonical timestamp format used in all tables.
func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// NewID returns a random RFC 4122 v4 UUID. No external UUID dependency.
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is unrecoverable; panic is appropriate here.
		panic(fmt.Sprintf("db.NewID: crypto/rand failed: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
