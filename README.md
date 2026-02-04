# Bulletproof Backup

Snapshot-based backup and versioning for AI agents. Detect drift, diagnose attacks, and restore to known-good states.

## What It Does

AI agents change over time—skills evolve, personalities drift, configurations shift. When an agent is compromised through personality attacks, skill weapons, or prompt injection, you need tools to:

- **Detect drift** — Binary search through snapshots to find exactly when changes occurred
- **Diagnose attacks** — Compare files across time with unified diffs
- **Restore safely** — Roll back to any previous snapshot with verification

**For the full story**, see [specs/product-story.md](specs/product-story.md)
**For technical details**, see [specs/requirements.md](specs/requirements.md)

## Quick Start

### 1. Initialize configuration

```bash
bulletproof init
```

Prompts for backup destination path. Automatically detects if destination is a git repository.

### 2. Create your first backup

```bash
bulletproof backup
```

Creates snapshot with timestamp ID (e.g., `20250203-120000`)

### 3. View snapshots

```bash
bulletproof snapshots
```

Lists all available snapshots with short IDs (1, 2, 3...) and timestamps.

### 4. Compare changes

```bash
bulletproof diff 5 3
```

Shows unified diff between snapshots 5 and 3.

### 5. Restore a snapshot

```bash
bulletproof restore 2
```

Restores to snapshot 2 (creates safety backup first).

## Storage Options

Bulletproof supports two backup approaches, automatically detected based on your destination:

### Multi-Folder Backups

Works anywhere—local disk, Dropbox, Google Drive, OneDrive, network shares:

```yaml
destination:
  path: ~/bulletproof-backups
```

Creates timestamped subdirectories:
```
~/bulletproof-backups/
├── 20250203-120000/
├── 20250202-180000/
└── 20250201-180000/
```

### Git Repository Backups

If your destination is a git repository, Bulletproof automatically uses git operations:

```yaml
destination:
  path: ~/bulletproof-repo  # Must be a git repository
```

Each backup creates a git commit and tag. Git deduplication saves storage space.

## What Gets Backed Up

- Skills and capabilities (`workspace/skills/`)
- Personality definition (`workspace/SOUL.md`)
- Agent configuration (`openclaw.json`)
- Conversation logs (`workspace/memory/`)
- Custom data exports (via pre-backup scripts)

Default exclusions: `*.log`, `*.tmp`, `node_modules/`, `.git/`

## Installation

### From Release (Recommended)

Download the latest release for your platform:

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/bulletproof-bot/backup/releases/latest/download/bulletproof_darwin_arm64.tar.gz | tar xz
sudo mv bulletproof /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -L https://github.com/bulletproof-bot/backup/releases/latest/download/bulletproof_darwin_amd64.tar.gz | tar xz
sudo mv bulletproof /usr/local/bin/
```

**Linux:**
```bash
curl -L https://github.com/bulletproof-bot/backup/releases/latest/download/bulletproof_linux_amd64.tar.gz | tar xz
sudo mv bulletproof /usr/local/bin/
```

**Windows:**
Download the `.zip` file from [releases](https://github.com/bulletproof-bot/backup/releases/latest) and extract `bulletproof.exe` to your PATH.

### From Source

Requires Go 1.21 or later:

```bash
git clone https://github.com/bulletproof-bot/backup.git
cd backup
make build
sudo cp bin/bulletproof /usr/local/bin/
```

## Commands

- `bulletproof init` - Initialize configuration
- `bulletproof backup` - Create snapshot of current state
- `bulletproof restore <id>` - Restore to specific snapshot
- `bulletproof snapshots` - List all available snapshots
- `bulletproof diff [id1] [id2]` - Compare snapshots or current state
- `bulletproof config` - View or modify configuration
- `bulletproof skill` - Learn advanced usage and drift diagnosis
- `bulletproof version` - Show version information

Run `bulletproof --help` for detailed command usage.

## Configuration

Config file: `~/.config/bulletproof/config.yaml`

Example configuration:

```yaml
destination:
  path: ~/bulletproof-backups

exclude:
  - "*.log"
  - "*.tmp"
  - node_modules/
  - .git/
  - .DS_Store
```

For advanced configuration (custom scripts, multiple sources, analytics), see [specs/requirements.md](specs/requirements.md#configuration).

## Development

See [CLAUDE.md](CLAUDE.md) for development setup, architecture details, and contribution guidelines.

```bash
# Build
make build

# Run tests
make test

# Run all checks (format, vet, lint, test)
make check
```

## Documentation

- [Product Story](specs/product-story.md) - User journeys, security context, and feature overview
- [Requirements](specs/requirements.md) - Complete technical specification
- [CLAUDE.md](CLAUDE.md) - Development guide for contributors

## License

MIT
