package history

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/persist/ent"
	"github.com/ca-x/nowledge-mem-snap/internal/persist/ent/runrecord"
)

type Run struct {
	ID         string         `json:"id"`
	TaskKey    string         `json:"task_key"`
	TaskName   string         `json:"task_name"`
	SourceKey  string         `json:"source_key"`
	Status     string         `json:"status"`
	ObjectName string         `json:"object_name"`
	Encrypted  bool           `json:"encrypted"`
	SizeBytes  int64          `json:"size_bytes"`
	Error      string         `json:"error,omitempty"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Targets    []TargetResult `json:"targets"`
}

type TargetResult struct {
	TargetKey        string    `json:"target_key"`
	TargetName       string    `json:"target_name"`
	Status           string    `json:"status"`
	Bytes            int64     `json:"bytes"`
	RetentionDeleted int       `json:"retention_deleted,omitempty"`
	Error            string    `json:"error,omitempty"`
	FinishedAt       time.Time `json:"finished_at"`
}

type Store struct {
	client        *ent.Client
	tenant        string
	limit         int
	retentionDays int
	mu            sync.Mutex
}

func NewStore(client *ent.Client, tenant string, limit int) *Store {
	return NewStoreWithRetention(client, tenant, limit, 180)
}

func NewStoreWithRetention(client *ent.Client, tenant string, limit int, retentionDays int) *Store {
	if limit <= 0 {
		limit = 100
	}
	if retentionDays <= 0 {
		retentionDays = 180
	}
	return &Store{client: client, tenant: tenant, limit: limit, retentionDays: retentionDays}
}

func (s *Store) Tenant() string {
	if s == nil {
		return ""
	}
	return s.tenant
}

func (s *Store) List() ([]Run, error) {
	rows, err := s.client.RunRecord.Query().
		Where(runrecord.Tenant(s.tenant)).
		Where(runrecord.StartedAtGTE(time.Now().UTC().AddDate(0, 0, -s.retentionDays))).
		Order(ent.Desc(runrecord.FieldStartedAt)).
		Limit(s.limit).
		All(context.Background())
	if err != nil {
		return nil, err
	}
	runs := make([]Run, 0, len(rows))
	for _, row := range rows {
		var run Run
		if err := json.Unmarshal([]byte(row.Payload), &run); err != nil {
			return nil, err
		}
		if run.Targets == nil {
			run.Targets = []TargetResult{}
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (s *Store) Append(run Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := json.Marshal(run)
	if err != nil {
		return err
	}
	ctx := context.Background()
	row, err := s.client.RunRecord.Query().
		Where(runrecord.Tenant(s.tenant), runrecord.RunID(run.ID)).
		Only(ctx)
	if ent.IsNotFound(err) {
		err = s.client.RunRecord.Create().
			SetTenant(s.tenant).
			SetRunID(run.ID).
			SetPayload(string(payload)).
			SetStartedAt(run.StartedAt).
			Exec(ctx)
	} else if err == nil {
		err = row.Update().
			SetPayload(string(payload)).
			SetStartedAt(run.StartedAt).
			Exec(ctx)
	}
	if err != nil {
		return err
	}
	return s.pruneLocked(ctx)
}

func (s *Store) pruneLocked(ctx context.Context) error {
	rows, err := s.client.RunRecord.Query().
		Where(runrecord.Tenant(s.tenant)).
		All(ctx)
	if err != nil {
		return err
	}
	if len(rows) <= s.limit {
		return s.pruneByAgeLocked(ctx)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].StartedAt.After(rows[j].StartedAt)
	})
	for _, row := range rows[s.limit:] {
		if err := s.client.RunRecord.DeleteOne(row).Exec(ctx); err != nil {
			return err
		}
	}
	return s.pruneByAgeLocked(ctx)
}

func (s *Store) pruneByAgeLocked(ctx context.Context) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -s.retentionDays)
	_, err := s.client.RunRecord.Delete().
		Where(runrecord.Tenant(s.tenant), runrecord.StartedAtLT(cutoff)).
		Exec(ctx)
	return err
}
