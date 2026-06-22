# High-Performance Task Manager CLI

A concurrent task management system built in Go with HTTP API support.

## Tech Stack
- Go
- Concurrency (goroutines, mutexes)

## Features
- Add, list, toggle tasks via CLI or HTTP API
- Thread-safe operations with mutex locking
- JSON API for frontend integration

## Run
```bash
go run main.go
```

## HTTP API
- GET /api/tasks - List all tasks
- POST /api/tasks - Create task (body: {"title": "..."})
- PATCH /api/tasks?id=N - Toggle task status
