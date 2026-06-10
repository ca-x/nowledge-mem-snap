package archive

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestObjectNameExpandsTokens(t *testing.T) {
	at := time.Date(2026, 6, 10, 12, 30, 45, 0, time.UTC)
	got := ObjectName("backups/{task}/{date}/{timestamp}", "main", at, PlainExtension)
	want := "backups/main/2026-06-10/20260610T123045Z.zip"
	if got != want {
		t.Fatalf("ObjectName() = %q, want %q", got, want)
	}
}

func TestBuildEncryptionOptional(t *testing.T) {
	plain, err := Build([]byte("zip"), BuildOptions{
		Prefix:    "snap/{task}/{timestamp}",
		TaskKey:   "default",
		CreatedAt: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Build plain: %v", err)
	}
	if plain.Encrypted {
		t.Fatal("plain artifact marked encrypted")
	}
	if !strings.HasSuffix(plain.Name, ".zip") {
		t.Fatalf("plain name = %q", plain.Name)
	}

	encrypted, err := Build([]byte("zip"), BuildOptions{
		Prefix:             "snap/{task}/{timestamp}",
		TaskKey:            "default",
		CreatedAt:          time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
		Encrypt:            true,
		EncryptionPassword: "secret",
	})
	if err != nil {
		t.Fatalf("Build encrypted: %v", err)
	}
	if !encrypted.Encrypted {
		t.Fatal("encrypted artifact not marked encrypted")
	}
	if !strings.HasSuffix(encrypted.Name, ".zip.aes.json") {
		t.Fatalf("encrypted name = %q", encrypted.Name)
	}
	var env map[string]any
	if err := json.Unmarshal(encrypted.Data, &env); err != nil {
		t.Fatalf("encrypted artifact is not JSON: %v", err)
	}
}

func TestRetentionDirectoryUsesStableTaskPrefix(t *testing.T) {
	got := RetentionDirectory("nowledge-mem/{task}/{timestamp}", "default")
	want := "nowledge-mem/default"
	if got != want {
		t.Fatalf("RetentionDirectory() = %q, want %q", got, want)
	}

	root := RetentionDirectory("{timestamp}", "default")
	if root != "." {
		t.Fatalf("RetentionDirectory root template = %q, want .", root)
	}
}
