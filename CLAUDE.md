# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bulletproof is a CLI tool for backing up OpenClaw AI agents with snapshot-based versioning. It tracks changes over time and enables rollback to previous states. The tool was migrated from Dart to Go to leverage Go's cross-compilation and single-binary distribution.

### CLI Commands

- `bulletproof init` - Initialize configuration (prompts for destination path)
- `bulletproof backup` - Create snapshot of current state
- `bulletproof restore <id>` - Restore to specific snapshot (creates safety backup first)
- `bulletproof snapshots` - List all available snapshots with short IDs (1, 2, 3...)
- `bulletproof diff [id1] [id2]` - Compare snapshots or current state with unified diffs
- `bulletproof config` - View or modify configuration
- `bulletproof skill` - Advanced usage and drift diagnosis guide
- `bulletproof version` - Show version information with update checking

### Documentation

- `specs/product-story.md` - User journeys, security context (personality attacks, skill weapons), and feature overview
- `specs/requirements.md` - Complete technical specification with edge cases and advanced configuration
- `README.md` - User-facing documentation with quick start guide and installation instructions

## Build and Development Commands

```bash
# Build the binary
make build

# Install to GOPATH/bin
make install

# Run all checks (format, vet, lint, test) - use before committing
make check

# Format code
make fmt
go fmt ./...

# Run go vet
make vet
go vet ./...

# Run linter (requires golangci-lint)
make lint

# Run all tests
make test
go test ./...

# Run tests for a specific package
go test ./internal/types
go test ./internal/config

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Cross-compile for all platforms
make build-all

# Clean build artifacts
make clean
```

## High-Level Architecture

### Package Structure Philosophy

The codebase uses a **layered architecture** with clear separation of concerns:

```
internal/
├── types/          # Shared data structures (Snapshot, SnapshotDiff, etc.)
├── config/         # Configuration and OpenClaw detection
├── backup/         # Core backup orchestration
│   ├── destinations/  # Backup destination implementations
│   └── *.go        # BackupEngine, Destination interface
├── commands/       # CLI command implementations
└── utils/          # Shared utilities (file ops, hashing)
```

**Key architectural decision:** The `types` package was created to break a circular dependency between `backup` and `destinations`. Both packages need to reference `Snapshot` and related types, so these were extracted into a shared `types` package that both can import.

### Data Flow

1. **Commands layer** (`internal/commands`) - Cobra CLI commands that parse user input
2. **Engine layer** (`internal/backup`) - BackupEngine orchestrates backup/restore operations
3. **Destination layer** (`internal/backup/destinations`) - Implements storage backends (local, git, sync)
4. **Types layer** (`internal/types`) - Defines Snapshot data structures used across layers

### Destination Interface Pattern

The `Destination` interface (defined in `internal/backup/destination.go`) allows pluggable storage backends:

- **LocalDestination**: Timestamped folders on local filesystem (multi-folder backups)
- **GitDestination**: Git commits + tags with optional remote push (using go-git library)
- **SyncDestination**: Non-timestamped folders for cloud sync services (Dropbox, Google Drive)

The factory pattern in `backup/engine.go::createDestination()` instantiates the appropriate implementation based on config. **Auto-detection**: If the destination path is a git repository, GitDestination is automatically used; otherwise LocalDestination is used for regular directories.

### OpenClaw Integration

The tool backs up OpenClaw agent installations, detecting them at:
- Default: `~/.openclaw/`
- Docker: `/data/.openclaw`, `/openclaw`, `/app/.openclaw`

Critical files backed up (see `config/openclaw.go::GetBackupTargets()`):
- `openclaw.json` - main configuration
- `workspace/SOUL.md` - agent personality
- `workspace/AGENTS.md`, `workspace/TOOLS.md` - agent definitions
- `workspace/skills/` - skill modules
- `workspace/memory/` - conversation logs

### Snapshot System

Snapshots are point-in-time backups with SHA-256 file hashing:
- **ID format**: Full timestamp `yyyyMMdd-HHmmss` used internally and for folder names
- **User-facing IDs**: CLI displays short IDs (1, 2, 3...) for convenience, sorted by timestamp
- **Diffing**: Compares snapshots to show added/modified/removed files with unified diffs
- **Skip optimization**: If no changes detected, backup is skipped (avoids duplicate snapshots)

## Important Patterns and Conventions

### Error Handling

- **Never swallow errors** - all errors propagate up with context using `fmt.Errorf("context: %w", err)`
- **No try-catch blocks** - errors are values, not exceptions
- Use error wrapping to maintain error chains: `if err != nil { return fmt.Errorf("failed to X: %w", err) }`

### Configuration

- Config file: `~/.config/bulletproof/config.yaml`
- YAML with inline comments for readability (see `config/config.go::Save()`)
- Default exclusions: `*.log`, `node_modules/`, `.git/`

### Testing

- Tests live next to implementation files (e.g., `snapshot_test.go` next to `snapshot.go`)
- Use table-driven tests for multiple test cases (see `config_test.go`)
- Use `t.TempDir()` for test isolation - automatically cleaned up
- Successful tests are silent; failures reported via standard `testing` package

### Git Operations

Uses `github.com/go-git/go-git/v5` for pure Go git implementation:
- No external git binary dependency
- RefSpec is in `config` package, not `plumbing`: `config.RefSpec("refs/tags/*:refs/tags/*")`
- Tags created for each snapshot
- Push with `--follow-tags` to sync tags to remote

## Migration Context (Dart → Go)

This codebase was migrated from Dart to preserve functionality while gaining Go benefits:
- Single binary distribution (no runtime dependency)
- Easy cross-compilation for Linux/macOS/Windows
- Strong standard library for file operations

The migration preserved:
- All 6 core CLI commands (init, backup, restore, diff, history, config)
- Snapshot-based backup system with diffing
- Three destination types (local, git, sync)
- OpenClaw detection and integration

Deferred/removed:
- Platform-specific schedulers (cron/launchd/Task Scheduler) - removed as placeholder code
- Service management commands - removed as placeholder code

## Key Dependencies

- `github.com/spf13/cobra` - CLI framework for command structure
- `gopkg.in/yaml.v3` - YAML config parsing
- `github.com/go-git/go-git/v5` - Pure Go git implementation

## Architecture Best Practices

- **TDD (Test-Driven Development)** - write the tests first; the implementation code isn't done until the tests pass.
- **DRY (Don't Repeat Yourself)** – eliminate duplicated logic by extracting shared utilities and modules.
- **Separation of Concerns** – each module should handle one distinct responsibility.
- **Single Responsibility Principle (SRP)** – every class/module/function/file should have exactly one reason to change.
- **Clear Abstractions & Contracts** – expose intent through small, stable interfaces and hide implementation details.
- **Low Coupling, High Cohesion** – keep modules self-contained, minimize cross-dependencies.
- **Scalability & Statelessness** – design components to scale horizontally and prefer stateless services when possible.
- **Observability & Testability** – build in logging, metrics, tracing, and ensure components can be unit/integration tested.
- **KISS (Keep It Simple, Sir)** - keep solutions as simple as possible.
- **YAGNI (You're Not Gonna Need It)** – avoid speculative complexity or over-engineering.
- **Don't Swallow Errors** by catching exceptions, silently filling in required but missing values or adding timeouts when something hangs unexpectedly. All of those are exceptions that should be thrown so that the errors can be seen, root causes can be found and fixes can be applied.
- **No Placeholder Code** - we're building production code here, not toys.
- **No Comments for Removed Functionality** - the source is not the place to keep history of what's changed; it's the place to implement the current requirements only.
- **Layered Architecture** - organize code into clear tiers where each layer depends only on the one(s) below it, keeping logic cleanly separated.
- **Prefer Non-Nullable Variables** when possible; use nullability sparingly.
- **Prefer Async Notifications** when possible over inefficient polling.
- **Consider First Principles** to assess your current architecture against the one you'd use if you started over from scratch.
- **Eliminate Race Conditions** that might cause dropped or corrupted data
- **Write for Maintainability** so that the code is clear and readable and easy to maintain by future developers.
- **Arrange Project Idiomatically** for the language and framework being used, including recommended lints, static analysis tools, folder structure and gitignore entries.

## Go Idioms and Best Practices

### Code Quality Tools

- **golangci-lint** configuration in `.golangci.yml` with comprehensive linters enabled
- **GitHub Actions** workflow (`.github/workflows/go.yml`) runs on push and PR
- **Package documentation** - each package has a `doc.go` file with package-level comments
- Use `make check` before committing to run all quality checks

### Linting Rules

The project uses golangci-lint with these key linters:
- `errcheck` - ensures all errors are checked
- `gosimple`, `staticcheck` - code simplification and static analysis
- `govet` - standard Go vet checks
- `revive` - fast, extensible linter for style
- `gosec` - security checker
- `errorlint` - proper error wrapping with `%w`

### CI/CD

GitHub Actions workflow runs:
1. Format check (gofmt)
2. Vet check
3. Tests with race detection and coverage
4. golangci-lint
5. Cross-platform builds

## Release Process

### Version Management

Version information is embedded at build time using Go's `-ldflags`:

```go
// internal/version/version.go
var (
    Version   = "dev"      // Set via -X flag
    GitCommit = "none"     // Set via -X flag
    BuildDate = "unknown"  // Set via -X flag
)
```

Build flags in Makefile and .goreleaser.yml inject these values:
```
-X github.com/bulletproof-bot/backup/internal/version.Version={{.Version}}
-X github.com/bulletproof-bot/backup/internal/version.GitCommit={{.ShortCommit}}
-X github.com/bulletproof-bot/backup/internal/version.BuildDate={{.Date}}
```

### Automatic Update Checking

The CLI checks for newer versions via GitHub API (see `internal/version/version.go::CheckForUpdate()`):
- Runs asynchronously after command execution (non-blocking)
- Compares current version against latest GitHub release
- Displays update notice if newer version available
- Skips check for dev builds

### Creating a Release

The project uses [GoReleaser](https://goreleaser.com/) for automated multi-platform builds:

1. **Tag the version:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions automatically:**
   - Runs tests and linting
   - Builds binaries for Linux, macOS (Intel/ARM), Windows
   - Creates GitHub release with installation instructions
   - Generates checksums and archives

3. **Configuration files:**
   - `.goreleaser.yml` - GoReleaser build configuration
   - `.github/workflows/release.yml` - GitHub Actions release workflow
   - `.github/workflows/go.yml` - CI workflow (runs on push/PR)

### Local Release Testing

```bash
# Test cross-platform builds locally
make build-all

# Test version information
./bin/bulletproof version

# Create a snapshot release (requires goreleaser installed)
make release
```

### Release Workflow

The release workflow (`.github/workflows/release.yml`):
- Triggers on tags matching `v*`
- Sets up Go 1.21
- Runs GoReleaser with GitHub token for release creation
- Produces artifacts:
  - `bulletproof_VERSION_linux_amd64.tar.gz`
  - `bulletproof_VERSION_darwin_amd64.tar.gz`
  - `bulletproof_VERSION_darwin_arm64.tar.gz`
  - `bulletproof_VERSION_windows_amd64.zip`
  - `checksums.txt`

## Module Path

`github.com/bulletproof-bot/backup` - used in import statements throughout the codebase.
