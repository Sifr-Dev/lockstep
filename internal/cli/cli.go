package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	taskmanager "github.com/mohamedsalahi/task_manager_cli/internal/task"
)

var ErrExit = errors.New("exit")

const usage = `Commands:
  add <title>  create a pending task
  list          list tasks and status
  done <id>     mark a task as done
  delete <id>   remove a task
  help          show this help
  quit          leave the CLI`

// CLI executes task commands against the supplied shared manager.
type CLI struct {
	manager *taskmanager.Manager
	input   io.Reader
	output  io.Writer
}

// New creates a command-line interface backed by manager.
func New(manager *taskmanager.Manager, input io.Reader, output io.Writer) *CLI {
	return &CLI{
		manager: manager,
		input:   input,
		output:  output,
	}
}

// Run reads commands until EOF or the quit command.
func (cli *CLI) Run() error {
	return cli.RunContext(context.Background())
}

// RunContext reads commands until EOF, the quit command, or context cancellation.
func (cli *CLI) RunContext(ctx context.Context) error {
	fmt.Fprintln(cli.output, "SIFR task manager // in-memory session")
	fmt.Fprintln(cli.output, usage)

	scanner := bufio.NewScanner(cli.input)
	scanner.Buffer(make([]byte, 1024), 64<<10)
	lines := make(chan string)
	scanResult := make(chan error, 1)

	go func() {
		defer close(lines)
		for scanner.Scan() {
			select {
			case lines <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
		scanResult <- scanner.Err()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line, open := <-lines:
			if !open {
				return <-scanResult
			}
			if err := cli.Execute(line); err != nil {
				if errors.Is(err, ErrExit) {
					return nil
				}
				fmt.Fprintf(cli.output, "error: %v\n", err)
			}
		}
	}
}

// Execute runs one command line.
func (cli *CLI) Execute(line string) error {
	command, arguments := splitCommand(line)
	switch command {
	case "":
		return nil
	case "add":
		return cli.add(arguments)
	case "list":
		return cli.list()
	case "done":
		return cli.done(arguments)
	case "delete":
		return cli.delete(arguments)
	case "help":
		fmt.Fprintln(cli.output, usage)
		return nil
	case "quit", "exit":
		return ErrExit
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func (cli *CLI) add(title string) error {
	created, err := cli.manager.Add(title)
	if err != nil {
		return err
	}
	fmt.Fprintf(cli.output, "added task %d\n", created.ID)
	return nil
}

func (cli *CLI) list() error {
	tasks := cli.manager.List()
	if len(tasks) == 0 {
		fmt.Fprintln(cli.output, "no tasks")
		return nil
	}

	for _, task := range tasks {
		fmt.Fprintf(cli.output, "%d\t%s\t%s\n", task.ID, task.Status, task.Title)
	}
	return nil
}

func (cli *CLI) done(rawID string) error {
	id, err := parseID(rawID)
	if err != nil {
		return err
	}

	completed, err := cli.manager.Complete(id)
	if err != nil {
		return err
	}
	fmt.Fprintf(cli.output, "task %d marked done\n", completed.ID)
	return nil
}

func (cli *CLI) delete(rawID string) error {
	id, err := parseID(rawID)
	if err != nil {
		return err
	}
	if err := cli.manager.Delete(id); err != nil {
		return err
	}
	fmt.Fprintf(cli.output, "task %d deleted\n", id)
	return nil
}

func splitCommand(line string) (string, string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", ""
	}

	separator := strings.IndexFunc(trimmed, func(character rune) bool {
		return character == ' ' || character == '\t' || character == '\r' || character == '\n'
	})
	if separator == -1 {
		return strings.ToLower(trimmed), ""
	}

	return strings.ToLower(trimmed[:separator]), strings.TrimSpace(trimmed[separator+1:])
}

func parseID(rawID string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(rawID))
	if err != nil || id <= 0 {
		return 0, errors.New("task ID must be a positive integer")
	}
	return id, nil
}
