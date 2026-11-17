package task

import (
	"errors"
	"strings"
	"sync"
	"unicode"
)

const MaxTitleRunes = 200

type Status string

const (
	StatusPending Status = "pending"
	StatusDone    Status = "done"
)

var (
	ErrInvalidTitle = errors.New("title must contain between 1 and 200 printable characters")
	ErrTaskNotFound = errors.New("task not found")
)

type Task struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status Status `json:"status"`
}

// Manager is the concurrency-safe source of truth for task state.
type Manager struct {
	mu     sync.RWMutex
	tasks  []Task
	nextID int
}

// NewManager creates an empty in-memory task manager.
func NewManager() *Manager {
	return &Manager{
		tasks:  make([]Task, 0),
		nextID: 1,
	}
}

// Add validates and appends a pending task.
func (manager *Manager) Add(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if !validTitle(title) {
		return Task{}, ErrInvalidTitle
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	created := Task{
		ID:     manager.nextID,
		Title:  title,
		Status: StatusPending,
	}
	manager.nextID++
	manager.tasks = append(manager.tasks, created)
	return created, nil
}

// List returns a copy of the current task snapshot.
func (manager *Manager) List() []Task {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	tasks := make([]Task, len(manager.tasks))
	copy(tasks, manager.tasks)
	return tasks
}

// Get returns one task by ID.
func (manager *Manager) Get(id int) (Task, error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	for _, task := range manager.tasks {
		if task.ID == id {
			return task, nil
		}
	}

	return Task{}, ErrTaskNotFound
}

// Toggle changes a task between pending and done and returns a value copy.
func (manager *Manager) Toggle(id int) (Task, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	for index := range manager.tasks {
		if manager.tasks[index].ID != id {
			continue
		}
		if manager.tasks[index].Status == StatusPending {
			manager.tasks[index].Status = StatusDone
		} else {
			manager.tasks[index].Status = StatusPending
		}
		return manager.tasks[index], nil
	}

	return Task{}, ErrTaskNotFound
}

// Complete marks a task as done without changing it if it is already done.
func (manager *Manager) Complete(id int) (Task, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	for index := range manager.tasks {
		if manager.tasks[index].ID != id {
			continue
		}
		manager.tasks[index].Status = StatusDone
		return manager.tasks[index], nil
	}

	return Task{}, ErrTaskNotFound
}

// Delete removes a task by ID.
func (manager *Manager) Delete(id int) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	for index := range manager.tasks {
		if manager.tasks[index].ID == id {
			manager.tasks = append(manager.tasks[:index], manager.tasks[index+1:]...)
			return nil
		}
	}

	return ErrTaskNotFound
}

func validTitle(title string) bool {
	if title == "" || len([]rune(title)) > MaxTitleRunes {
		return false
	}
	for _, character := range title {
		if !unicode.IsPrint(character) {
			return false
		}
	}
	return true
}
