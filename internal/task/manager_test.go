package task

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestManagerAddsValidatedTasks(t *testing.T) {
	manager := NewManager()

	created, err := manager.Add("  Write documentation  ")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if created.ID != 1 {
		t.Fatalf("ID = %d, want 1", created.ID)
	}
	if created.Title != "Write documentation" {
		t.Fatalf("Title = %q, want %q", created.Title, "Write documentation")
	}
	if created.Status != StatusPending {
		t.Fatalf("Status = %q, want %q", created.Status, StatusPending)
	}
}

func TestManagerRejectsInvalidTitles(t *testing.T) {
	manager := NewManager()

	for _, title := range []string{"", "   ", "line\nbreak", "right\u202eoverride", strings.Repeat("x", MaxTitleRunes+1)} {
		if _, err := manager.Add(title); !errors.Is(err, ErrInvalidTitle) {
			t.Fatalf("Add(%q) error = %v, want ErrInvalidTitle", title, err)
		}
	}
}

func TestManagerListReturnsIndependentSnapshot(t *testing.T) {
	manager := NewManager()
	created, err := manager.Add("Original title")
	if err != nil {
		t.Fatal(err)
	}

	tasks := manager.List()
	tasks[0].Title = "Mutated outside manager"
	tasks[0].Status = StatusDone

	fresh := manager.List()
	if fresh[0].Title != created.Title || fresh[0].Status != StatusPending {
		t.Fatalf("List() exposed mutable manager state: %+v", fresh[0])
	}
}

func TestManagerCompletesTasksIdempotently(t *testing.T) {
	manager := NewManager()
	created, err := manager.Add("Complete me")
	if err != nil {
		t.Fatal(err)
	}

	completed, err := manager.Complete(created.ID)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if completed.Status != StatusDone {
		t.Fatalf("Status = %q, want %q", completed.Status, StatusDone)
	}

	completedAgain, err := manager.Complete(created.ID)
	if err != nil {
		t.Fatalf("second Complete() error = %v", err)
	}
	if completedAgain.Status != StatusDone {
		t.Fatalf("second Status = %q, want %q", completedAgain.Status, StatusDone)
	}
}

func TestManagerTogglesAndDeletesTasks(t *testing.T) {
	manager := NewManager()
	created, err := manager.Add("Toggle me")
	if err != nil {
		t.Fatal(err)
	}

	toggled, err := manager.Toggle(created.ID)
	if err != nil {
		t.Fatalf("Toggle() error = %v", err)
	}
	if toggled.Status != StatusDone {
		t.Fatalf("Status = %q, want %q", toggled.Status, StatusDone)
	}

	toggled.Status = StatusPending
	fresh := manager.List()
	if fresh[0].Status != StatusDone {
		t.Fatalf("Toggle() exposed mutable manager state: %+v", fresh[0])
	}

	if err := manager.Delete(created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(manager.List()) != 0 {
		t.Fatalf("List() after Delete() = %+v", manager.List())
	}
}

func TestManagerReportsMissingTasks(t *testing.T) {
	manager := NewManager()

	if _, err := manager.Toggle(404); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("Toggle() error = %v, want ErrTaskNotFound", err)
	}
	if _, err := manager.Complete(404); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("Complete() error = %v, want ErrTaskNotFound", err)
	}
	if err := manager.Delete(404); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("Delete() error = %v, want ErrTaskNotFound", err)
	}
}

func TestManagerSupportsConcurrentAdds(t *testing.T) {
	manager := NewManager()
	const count = 100

	var waitGroup sync.WaitGroup
	for index := 0; index < count; index++ {
		waitGroup.Add(1)
		go func(number int) {
			defer waitGroup.Done()
			if _, err := manager.Add("Concurrent task"); err != nil {
				t.Errorf("Add() error = %v", err)
			}
		}(index)
	}
	waitGroup.Wait()

	tasks := manager.List()
	if len(tasks) != count {
		t.Fatalf("task count = %d, want %d", len(tasks), count)
	}
	seen := make(map[int]struct{}, count)
	for _, task := range tasks {
		if _, duplicate := seen[task.ID]; duplicate {
			t.Fatalf("duplicate ID %d", task.ID)
		}
		seen[task.ID] = struct{}{}
	}
}
