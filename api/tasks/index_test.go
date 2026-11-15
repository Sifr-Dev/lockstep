package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handler "github.com/mohamedsalahi/task_manager_cli/api/tasks"
	taskmanager "github.com/mohamedsalahi/task_manager_cli/internal/task"
)

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type taskEnvelope struct {
	Task taskmanager.Task `json:"task"`
}

type listEnvelope struct {
	Tasks []taskmanager.Task `json:"tasks"`
}

func TestHandlerCreateListsAndMutatesSharedManager(t *testing.T) {
	manager := taskmanager.NewManager()
	server := handler.NewHandler(manager)

	created := requestJSON(t, server, http.MethodPost, "/api/tasks", `{"title":"  Ship tested code  "}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d; body = %s", created.Code, http.StatusCreated, created.Body.String())
	}
	if created.Header().Get("Location") != "/api/tasks/1" {
		t.Fatalf("Location = %q, want %q", created.Header().Get("Location"), "/api/tasks/1")
	}
	if contentType := created.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", contentType)
	}

	var createBody taskEnvelope
	decodeJSON(t, created, &createBody)
	if createBody.Task.ID != 1 || createBody.Task.Title != "Ship tested code" || createBody.Task.Status != taskmanager.StatusPending {
		t.Fatalf("created task = %+v", createBody.Task)
	}
	if len(manager.List()) != 1 {
		t.Fatalf("manager task count = %d, want 1", len(manager.List()))
	}

	listed := requestJSON(t, server, http.MethodGet, "/api/tasks", "")
	if listed.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", listed.Code, http.StatusOK)
	}
	var listBody listEnvelope
	decodeJSON(t, listed, &listBody)
	if len(listBody.Tasks) != 1 || listBody.Tasks[0] != createBody.Task {
		t.Fatalf("listed tasks = %+v, want %+v", listBody.Tasks, createBody.Task)
	}

	toggled := requestJSON(t, server, http.MethodPatch, "/api/tasks/1", "")
	if toggled.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, want %d; body = %s", toggled.Code, http.StatusOK, toggled.Body.String())
	}
	var toggleBody taskEnvelope
	decodeJSON(t, toggled, &toggleBody)
	if toggleBody.Task.Status != taskmanager.StatusDone {
		t.Fatalf("toggled status = %q, want %q", toggleBody.Task.Status, taskmanager.StatusDone)
	}

	deleted := requestJSON(t, server, http.MethodDelete, "/api/tasks/1", "")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want %d", deleted.Code, http.StatusNoContent)
	}
	if deleted.Body.Len() != 0 {
		t.Fatalf("DELETE body = %q, want empty", deleted.Body.String())
	}
	if len(manager.List()) != 0 {
		t.Fatalf("manager task count after delete = %d, want 0", len(manager.List()))
	}
}

func TestHandlerCreateValidation(t *testing.T) {
	manager := taskmanager.NewManager()
	server := handler.NewHandler(manager)

	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantCode    string
	}{
		{
			name:        "missing media type",
			contentType: "",
			body:        `{"title":"Valid"}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    "unsupported_media_type",
		},
		{
			name:        "wrong media type",
			contentType: "text/plain",
			body:        `{"title":"Valid"}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    "unsupported_media_type",
		},
		{
			name:        "malformed JSON",
			contentType: "application/json",
			body:        `{"title":`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_json",
		},
		{
			name:        "unknown field",
			contentType: "application/json",
			body:        `{"title":"Valid","status":"done"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_json",
		},
		{
			name:        "multiple objects",
			contentType: "application/json",
			body:        `{"title":"Valid"}{"title":"Second"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_json",
		},
		{
			name:        "missing title",
			contentType: "application/json",
			body:        `{}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_title",
		},
		{
			name:        "whitespace title",
			contentType: "application/json",
			body:        `{"title":"   "}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_title",
		},
		{
			name:        "control character in title",
			contentType: "application/json",
			body:        `{"title":"line\nbreak"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_title",
		},
		{
			name:        "oversized title",
			contentType: "application/json",
			body:        `{"title":"` + strings.Repeat("x", taskmanager.MaxTitleRunes+1) + `"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_title",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := requestJSON(t, server, http.MethodPost, "/api/tasks", test.body, test.contentType)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			assertError(t, response, test.wantCode)
			if len(manager.List()) != 0 {
				t.Fatalf("invalid request changed manager state")
			}
		})
	}
}

func TestHandlerRejectsOversizedBodies(t *testing.T) {
	manager := taskmanager.NewManager()
	server := handler.NewHandler(manager)

	response := requestJSON(t, server, http.MethodPost, "/api/tasks", `{"title":"`+strings.Repeat("x", 70<<10)+`"}`)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}
	assertError(t, response, "request_too_large")
	if len(manager.List()) != 0 {
		t.Fatalf("oversized request changed manager state")
	}
}

func TestHandlerItemValidationAndMissingTasks(t *testing.T) {
	manager := taskmanager.NewManager()
	server := handler.NewHandler(manager)

	for _, path := range []string{"/api/tasks/nope", "/api/tasks/0", "/api/tasks/-1"} {
		response := requestJSON(t, server, http.MethodPatch, path, "")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("PATCH %s status = %d, want %d", path, response.Code, http.StatusBadRequest)
		}
		assertError(t, response, "invalid_id")
	}

	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		response := requestJSON(t, server, method, "/api/tasks/404", "")
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want %d", method, response.Code, http.StatusNotFound)
		}
		assertError(t, response, "task_not_found")
	}
}

func TestHandlerRejectsUnsupportedMethods(t *testing.T) {
	manager := taskmanager.NewManager()
	server := handler.NewHandler(manager)

	tests := []struct {
		method    string
		path      string
		wantAllow string
		wantCode  string
	}{
		{method: http.MethodPut, path: "/api/tasks", wantAllow: "GET, POST", wantCode: "method_not_allowed"},
		{method: http.MethodPost, path: "/api/tasks/1", wantAllow: "PATCH, DELETE", wantCode: "method_not_allowed"},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			response := requestJSON(t, server, test.method, test.path, "")
			if response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
			}
			if allow := response.Header().Get("Allow"); allow != test.wantAllow {
				t.Fatalf("Allow = %q, want %q", allow, test.wantAllow)
			}
			assertError(t, response, test.wantCode)
		})
	}
}

func requestJSON(t *testing.T, server http.Handler, method, path, body string, contentType ...string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if len(contentType) == 0 {
		contentType = []string{"application/json"}
	}
	request.Header.Set("Content-Type", contentType[0])

	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode JSON %q: %v", response.Body.String(), err)
	}
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, wantCode string) {
	t.Helper()

	var body errorEnvelope
	decodeJSON(t, response, &body)
	if body.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q; body = %s", body.Error.Code, wantCode, response.Body.String())
	}
	if body.Error.Message == "" {
		t.Fatalf("error message is empty; body = %s", response.Body.String())
	}
}
