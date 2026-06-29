package backup

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

func TestTaskWithSelectedTargetsUsesRequestedSubset(t *testing.T) {
	task := config.TaskConfig{
		Key:        "task",
		TargetKeys: []string{"target-a", "target-b", "target-c"},
	}

	got, err := taskWithSelectedTargets(task, []string{" target-b ", "target-a", "target-b"})
	if err != nil {
		t.Fatalf("taskWithSelectedTargets: %v", err)
	}
	want := []string{"target-b", "target-a"}
	if !reflect.DeepEqual(got.TargetKeys, want) {
		t.Fatalf("TargetKeys = %#v, want %#v", got.TargetKeys, want)
	}
	if !reflect.DeepEqual(task.TargetKeys, []string{"target-a", "target-b", "target-c"}) {
		t.Fatalf("source task was mutated: %#v", task.TargetKeys)
	}
}

func TestTaskWithSelectedTargetsRejectsEmptySelection(t *testing.T) {
	task := config.TaskConfig{
		Key:        "task",
		TargetKeys: []string{"target-a"},
	}

	_, err := taskWithSelectedTargets(task, []string{" "})
	if !errors.Is(err, ErrInvalidTargetSelection) {
		t.Fatalf("error = %v, want ErrInvalidTargetSelection", err)
	}
}

func TestTaskWithSelectedTargetsRejectsTargetsOutsideTask(t *testing.T) {
	task := config.TaskConfig{
		Key:        "task",
		TargetKeys: []string{"target-a"},
	}

	_, err := taskWithSelectedTargets(task, []string{"target-b"})
	if !errors.Is(err, ErrInvalidTargetSelection) {
		t.Fatalf("error = %v, want ErrInvalidTargetSelection", err)
	}
}
