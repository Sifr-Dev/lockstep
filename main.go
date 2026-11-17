// task_manager_cli — task manager with a shared core:
// stdin REPL by default, `-http :PORT` serves the JSON API used by the lab.
// In-memory only; nothing is persisted.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	handler "github.com/mohamedsalahi/task_manager_cli/api/tasks"
	"github.com/mohamedsalahi/task_manager_cli/internal"
)

func main() {
	port := flag.String("http", "", "serve the JSON API on this address (e.g. :8080)")
	flag.Parse()
	tm := internal.NewTaskManager()

	if *port != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/tasks", handler.Tasks(tm))
		fmt.Printf("task_manager_cli listening on %s (GET/POST/PATCH/DELETE /api/tasks)\n", *port)
		if err := http.ListenAndServe(*port, mux); err != nil {
			fmt.Fprintln(os.Stderr, "fatal:", err)
			os.Exit(1)
		}
		return
	}

	fmt.Println("=== Task Manager CLI ===")
	fmt.Println("commands: add <title>, list, done <id>, delete <id>, exit")
	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !sc.Scan() {
			return
		}
		parts := strings.Fields(strings.TrimSpace(sc.Text()))
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "add":
			if len(parts) < 2 {
				fmt.Println("usage: add <title>")
				continue
			}
			t := tm.Add(strings.Join(parts[1:], " "))
			fmt.Printf("added #%d %q\n", t.ID, t.Title)
		case "list":
			for _, t := range tm.List() {
				mark := " "
				if t.Status == internal.StatusDone {
					mark = "x"
				}
				fmt.Printf("[%s] #%d %s\n", mark, t.ID, t.Title)
			}
		case "done":
			if len(parts) != 2 {
				fmt.Println("usage: done <id>")
				continue
			}
			var id int
			if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
				fmt.Println("invalid id")
				continue
			}
			if t, ok := tm.Toggle(id); ok {
				fmt.Printf("#%d -> %s\n", t.ID, t.Status)
			} else {
				fmt.Printf("no task #%d\n", id)
			}
		case "delete":
			if len(parts) != 2 {
				fmt.Println("usage: delete <id>")
				continue
			}
			var id int
			if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
				fmt.Println("invalid id")
				continue
			}
			if tm.Delete(id) {
				fmt.Printf("deleted #%d\n", id)
			} else {
				fmt.Printf("no task #%d\n", id)
			}
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Printf("unknown command: %s\n", parts[0])
		}
	}
}
