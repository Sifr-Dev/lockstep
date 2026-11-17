<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/banner-dark.svg">
    <img alt="task_manager_cli — built from zero" src=".github/banner-light.svg" width="100%">
  </picture>
</p>

# task_manager_cli

a task manager with one shared core: a stdin REPL by default, and a JSON
HTTP API (`/api/tasks`) for the portfolio lab. in-memory only — nothing is
persisted.

## repl

```console
$ go run .
commands: add <title>, list, done <id>, delete <id>, exit
> add write release notes
added #3 "write release notes"
> done 3
#3 -> done
```

## http api

```sh
go run . -http :8080
curl localhost:8080/api/tasks                    # GET list
curl -X POST localhost:8080/api/tasks -d '{"title":"bench"}'        # 201
curl -X PATCH 'localhost:8080/api/tasks?id=3'                      # toggle pending<->done, 200
curl -X DELETE 'localhost:8080/api/tasks?id=3'                     # 200 {\"deleted\":true}
```

status codes: `201` create · `200` list/toggle/delete · `400` bad input ·
`404` unknown id · `405` wrong verb. CORS: origin `*`, methods incl. OPTIONS.

## design

- **one `internal.TaskManager`** shared by REPL and HTTP handler (the HTTP
  layer used to keep a second, divergent copy; both front-ends now parse
  identically)
- `internal`: `Add/List/Toggle/Delete/Get`, `sync.Mutex`, seeded examples
- old REPL was a stub (no-ops); it is now fully functional against the same core
- in-memory only: restart = fresh state (matches the old behavior; no file persistence claimed)

## verified (2026-11-17)

`go vet ./...` exit 0 · `go test -race ./...` PASS · REPL scripted
add/list/done OK · HTTP: list 200 · create 201 · toggle 200 · delete 200 ·
unknown id 404 · `go run . -http :8080` curl-verified

## license

`MIT` — see `LICENSE`. built from zero.
