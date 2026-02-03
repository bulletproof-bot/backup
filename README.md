# bulletproof backup

Back up your AI agent. Track changes over time. Rollback when things go wrong.

## What it does

Your agent changes over time — skills get added, personality drifts, memories accumulate. This tool captures snapshots so you can:

- **See what changed** — Git-based versioning shows diffs between any two points in time
- **Roll back** — Restore your agent to any previous state
- **Export anywhere** — Your cloud storage (Dropbox, Google Drive, S3), your data

## What gets backed up

- Skills and capabilities
- Soul file (personality definition)
- Conversations and memories
- System prompts
- Configuration

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

## Quick Start

### 1. Initialize configuration

```bash
bulletproof init
```

This will guide you through setting up backup destinations (local folder, git repo, or sync folder).

### 2. Create your first backup

```bash
bulletproof backup -m "Initial backup"
```

### 3. View backup history

```bash
bulletproof history
```

### 4. See what changed

```bash
bulletproof diff
```

### 5. Restore to a previous version

```bash
bulletproof restore <snapshot-id>
```

## Commands

- `bulletproof init` - Interactive setup wizard
- `bulletproof backup` - Create a new backup snapshot
- `bulletproof restore <id>` - Restore to a specific snapshot
- `bulletproof diff` - Show changes since last backup
- `bulletproof history` - List all backup snapshots
- `bulletproof config` - View or modify configuration
- `bulletproof version` - Show version information

Run `bulletproof --help` for detailed command usage.

## Configuration

Config file: `~/.config/bulletproof/config.yaml`

### Backup Destinations

- **Local**: Timestamped folders on your filesystem
- **Git**: Git repository with commit history and tags (supports remote push)
- **Sync**: Non-timestamped folder for cloud sync services (Dropbox, Google Drive, etc.)

### Default Exclusions

The following patterns are excluded from backups by default:
- `*.log`
- `node_modules/`
- `.git/`
- Temporary and cache files

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

## License

MIT

