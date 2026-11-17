<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/banner-dark.svg">
    <img alt="task_manager_cli — built from zero" src=".github/banner-light.svg" width="100%">
  </picture>
</p>

# `task_manager_cli`

A small, concurrency-safe Go task manager with an interactive CLI and a JSON HTTP API. Both interfaces use one `internal/task.Manager` instance when they run in the same process.

The implementation is deliberately narrow: tasks are in memory, task IDs are positive integers, titles are bounded printable text, and status is `pending` or `done`. No benchmark currently supports a “high-performance” claim.

## implemented surface

| Surface | Operation | Behavior |
|---|---|---|
| Manager | `Add` | validates and appends a pending task |
| Manager | `List` / `Get` | returns independent task snapshots |
| Manager | `Toggle` | changes `pending` to `done` or back |
| Manager | `Complete` | idempotently marks a task `done` |
| Manager | `Delete` | removes a task by ID |
| CLI | `add`, `list`, `done`, `delete` | operates on the injected manager |
| HTTP | collection and item routes | operates on the same manager when combined with the CLI |

The manager uses an `RWMutex`; callers receive task values or copied slices rather than pointers into mutable internal state.

## run locally

The interactive CLI only:

```bash
go run .
```

The CLI and HTTP API in one process, sharing the same manager:

```bash
go run . --http --addr 127.0.0.1:8080
```

The HTTP API only:

```bash
go run . serve --addr 127.0.0.1:8080
```

`--http` prints the selected address before starting the CLI. The default bind address is loopback-only.

## CLI contract

```text
add <title>  create a pending task
list         list task IDs, status, and titles
done <id>    idempotently mark a task as done
delete <id>  remove a task
help         show command help
quit         leave the CLI
```

`done` sets a task to `done` and leaves an already-completed task unchanged. The HTTP `PATCH` operation intentionally toggles the status instead.

Invalid commands, titles, and IDs produce an error and do not stop the CLI. A task ID must be a positive integer.

## HTTP contract

The canonical item route is `/api/tasks/{id}`. The old published query-string form `PATCH /api/tasks?id=N` is not part of this contract.

| Method | Path | Success | Meaning |
|---|---|---:|---|
| `GET` | `/api/tasks` | `200` | list tasks |
| `POST` | `/api/tasks` | `201` | create a task; sets `Location` |
| `GET` | `/api/tasks/{id}` | `200` | read one task |
| `PATCH` | `/api/tasks/{id}` | `200` | toggle one task |
| `DELETE` | `/api/tasks/{id}` | `204` | delete one task |

Create requests require `Content-Type: application/json`:

```bash
curl -i http://127.0.0.1:8080/api/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Write the integration test"}'
```

```json
{
  "task": {
    "id": 1,
    "title": "Write the integration test",
    "status": "pending"
  }
}
```

The `Location` header is `/api/tasks/1`, which is directly retrievable. List responses use `{"tasks":[...]}`. Errors use one stable envelope:

```json
{
  "error": {
    "code": "invalid_title",
    "message": "title must contain between 1 and 200 printable characters"
  }
}
```

| Status | Error code | Condition |
|---:|---|---|
| `400` | `invalid_json`, `invalid_title`, `invalid_id` | malformed body, invalid title, or non-positive/malformed ID |
| `404` | `task_not_found`, `route_not_found` | missing task or unmatched route |
| `405` | `method_not_allowed` | unsupported method; the response includes `Allow` |
| `413` | `request_too_large` | request body exceeds 64 KiB |
| `415` | `unsupported_media_type` | create request is not `application/json` |

## state, deployment, and security boundaries

- **In-memory only.** State is process-local and resets when the process restarts.
- **Sharing is process-scoped.** CLI and HTTP requests share state only when the interfaces use the same manager instance. `--http` provides that topology; separately deployed processes or serverless instances do not share memory.
- **No durable history.** There is no database, file store, write-ahead log, or recovery path.
- **No authentication or authorization.** The API is intended for a local or trusted single-user environment, not a multi-user public service.
- **Same-origin by default.** No wildcard or permissive CORS headers are emitted, and preflight requests are not granted.
- **Bounded input.** Titles are trimmed, limited to 200 printable Unicode characters, and request bodies are limited to 64 KiB.
- **Deployment is not claimed.** The exported `Handler` remains available for function-style adapters, but no hosted route mapping or durable deployment has been verified for this repository.

## source layout

```text
main.go                    CLI and optional HTTP runtime
internal/cli/              command parsing and terminal output
internal/task/             concurrency-safe manager and domain model
api/tasks/index.go         injectable HTTP handler and JSON contract
.github/workflows/ci.yml   formatting, vet, test, race, and build gate
```

## validation

```bash
gofmt -l .
go vet ./...
go test -shuffle=on -count=1 ./...
go test -race -count=1 ./...
go build ./...
```

The test suite covers manager validation and concurrency, CLI command/error behavior, HTTP status and schema contracts, body limits, missing routes, concurrent creates, production CLI/API wiring, and shared-state visibility.

## identity

The README header uses the approved SIFR identity V2 asset from `logo/png/readme-header-dark.png`, copied unchanged to `.github/assets/readme-header-dark.png`. Its SHA-256 is `68b65437c8d9f5164e6d88c40424ffaa06d967c342bf4023caf288a9f3c6547c`.

## license

No project license has been declared. Add one only after the owner selects the intended terms.
