package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Println("=== Task Manager CLI ===")
	fmt.Println("Commands: add <title>, list, done <id>, delete <id>, exit")
	fmt.Println()
	
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		if input == "exit" {
			fmt.Println("Goodbye!")
			break
		}
		
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}
		
		switch parts[0] {
		case "list":
			fmt.Println("Tasks: (use HTTP API at /api/tasks)")
		default:
			fmt.Printf("Unknown command: %s\n", parts[0])
		}
	}
}
