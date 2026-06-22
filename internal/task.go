package internal

import "sync"

type Task struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type TaskManager struct {
	tasks  []Task
	nextID int
	mux    sync.Mutex
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: []Task{
			{ID: 1, Title: "Initialize neural network", Status: "done"},
			{ID: 2, Title: "Deploy microservices", Status: "pending"},
		},
		nextID: 3,
	}
}

func (tm *TaskManager) Add(title string) Task {
	tm.mux.Lock()
	defer tm.mux.Unlock()
	task := Task{ID: tm.nextID, Title: title, Status: "pending"}
	tm.nextID++
	tm.tasks = append(tm.tasks, task)
	return task
}

func (tm *TaskManager) List() []Task {
	tm.mux.Lock()
	defer tm.mux.Unlock()
	result := make([]Task, len(tm.tasks))
	copy(result, tm.tasks)
	return result
}

func (tm *TaskManager) Toggle(id int) *Task {
	tm.mux.Lock()
	defer tm.mux.Unlock()
	for i, t := range tm.tasks {
		if t.ID == id {
			if t.Status == "pending" {
				tm.tasks[i].Status = "done"
			} else {
				tm.tasks[i].Status = "pending"
			}
			return &tm.tasks[i]
		}
	}
	return nil
}
