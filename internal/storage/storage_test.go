package storage

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/lib-x/nowledge-mem-snap/internal/config"
)

func TestApplyRetentionKeepLastIsScopedToTaskDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := Target{Key: "local", Name: "Local", Fs: fs}
	task := config.TaskConfig{
		Key:          "task-a",
		ObjectPrefix: "backups/{task}/{timestamp}",
		Retention: config.RetentionConfig{
			Mode:     "keep_last",
			KeepLast: 2,
		},
	}

	files := []struct {
		name string
		age  int
	}{
		{"backups/task-a/old.zip", 4},
		{"backups/task-a/middle.zip", 3},
		{"backups/task-a/new.zip", 2},
		{"backups/task-b/other.zip", 9},
		{"backups/task-a/readme.txt", 10},
	}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	for _, file := range files {
		if err := afero.WriteFile(fs, file.name, []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", file.name, err)
		}
		ts := now.AddDate(0, 0, -file.age)
		if err := fs.Chtimes(file.name, ts, ts); err != nil {
			t.Fatalf("chtimes %s: %v", file.name, err)
		}
	}

	deleted, err := ApplyRetention(context.Background(), target, task, now)
	if err != nil {
		t.Fatalf("ApplyRetention: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if _, err := fs.Stat("backups/task-a/old.zip"); err == nil {
		t.Fatal("old task-a backup still exists")
	}
	for _, name := range []string{
		"backups/task-a/middle.zip",
		"backups/task-a/new.zip",
		"backups/task-b/other.zip",
		"backups/task-a/readme.txt",
	} {
		if _, err := fs.Stat(name); err != nil {
			t.Fatalf("%s should remain: %v", name, err)
		}
	}
}

func TestApplyRetentionKeepAfterDateUsesLocalTimezone(t *testing.T) {
	oldLocal := time.Local
	loc := time.FixedZone("UTC+8", 8*60*60)
	time.Local = loc
	defer func() { time.Local = oldLocal }()

	fs := afero.NewMemMapFs()
	target := Target{Key: "local", Name: "Local", Fs: fs}
	task := config.TaskConfig{
		Key:          "task-a",
		ObjectPrefix: "backups/{task}/{timestamp}",
		Retention: config.RetentionConfig{
			Mode:      "keep_after",
			KeepAfter: "2026-06-10",
		},
	}
	files := map[string]time.Time{
		"backups/task-a/before.zip":   time.Date(2026, 6, 9, 15, 59, 0, 0, time.UTC),
		"backups/task-a/boundary.zip": time.Date(2026, 6, 9, 16, 0, 0, 0, time.UTC),
		"backups/task-a/after.zip":    time.Date(2026, 6, 10, 1, 0, 0, 0, time.UTC),
	}
	for name, ts := range files {
		if err := afero.WriteFile(fs, name, []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		if err := fs.Chtimes(name, ts, ts); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}

	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc)
	deleted, err := ApplyRetention(context.Background(), target, task, now)
	if err != nil {
		t.Fatalf("ApplyRetention: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if _, err := fs.Stat("backups/task-a/before.zip"); err == nil {
		t.Fatal("before.zip still exists")
	}
	for _, name := range []string{"backups/task-a/boundary.zip", "backups/task-a/after.zip"} {
		if _, err := fs.Stat(name); err != nil {
			t.Fatalf("%s should remain: %v", name, err)
		}
	}
}

func TestApplyRetentionKeepBeforeDateUsesLocalTimezone(t *testing.T) {
	oldLocal := time.Local
	loc := time.FixedZone("UTC+8", 8*60*60)
	time.Local = loc
	defer func() { time.Local = oldLocal }()

	fs := afero.NewMemMapFs()
	target := Target{Key: "local", Name: "Local", Fs: fs}
	task := config.TaskConfig{
		Key:          "task-a",
		ObjectPrefix: "backups/{task}/{timestamp}",
		Retention: config.RetentionConfig{
			Mode:       "keep_before",
			KeepBefore: "2026-06-10",
		},
	}
	files := map[string]time.Time{
		"backups/task-a/before.zip":   time.Date(2026, 6, 9, 15, 59, 0, 0, time.UTC),
		"backups/task-a/boundary.zip": time.Date(2026, 6, 9, 16, 0, 0, 0, time.UTC),
		"backups/task-a/after.zip":    time.Date(2026, 6, 10, 1, 0, 0, 0, time.UTC),
	}
	for name, ts := range files {
		if err := afero.WriteFile(fs, name, []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		if err := fs.Chtimes(name, ts, ts); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}

	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc)
	deleted, err := ApplyRetention(context.Background(), target, task, now)
	if err != nil {
		t.Fatalf("ApplyRetention: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}
	if _, err := fs.Stat("backups/task-a/before.zip"); err != nil {
		t.Fatalf("before.zip should remain: %v", err)
	}
	for _, name := range []string{"backups/task-a/boundary.zip", "backups/task-a/after.zip"} {
		if _, err := fs.Stat(name); err == nil {
			t.Fatalf("%s still exists", name)
		}
	}
}
