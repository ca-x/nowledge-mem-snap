package persist

import (
	"strings"
	"testing"
)

func TestNormalizeDatabaseType(t *testing.T) {
	tests := map[string]string{
		"":           "sqlite",
		"sqlite":     "sqlite",
		"sqlite3":    "sqlite",
		"postgres":   "postgres",
		"postgresql": "postgres",
		"pg":         "postgres",
		"mysql":      "mysql",
		"mariadb":    "mysql",
	}
	for input, want := range tests {
		if got := normalizeDatabaseType(input); got != want {
			t.Fatalf("normalizeDatabaseType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSQLiteDSNIncludesRecommendedPragmas(t *testing.T) {
	dsn := sqliteDSN("/app/data/data.db")
	for _, want := range []string{
		"cache=shared",
		"_pragma=foreign_keys(1)",
		"_pragma=journal_mode(WAL)",
		"_pragma=synchronous(NORMAL)",
		"_pragma=busy_timeout(10000)",
	} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("sqliteDSN() missing %q in %q", want, dsn)
		}
	}
}
