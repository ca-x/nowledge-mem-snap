package persist

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"entgo.io/ent/dialect"

	"github.com/lib-x/nowledge-mem-snap/internal/persist/ent"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib-x/entsqlite"
	_ "github.com/lib/pq"
)

const (
	DBFilename    = "data.db"
	sqlitePragmas = "cache=shared&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(10000)"
)

func DBPath(dir string) string {
	return filepath.Join(dir, DBFilename)
}

func OpenClient(dir string) (*ent.Client, error) {
	databaseType := normalizeDatabaseType(os.Getenv("NMEM_SNAP_DATABASE_TYPE"))
	driverName := entDriverName(databaseType)

	dsn := strings.TrimSpace(os.Getenv("NMEM_SNAP_DATABASE_DSN"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("NMEM_SNAP_DATABASE_URL"))
	}
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		if databaseType != "sqlite" {
			return nil, fmt.Errorf("NMEM_SNAP_DATABASE_DSN is required for NMEM_SNAP_DATABASE_TYPE=%s", databaseType)
		}
		dbPath, err := filepath.Abs(DBPath(dir))
		if err != nil {
			return nil, fmt.Errorf("resolve database path: %w", err)
		}
		if err := ensureDatabaseFile(dbPath); err != nil {
			return nil, fmt.Errorf("create database file: %w", err)
		}
		dsn = sqliteDSN(dbPath)
	}

	client, err := ent.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", driverName, err)
	}
	if err := client.Schema.Create(context.Background()); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("apply %s schema: %w", driverName, err)
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

func normalizeDatabaseType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "sqlite", "sqlite3":
		return "sqlite"
	case "postgres", "postgresql", "pg":
		return "postgres"
	case "mysql", "mariadb":
		return "mysql"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func entDriverName(databaseType string) string {
	switch normalizeDatabaseType(databaseType) {
	case "sqlite":
		return dialect.SQLite
	case "postgres":
		return dialect.Postgres
	case "mysql":
		return dialect.MySQL
	default:
		return normalizeDatabaseType(databaseType)
	}
}
