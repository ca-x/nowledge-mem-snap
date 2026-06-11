package storage

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"golang.org/x/net/webdav"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

func TestReadReadsRemoteObject(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := Target{Key: "local", Name: "Local", Fs: fs}
	if err := afero.WriteFile(fs, "backups/task-a/export.zip", []byte("archive"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := Read(context.Background(), target, "backups/task-a/export.zip")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != "archive" {
		t.Fatalf("Read data = %q, want archive", got)
	}
}

func TestReadRejectsUnsafePath(t *testing.T) {
	target := Target{Key: "local", Name: "Local", Fs: afero.NewMemMapFs()}

	_, err := Read(context.Background(), target, "../secret.zip")
	if err == nil {
		t.Fatal("Read succeeded with unsafe path")
	}
}

func TestListBackupObjectsRequiresNonRootPrefix(t *testing.T) {
	target := Target{Key: "local", Name: "Local", Fs: afero.NewMemMapFs()}

	for _, prefix := range []string{"", ".", "/"} {
		t.Run(prefix, func(t *testing.T) {
			_, err := ListBackupObjects(context.Background(), target, prefix)
			if err == nil {
				t.Fatal("ListBackupObjects succeeded with root prefix")
			}
		})
	}
}

func TestListBackupObjectsReturnsNewestFirst(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := Target{Key: "local", Name: "Local", Fs: fs}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	files := []struct {
		name string
		age  time.Duration
		body string
	}{
		{"backups/task-a/old.zip", 4 * time.Hour, "old"},
		{"backups/task-a/encrypted.zip.aes.json", 2 * time.Hour, "encrypted"},
		{"backups/task-a/new.zip", time.Hour, "new"},
		{"backups/task-a/notes.txt", 30 * time.Minute, "notes"},
		{"backups/task-b/other.zip", 10 * time.Minute, "other"},
	}
	for _, file := range files {
		if err := afero.WriteFile(fs, file.name, []byte(file.body), 0644); err != nil {
			t.Fatalf("write %s: %v", file.name, err)
		}
		ts := now.Add(-file.age)
		if err := fs.Chtimes(file.name, ts, ts); err != nil {
			t.Fatalf("chtimes %s: %v", file.name, err)
		}
	}

	objects, err := ListBackupObjects(context.Background(), target, "backups/task-a")
	if err != nil {
		t.Fatalf("ListBackupObjects: %v", err)
	}
	if len(objects) != 3 {
		t.Fatalf("len(objects) = %d, want 3", len(objects))
	}
	wantNames := []string{
		"backups/task-a/new.zip",
		"backups/task-a/encrypted.zip.aes.json",
		"backups/task-a/old.zip",
	}
	for i, want := range wantNames {
		if objects[i].Name != want {
			t.Fatalf("objects[%d].Name = %q, want %q", i, objects[i].Name, want)
		}
	}
	if !objects[1].Encrypted {
		t.Fatal("encrypted backup was not marked encrypted")
	}
	if objects[0].SizeBytes != int64(len("new")) {
		t.Fatalf("size_bytes = %d, want %d", objects[0].SizeBytes, len("new"))
	}
}

func TestListBackupObjectsMissingPrefixIsEmpty(t *testing.T) {
	target := Target{Key: "local", Name: "Local", Fs: afero.NewMemMapFs()}

	objects, err := ListBackupObjects(context.Background(), target, "missing")
	if err != nil {
		t.Fatalf("ListBackupObjects: %v", err)
	}
	if len(objects) != 0 {
		t.Fatalf("len(objects) = %d, want 0", len(objects))
	}
}

func TestListBackupObjectsHonorsContext(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := Target{Key: "local", Name: "Local", Fs: fs}
	if err := afero.WriteFile(fs, "backups/task-a/export.zip", []byte("x"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ListBackupObjects(ctx, target, "backups/task-a")
	if !errors.Is(err, context.Canceled) && !errors.Is(err, os.ErrClosed) {
		t.Fatalf("ListBackupObjects error = %v, want context canceled", err)
	}
}

func TestBrowseBackupDirectoriesReturnsDirectoriesAndRootObjects(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := Target{Key: "local", Name: "Local", Fs: fs}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	files := map[string]time.Duration{
		"root.zip":                     time.Hour,
		"backups/task-a/export.zip":    2 * time.Hour,
		"backups/task-b/export.zip":    3 * time.Hour,
		"backups/task-b/readme.txt":    4 * time.Hour,
		"backups/task-c/export.tar.gz": 5 * time.Hour,
	}
	for name, age := range files {
		if err := afero.WriteFile(fs, name, []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		ts := now.Add(-age)
		if err := fs.Chtimes(name, ts, ts); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}

	browse, err := BrowseBackupDirectories(context.Background(), target, "")
	if err != nil {
		t.Fatalf("BrowseBackupDirectories: %v", err)
	}
	if len(browse.Objects) != 1 || browse.Objects[0].Name != "root.zip" {
		t.Fatalf("root objects = %#v, want root.zip", browse.Objects)
	}
	gotDirs := make(map[string]int)
	for _, dir := range browse.Directories {
		gotDirs[dir.Name] = dir.ObjectCount
	}
	wantDirs := map[string]int{
		"backups/task-a": 1,
		"backups/task-b": 1,
	}
	if len(gotDirs) != len(wantDirs) {
		t.Fatalf("directories = %#v, want %#v", gotDirs, wantDirs)
	}
	for name, count := range wantDirs {
		if gotDirs[name] != count {
			t.Fatalf("directory %s count = %d, want %d", name, gotDirs[name], count)
		}
	}
}

func TestBrowseBackupDirectoriesPrefixShowsDirectFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := Target{Key: "local", Name: "Local", Fs: fs}
	if err := afero.WriteFile(fs, "backups/export.zip.aes.json", []byte("x"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	browse, err := BrowseBackupDirectories(context.Background(), target, "backups")
	if err != nil {
		t.Fatalf("BrowseBackupDirectories: %v", err)
	}
	if len(browse.Directories) != 0 {
		t.Fatalf("directories = %#v, want none", browse.Directories)
	}
	if len(browse.Objects) != 1 || browse.Objects[0].Name != "backups/export.zip.aes.json" || !browse.Objects[0].Encrypted {
		t.Fatalf("objects = %#v, want encrypted direct backup", browse.Objects)
	}
}

func TestBrowseBackupDirectoriesWorksWithWebDAVTarget(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	fixtures := map[string]time.Duration{
		"root.zip":                           time.Hour,
		"backups/task-a/export.zip":          2 * time.Hour,
		"backups/task-b/export.zip.aes.json": 3 * time.Hour,
		"backups/task-b/notes.txt":           4 * time.Hour,
	}
	for name, age := range fixtures {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(full, []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		ts := now.Add(-age)
		if err := os.Chtimes(full, ts, ts); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}
	srv := httptest.NewServer(&webdav.Handler{
		FileSystem: webdav.Dir(root),
		LockSystem: webdav.NewMemLS(),
	})
	t.Cleanup(srv.Close)

	fs, err := newWebDAVFS(context.Background(), config.WebDAVConfig{URL: srv.URL})
	if err != nil {
		t.Fatalf("newWebDAVFS: %v", err)
	}
	target := Target{Key: "webdav", Name: "WebDAV", Fs: fs}

	browse, err := BrowseBackupDirectories(context.Background(), target, "")
	if err != nil {
		t.Fatalf("BrowseBackupDirectories: %v", err)
	}
	if len(browse.Objects) != 1 || browse.Objects[0].Name != "root.zip" {
		t.Fatalf("root objects = %#v, want root.zip", browse.Objects)
	}
	gotDirs := make(map[string]int)
	for _, dir := range browse.Directories {
		gotDirs[dir.Name] = dir.ObjectCount
	}
	for _, name := range []string{"backups/task-a", "backups/task-b"} {
		if gotDirs[name] != 1 {
			t.Fatalf("directory %s count = %d, want 1; all dirs = %#v", name, gotDirs[name], gotDirs)
		}
	}

	objects, err := ListBackupObjects(context.Background(), target, "backups/task-b")
	if err != nil {
		t.Fatalf("ListBackupObjects: %v", err)
	}
	if len(objects) != 1 || objects[0].Name != "backups/task-b/export.zip.aes.json" || !objects[0].Encrypted {
		t.Fatalf("objects = %#v, want encrypted backup in backups/task-b", objects)
	}
}

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
