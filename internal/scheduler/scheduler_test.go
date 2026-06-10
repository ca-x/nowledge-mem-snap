package scheduler

import (
	"testing"
	"time"

	"github.com/lib-x/nowledge-mem-snap/internal/config"
)

func TestNextRunOnceUsesCurrentLocation(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc)
	got, ok := nextRun(now, config.ScheduleConfig{
		Type:  "once",
		RunAt: "2026-06-10T13:30",
	})
	if !ok {
		t.Fatal("nextRun returned ok=false")
	}
	want := time.Date(2026, 6, 10, 13, 30, 0, 0, loc)
	if !got.Equal(want) || got.Location() != loc {
		t.Fatalf("nextRun = %v (%s), want %v (%s)", got, got.Location(), want, want.Location())
	}
}

func TestNextRunOncePastTimeRunsImmediately(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc)
	got, ok := nextRun(now, config.ScheduleConfig{
		Type:  "once",
		RunAt: "2026-06-10T11:30",
	})
	if !ok {
		t.Fatal("nextRun returned ok=false")
	}
	if !got.Equal(now) || got.Location() != loc {
		t.Fatalf("nextRun = %v (%s), want %v (%s)", got, got.Location(), now, now.Location())
	}
}

func TestNextRunWeeklyUsesCurrentLocation(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc)
	got, ok := nextRun(now, config.ScheduleConfig{
		Type:    "weekly",
		Time:    "03:00",
		Weekday: "thursday",
	})
	if !ok {
		t.Fatal("nextRun returned ok=false")
	}
	want := time.Date(2026, 6, 11, 3, 0, 0, 0, loc)
	if !got.Equal(want) || got.Location() != loc {
		t.Fatalf("nextRun = %v (%s), want %v (%s)", got, got.Location(), want, want.Location())
	}
}
