// Package internal: the shared task manager — used by both the REPL (main.go)
// and the HTTP handler (api/tasks). One implementation, no copies.
// In-memory only; nothing is persisted.
package internal

import "sync"

type Task struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"` // "pending" | "done"
}

const (
	StatusPending = "pending"
	StatusDone    = "done"
)

func validStatus(s string) bool { return s == StatusPending || s == StatusDone }

type TaskManager struct {
	tasks  []Task
	nextID int
	mux    sync.Mutex
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: []Task{
			{ID: 1, Title: "Initialize neural network", Status: StatusDone},
			{ID: 2, Title: "Deploy microservices", Status: StatusPending},
		},
		nextID: 3,
	}
}

func (tm *TaskManager) Add(title string) Task {
	tm.mux.Lock()
	defer tm.mux.Unlock()
	t := Task{ID: tm.nextID, Title: title, Status: StatusPending}
	tm.nextID++
	tm.tasks = append(tm.tasks, t)
	return t
}

func (tm *TaskManager) List() []Task {
	tm.mux.Lock()
	defer tm.mux.Unlock()
	out := make([]Task, len(tm.tasks))
	copy(out, tm.tasks)
	return out
}

// Toggle flips pending<->done. Returns the task and whether it was found.
func (tm *TaskManager) Toggle(id int) (Task, bool) {
	tm.mux.Lock()
	defer tm.mux.Unlock()
	for i := range tm.tasks {
		if tm.tasks[i].ID == id {
			if tm.tasks[i].Status == StatusPending {
				tm.tasks[i].Status = StatusDone
			} else {
				tm.tasks[i].Status = StatusPending
			}
			return tm.tasks[i], true
		}
	}
	return Task{}, false
}

// Delete removes a task. Returns false when the ID does not exist.
func (tm *TaskManager) Delete(id int) bool {
	tm.mux.Lock()
	defer tm.mux.Unlock()
	for i := range tm.tasks {
		if tm.tasks[i].ID == id {
			tm.tasks = append(tm.tasks[:i], tm.tasks[i+1:]...)
			return true
		}
	}
	return false
}

func (tm *TaskManager) Get(id int) (Task, bool) {
	tm.mux.Lock()
	defer tm.mux.Unlock()
	for _, t := range tm.tasks {
		if t.ID == id {
			return t, true
		}
	}
	return Task{}, false
}
