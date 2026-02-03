# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bulletproof is a CLI tool for backing up OpenClaw AI agents with snapshot-based versioning. It tracks changes over time and enables rollback to previous states. The tool was migrated from Dart to Go to leverage Go's cross-compilation and single-binary distribution.

## Build and Development Commands

```bash
# Build the binary
make build

# Install to GOPATH/bin
make install

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/types
go test ./internal/config

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

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

- **LocalDestination**: Timestamped folders on local filesystem
- **GitDestination**: Git commits + tags with optional remote push (using go-git library)
- **SyncDestination**: Non-timestamped folders for cloud sync services (Dropbox, Google Drive)

The factory pattern in `backup/engine.go::createDestination()` instantiates the appropriate implementation based on config.

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
- **ID format**: `yyyyMMdd-HHmmss` timestamp
- **Diffing**: Compares snapshots to show added/modified/removed files
- **Skip optimization**: If no changes detected, backup is skipped

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

## Module Path

`github.com/bulletproof-bot/backup` - used in import statements throughout the codebase.
