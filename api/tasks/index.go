// Package handler: JSON HTTP surface. Shares internal.TaskManager — no copies.
// GET list · POST create (201) · PATCH toggle ?id=N · DELETE ?id=N
// status codes: 200 list/toggle/delete, 201 create, 400 bad input, 404 missing, 405 wrong verb
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mohamedsalahi/task_manager_cli/internal"
)

func Tasks(tm *internal.TaskManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		idFromQuery := func() (int, bool) {
			s := r.URL.Query().Get("id")
			id, err := strconv.Atoi(s)
			return id, err == nil
		}

		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(tm.List())
		case http.MethodPost:
			var body struct{ Title string `json:"title"` }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "title required"})
				return
			}
			t := tm.Add(body.Title)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(t)
		case http.MethodPatch:
			id, ok := idFromQuery()
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "numeric id required (?id=N)"})
				return
			}
			t, found := tm.Toggle(id)
			if !found {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "no such task id"})
				return
			}
			_ = json.NewEncoder(w).Encode(t)
		case http.MethodDelete:
			id, ok := idFromQuery()
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "numeric id required (?id=N)"})
				return
			}
			if !tm.Delete(id) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "no such task id"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]bool{"deleted": true})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		}
	}
}
