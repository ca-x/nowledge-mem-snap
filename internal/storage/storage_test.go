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
