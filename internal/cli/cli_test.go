package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handler "github.com/Sifr-Dev/lockstep/api/tasks"
	"github.com/Sifr-Dev/lockstep/internal/cli"
	taskmanager "github.com/Sifr-Dev/lockstep/internal/task"
)

func TestCLIImplementsAdvertisedCommands(t *testing.T) {
	manager := taskmanager.NewManager()
	var output bytes.Buffer
	terminal := cli.New(manager, strings.NewReader(""), &output)

	if err := terminal.Execute("add Write a focused test"); err != nil {
		t.Fatalf("add error = %v", err)
	}
	if err := terminal.Execute("list"); err != nil {
		t.Fatalf("list error = %v", err)
	}
	if err := terminal.Execute("done 1"); err != nil {
		t.Fatalf("done error = %v", err)
	}
	if err := terminal.Execute("delete 1"); err != nil {
		t.Fatalf("delete error = %v", err)
	}
	if err := terminal.Execute("list"); err != nil {
		t.Fatalf("list error = %v", err)
	}

	text := output.String()
	for _, expected := range []string{
		"added task 1",
		"1\tpending\tWrite a focused test",
		"task 1 marked done",
		"task 1 deleted",
		"no tasks",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("output %q does not contain %q", text, expected)
		}
	}
}

func TestCLIRejectsInvalidCommandsAndIDs(t *testing.T) {
	manager := taskmanager.NewManager()
	var output bytes.Buffer
	terminal := cli.New(manager, strings.NewReader(""), &output)

	for _, command := range []string{
		"add",
		"add    ",
		"done 0",
		"done -1",
		"done abc",
		"delete 0",
		"delete abc",
		"unknown",
	} {
		if err := terminal.Execute(command); err == nil {
			t.Fatalf("Execute(%q) error = nil, want an error", command)
		}
	}

	if len(manager.List()) != 0 {
		t.Fatalf("invalid commands changed manager state: %+v", manager.List())
	}
}

func TestCLIRunReportsCommandErrorsAndStopsOnQuit(t *testing.T) {
	manager := taskmanager.NewManager()
	var output bytes.Buffer
	terminal := cli.New(manager, strings.NewReader("unknown\nadd Still works\nquit\nadd Never runs\n"), &output)

	if err := terminal.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	text := output.String()
	if !strings.Contains(text, `error: unknown command "unknown"`) {
		t.Fatalf("output does not report unknown command: %q", text)
	}
	if !strings.Contains(text, "added task 1") {
		t.Fatalf("output does not show valid command after an error: %q", text)
	}
	if strings.Contains(text, "added task 2") {
		t.Fatalf("command after quit was executed: %q", text)
	}
}

func TestCLIRunContextStopsWhenCanceled(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	terminal := cli.New(taskmanager.NewManager(), reader, io.Discard)

	if err := terminal.RunContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("RunContext() error = %v, want context.Canceled", err)
	}
}

func TestCLIAndHTTPUseTheSameManagerState(t *testing.T) {
	manager := taskmanager.NewManager()
	var output bytes.Buffer
	terminal := cli.New(manager, strings.NewReader(""), &output)
	server := handler.NewHandler(manager)

	if err := terminal.Execute("add Created by CLI"); err != nil {
		t.Fatalf("CLI add error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", response.Code, http.StatusOK)
	}

	var list struct {
		Tasks []taskmanager.Task `json:"tasks"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(list.Tasks) != 1 || list.Tasks[0].Title != "Created by CLI" {
		t.Fatalf("HTTP list = %+v, want the CLI task", list.Tasks)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"title":"Created by HTTP"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}

	output.Reset()
	if err := terminal.Execute("list"); err != nil {
		t.Fatalf("CLI list error = %v", err)
	}
	text := output.String()
	if !strings.Contains(text, "Created by CLI") || !strings.Contains(text, "Created by HTTP") {
		t.Fatalf("CLI list does not contain tasks from both interfaces: %q", text)
	}
}
