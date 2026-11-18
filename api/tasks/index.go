package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	taskmanager "github.com/Sifr-Dev/task_manager_cli/internal/task"
)

const maxRequestBodyBytes = 64 << 10

type API struct {
	manager *taskmanager.Manager
}

type listResponse struct {
	Tasks []taskmanager.Task `json:"tasks"`
}

type taskResponse struct {
	Task taskmanager.Task `json:"task"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type createTaskRequest struct {
	Title string `json:"title"`
}

// NewHandler creates HTTP routes backed by the supplied manager.
func NewHandler(manager *taskmanager.Manager) http.Handler {
	api := &API{manager: manager}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", api.handleCollection)
	mux.HandleFunc("/api/tasks/{id}", api.handleItem)
	mux.HandleFunc("/", api.handleNotFound)
	return mux
}

var (
	defaultManager = taskmanager.NewManager()
	defaultHandler = NewHandler(defaultManager)
)

// Handler serves the package-level API used by function-style deployments.
// Long-running processes should call NewHandler with their own shared manager.
func Handler(w http.ResponseWriter, r *http.Request) {
	defaultHandler.ServeHTTP(w, r)
}

func (api *API) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.listTasks(w)
	case http.MethodPost:
		api.createTask(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (api *API) handleItem(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.getTask(w, r)
	case http.MethodPatch:
		api.toggleTask(w, r)
	case http.MethodDelete:
		api.deleteTask(w, r)
	default:
		w.Header().Set("Allow", "GET, PATCH, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (api *API) handleNotFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "route_not_found", "route not found")
}

func (api *API) listTasks(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, listResponse{Tasks: api.manager.List()})
}

func (api *API) createTask(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return
	}
	if r.ContentLength > maxRequestBodyBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request createTaskRequest
	if err := decoder.Decode(&request); err != nil {
		if isMaxBytesError(err) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be a JSON object with a title field")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if isMaxBytesError(err) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain exactly one JSON object")
		return
	}

	created, err := api.manager.Add(request.Title)
	if err != nil {
		if errors.Is(err, taskmanager.ErrInvalidTitle) {
			writeError(w, http.StatusBadRequest, "invalid_title", taskmanager.ErrInvalidTitle.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "task could not be created")
		return
	}

	w.Header().Set("Location", "/api/tasks/"+strconv.Itoa(created.ID))
	writeJSON(w, http.StatusCreated, taskResponse{Task: created})
}

func (api *API) getTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	found, err := api.manager.Get(id)
	if err != nil {
		if errors.Is(err, taskmanager.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "task_not_found", taskmanager.ErrTaskNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "task could not be read")
		return
	}

	writeJSON(w, http.StatusOK, taskResponse{Task: found})
}

func (api *API) toggleTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	toggled, err := api.manager.Toggle(id)
	if err != nil {
		if errors.Is(err, taskmanager.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "task_not_found", taskmanager.ErrTaskNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "task could not be updated")
		return
	}

	writeJSON(w, http.StatusOK, taskResponse{Task: toggled})
}

func (api *API) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	if err := api.manager.Delete(id); err != nil {
		if errors.Is(err, taskmanager.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "task_not_found", taskmanager.ErrTaskNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "task could not be deleted")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func isMaxBytesError(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func parseTaskID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(strings.TrimSpace(r.PathValue("id")))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_id", "task ID must be a positive integer")
		return 0, false
	}
	return id, true
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		encoded = []byte(`{"error":{"code":"internal_error","message":"response encoding failed"}}`)
		status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}
