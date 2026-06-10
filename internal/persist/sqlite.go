package persist

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/lib-x/nowledge-mem-snap/internal/persist/ent"

	_ "github.com/lib-x/entsqlite"
)

const (
	DBFilename    = "data.db"
	sqlitePragmas = "cache=shared&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(10000)"
)

func DBPath(dir string) string {
	return filepath.Join(dir, DBFilename)
}

func OpenClient(dir string) (*ent.Client, error) {
	dsn := strings.TrimSpace(os.Getenv("NMEM_SNAP_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		dbPath, err := filepath.Abs(DBPath(dir))
		if err != nil {
			return nil, fmt.Errorf("resolve database path: %w", err)
		}
		if err := ensureDatabaseFile(dbPath); err != nil {
			return nil, fmt.Errorf("create database file: %w", err)
		}
		dsn = sqliteDSN(dbPath)
	}

	client, err := ent.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	if err := client.Schema.Create(context.Background()); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("apply sqlite schema: %w", err)
	}
	return client, nil
}

func ensureDatabaseFile(dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}

func sqliteDSN(dbPath string) string {
	return (&url.URL{
		Scheme:   "file",
		Path:     filepath.ToSlash(dbPath),
		RawQuery: sqlitePragmas,
	}).String()
}
