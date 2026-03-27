# Go Modules Initialization Design

**Date:** 2026-03-26
**Status:** Approved

## Overview

Initialize two Go modules in the project-neo monorepo — `apps/workers` and `packages/graphql-api` — enabling the Go workspace (`go.work`) and unblocking Go CI cache. Both modules are created atomically in a single commit.

## Goals

- Establish the Go workspace with both modules registered
- Provide minimal but compilable entry points for each service
- Wire both modules into Turborepo so `bun run build`, `bun run dev`, and `bun run lint` work across the monorepo
- Enable Go dependency caching in CI (requires `go.sum` files to be present and committed)

## Non-Goals

- No business logic, HTTP servers, or service skeletons
- No tests (deferred for later phase)
- No shared packages or inter-module imports

## Files Created

### `apps/workers/go.mod`
```
module project-neo/workers

go 1.24.4
```

### `apps/workers/go.sum`
Run `go mod tidy` inside `apps/workers/`. For a module with no external dependencies, `go mod tidy` may produce no `go.sum` file at all — this is fine. `modules-download-mode: readonly` in `.golangci.yml` only requires `go.sum` entries to match what is declared in `require` blocks; an empty `require` block has nothing to check. If the file is not created, do not force-create it.

### `apps/workers/main.go`
```go
package main

func main() {}
```

### `apps/workers/package.json`
```json
{
  "name": "@project-neo/workers",
  "private": true,
  "scripts": {
    "build": "go build ./...",
    "dev": "go run .",
    "lint": "golangci-lint run --config ../../.golangci.yml ./..."
  }
}
```

### `packages/graphql-api/go.mod`
```
module project-neo/graphql-api

go 1.24.4
```

### `packages/graphql-api/go.sum`
Run `go mod tidy` inside `packages/graphql-api/`. Same rationale as `apps/workers/go.sum` above — only commit if the file is actually created by `go mod tidy`.

### `packages/graphql-api/main.go`
```go
package main

func main() {}
```

### `packages/graphql-api/package.json`
```json
{
  "name": "@project-neo/graphql-api",
  "private": true,
  "scripts": {
    "build": "go build ./...",
    "dev": "go run .",
    "lint": "golangci-lint run --config ../../.golangci.yml ./..."
  }
}
```

## Files Modified

### `go.work`
Replace the entire file content with:
```
go 1.24.4

use (
	./apps/workers
	./packages/graphql-api
)
```
(Note: indentation uses tabs as required by Go workspace file format.)

### `.github/workflows/ci.yml`
Enable Go cache — change:
```yaml
cache: false  # Disable until we have go.sum files
```
to:
```yaml
cache: true
```

### `.github/workflows/release.yml`
Same change as `ci.yml` — also has `cache: false` with the same comment. Update to `cache: true`.

### `bun.lock`
Adding two new `package.json` files to the Bun workspace will cause `bun install` to update `bun.lock`. Run `bun install` after creating the `package.json` files and commit the updated lockfile alongside everything else.

## Module Naming

Module paths use the short form `project-neo/<name>` (not the full GitHub URL). Since both modules are private and resolved via `go.work`, the path is a local identifier only — not a fetchable URL. This keeps import paths concise.

## Turborepo Integration

Each module's `package.json` exposes `build`, `dev`, and `lint` scripts, which Turborepo picks up via the root `turbo.json` pipeline. The Bun workspace glob (`apps/*`, `packages/*`) will include both new `package.json` files automatically.

**Known limitation — turbo.json outputs:** The root `turbo.json` defines `build` outputs as `["dist/**", "build/**", ".next/**"]`. A `go build ./...` places the binary in the module root, not those paths. Turborepo will not cache Go build output correctly. Deferred until the services have real build targets.

**Known limitation — lint-staged:** The root `package.json` lint-staged config runs `golangci-lint run --fix --config .golangci.yml` from the repo root when `.go` files are staged. With `go.work` present, modern versions of golangci-lint resolve module context via the workspace file, so this may work. However, if it fails on pre-commit, the fix is to update the lint-staged pattern to run per-directory rather than from the root. This is deferred — validate on first commit that touches Go files.

## Linting

The per-module `lint` script (`golangci-lint run --config ../../.golangci.yml ./...`) runs from the module root, correctly resolving the relative config path. This is the canonical linting path via Turborepo.

The root `lint:go` script (`golangci-lint run --config .golangci.yml ./...`) runs from the repo root and may behave inconsistently with `go.work` module boundaries. It is superseded by per-module linting and can be addressed in a follow-up.

The empty `func main() {}` stubs are expected to pass all configured linters cleanly.

## CI Impact

- `go.sum` files committed → `cache: true` in both workflows is now meaningful
- `go test ./...` step in CI will now actively run (finds `.go` files), and the empty modules will produce zero test output cleanly
- `go work sync` step is a no-op for dependency-free modules but validates workspace integrity

## Commit Strategy

Single commit. The exact sequence before committing:

1. Create all module files (`go.mod`, `main.go`, `package.json` for both modules)
2. Update `go.work` to the final content shown above
3. Run `go mod tidy` inside each module directory (commits `go.sum` if created)
4. Run `bun install` from the repo root (updates `bun.lock`)
5. Update `cache: false` → `cache: true` in both `ci.yml` and `release.yml`
6. Stage and commit everything atomically — `bun.lock` **must** be in the same commit as the `package.json` files, otherwise CI (`bun install --frozen-lockfile`) will fail on the PR

Commit message:
```
feat(workers): initialize Go modules and enable workspace
```

Scope `workers` is used as the primary scope since it represents the first Go service. The `graphql-api` module is part of the same atomic workspace initialization.
