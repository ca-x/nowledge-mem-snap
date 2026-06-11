package archive

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestObjectNameExpandsTokens(t *testing.T) {
	at := time.Date(2026, 6, 10, 12, 30, 45, 0, time.UTC)
	got := ObjectName("backups/{task}/{task_id}/{date}/{timestamp}", "018ff3c8-a1ec-74f8-9381-fc7d6fb17f51", "Main Mem", at, PlainExtension)
	want := "backups/Main-Mem/018ff3c8-a1ec-74f8-9381-fc7d6fb17f51/2026-06-10/20260610T123045Z.zip"
	if got != want {
		t.Fatalf("ObjectName() = %q, want %q", got, want)
	}
}

func TestBuildEncryptionOptional(t *testing.T) {
	plain, err := Build([]byte("zip"), BuildOptions{
		Prefix:    "snap/{task}/{timestamp}",
		TaskKey:   "default",
		TaskName:  "Default backup",
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
		TaskName:           "Default backup",
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

func TestDecryptRoundTrip(t *testing.T) {
	plain := []byte("zip data")
	encrypted, err := Encrypt(plain, "secret", Metadata{TaskKey: "task", TaskName: "Task"})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, metadata, err := Decrypt(encrypted, "secret")
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(got) != string(plain) {
		t.Fatalf("Decrypt data = %q, want %q", got, plain)
	}
	if metadata.TaskKey != "task" {
		t.Fatalf("metadata task = %q", metadata.TaskKey)
	}
}

func TestDecryptRejectsWrongPassword(t *testing.T) {
	encrypted, err := Encrypt([]byte("zip data"), "secret", Metadata{TaskKey: "task"})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if _, _, err := Decrypt(encrypted, "wrong"); err == nil {
		t.Fatal("Decrypt with wrong password succeeded")
	}
}

func TestRetentionDirectoryUsesStableTaskPrefix(t *testing.T) {
	got := RetentionDirectory("nowledge-mem/{task}/{timestamp}", "018ff3c8-a1ec-74f8-9381-fc7d6fb17f51", "Default backup")
	want := "nowledge-mem/Default-backup"
	if got != want {
		t.Fatalf("RetentionDirectory() = %q, want %q", got, want)
	}

	root := RetentionDirectory("{timestamp}", "default", "Default backup")
	if root != "." {
		t.Fatalf("RetentionDirectory root template = %q, want .", root)
	}
}
