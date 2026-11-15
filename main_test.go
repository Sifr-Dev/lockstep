package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestParseCLIOptions(t *testing.T) {
	parsed, err := parseCLIOptions(nil, io.Discard)
	if err != nil {
		t.Fatalf("parseCLIOptions() error = %v", err)
	}
	if parsed.withHTTP || parsed.address != defaultAddress {
		t.Fatalf("default options = %+v", parsed)
	}

	parsed, err = parseCLIOptions([]string{"--http", "--addr", "127.0.0.1:9090"}, io.Discard)
	if err != nil {
		t.Fatalf("parseCLIOptions() error = %v", err)
	}
	if !parsed.withHTTP || parsed.address != "127.0.0.1:9090" {
		t.Fatalf("HTTP options = %+v", parsed)
	}

	if _, err := parseCLIOptions([]string{"unexpected"}, io.Discard); err == nil {
		t.Fatal("parseCLIOptions() accepted an unexpected positional argument")
	}
}

func TestRunCLIWithoutHTTPServer(t *testing.T) {
	var output bytes.Buffer
	if err := run(context.Background(), nil, strings.NewReader("add CLI only\nlist\nquit\n"), &output, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	text := output.String()
	if !strings.Contains(text, "added task 1") || !strings.Contains(text, "CLI only") {
		t.Fatalf("run() output = %q", text)
	}
	if strings.Contains(text, "HTTP API listening") {
		t.Fatalf("run() started an unexpected HTTP server: %q", text)
	}
}

func TestRunSharesManagerBetweenCLIAndHTTP(t *testing.T) {
	inputReader, inputWriter := io.Pipe()
	output := &lockedBuffer{}
	result := make(chan error, 1)

	go func() {
		result <- run(
			context.Background(),
			[]string{"--http", "--addr", "127.0.0.1:0"},
			inputReader,
			output,
			io.Discard,
		)
	}()

	url := waitForServerURL(t, output)
	if _, err := io.WriteString(inputWriter, "add Created through CLI\n"); err != nil {
		t.Fatalf("write CLI command: %v", err)
	}
	waitForOutput(t, output, "added task 1")

	response, err := http.Get(url + "/api/tasks")
	if err != nil {
		t.Fatalf("GET shared tasks: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var list struct {
		Tasks []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&list); err != nil {
		t.Fatalf("decode task list: %v", err)
	}
	if len(list.Tasks) != 1 || list.Tasks[0].Title != "Created through CLI" {
		t.Fatalf("shared task list = %+v", list.Tasks)
	}

	if _, err := io.WriteString(inputWriter, "list\nquit\n"); err != nil {
		t.Fatalf("write CLI exit: %v", err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not stop after the quit command")
	}
	_ = inputWriter.Close()
}

func TestServeStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	output := &lockedBuffer{}
	result := make(chan error, 1)

	go func() {
		result <- run(ctx, []string{"serve", "--addr", "127.0.0.1:0"}, strings.NewReader(""), output, io.Discard)
	}()

	_ = waitForServerURL(t, output)
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("serve error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not stop after context cancellation")
	}
}

func TestHelpIsNotAnError(t *testing.T) {
	if _, err := parseCLIOptions([]string{"--help"}, io.Discard); !errorsIsFlagHelp(err) {
		t.Fatalf("parseCLIOptions(--help) error = %v, want flag.ErrHelp", err)
	}
}

func errorsIsFlagHelp(err error) bool {
	return err == flag.ErrHelp
}

func waitForServerURL(t *testing.T, output *lockedBuffer) string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, line := range strings.Split(output.String(), "\n") {
			if strings.HasPrefix(line, "HTTP API listening on ") {
				return strings.TrimPrefix(line, "HTTP API listening on ")
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not start; output = %q", output.String())
	return ""
}

func waitForOutput(t *testing.T, output *lockedBuffer, expected string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(output.String(), expected) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("output did not contain %q; output = %q", expected, output.String())
}

type lockedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (buffer *lockedBuffer) Write(data []byte) (int, error) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	return buffer.buffer.Write(data)
}

func (buffer *lockedBuffer) String() string {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	return buffer.buffer.String()
}
