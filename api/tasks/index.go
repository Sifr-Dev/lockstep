package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Task struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

var (
	tasks = []Task{
		{ID: 1, Title: "Initialize neural network", Status: "done"},
		{ID: 2, Title: "Deploy microservices", Status: "pending"},
	}
	nextId  = 3
	taskMux sync.Mutex
)

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		taskMux.Lock()
		json.NewEncoder(w).Encode(tasks)
		taskMux.Unlock()
		return
	}

	if r.Method == "POST" {
		var reqBody struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil || reqBody.Title == "" {
			http.Error(w, `{"error":"Title required"}`, http.StatusBadRequest)
			return
		}

		taskMux.Lock()
		newTask := Task{ID: nextId, Title: reqBody.Title, Status: "pending"}
		nextId++
		tasks = append(tasks, newTask)
		taskMux.Unlock()

		json.NewEncoder(w).Encode(newTask)
		return
	}

	if r.Method == "PATCH" {
		// Extremely simple path parser for /api/task?id=X since Vercel passes query params
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
            // fallback: check path, e.g. /api/tasks/1
            parts := strings.Split(r.URL.Path, "/")
            if len(parts) > 0 {
                idStr = parts[len(parts)-1]
            }
        }

		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"Invalid ID"}`, http.StatusBadRequest)
			return
		}

		taskMux.Lock()
		defer taskMux.Unlock()
		for i, t := range tasks {
			if t.ID == id {
				if t.Status == "pending" {
					tasks[i].Status = "done"
				} else {
					tasks[i].Status = "pending"
				}
				json.NewEncoder(w).Encode(tasks[i])
				return
			}
		}

		http.Error(w, `{"error":"Not found"}`, http.StatusNotFound)
		return
	}
}
