package examples

import (
	"testing"

	"github.com/mohamedsalahi/task_manager_cli/internal"
)

func TestTaskManagerUsage(t *testing.T) {
	tm := internal.NewTaskManager()

	tasks := tm.List()
	if len(tasks) != 2 {
		t.Fatalf("expected 2 initial tasks, got %d", len(tasks))
	}

	task := tm.Add("Write documentation")
	if task.Title != "Write documentation" {
		t.Fatalf("expected title 'Write documentation', got %s", task.Title)
	}
	if task.Status != "pending" {
		t.Fatalf("expected status 'pending', got %s", task.Status)
	}

	updated, ok := tm.Toggle(task.ID)
	if !ok || updated.Status != "done" {
		t.Fatal("expected task to be toggled to 'done'")
	}
	if !tm.Delete(task.ID) {
		t.Fatal("expected task to be deleted")
	}
	if _, ok := tm.Get(task.ID); ok {
		t.Fatal("task still present after delete")
	}
}
