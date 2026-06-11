# Remote Restore Wizard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an authenticated Restore tab that restores a portable Nowledge Mem export from an existing S3/WebDAV target into an existing Nowledge Mem API source.

**Architecture:** Add a focused `internal/restore` package for object listing, archive download/decryption, asynchronous restore jobs, and Nowledge Mem import polling. Add server endpoints under `/api/restore/*` and a split frontend wizard under `web/src/restore/*`. Keep backup tasks, schedules, retention, and backup run history separate from restore.

**Tech Stack:** Go `net/http`, `slog`, `afero`, `github.com/lib-x/nowledgemem-go`, React 18, TypeScript, animal-island-ui, lucide-react.

---

## File Structure

- Modify `internal/archive/archive.go`: add `Decrypt` and `Inspect`-style helpers for encrypted envelopes.
- Modify `internal/archive/archive_test.go`: add decryption round-trip and wrong-password tests.
- Modify `internal/storage/storage.go`: add safe `Read` and `ListBackupObjects` helpers using existing target filesystems.
- Modify `internal/storage/storage_test.go`: cover object path validation and listing filters.
- Create `internal/restore/restore.go`: restore request/result types, manager, job state, object listing, job start/status, import polling.
- Create `internal/restore/restore_test.go`: cover validation, plain ZIP restore, encrypted restore, and status transitions with an `httptest` Nowledge Mem server.
- Modify `internal/server/server.go`: wire restore endpoints, merge stored secrets, enforce tenant-owned target/source selection.
- Create or modify `internal/server/restore_test.go`: exercise HTTP-level restore validation and happy path.
- Modify `web/src/types.ts`: add restore API types.
- Create `web/src/restore/types.ts`: wizard draft and step-local types.
- Create `web/src/restore/restoreDefaults.ts`: import flag definitions and default options.
- Create `web/src/restore/RestoreStepper.tsx`: visual step progress.
- Create `web/src/restore/RestoreSummary.tsx`: sticky/collapsible readiness summary.
- Create `web/src/restore/RestoreTargetStep.tsx`: saved target selection.
- Create `web/src/restore/RestoreObjectStep.tsx`: prefix scan/manual path/object selection/password prompt.
- Create `web/src/restore/RestoreDestinationStep.tsx`: Nowledge Mem API source selection.
- Create `web/src/restore/RestoreOptionsStep.tsx`: import mode and include flags.
- Create `web/src/restore/RestoreProgressStep.tsx`: job start/status polling UI.
- Create `web/src/restore/RestoreWizard.tsx`: owns wizard state and navigation.
- Create `web/src/pages/RestorePage.tsx`: filters config into restore-ready target/source lists.
- Modify `web/src/main.tsx`: register Restore tab only.
- Modify `web/src/i18n.tsx`: add English and Chinese restore strings.
- Modify `web/src/styles.css`: add restore wizard layout, stepper, object table, summary, and progress styles.
- Modify `README.md` and `README.zh-CN.md`: document restore support and constraints.

---

## Task 1: Archive Decryption

**Files:**
- Modify: `internal/archive/archive.go`
- Modify: `internal/archive/archive_test.go`

- [ ] **Step 1: Add failing tests**

Add tests:

```go
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
```

- [ ] **Step 2: Run failing tests**

Run: `go test ./internal/archive`

Expected: fail because `Decrypt` is undefined.

- [ ] **Step 3: Implement decryption**

Add `Decrypt(encrypted []byte, password string) ([]byte, Metadata, error)` to `archive.go`. It must:

- decode `encryptedEnvelope`
- validate `Format == envelopeFormat`
- require non-empty password
- base64-decode salt, nonce, ciphertext
- derive the scrypt key using envelope `N/R/P`
- open AES-GCM with `aad(env.Metadata)`
- return plaintext and metadata

- [ ] **Step 4: Verify archive tests**

Run: `go test ./internal/archive`

Expected: pass.

---

## Task 2: Storage Read And Restore Object Listing

**Files:**
- Modify: `internal/storage/storage.go`
- Modify: `internal/storage/storage_test.go`

- [ ] **Step 1: Add storage tests**

Add tests using `afero.NewMemMapFs()`:

```go
func TestReadRejectsUnsafeObjectPath(t *testing.T) {
	target := Target{Fs: afero.NewMemMapFs()}
	if _, err := Read(context.Background(), target, "../secret.zip"); err == nil {
		t.Fatal("Read accepted unsafe path")
	}
}

func TestListBackupObjectsFiltersByPrefixAndSuffix(t *testing.T) {
	fs := afero.NewMemMapFs()
	mustWriteFile(t, fs, "backups/a.zip", "zip")
	mustWriteFile(t, fs, "backups/b.zip.aes.json", "{}")
	mustWriteFile(t, fs, "backups/c.txt", "no")
	target := Target{Fs: fs}
	got, err := ListBackupObjects(context.Background(), target, "backups")
	if err != nil {
		t.Fatalf("ListBackupObjects: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("objects = %#v", got)
	}
}
```

- [ ] **Step 2: Run failing tests**

Run: `go test ./internal/storage`

Expected: fail because `Read` and exported `ListBackupObjects` are undefined.

- [ ] **Step 3: Implement helpers**

Add:

```go
type BackupObject struct {
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	ModTime   time.Time `json:"modified_at"`
	Encrypted bool      `json:"encrypted"`
}

func Read(ctx context.Context, target Target, objectName string) ([]byte, error)
func ListBackupObjects(ctx context.Context, target Target, prefix string) ([]BackupObject, error)
```

`ListBackupObjects` must reject empty prefix, walk only that prefix, filter `.zip` and `.zip.aes.json`, sort newest first, and return clean object names.

- [ ] **Step 4: Verify storage tests**

Run: `go test ./internal/storage`

Expected: pass.

---

## Task 3: Restore Manager Backend

**Files:**
- Create: `internal/restore/restore.go`
- Create: `internal/restore/restore_test.go`

- [ ] **Step 1: Add restore tests**

Cover:

- listing rejects empty prefix
- start rejects non-`.zip` object
- encrypted object requires password
- plain ZIP upload calls `/data/import/upload`
- status polling reaches completed

- [ ] **Step 2: Run failing tests**

Run: `go test ./internal/restore`

Expected: fail because package is new/incomplete.

- [ ] **Step 3: Implement manager**

Core exported API:

```go
type Manager struct {
	logger *slog.Logger
	factory *storage.Factory
}

func NewManager(logger *slog.Logger) *Manager
func (m *Manager) ListObjects(ctx context.Context, target config.TargetConfig, prefix string) ([]storage.BackupObject, error)
func (m *Manager) Start(ctx context.Context, req StartRequest) (*Job, error)
func (m *Manager) Get(id string) (*Job, bool)
```

`StartRequest` contains target config, destination source config, object name, encryption password, and import options.

Job state is protected by a mutex and records `queued/downloading/decrypting/uploading/importing/completed/failed`.

- [ ] **Step 4: Implement import upload and polling**

Use `mem.NewClient(mem.WithBaseURL(...), mem.WithAPIKey(...))`, call `client.Data.UploadImport`, then poll `client.Data.ImportStatus` until `completed`, `failed`, or context cancellation.

- [ ] **Step 5: Verify restore tests**

Run: `go test ./internal/restore`

Expected: pass.

---

## Task 4: Restore HTTP API

**Files:**
- Modify: `internal/server/server.go`
- Create: `internal/server/restore_test.go`

- [ ] **Step 1: Add HTTP tests**

Test:

- unauthenticated restore endpoints require auth through existing API middleware
- object listing uses tenant target and rejects missing target
- job start rejects destination source that is not `nowledgemem_api`

- [ ] **Step 2: Run failing server tests**

Run: `go test ./internal/server`

Expected: fail because endpoints are missing.

- [ ] **Step 3: Wire manager and routes**

Add `restoreManager *restore.Manager` to `Server`, initialize it in `New`, and register:

- `POST /api/restore/objects`
- `POST /api/restore/jobs`
- `GET /api/restore/jobs/{id}`

- [ ] **Step 4: Implement handlers**

Handlers load current user config, look up target/source by key, merge saved secrets, call restore manager, and return JSON.

- [ ] **Step 5: Verify server tests**

Run: `go test ./internal/server`

Expected: pass.

---

## Task 5: Restore Frontend Types And Defaults

**Files:**
- Modify: `web/src/types.ts`
- Create: `web/src/restore/types.ts`
- Create: `web/src/restore/restoreDefaults.ts`

- [ ] **Step 1: Add TypeScript contracts**

Define `RestoreObject`, `RestoreImportOptions`, `RestoreJob`, `RestoreDraft`, `RestoreStepKey`, and flag metadata.

- [ ] **Step 2: Add defaults**

Default import options set all `include_*` flags to `true` and `mode` to `""` until API mode enum is confirmed by runtime/server config.

- [ ] **Step 3: Verify TypeScript**

Run: `npm --prefix web run build`

Expected: pass or fail only due missing components from later tasks if references were added early.

---

## Task 6: Restore Wizard Components

**Files:**
- Create: `web/src/restore/RestoreStepper.tsx`
- Create: `web/src/restore/RestoreSummary.tsx`
- Create: `web/src/restore/RestoreTargetStep.tsx`
- Create: `web/src/restore/RestoreObjectStep.tsx`
- Create: `web/src/restore/RestoreDestinationStep.tsx`
- Create: `web/src/restore/RestoreOptionsStep.tsx`
- Create: `web/src/restore/RestoreProgressStep.tsx`
- Create: `web/src/restore/RestoreWizard.tsx`

- [ ] **Step 1: Implement presentation components**

Each component receives data and callbacks through props. No component reads global config directly.

- [ ] **Step 2: Implement wizard state**

`RestoreWizard` owns draft state, step validation, object scanning, job start, and job polling.

- [ ] **Step 3: Verify frontend build**

Run: `npm --prefix web run build`

Expected: pass after page registration in Task 7.

---

## Task 7: Restore Page, Tab Registration, I18n, Styles

**Files:**
- Create: `web/src/pages/RestorePage.tsx`
- Modify: `web/src/main.tsx`
- Modify: `web/src/i18n.tsx`
- Modify: `web/src/styles.css`

- [ ] **Step 1: Add RestorePage**

Filter config into enabled targets and enabled `nowledgemem_api` destination sources. Render `RestoreWizard`.

- [ ] **Step 2: Register tab**

Add Restore tab to `dashboardTabs` without moving existing tabs.

- [ ] **Step 3: Add i18n strings**

Add English and Chinese keys for step labels, field labels, validation messages, buttons, progress states, and terminal states.

- [ ] **Step 4: Add styles**

Add responsive styles for restore layout, stepper, object list, summary panel, import option grid, danger block, and progress timeline.

- [ ] **Step 5: Verify frontend build**

Run: `npm --prefix web run build`

Expected: pass.

---

## Task 8: Documentation And Full Verification

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`

- [ ] **Step 1: Document restore**

Document:

- Restore tab supports saved S3/WebDAV targets.
- Restore source objects must be `.zip` or `.zip.aes.json` portable Nowledge Mem exports.
- Destination is a saved Nowledge Mem API source.
- Directory snapshots are not application-level restore archives.
- Encrypted restore password is used only for the restore request.

- [ ] **Step 2: Run full verification**

Run:

```bash
npm --prefix web run build
go test ./...
go build -trimpath ./...
docker compose -f docker-compose.yml config
```

Expected: all commands exit 0.

- [ ] **Step 3: Commit implementation**

Commit message:

```bash
git add internal web README.md README.zh-CN.md
git commit -m "feat: add remote restore wizard"
```

---

## Self-Review

- Spec coverage: backend object listing/start/status, encrypted restore, frontend wizard, i18n, safety constraints, tests, and docs are covered.
- Placeholder scan: no TBD/TODO placeholders are used.
- Type consistency: restore object/job/import option names match between API and frontend tasks.

