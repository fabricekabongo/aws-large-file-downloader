# AGENTS.md

Guidance for agents working in this repository (`aws-large-file-downloader`).

## 1) Project map (quick orientation)

- Entry point: `main.go` creates root Cobra command and executes it.
- CLI layer: `cmd/`
  - `root.go` defines root command, logging level parsing, and subcommands.
  - `download.go` defines `download` command and wiring to concrete S3 + download service.
- Application orchestration: `internal/app/app.go` runs the Bubble Tea TUI and initializes tracing span.
- Domain logic: `internal/download/service.go`
  - Parses S3 URIs.
  - Performs safe local download via temp file then atomic rename.
- Infrastructure adapters:
  - `internal/s3client/client.go`: AWS SDK v2 wrapper (`GetObject` + `io.Copy`).
  - `internal/logging/`: slog logger setup.
  - `internal/telemetry/`: OpenTelemetry initialization.
- TUI: `internal/tui/model.go` contains Bubble Tea model/update/view.

## 2) Domain intent

The core domain is **reliable large-file transfer from S3 to local disk** with a simple CLI/TUI experience.

When changing code, optimize for:
- Correctness and recoverability (temp files, explicit errors, context cancellation).
- Predictable behavior for long-running downloads.
- Clear boundaries between domain (`internal/download`) and infrastructure (`internal/s3client`, telemetry/logging).

## 3) Architecture constraints to preserve

- Keep **domain logic in `internal/download`** and avoid leaking AWS SDK types there.
- Keep command handlers thin: validate flags, wire dependencies, delegate to service.
- Preserve context propagation (`cmd.Context()` / passed `ctx`) for cancellation, telemetry, and logging.
- Prefer dependency injection seams already present in tests (e.g., command built with interface service).

## 4) Testing workflow (expected)

Use small, focused changes and run tests frequently.

Recommended loop:
1. Add/adjust a focused test near the changed behavior.
2. Make minimal code change.
3. Run targeted test package.
4. Run full test suite before commit.

Useful commands:
- `go test ./...`
- `go test ./cmd ./internal/download ./internal/logging ./internal/telemetry`

If adding behavior that spans CLI -> service, prefer an end-to-end style command test in `cmd/*_test.go` plus a unit test in `internal/download/service_test.go`.

## 5) Security and performance guardrails

- Never log secrets, credentials, or sensitive object contents.
- Keep streaming behavior (`io.Copy`) for large objects; avoid buffering full files in memory.
- Maintain atomic write pattern (temp file + rename) to reduce partial-file risk.
- Preserve strict S3 URI validation (`s3://bucket/key`) unless intentionally expanding accepted formats.

## 6) Developer experience guardrails

- Keep errors actionable and user-facing text clear.
- Prefer explicit names over clever abstractions.
- Avoid adding deep framework complexity for simple command flows.
- Add tests with every behavior change; avoid untested refactors.

## 7) Suggested exploration order for new agents

1. `go test ./...` to establish baseline.
2. Read `cmd/root.go` and `cmd/download.go` for control flow.
3. Read `internal/download/service.go` for domain behavior.
4. Read `internal/s3client/client.go` for AWS interaction boundary.
5. Read TUI + app orchestration (`internal/tui/model.go`, `internal/app/app.go`) if touchpoints include interactive mode.

## 8) Common pitfalls

- Duplicating logic between command layer and service instead of centralizing in `internal/download`.
- Breaking command testability by hard-coding concrete dependencies.
- Ignoring context from Cobra command.
- Writing tests that require live AWS instead of mocking client interfaces.
