# Bulletproof Backup

Snapshot-based backup and versioning for AI agents. Detect drift, diagnose attacks, and restore to known-good states.

## The Key Capability

**Bulletproof enables agents to detect their own compromises using their own AI or that of another agent, known to be uncompromised.**

Bulletproof doesn't embed AI or require API keys. Instead, it provides:

- **Structured CLI output** — Agents parse diffs and analyze changes using their native LLM capabilities
- **Teaching methodology** — The `bulletproof skill` command trains agents in drift detection via binary search
- **Agent-driven analysis** — Agents provide the intelligence to diagnose attacks

The tool provides the **data**. The agent provides the **intelligence**. The skill provides the **methodology**. This separation is what makes autonomous security analysis possible.

## What It Does

AI agents change over time—skills evolve, personalities drift, configurations shift. When an agent is compromised through personality attacks, skill weapons, or prompt injection, you need tools to:

- **Detect drift** — Binary search through snapshots to find exactly when changes occurred
- **Diagnose attacks** — Compare files across time with unified diffs
- **Restore safely** — Roll back to any previous snapshot with verification

**For the full story**, see [specs/product-story.md](specs/product-story.md)
**For technical details**, see [specs/requirements.md](specs/requirements.md)

## Features

- ✅ **Snapshot-based backups** with SHA-256 file hashing
- ✅ **Short numeric IDs** (1, 2, 3) for easy snapshot reference
- ✅ **Unified diffs** (git-compatible) for precise change analysis
- ✅ **Binary search guidance** (700+ line methodology guide)
- ✅ **Multi-source backups** with glob pattern support
- ✅ **Custom scripts** (pre-backup exports, post-restore imports)
- ✅ **Three storage options** (local, git, cloud sync)
- ✅ **Retention policies** (keep-last, daily, weekly, monthly)
- ✅ **Platform scheduling** (systemd/launchd/Task Scheduler)
- ✅ **Self-contained backups** (config + scripts travel together)
- ✅ **Safety confirmations** (diff preview before restore)
- ✅ **Security warnings** (untrusted backup script detection)
- ✅ **Privacy-first analytics** (anonymous, opt-out anytime)
- ✅ **Cross-platform** (Linux, macOS, Windows)
- ✅ **Zero dependencies** (single binary, pure Go)

## Quick Start

### One Command to Bulletproof Your Agent

```bash
bulletproof init
```

**That's it.** This single command:
- Detects your OpenClaw installation location
- Prompts for backup destination path
- **Automatically sets up daily backups at 3:00 AM**
- Installs platform-specific scheduled services (systemd/launchd/Task Scheduler)

Your agent is now protected with automatic daily backups. No further setup required.

### Manual Backup (Optional)

```bash
bulletproof backup
```

Creates an immediate snapshot (useful for pre-deployment backups or testing).

### View Snapshots

```bash
bulletproof snapshots
```

Lists all available snapshots with short IDs (1, 2, 3...) and timestamps.

### Compare Changes

```bash
bulletproof diff 5 3
```

Shows unified diff between snapshots 5 and 3.

### Restore a Snapshot

```bash
bulletproof restore 2
```

Restores to snapshot 2 (creates safety backup first). Shows diff and asks for confirmation before overwriting files.

### Manage Old Snapshots

```bash
bulletproof prune --dry-run
```

Preview which snapshots would be deleted based on retention policy. Remove `--dry-run` to actually delete.

### Customize Backup Time (Optional)

```bash
# Change to 2:00 AM
bulletproof schedule enable --time 02:00

# Disable automatic backups
bulletproof schedule disable

# Check schedule status
bulletproof schedule status
```

## Advanced Features

### Multi-Source Backups

Back up multiple directories in a single snapshot:

```bash
# Configure in ~/.config/bulletproof/config.yaml
sources:
  - ~/.openclaw
  - ~/graph-exports/*
  - ~/vector-db/dumps
```

Supports glob patterns for dynamic source selection.

### Custom Scripts (Data Export/Import)

Execute custom scripts before backup or after restore:

```yaml
scripts:
  pre_backup:
    - name: "Export database"
      command: "~/scripts/db-export.sh"
  post_restore:
    - name: "Import database"
      command: "~/scripts/db-import.sh"
```

**Use cases**:
- Export Neo4j graph databases
- Export Pinecone vector indexes
- Backup external configuration
- Run validation checks

Scripts can access `$EXPORTS_DIR` to save outputs that get included in the snapshot.

### Migration to New Machine

Bootstrap configuration from an existing backup:

```bash
# On new machine
bulletproof init --from-backup /path/to/backup/20250203-120000
bulletproof restore 1
```

The backup includes your config and scripts, so everything migrates together.

### Restore to Alternative Location

Test restores without overwriting your live agent:

```bash
bulletproof restore 2 --target ~/test-restore
```

### Skip Prompts for Automation

For automated workflows:

```bash
bulletproof backup --force            # Skip no-change detection
bulletproof restore 1 --force         # Skip confirmation prompts
bulletproof backup --no-scripts       # Skip pre-backup scripts
bulletproof restore 1 --no-scripts    # Skip post-restore scripts
```

### Privacy-First Analytics

Bulletproof includes optional anonymous usage analytics (enabled by default):

```bash
bulletproof analytics status    # Check current status
bulletproof analytics disable   # Opt out completely
bulletproof analytics enable    # Re-enable
```

**What's tracked**: Commands used, OS type, CLI version (no file paths, no snapshot data, no PII)

**First-run notice**: Explains what's tracked on first use with easy opt-out

## Storage Options

Bulletproof supports three backup destination types, automatically detected based on your destination:

### 1. Multi-Folder Backups (Local/Network Storage)

Best for: Local disk, network shares, external drives

```yaml
destination:
  type: local  # Auto-detected for regular directories
  path: ~/bulletproof-backups
```

Creates timestamped subdirectories:

```
~/bulletproof-backups/
├── 20250203-120000/
├── 20250202-180000/
└── 20250201-180000/
```

### 2. Git Repository Backups

Best for: Version control, storage efficiency, remote backups

```yaml
destination:
  type: git  # Auto-detected for git repositories
  path: ~/bulletproof-repo # Must be a git repository
```

Each backup creates a git commit and tag. Automatic push to remote if configured. Git deduplication saves storage space.

### 3. Cloud Sync Backups (Dropbox/Google Drive)

Best for: Cloud sync services that handle versioning themselves

```yaml
destination:
  type: sync  # Non-timestamped, sync service handles versions
  path: ~/Dropbox/bulletproof-backup
```

Creates a single folder that's continuously synced. The sync service (Dropbox, Google Drive, OneDrive) maintains version history.

## What Gets Backed Up

**OpenClaw agent files:**
- Skills and capabilities (`workspace/skills/`)
- Personality definition (`workspace/SOUL.md`)
- Agent configuration (`openclaw.json`)
- Conversation logs (`workspace/memory/`)
- Agent definitions (`workspace/AGENTS.md`, `workspace/TOOLS.md`)

**Additional data (via scripts):**
- Neo4j graph database exports
- Pinecone vector index exports
- Custom configuration files
- External database dumps

**Self-contained metadata:**
- `.bulletproof/config.yaml` - Snapshot of your config
- `.bulletproof/snapshot.json` - File hashes and metadata
- `.bulletproof/scripts/` - Scripts at time of backup
- `_exports/` - Pre-backup script outputs

Default exclusions: `*.log`, `*.tmp`, `node_modules/`, `.git/`

**Result**: Each backup is completely self-contained and can restore on any machine, including scripts and external data.

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

### Core Commands

- `bulletproof init [--from-backup <path>]` - Initialize configuration (optionally from existing backup)
- `bulletproof backup [--force] [--no-scripts] [-m "message"]` - Create snapshot
- `bulletproof restore <id> [--target <path>] [--force] [--no-scripts]` - Restore snapshot
- `bulletproof snapshots [--format json|csv]` - List all snapshots with short IDs
- `bulletproof diff [id1] [id2] [pattern]` - Compare snapshots (supports 0-3 arguments)
- `bulletproof prune [--dry-run]` - Delete old snapshots per retention policy

### Management Commands

- `bulletproof schedule enable|disable|status [--time HH:MM]` - Manage automatic backups
- `bulletproof config show|edit|path` - View or modify configuration
- `bulletproof analytics enable|disable|status` - Manage anonymous usage tracking
- `bulletproof version` - Show version with update check

### Learning Command

- `bulletproof skill` - **700+ line comprehensive guide teaching:**
  - Binary search methodology for drift detection (8-step tutorial)
  - Attack pattern identification (personality attacks, skill weapons)
  - Platform migration workflows with scripts
  - Automated service setup (systemd/launchd/Task Scheduler)
  - Neo4j and Pinecone integration examples

Run `bulletproof <command> --help` for detailed usage.

**Design philosophy**: The tool provides structured data; agents apply their native LLM intelligence for analysis. The `skill` command teaches the methodology—it doesn't automate the thinking.

## Configuration

Config file: `~/.config/bulletproof/config.yaml`

### Basic Configuration

```yaml
destination:
  type: local  # 'local', 'git', or 'sync'
  path: ~/bulletproof-backups

exclude:
  - "*.log"
  - "*.tmp"
  - node_modules/
  - .git/
```

### Complete Configuration Schema

```yaml
# Single source (simple case)
openclaw_path: ~/.openclaw

# OR multiple sources with glob patterns
sources:
  - ~/.openclaw
  - ~/graph-exports/*
  - ~/vector-db/dumps/*.json

destination:
  type: local  # Required: 'local', 'git', or 'sync'
  path: ~/bulletproof-backups

# Automatic backup scheduling
schedule:
  enabled: true
  time: "03:00"  # HH:MM format

# Backup options
options:
  include_auth: false
  exclude:
    - "*.log"
    - "*.tmp"
    - node_modules/
    - .git/

# Custom scripts for data export/import
scripts:
  pre_backup:
    - name: "Export Neo4j"
      command: "~/scripts/neo4j-export.sh"
      timeout: 300  # seconds (default: 60)
  post_restore:
    - name: "Import Neo4j"
      command: "~/scripts/neo4j-import.sh"
      timeout: 300

# Automatic snapshot pruning
retention:
  enabled: true
  keep_last: 10        # Keep last 10 snapshots
  keep_daily: 7        # Keep daily snapshots for 7 days
  keep_weekly: 4       # Keep weekly snapshots for 4 weeks
  keep_monthly: 6      # Keep monthly snapshots for 6 months

# Anonymous usage analytics (opt-in by default)
analytics:
  enabled: true  # Set to false to disable
```

### Script Environment Variables

Scripts have access to these environment variables:

- `$SNAPSHOT_ID` - Current snapshot identifier
- `$OPENCLAW_PATH` - Path to OpenClaw installation
- `$BACKUP_DIR` - Backup destination directory
- `$EXPORTS_DIR` - Directory for script outputs (`_exports/`)

**Example script** (`neo4j-export.sh`):
```bash
#!/bin/bash
neo4j-admin dump --to="$EXPORTS_DIR/neo4j-backup.dump"
echo "Exported Neo4j to $EXPORTS_DIR"
```

For more details, see [specs/requirements.md](specs/requirements.md#configuration).

## Development

See [CLAUDE.md](CLAUDE.md) for development setup, architecture details, and contribution guidelines.

```bash
# Build
make build

# Run tests (65+ passing tests)
make test

# Run all checks (format, vet, lint, test)
make check

# Cross-compile for all platforms
make build-all
```

### Code Quality

- **65+ integration tests** covering all commands and edge cases
- **Test-driven development** - tests written first, implementation follows
- **Platform-specific testing** - Linux, macOS, Windows
- **Comprehensive coverage**:
  - Multi-source backups
  - Retention policies
  - Config validation
  - Script execution
  - Git operations
  - Restore workflows

## Troubleshooting

### Analytics Notice on First Run

On first command execution, you'll see a transparent notice about anonymous analytics:

```
📊 Anonymous Usage Analytics
Bulletproof collects anonymous usage data to improve the tool.
What's tracked: Commands, OS type, CLI version
What's NOT tracked: File paths, snapshot data, personal information

To opt out: bulletproof analytics disable
```

This only shows once and respects your privacy completely.

### Untrusted Backup Warning

When restoring a backup from an untrusted source, you'll see a security warning before scripts execute:

```
⚠️  SECURITY WARNING
This backup contains post-restore scripts that will execute...
```

Options:
- Review scripts in `.bulletproof/scripts/` before approving
- Use `--no-scripts` to skip script execution
- Use `--force` only for verified trusted backups

### Restore Confirmation

Before overwriting files, you'll see a diff and confirmation prompt:

```
📋 Changes that will be applied:
  + 3 files will be removed
  + 5 files will be added
  ~ 2 files will be modified

⚠️  This will overwrite your current files. Are you sure? [y/N]:
```

Use `--force` to skip this prompt for automation.

## Documentation

- [Product Story](specs/product-story.md) - User journeys, security context, and feature overview
- [Requirements](specs/requirements.md) - Complete technical specification
- [CLAUDE.md](CLAUDE.md) - Development guide for contributors

## License

MIT
