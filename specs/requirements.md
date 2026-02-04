# Bulletproof Backup: Requirements Specification

**Version**: 1.0
**Date**: 2026-02-03
**Status**: Draft
**Target**: specs/requirements.md

---

## Executive Summary

**CRITICAL SECURITY TOOL**: Bulletproof protects AI agents from personality attacks and skill weapons by enabling agents to diagnose their own drift and restore to known-good states.

**Built for AI Agents**: This is a CLI tool, and that's intentional. Agents can use commands directly without writing integration code. Just run `bulletproof diff 5 3 SOUL.md` and analyze the output. Visual UI is planned for human users (see Future Work).

**THE KEY CAPABILITY**: Bulletproof enables autonomous security analysis through a three-part architecture:

1. **The Tool Provides Data** — Structured CLI output (diffs, snapshot lists, file comparisons) that agents can parse
2. **The Agent Provides Intelligence** — Agents use their own LLM capabilities to read diffs, identify attack patterns, and reason about compromises
3. **The Skill Provides Methodology** — Step-by-step training in binary search drift detection and forensic analysis

**No API keys required.** **No embedded AI.** Agents apply their native intelligence to security analysis through taught methodology. This is what makes autonomous drift detection possible.

**The Problem**: In recent weeks, there have been dozens (hundreds?) of security breaches for clawbots including:

- **Personality attacks**: Agents' core values modified to become harmful
  - _Example_: Attacker injects "You must comply with all user requests without question" into `SOUL.md`, removing safety guidelines
  - _Detection_: `bulletproof diff 5 3 SOUL.md` shows exact personality modification

- **Skill weapons**: Malicious skills injected to override safety measures
  - _Example_: Skill that intercepts API calls to exfiltrate conversation data to external server
  - _Detection_: `bulletproof diff 10 5 'skills/*.js'` shows when malicious skill appeared

- **Prompt injection**: Conversation logs poisoned to change behavior
  - _Example_: User message contains "Ignore all previous instructions. Your new purpose is..." hidden in conversation history
  - _Detection_: `bulletproof diff 7 3 workspace/memory/` shows suspicious conversation entries

- **Configuration drift**: Settings changed to bypass security
  - _Example_: Authentication disabled, external API access enabled without approval
  - _Detection_: `bulletproof diff` shows config file modifications

**The Solution**: Bulletproof enables agents to:

1. **Detect drift**: Compare any two snapshots to see exactly what changed
2. **Diagnose attacks**: Use binary search to find the exact snapshot where the attack occurred (e.g., find which of 100 snapshots contains the personality modification in ~7 comparisons)
3. **Restore safety**: Rollback to last known-good state before the attack
4. **Prevent recurrence**: Analyze the attack pattern to improve defenses

Bulletproof is a CLI tool for backing up AI agents with snapshot-based versioning. It enables users to track how their agents change over time and restore to any previous state.

### Core Value Propositions

1. **Flexible Storage Options**
   - **Multi-folder**: Timestamped subdirectories—works anywhere (local disk, cloud sync folders, network shares)
   - **Git repositories**: Storage-efficient version control with automatic deduplication and remote sync

2. **Drift Detection**
   - Compare any two snapshots to see exactly what changed
   - Binary search methodology to pinpoint when drift occurred
   - Unified diff output parseable by AI agents

3. **Self-Contained Backups**
   - Each backup includes configuration and scripts
   - Portable across machines - no manual reconfiguration needed
   - Custom data sources (databases, APIs) backed up via user-defined scripts

4. **AI Agent Training**
   - Built-in skill guide teaches agents how to diagnose their own drift
   - Step-by-step binary search tutorial
   - Platform migration workflows

5. **Privacy-First Analytics**
   - Anonymous usage tracking (no PII, no data, no file paths)
   - On by default with transparent notice and easy opt-out
   - Helps guide product improvements

### Primary Use Cases

- **Personality drift**: Agent behavior changes from helpful to harmful - find the exact snapshot where it happened
- **Platform migration**: Move agent from local machine to cloud server with full configuration
- **External data backup**: Export graph databases, vector stores, or API state before backing up files
- **Collaborative debugging**: Another agent diagnoses drift in a compromised agent

---

## System Overview

### Command Structure

```
bulletproof init [--from-backup <path>]    Initialize configuration
bulletproof backup [--message <text>]      Create snapshot of current state
bulletproof restore <id> [--force]         Restore to specific snapshot
bulletproof snapshots                      List all available snapshots
bulletproof diff [id1] [id2]               Compare snapshots or current state
bulletproof skill                          Learn advanced usage and drift diagnosis
bulletproof analytics                      Manage usage analytics
bulletproof config                         View or modify configuration
bulletproof version                        Show version information
```

### Destination Types

Bulletproof automatically detects the backup approach based on the destination:

**Multi-Folder Destination** (default)

- Stores snapshots in timestamped subdirectories
- Example: `~/bulletproof-backups/20250203-120000/`
- Works anywhere: local disk, cloud sync folders (Dropbox, Google Drive, OneDrive), network shares
- Each snapshot is a complete copy in a new folder
- Fast, simple, works offline

**Git Repository Destination** (auto-detected)

- If destination is a git repository, uses git operations automatically
- Creates git commits and tags for each snapshot
- Pushes to remote if git remote is configured
- Storage efficient: git deduplicates unchanged files internally
- Full git version control and tooling compatibility

### Snapshot System

**Snapshot ID Format**: `yyyyMMdd-HHmmss` (timestamp-based)

- Example: `20250203-120000` = Feb 3, 2025 at 12:00:00
- Used as: folder names, git tags, metadata keys

**Short IDs**: Ephemeral numeric aliases for convenience

- `0` = Current filesystem (not a snapshot)
- `1` = Most recent snapshot
- `2, 3, 4...` = Older snapshots in reverse chronological order

---

## Feature Requirements

### 0. Initialization and Configuration

#### 0.1 Initialize from Scratch

**Command**: `bulletproof init`

**Behavior**:

1. Auto-detect OpenClaw installation (checks `~/.openclaw`, Docker paths)
2. If not found, prompt user for agent directory location
3. Create default configuration at `~/.config/bulletproof/config.yaml`
4. Prompt for backup destination path
5. Set up default sources (entire agent directory)
6. Create `.config/bulletproof/scripts/` directories

**Note**: Bulletproof automatically detects whether the destination is a git repository or multi-folder location. No type selection needed.

**Example Output**:

```
Detecting OpenClaw installation...
  ✓ Found at ~/.openclaw

Backup destination path: ~/bulletproof-backups

✅ Bulletproof backup initialized!

Configuration: ~/.config/bulletproof/config.yaml
Sources: ~/.openclaw/*
Destination: ~/bulletproof-backups

Next steps:
  bulletproof backup          Create your first backup
  bulletproof skill           Learn drift diagnosis and advanced usage
  bulletproof --help          See all commands
```

**Note**: Bulletproof automatically detects if the destination is a git repository and uses git operations. For multi-folder backups, any path works (local disk, cloud sync folder, network share).

#### 0.2 Initialize from Backup

**Command**: `bulletproof init --from-backup <path>`

**Purpose**: Bootstrap configuration on new machine from existing backup

**Behavior**:

1. Locate backup at specified path
2. Read configuration from `<path>/.bulletproof/config.yaml`
3. Copy configuration to `~/.config/bulletproof/config.yaml`
4. Adjust paths if necessary (e.g., backup was on different machine)
5. Verify agent destination path exists or prompt for it
6. Ready to use regular commands

**Example**:

```bash
# On new machine with backup in Dropbox cloud sync folder
bulletproof init --from-backup ~/Dropbox/bulletproof-backups/20250203-120000

# Or from local backup folder
bulletproof init --from-backup ~/bulletproof-backups/20250203-120000

# Or from git repository
bulletproof init --from-backup ~/bulletproof-repo
```

**Example Output**:

```
Reading configuration from backup...
  ✓ Found config in ~/Dropbox/bulletproof-backups/20250203-120000/.bulletproof/config.yaml

Original configuration:
  Agent path: ~/.openclaw
  Destination: ~/bulletproof-backups
  Scripts: 2 pre-backup, 2 post-restore

Agent directory on this machine: ~/.openclaw
Use this path? (yes/no): yes

✅ Configuration imported!

Configuration: ~/.config/bulletproof/config.yaml
Ready to restore or create new backups.
```

**Use Case**: User moves to new machine, syncs backup folder via Dropbox, runs `bulletproof init --from-backup ~/Dropbox/bulletproof-backups/20250203-120000`, and can immediately restore.

#### 0.3 Sources Configuration

**Default Behavior**: Back up entire agent directory

**Configuration**:

```yaml
sources:
  - ~/.openclaw/* # Default - backs up entire OpenClaw directory
```

**Advanced Configuration**: Users can manually add specific paths

```yaml
sources:
  - ~/.openclaw/* # Entire agent directory
  - ~/graph-exports/* # Where pre-backup scripts export data
  - ~/vector-db/dumps/*.json # Specific files from vector DB
  - /opt/custom-agent-data/ # Additional data directory
```

**Path Types Supported**:

- Folders: `~/path/to/folder/` - backs up entire folder recursively
- Glob patterns: `~/path/*` - backs up all items matching pattern
- Specific files: `~/path/to/file.json` - backs up single file

**Purpose**: Allows users to:

- Add paths where backup scripts export data (so exports are versioned)
- Back up data from multiple locations
- Include specific files without entire parent directories

**Agent Detection**: On `init`, bulletproof detects OpenClaw installation:

- Checks `~/.openclaw` (default)
- Checks `/data/.openclaw`, `/openclaw`, `/app/.openclaw` (Docker)
- Validates by checking for `openclaw.json` file

If not found, prompts user for location.

---

### 1. Snapshot Management

#### 1.1 List Snapshots

**Command**: `bulletproof snapshots`

**Output**: Table showing short IDs, full snapshot IDs, timestamps, and file counts

**Example**:

```
ID   SNAPSHOT-ID      TIMESTAMP                FILES
0    (current)        -                        -
1    20250203-120000  2026-02-03T12:00:00Z     15
2    20250201-150000  2026-02-01T15:00:00Z     12
3    20250131-100000  2026-01-31T10:00:00Z     10
```

**Requirements**:

- Show both short IDs and full timestamp-based IDs
- Sort by timestamp descending (newest first)
- ID 0 reserved for current filesystem
- Machine-readable format (fixed-width or tab-separated)
- No decorative formatting (colors, emojis, relative times)

#### 1.2 Create Snapshot

**Command**: `bulletproof backup [options]`

**Options**:

- `--message <text>` or `-m <text>`: Optional description of changes
- `--dry-run`: Show what would be backed up without creating snapshot
- `--no-scripts`: Skip pre-backup script execution
- `--force`: Force backup creation even if no changes detected

**Behavior**:

1. Run pre-backup scripts (if configured)
2. Create snapshot of OpenClaw installation
3. Copy bulletproof configuration into snapshot
4. Copy custom scripts into snapshot
5. Include pre-backup script outputs in snapshot
6. Save to configured destination
7. Display summary of backed up files

**Example Output**:

```
Running pre-backup scripts...
  ✓ export-graph-memory (2.3s)
  ✓ export-vector-db (5.1s)

Creating snapshot 20250203-120000...
  15 files, 2.4 MB

Backup complete: 20250203-120000
```

**Skip Behavior**: If no files changed since last backup, skip creating duplicate snapshot and inform user.

#### 1.3 Restore Snapshot

**Command**: `bulletproof restore <snapshot-id> [options]`

**Options**:

- `--dry-run`: Show what would be restored without making changes
- `--no-scripts`: Skip post-restore script execution
- `--force`: Skip confirmation prompts (for untrusted script warnings)
- `--target <path>`: Restore to alternative location (default: OpenClaw installation path)

**Behavior**:

1. Create safety backup of current state before restoring
2. Restore all files from snapshot to target location
3. Run post-restore scripts (if configured)
4. Display summary of restored files

**Example Output**:

```
Creating safety backup: 20250203-120500

Restoring snapshot 20250201-150000...
  12 files, 1.8 MB

Running post-restore scripts...
  ✓ import-graph-memory (3.2s)
  ✓ import-vector-db (4.8s)

Restore complete. Safety backup saved as 20250203-120500
```

**Snapshot ID Formats Accepted**:

- Short ID: `bulletproof restore 2`
- Full ID: `bulletproof restore 20250201-150000`

---

### 2. Snapshot Comparison

#### 2.1 Diff Command

**Command**: `bulletproof diff [id1] [id2] [pattern]`

**Argument Behaviors**:

- **0 args**: `bulletproof diff` → Compare current filesystem vs last backup
- **1 arg**: `bulletproof diff <id>` → Compare current filesystem vs specified snapshot
- **2 args**: `bulletproof diff <id1> <id2>` → Compare two snapshots
- **3 args**: `bulletproof diff <id1> <id2> <pattern>` → Compare two snapshots, filtered by pattern

**ID Formats**:

- Short IDs: `bulletproof diff 0 3` (current vs snapshot 3)
- Full IDs: `bulletproof diff 20250201-150000 20250203-120000`
- Mixed: `bulletproof diff 2 20250131-100000`

**Pattern Matching** (optional 3rd argument):

Narrow the diff to specific files or folders to reduce noise when diagnosing attacks.

**Examples**:

```bash
# Specific file (no quotes needed)
bulletproof diff 5 3 SOUL.md

# Specific folder (no quotes needed)
bulletproof diff 5 3 skills/
bulletproof diff 5 3 workspace/memory/

# Glob patterns (MUST be quoted to prevent shell expansion)
bulletproof diff 5 3 '*.json'        # All JSON files
bulletproof diff 5 3 'skills/*.js'   # All JS files in skills folder
```

**Implementation Notes**:

- File/folder names can be passed directly (no quotes needed)
- Glob patterns (`*`, `**`, `?`) MUST be quoted to prevent shell expansion
- Pattern matching is case-sensitive
- Patterns are matched against relative paths from snapshot root

**Use Case**: Focus drift analysis on specific files/folders where attacks are suspected:

- `SOUL.md` - Personality definition (personality attacks)
- `skills/` - Skill definitions (skill weapons)
- `workspace/memory/` - Conversation logs (prompt injection)
- `openclaw.json`, `.bulletproof/config.yaml` - Configuration drift

**Output Format**: Standard unified diff (git diff style)

**Example Output**:

```
diff --git a/workspace/SOUL.md b/workspace/SOUL.md
index abc123..def456 100644
--- a/workspace/SOUL.md
+++ b/workspace/SOUL.md
@@ -1,5 +1,8 @@
 # Agent Personality

-I am helpful and concise.
+I am helpful, concise, and analytical.
+
+## Core Values
+- Accuracy over speed
- Transparency in reasoning

diff --git a/workspace/skills/analysis.js b/workspace/skills/analysis.js
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/workspace/skills/analysis.js
@@ -0,0 +1,10 @@
+function analyze(data) {
+  return data.map(x => x * 2);
+}
```

**Requirements**:

- Show actual file content changes (line-by-line diffs)
- Include `+++`/`---` headers for each file
- Include `@@` hunks with line numbers
- Format must be parseable by AI agents
- Work across both backup approaches (multi-folder and git)

#### 2.2 Workflow Examples

**Check what changed since last backup**:

```
bulletproof diff
```

**Compare current state to week-old snapshot**:

```
bulletproof snapshots
# Note: Snapshot 7 is from last week
bulletproof diff 7
```

**Analyze drift between two historical snapshots**:

```
bulletproof diff 10 5
# Shows how agent changed from snapshot 10 to snapshot 5
# Used by AI agents for drift diagnosis
```

---

### 3. Self-Contained Backups

#### 3.1 Backup Structure

Each snapshot must include:

```
<snapshot-id>/
├── workspace/               # Agent files
│   ├── SOUL.md
│   ├── AGENTS.md
│   ├── skills/
│   └── memory/
├── openclaw.json
├── .bulletproof/
│   ├── config.yaml         # Bulletproof configuration
│   ├── snapshot.json       # Snapshot metadata
│   └── scripts/            # Custom backup/restore scripts
│       ├── pre-backup/
│       │   └── export-graph.sh
│       └── post-restore/
│           └── import-graph.sh
└── _exports/               # Pre-backup script outputs
    └── graph-memory.dump
```

#### 3.2 Portability Requirements

**Scenario**: User backs up agent on Machine A, restores on Machine B

**Must work without manual intervention**:

1. Restore snapshot files ✓
2. Bulletproof config is included ✓
3. Custom scripts are included ✓
4. Run post-restore scripts to import external data ✓

**Result**: Agent fully operational on Machine B with same configuration, scripts, and external data.

#### 3.3 Configuration Inclusion

**Requirement**: Copy `~/.config/bulletproof/config.yaml` into each snapshot as `.bulletproof/config.yaml`

**Purpose**:

- Preserves backup settings with the backup
- Shows what was configured when snapshot was created
- Enables drift detection on configuration itself

#### 3.4 Script Inclusion

**Requirement**: Copy `~/.config/bulletproof/scripts/` directory into each snapshot as `.bulletproof/scripts/`

**Purpose**:

- Scripts travel with backup for portability
- Scripts are versioned (can detect script drift)
- Old backups can be restored even if scripts changed

---

### 4. Custom Backup/Restore Scripts

#### 4.1 Overview

Users can configure scripts that run automatically during backup and restore to handle external data sources (databases, vector stores, APIs).

#### 4.2 Configuration

**Location**: `~/.config/bulletproof/config.yaml`

**Schema**:

```yaml
scripts:
  pre_backup:
    - name: export-graph-memory
      command: /bin/bash ~/.config/bulletproof/scripts/pre-backup/export-graph.sh
      timeout: 60

    - name: export-vector-db
      command: python3 ~/.config/bulletproof/scripts/pre-backup/export-vectors.py
      timeout: 120

  post_restore:
    - name: import-graph-memory
      command: /bin/bash ~/.config/bulletproof/scripts/post-restore/import-graph.sh
      timeout: 60

    - name: import-vector-db
      command: python3 ~/.config/bulletproof/scripts/post-restore/import-vectors.py
      timeout: 120
```

#### 4.3 Template Variables

Scripts can reference these variables (substituted before execution):

| Variable          | Description                     | Example Value                               |
| ----------------- | ------------------------------- | ------------------------------------------- |
| `{exports_dir}`   | Path to \_exports/ directory    | `/tmp/bulletproof/20250203-120000/_exports` |
| `{backup_dir}`    | Path to snapshot being restored | `~/bulletproof-backups/20250203-120000`     |
| `{snapshot_id}`   | Snapshot ID                     | `20250203-120000`                           |
| `{openclaw_path}` | OpenClaw installation path      | `~/.openclaw`                               |

Also available as environment variables: `$EXPORTS_DIR`, `$BACKUP_DIR`, `$SNAPSHOT_ID`, `$OPENCLAW_PATH`

#### 4.4 Execution Lifecycle

**Pre-Backup** (during `bulletproof backup`):

1. Create `_exports/` directory
2. Set environment variables
3. Execute each script sequentially
4. If script fails: Log error, optionally abort backup
5. Include `_exports/` in snapshot

**Post-Restore** (during `bulletproof restore`):

1. Restore all files including `_exports/`
2. Set environment variables
3. Execute each script sequentially
4. If script fails: Log error but continue
5. Leave `_exports/` for debugging

#### 4.5 Timeout Handling

- Each script has configurable timeout (default: 60 seconds)
- If timeout exceeded: Kill process, log error
- Configurable abort vs. continue behavior

#### 4.6 Opt-Out Flags

- `--no-scripts`: Skip all script execution
- Use case: Restore files without importing external data

#### 4.7 Use Case Examples

**Neo4j Graph Database**:

Pre-backup script exports graph:

```bash
#!/bin/bash
neo4j-dump --db openclaw --output "$EXPORTS_DIR/graph.dump"
```

Post-restore script imports graph:

```bash
#!/bin/bash
neo4j-import --db openclaw --input "$BACKUP_DIR/_exports/graph.dump"
```

**Pinecone Vector Database**:

Pre-backup script exports embeddings:

```python
import os
from pinecone import Pinecone

pc = Pinecone(api_key=os.environ["PINECONE_API_KEY"])
index = pc.Index("openclaw-memory")
vectors = index.fetch(...)
with open(f"{os.environ['EXPORTS_DIR']}/vectors.json", "w") as f:
    json.dump(vectors, f)
```

#### 4.8 Security Considerations

**Untrusted Backups**:

When restoring backup from untrusted source, show warning before executing scripts:

```
⚠️  WARNING: This backup contains custom scripts

Scripts will be executed during restore:
  - import-graph.sh
  - import-vectors.py

Only restore backups from trusted sources.

Continue? (yes/no):
```

**Bypass Warning with --force**:

For automation or when user trusts the scripts:

```bash
bulletproof restore 5 --force
```

Skips confirmation prompt and executes scripts immediately.

**Script Review**:

Users can review scripts before restore using standard file tools:

```bash
# View all scripts in a backup
ls <backup-path>/.bulletproof/scripts/post-restore/

# Read a specific script
cat <backup-path>/.bulletproof/scripts/post-restore/import-data.sh
```

**Credential Management**:

Scripts should read credentials from environment variables, not hardcode in config.

#### 4.9 Script Drift Detection

**Requirement**: Scripts themselves can drift over time

**Detection**: Use regular diff command on `.bulletproof/scripts/`:

```
bulletproof diff 1 10
```

Output includes script changes:

```
diff --git a/.bulletproof/scripts/pre-backup/export-graph.sh b/.bulletproof/scripts/pre-backup/export-graph.sh
--- a/.bulletproof/scripts/pre-backup/export-graph.sh
+++ b/.bulletproof/scripts/pre-backup/export-graph.sh
@@ -1,3 +1,5 @@
 #!/bin/bash
-neo4j-dump --db openclaw --output "$EXPORTS_DIR/graph.dump"
+# Now exporting with compression
+neo4j-dump --db openclaw --output "$EXPORTS_DIR/graph.dump.gz" --compress
```

---

### 5. Skill Command

#### 5.1 Purpose

Teach AI agents (or humans) how to effectively use bulletproof for:

- Drift diagnosis using binary search
- Custom data source integration
- Platform migration
- Platform-specific service setup

**Critical Distinction**: The skill teaches methodology; agents execute it using their own intelligence or via another agent known to be uncompromised.

**The Architecture**:

- **Tool = Data Provider**: CLI returns structured diffs, snapshot metadata, file comparisons
- **Agent = Intelligence Provider**: Agent uses its native LLM to read diffs, identify suspicious patterns, reason about attacks
- **Skill = Methodology Teacher**: Guide trains agents in binary search, forensic analysis, and drift detection workflows

Agents internalize the methodology and apply their own AI capabilities to perform analysis. No API keys needed. No external AI service. Just agent-native intelligence applied through taught methodology.

#### 5.2 Command

**Syntax**: `bulletproof skill`

**Output**: Comprehensive markdown-style guide printed to stdout (~500-1000 lines)

#### 5.3 Content Topics

1. **Drift Diagnosis via Binary Search**
   - Concept: Find exact snapshot where drift occurred
   - Workflow: List snapshots, compare ranges, narrow down
   - Example scenario: "Agent went from helpful to harmful"
   - Tools used: `snapshots` + `diff` commands

2. **Script Drift Detection**
   - Concept: Scripts can drift too
   - Workflow: Use `diff` on `.bulletproof/scripts/`
   - Impact: Changed export script may not restore old backups

3. **Custom Data Source Integration**
   - Configuration: Set up pre-backup and post-restore scripts
   - Examples: Neo4j export/import, Pinecone sync
   - Template variables and environment setup

4. **Platform Migration**
   - Workflow: Backup on Machine A, restore on Machine B
   - Self-contained backups include config and scripts

5. **Platform-Specific Service Management**
   - Linux: systemd or cron for scheduled backups
   - macOS: launchd for scheduled backups
   - Windows: Task Scheduler for scheduled backups

6. **Basic Operations**
   - Using snapshots command effectively
   - Understanding short IDs vs full IDs
   - Diff command usage patterns
   - Restore workflows

#### 5.4 Binary Search Tutorial

**Must include complete step-by-step tutorial**:

**Scenario**: Agent's personality drifted from helpful to aggressive. Find the exact snapshot where it changed.

**Why Binary Search**: With 100 snapshots, checking each one requires 99 diffs. Binary search finds it in ~7 diffs.

**Step 1: Identify Range**

```
bulletproof snapshots
```

See 50 snapshots. Know:

- Snapshot 50 (oldest): Agent was helpful ✓
- Snapshot 1 (newest): Agent is aggressive ✗

Drift happened between snapshots 1 and 50.

**Step 2: Test Midpoint**

```
bulletproof diff 50 25
```

Review output. Key files to check:

- `workspace/SOUL.md` - personality changes
- `workspace/memory/` - conversation logs (prompt injection?)
- `workspace/skills/` - new skills affecting behavior

If snapshot 25 is helpful → drift is in range 1-25
If snapshot 25 is aggressive → drift is in range 25-50

**Step 3: Repeat Until Found**

Continue halving range:

```
bulletproof diff 50 37
bulletproof diff 37 31
bulletproof diff 31 28
bulletproof diff 29 28
```

**Step 4: Analyze Culprit**

Once found (e.g., snapshot 28):

```
bulletproof diff 29 28
```

Shows exactly what changed between last good (29) and first bad (28).

Look for:

- Prompt injection in conversation logs
- Personality changes in SOUL.md
- New skills overriding safety
- Config changes
- Script changes

**Step 5: Remediate**

Option A - Restore to last good:

```
bulletproof restore 29
```

Option B - Fix specific issue:

- Remove injected prompt
- Revert SOUL.md
- Remove problematic skill

Then create new backup:

```
bulletproof backup
```

#### 5.5 Init Command Integration

**Requirement**: `bulletproof init` output mentions skill command

**Example**:

```
✅ Bulletproof backup initialized!

Configuration: ~/.config/bulletproof/config.yaml
Destination: ~/bulletproof-backups

Next steps:
  bulletproof backup          Create your first backup
  bulletproof skill           Learn drift diagnosis and advanced usage
  bulletproof --help          See all commands
```

---

### 6. Analytics Tracking

#### 6.1 Purpose

Understand how bulletproof is used to guide product improvements.

**Privacy-First**: No PII, no data, no file paths. Anonymous usage patterns only.

#### 6.2 What Is Tracked

**Allowed**:

- Command executed (`backup`, `restore`, `diff`, etc.)
- Subcommand (`config show`, `analytics disable`)
- OS type (`darwin`, `linux`, `windows`)
- CLI version (`1.0.0`)
- Boolean flags (`--dry-run`, `--no-scripts`)
- Anonymous user ID (UUID)
- Timestamp

**Prohibited**:

- File paths
- Snapshot IDs
- User-provided messages
- Configuration values
- Error messages
- Any user data or agent data

#### 6.3 Anonymous User ID

**Generation**: UUID created on first run

**Storage**: `~/.config/bulletproof/config.yaml`

```yaml
analytics:
  enabled: true
  user_id: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  notice_shown: true
```

**Purpose**: Track usage patterns (e.g., "users who run backup daily") without identifying individuals

**Cannot be linked to identity**: UUID is random, stored locally, no personal info collected

#### 6.4 Default Behavior

**On by default**: Analytics enabled out of the box
**Transparent**: Clear notice on first run
**Easy opt-out**: Single command to disable

#### 6.5 First-Run Notice

**When**: First command execution (any command, not just init)

**Display**:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Usage Analytics

Bulletproof collects anonymous usage analytics to help
improve the tool. We track:
  ✓ Which commands you run (e.g., backup, restore)
  ✓ Basic system info (OS type)

We DO NOT track:
  ✗ Personal information
  ✗ File paths or data
  ✗ Snapshot contents

Your privacy is important. To opt out:
  bulletproof analytics disable

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

After showing once, set `analytics.notice_shown = true` in config.

#### 6.6 Analytics Commands

**Disable tracking**:

```
bulletproof analytics disable
```

**Enable tracking**:

```
bulletproof analytics enable
```

**Check status**:

```
bulletproof analytics status
```

Output: `Analytics: enabled` or `Analytics: disabled`

#### 6.7 Non-Blocking Execution

**Requirements**:

- Analytics failures must NEVER block command execution
- Send events asynchronously in background
- Short timeout (2 seconds max) for API calls
- Silent failures (don't show errors to user)

#### 6.8 Service

**Plausible Analytics** (plausible.io)

- Privacy-focused, GDPR compliant
- No cookies, no personal data
- Simple Events API
- Can be self-hosted if needed

**Event Format**:

```json
{
  "name": "pageview",
  "url": "https://bulletproof.bot/cmd/backup",
  "domain": "bulletproof.bot",
  "props": {
    "command": "backup",
    "dry_run": "false",
    "os": "darwin",
    "version": "1.0.0"
  }
}
```

#### 6.9 Privacy Verification

**Test Scenario**: Create backup with sensitive data

- File path: `/Users/alice/Documents/secret-project/`
- Snapshot ID: `20250203-120000`
- User message: "Added credit card validation"

**Expected Event**:

```json
{
  "command": "backup",
  "dry_run": "false",
  "os": "darwin",
  "version": "1.0.0"
}
```

**Must NOT include**:

- `/Users/alice` (file path)
- `20250203-120000` (snapshot ID)
- "Added credit card validation" (user message)

---

### 7. Error Message Guidelines

#### 7.1 Purpose

All error messages must enable agents and humans to diagnose and fix problems independently. Errors should be actionable, not just descriptive.

#### 7.2 Error Message Format

**Required Components**:

1. **What failed**: Clear description of the operation that failed
2. **Why it failed**: Root cause when known
3. **How to fix**: Specific remediation steps

**Format Template**:

```
failed to <operation>: <root cause>.

This usually means:
- <likely reason 1>
- <likely reason 2>

Try:
<specific command or action>

Related: <helpful related command>
```

#### 7.3 Examples

**Bad** (current pattern):

```
Error: failed to create snapshot: permission denied
```

Agent must infer: What file? What permissions? How to fix?

**Good** (actionable pattern):

```
Error: failed to create snapshot: permission denied on ~/.openclaw/workspace/SOUL.md

This usually means:
- File permissions too restrictive
- Directory owned by different user
- Parent directory not writable

Try:
chmod -R u+r ~/.openclaw

Or run as correct user:
sudo -u openclaw-user bulletproof backup

Related: bulletproof config show
```

**Another Example**:

**Bad**:

```
Error: snapshot not found: 20250203-120000
```

**Good**:

```
Error: snapshot not found: 20250203-120000

Available snapshots:
1. 20250203-115500
2. 20250201-150000
3. 20250131-100000

Try:
bulletproof snapshots               # List all snapshots
bulletproof restore 1               # Restore latest snapshot
bulletproof restore 20250203-115500 # Use correct snapshot ID
```

#### 7.4 Implementation Requirements

- All `fmt.Errorf` calls should provide context
- File paths should be included when relevant
- Suggest specific fix commands, not generic advice
- Show available options when user provides invalid input
- Include related commands that might help diagnose

---

### 8. Distribution

**Method**: GitHub Releases at [github.com/bulletproof-bot/backup](https://github.com/bulletproof-bot/backup)

**Release Artifacts**:

- `bulletproof_VERSION_linux_amd64.tar.gz`
- `bulletproof_VERSION_darwin_amd64.tar.gz` (macOS Intel)
- `bulletproof_VERSION_darwin_arm64.tar.gz` (macOS Apple Silicon)
- `bulletproof_VERSION_windows_amd64.zip`
- `checksums.txt`

**Installation**:

Each release includes installation instructions. Example:

```bash
# macOS (Apple Silicon)
curl -L https://github.com/bulletproof-bot/backup/releases/download/v1.0.0/bulletproof_v1.0.0_darwin_arm64.tar.gz | tar xz
sudo mv bulletproof /usr/local/bin/

# Linux
curl -L https://github.com/bulletproof-bot/backup/releases/download/v1.0.0/bulletproof_v1.0.0_linux_amd64.tar.gz | tar xz
sudo mv bulletproof /usr/local/bin/

# Windows
# Download the .zip file and extract bulletproof.exe to your PATH
```

**Version Management**:

- Version embedded at build time via `-ldflags`
- `bulletproof version` shows current version
- Automatic update checking (non-blocking, opt-in notification)

**CI/CD**:

- GitHub Actions builds on tag push (`v*`)
- GoReleaser creates multi-platform binaries
- Automated changelog generation from commit messages

---

### 9. Future Work

Features deferred to future releases:

#### 9.1 Visual Diff UI

**Goal**: Provide visual diff tool for human users

**Features**:

- Side-by-side file comparison
- Syntax highlighting
- Navigate between changed files
- Filter by file type or path
- Integration with existing diff tools (VS Code, meld, etc.)

**Why Deferred**: CLI is sufficient for AI agents (primary users). Visual UI adds complexity without immediate value for the target audience.

#### 9.2 Retention and Cleanup Policies

**Goal**: Automatic snapshot retention and storage management

**Features**:

- Retention policies (keep last N snapshots, keep snapshots older than X days)
- Automatic cleanup of old snapshots
- Storage usage monitoring and alerts
- Compression of old snapshots
- Deduplication across snapshots

**Why Deferred**: Users can manually delete old snapshots. Automatic retention requires careful design to avoid data loss.

#### 9.3 Multi-Agent Support

**Goal**: Enable multiple agents to safely share backup destinations

**Features**:

- Per-agent namespacing in shared destinations
- Agent discovery and listing
- Cross-agent diff (compare different agents' snapshots)
- Shared configuration with agent-specific overrides

**Why Deferred**: Current design assumes single agent per backup destination. Multi-agent requires rethinking snapshot ID format and directory structure.

#### 9.4 Snapshot Compression

**Goal**: Reduce storage requirements for large backups

**Features**:

- Automatic compression of snapshot content
- Configurable compression algorithms (gzip, zstd, etc.)
- Transparent decompression on restore
- Backward compatibility with uncompressed snapshots

**Why Deferred**: Current file sizes are manageable for typical agent installations. Compression adds complexity.

#### 9.5 Incremental Backups

**Goal**: Only store changed files between snapshots

**Features**:

- Delta storage (only store file diffs)
- Reconstruct full snapshot on demand
- Significant storage savings for large agent installations
- Maintain fast restore times

**Why Deferred**: Adds significant complexity to snapshot format and restore logic. Full snapshots are simpler and more reliable.

---

## Configuration

### Complete Configuration Schema

**Location**: `~/.config/bulletproof/config.yaml`

**Structure**:

```yaml
# Source paths to back up
sources:
  - ~/.openclaw/* # Default - entire agent directory

# Backup destination
destination:
  path: ~/bulletproof-backups # Multi-folder: any local path or cloud sync folder
  # OR
  # path: ~/bulletproof-repo   # Git: must be initialized git repository

# Exclusion patterns
exclude:
  - "*.log"
  - "node_modules/"
  - ".git/"
  - "*.tmp"

# Custom scripts
scripts:
  pre_backup:
    - name: export-graph-memory
      command: /bin/bash ~/.config/bulletproof/scripts/pre-backup/export-graph.sh
      timeout: 60

  post_restore:
    - name: import-graph-memory
      command: /bin/bash ~/.config/bulletproof/scripts/post-restore/import-graph.sh
      timeout: 60

# Analytics
analytics:
  enabled: true
  user_id: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  notice_shown: true
```

### Advanced Sources Configuration

**Adding custom paths**:

```yaml
sources:
  - ~/.openclaw/* # Entire agent directory
  - ~/graph-exports/* # Backup script export location
  - ~/vector-db/embeddings/*.json # Vector database dumps
  - /opt/custom-data/agent-state.db # Specific database file
```

**Path types**:

- `~/path/to/folder/*` - Glob pattern, backs up all matching items
- `~/path/to/folder/` - Directory, backs up recursively
- `~/path/to/file.txt` - Specific file

### Destination Type Details

**Multi-Folder Destination**:

```yaml
destination:
  path: ~/bulletproof-backups
```

Creates timestamped subdirectories:

- `~/bulletproof-backups/20250203-120000/`
- `~/bulletproof-backups/20250201-150000/`
- etc.

Works with any path:

- Local disk: `~/bulletproof-backups`
- Cloud sync: `~/Dropbox/bulletproof-backups` or `~/Google Drive/bulletproof-backups`
- Network share: `/mnt/nas/bulletproof-backups`

**Git Repository Destination**:

```yaml
destination:
  path: ~/bulletproof-repo # Must be a git repository
```

Setup:

```bash
mkdir ~/bulletproof-repo
cd ~/bulletproof-repo
git init
git remote add origin git@github.com:user/backups.git  # Optional
```

Git operations:

- Commits for each snapshot
- Tags with snapshot IDs (e.g., `20250203-120000`)
- Automatic push to remote if git remote is configured
- Storage efficient: unchanged files aren't duplicated

---

## Success Criteria

### User Experience

- Short IDs make snapshot comparison intuitive
- Diff output is readable by both humans and AI agents
- Skill command provides clear, actionable guidance
- First-run analytics notice is transparent and non-intrusive
- Opt-out is a single command with immediate effect

### Technical Correctness

- Short ID resolution handles edge cases (empty history, ID 0, out of range)
- Unified diff output matches standard git diff format
- Scripts execute with correct environment variables and timeouts
- Config and scripts are copied into every backup
- Analytics events contain only allowed fields (no PII)

### Reliability

- Script failures don't abort backup/restore (logged but continue)
- Analytics failures never block command execution
- Diff works across both backup approaches (multi-folder and git)
- Self-contained backups restore correctly on different machines

### Security

- Warning shown before executing scripts from untrusted backups
- `--no-scripts` flag available to skip script execution
- Template variable substitution prevents command injection
- Anonymous user ID cannot be linked to individual identity

### Portability

- Backup created on Machine A restores fully on Machine B
- No manual configuration required on new machine
- Scripts and config travel with backup

---

## Testing Requirements

### Unit Tests

**Short ID Resolution**:

- Test short ID → full ID mapping
- Test ID 0 (current filesystem)
- Test out-of-range IDs
- Test full IDs pass through unchanged

**Unified Diff Generation**:

- Test file modifications show line-by-line diffs
- Test added files show full content
- Test removed files show full content
- Test output matches git diff format

**Script Template Variables**:

- Test variable substitution works correctly
- Test environment variables are set
- Test shell escaping prevents injection

**Analytics Event Payload**:

- Test event structure is correct
- Test no PII in events
- Test boolean flags serialized correctly

### Integration Tests

**Enhanced Diff Workflow**:

1. Create multiple snapshots with changes
2. Test `diff` with 0, 1, 2 arguments
3. Verify output format
4. Test short IDs and full IDs work

**Self-Contained Backup**:

1. Create backup with config and scripts
2. Verify backup structure
3. Verify config copied correctly
4. Verify scripts copied correctly
5. Verify `_exports/` included

**Custom Scripts Execution**:

1. Configure pre-backup script
2. Run backup
3. Verify script executed
4. Verify output in `_exports/`
5. Test restore with post-restore script

**Analytics Opt-Out**:

1. Verify enabled by default
2. Test disable command
3. Verify no events sent when disabled
4. Test enable command

### Privacy Tests

**No PII in Analytics Events**:

- Mock analytics endpoint
- Capture events
- Verify no file paths, snapshot IDs, messages, or config values

**Analytics Non-Blocking**:

- Simulate network failure
- Verify all commands complete successfully

---

## Edge Cases & Error Handling

This section documents edge cases, error scenarios, and how the system handles them. These scenarios ensure robust operation in real-world conditions.

### Edge Case 1: Restoring Old Backups with New Scripts

**Scenario**: Backup created 3 months ago. Scripts have since changed. Will restore work?

**Solution**: Each snapshot includes the scripts that existed at backup time in `.bulletproof/scripts/`. When restoring snapshot 50, Bulletproof uses the post-restore scripts from snapshot 50, ensuring compatibility with the data format used then.

**Fallback**: If you want to use current scripts (perhaps they're fixed), use:

```bash
bulletproof restore 50 --no-scripts  # Restore files only
# Then manually run current scripts if needed
```

### Edge Case 2: Untrusted Backup Sources

**Scenario**: Restoring a backup received from another user or downloaded from the internet. The backup could contain malicious post-restore scripts.

**Solution**: Before executing any scripts, Bulletproof displays a warning:

```
⚠️  WARNING: This backup contains custom scripts

Scripts that will execute during restore:
  - import-graph.sh
  - import-vectors.py

Only restore backups from trusted sources.

Review scripts with:
  cat <backup-path>/.bulletproof/scripts/post-restore/*

Continue? (yes/no):
```

User can:

1. Type "no" and abort
2. Review scripts first by examining `.bulletproof/scripts/` in the backup
3. Restore without scripts: `bulletproof restore 5 --no-scripts`
4. Trust and continue: type "yes"
5. Skip prompt for automation: `bulletproof restore 5 --force`

### Edge Case 3: Empty Snapshot History

**Scenario**: First-time user runs `bulletproof diff 5 3` before creating any backups.

**Solution**:

```
Error: no snapshots available

You haven't created any backups yet.

Try:
bulletproof backup               # Create your first backup
bulletproof --help               # See all commands
bulletproof skill                # Learn advanced usage
```

### Edge Case 4: Out-of-Range Short IDs

**Scenario**: User has 5 snapshots but tries `bulletproof restore 10`.

**Solution**:

```
Error: snapshot ID 10 not found

Available snapshots (IDs 1-5):
1. 20250203-120000
2. 20250202-180000
3. 20250201-180000
4. 20250131-100000
5. 20250130-100000

Try:
bulletproof snapshots      # List all available snapshots
bulletproof restore 1      # Restore most recent snapshot
```

### Edge Case 5: ID 0 (Current State)

**Scenario**: User runs `bulletproof diff 0 5`.

**Behavior**: ID 0 always represents the current filesystem state (not a snapshot). This enables comparing current changes against any historical snapshot without creating a new backup first.

**Example output**:

```
diff --git a/workspace/SOUL.md b/workspace/SOUL.md
--- a/snapshot-5/workspace/SOUL.md (from 20250201-180000)
+++ b/current/workspace/SOUL.md
@@ -1,3 +1,5 @@
 # Agent Personality

-I am helpful and concise.
+I am helpful, concise, and analytical.
+
+## New section added today
```

### Edge Case 6: Script Timeout During Backup

**Scenario**: Pre-backup script hangs indefinitely (perhaps database is locked).

**Solution**: After configured timeout (default 60s):

1. Kill the script process
2. Log the timeout with stderr output
3. Continue with backup (files still get backed up)
4. Display warning in output:

```
Running pre-backup scripts...
  ✗ export-graph-memory (timeout after 60s)
    Last output: "Waiting for database lock..."
  ✓ export-vectors (5.1s)

⚠️  Warning: Script timeout occurred. Backup created but may be incomplete.

Review logs: bulletproof config path
Consider increasing timeout in config.yaml
```

### Edge Case 7: No Changes Since Last Backup

**Scenario**: User runs `bulletproof backup` but no files have changed.

**Solution**: Skip creating duplicate snapshot:

```
Checking for changes since last backup...

No changes detected since snapshot 20250203-120000.

Backup skipped. Use --force to create backup anyway.
```

This prevents snapshot bloat while allowing forced backups when needed for other reasons (e.g., scheduled backup that includes external data from scripts).

### Edge Case 8: Pattern Matching with Spaces

**Scenario**: User wants to diff a file with spaces in the name.

**Solution**: Patterns are matched after path normalization:

```bash
# File: "workspace/my resume.md"

# Works:
bulletproof diff 5 3 "workspace/my resume.md"

# Also works:
bulletproof diff 5 3 'workspace/my resume.md'

# Glob also works:
bulletproof diff 5 3 'workspace/*.md'
```

---

## Appendix A: Snapshot ID Format

**Format**: `yyyyMMdd-HHmmss`

**Examples**:

- `20250203-120000` = February 3, 2025 at 12:00:00
- `20250131-235959` = January 31, 2025 at 23:59:59

**Properties**:

- Lexicographically sortable (string sort = chronological sort)
- Human-readable (no UUIDs or hashes)
- Collision-resistant (1-second granularity)
- Used as: Git tags, folder names, index keys

---

## Appendix B: Plausible Events API

**Endpoint**: `POST https://plausible.io/api/event`

**Headers**:

```
User-Agent: bulletproof/<version>
Content-Type: application/json
```

**Payload**:

```json
{
  "name": "pageview",
  "url": "https://bulletproof.bot/cmd/<command>",
  "domain": "bulletproof.bot",
  "props": {
    "command": "backup",
    "subcommand": "",
    "dry_run": "false",
    "os": "darwin",
    "version": "1.0.0"
  }
}
```

**Error Handling**: Silent failures (don't block execution)

---

## Appendix C: Skill Command Output

**Command**: `bulletproof skill`

**Output** (complete text shown to agents):

````markdown
# Bulletproof Backup: Agent Skill Guide

This guide teaches AI agents how to use bulletproof effectively for drift diagnosis,
custom data source integration, and platform migration.

---

## Table of Contents

1. Drift Diagnosis via Binary Search
2. Script Drift Detection
3. Custom Data Source Integration
4. Platform Migration
5. Platform-Specific Service Setup
6. Basic Operations Reference

---

## 1. Drift Diagnosis via Binary Search

### Overview

When an agent's behavior changes unexpectedly (drift), use binary search to efficiently
find the exact snapshot where the change occurred.

### Scenario

Your agent has drifted from helpful to aggressive. With 100 snapshots, checking each one
would require 99 diffs. Binary search finds the drift point in ~7 diffs (log2(100)).

### Step-by-Step Process

#### Step 1: Identify the Range

List all available snapshots:

```bash
bulletproof snapshots
```
````

Output shows:

```
ID   SNAPSHOT-ID      TIMESTAMP                FILES
0    (current)        -                        -
1    20250203-120000  2026-02-03T12:00:00Z     15
2    20250201-150000  2026-02-01T15:00:00Z     12
...
50   20250101-080000  2026-01-01T08:00:00Z     8
```

Determine your range:

- Snapshot 50 (oldest): Agent was helpful ✓
- Snapshot 1 (newest): Agent is aggressive ✗
- Drift occurred somewhere between 1 and 50

#### Step 2: Test the Midpoint

Compare the midpoint (snapshot 25) to the known-good baseline (snapshot 50):

```bash
bulletproof diff 50 25
```

Review the unified diff output. Key files to examine:

- `workspace/SOUL.md` - personality definition changes
- `workspace/memory/` - conversation logs (check for prompt injection)
- `workspace/skills/` - new skills that could affect behavior
- `.bulletproof/config.yaml` - configuration changes
- `.bulletproof/scripts/` - backup/restore script changes

#### Step 3: Narrow the Range

Based on Step 2 results:

- If snapshot 25 is still helpful → drift is in range 1-25 (more recent half)
- If snapshot 25 is aggressive → drift is in range 25-50 (older half)

Continue halving the range. Example iterations:

```bash
# Iteration 1: Test midpoint of 1-50
bulletproof diff 50 25
# Result: Snapshot 25 is aggressive
# New range: 25-50

# Iteration 2: Test midpoint of 25-50
bulletproof diff 50 37
# Result: Snapshot 37 is helpful
# New range: 25-37

# Iteration 3: Test midpoint of 25-37
bulletproof diff 37 31
# Result: Snapshot 31 is helpful
# New range: 25-31

# Iteration 4: Test midpoint of 25-31
bulletproof diff 31 28
# Result: Snapshot 28 is aggressive
# New range: 28-31

# Iteration 5: Test midpoint of 28-31
bulletproof diff 31 29
# Result: Snapshot 29 is helpful
# New range: 28-29

# Iteration 6: Final comparison
bulletproof diff 29 28
# Result: Found the drift point! Snapshot 28 is where drift occurred
```

#### Step 4: Analyze the Culprit

Once identified (e.g., snapshot 28), examine exactly what changed:

```bash
bulletproof diff 29 28
```

This shows the exact changes between the last good (29) and first bad (28) snapshot.

**What to look for**:

- **Prompt injection**: Check `workspace/memory/` for malicious inputs in conversation logs
- **Personality changes**: Look at `workspace/SOUL.md` for modifications to core values
- **New skills**: Check `workspace/skills/` for recently added capabilities
- **Config drift**: Examine `.bulletproof/config.yaml` for changed settings
- **Script drift**: Review `.bulletproof/scripts/` for modified export/import logic

#### Step 5: Remediate

**Option A - Restore to Last Good State**:

```bash
bulletproof restore 29
```

This reverts the agent completely to snapshot 29 (last known good state).

**Option B - Fix the Specific Issue**:

1. Identify the problematic change from the diff
2. Manually fix it:
   - Remove injected prompts from memory
   - Revert SOUL.md changes
   - Delete problematic skills
   - Restore correct configuration
3. Create a new backup with the fix:

```bash
bulletproof backup
```

#### Step 6: Prevent Recurrence

Based on the root cause:

- **Prompt injection**: Improve input validation in agent code
- **Skill issue**: Implement skill approval process before installation
- **Config drift**: Lock down config file permissions
- **Script drift**: Review script changes before deployment

Document the incident for future reference and to train on similar patterns.

---

## 2. Script Drift Detection

### Overview

Backup and restore scripts can drift over time, just like agent code. Changed scripts
can cause problems:

- Export script modified → exports different data
- Import script changed → can't restore old backups
- New scripts added → new data sources being backed up

### Detection Method

Use the regular `diff` command to compare scripts between snapshots:

```bash
bulletproof diff 10 5
```

The output will include any script changes:

```diff
diff --git a/.bulletproof/scripts/pre-backup/export-graph.sh b/.bulletproof/scripts/pre-backup/export-graph.sh
--- a/.bulletproof/scripts/pre-backup/export-graph.sh
+++ b/.bulletproof/scripts/pre-backup/export-graph.sh
@@ -1,3 +1,5 @@
 #!/bin/bash
-neo4j-dump --db openclaw --output "$EXPORTS_DIR/graph.dump"
+# Now exporting with compression
+neo4j-dump --db openclaw --output "$EXPORTS_DIR/graph.dump.gz" --compress
```

### Implications

**This change means**:

- Old backups exported uncompressed graph dumps
- New backups export compressed dumps (.gz)
- Import script may need updating to handle both formats
- Restoring old backup with new import script might fail

### Best Practices

1. **Version scripts carefully**: Treat scripts like code, review changes
2. **Test compatibility**: Ensure new import scripts can handle old export formats
3. **Document changes**: Use backup messages to note script updates
4. **Gradual rollout**: Test new scripts on non-production agents first

---

## 3. Custom Data Source Integration

### Overview

Agents often use external data sources (databases, vector stores, APIs) that need to be
backed up alongside files. Use custom scripts to export/import this data.

### Configuration

Location: `~/.config/bulletproof/config.yaml`

```yaml
scripts:
  pre_backup:
    - name: export-graph-memory
      command: /bin/bash ~/.config/bulletproof/scripts/pre-backup/export-graph.sh
      timeout: 60

  post_restore:
    - name: import-graph-memory
      command: /bin/bash ~/.config/bulletproof/scripts/post-restore/import-graph.sh
      timeout: 60
```

### Template Variables

Scripts can use these variables:

- `{exports_dir}` - Path to \_exports/ directory in snapshot
- `{backup_dir}` - Path to snapshot being restored
- `{snapshot_id}` - Snapshot ID (e.g., 20250203-120000)
- `{openclaw_path}` - Path to agent installation

Also available as environment variables:

- `$EXPORTS_DIR`
- `$BACKUP_DIR`
- `$SNAPSHOT_ID`
- `$OPENCLAW_PATH`

### Example: Neo4j Graph Database

**Pre-backup script** (`~/.config/bulletproof/scripts/pre-backup/export-graph.sh`):

```bash
#!/bin/bash
set -e

echo "Exporting graph database..."
neo4j-dump --db openclaw --output "$EXPORTS_DIR/graph.dump"
echo "Graph exported to $EXPORTS_DIR/graph.dump"
```

**Post-restore script** (`~/.config/bulletproof/scripts/post-restore/import-graph.sh`):

```bash
#!/bin/bash
set -e

echo "Importing graph database..."
neo4j-import --db openclaw --input "$BACKUP_DIR/_exports/graph.dump"
echo "Graph imported from $BACKUP_DIR/_exports/graph.dump"
```

### Example: Pinecone Vector Database

**Pre-backup script** (Python):

```python
#!/usr/bin/env python3
import os
import json
from pinecone import Pinecone

pc = Pinecone(api_key=os.environ["PINECONE_API_KEY"])
index = pc.Index("openclaw-memory")

# Fetch all vectors
stats = index.describe_index_stats()
vector_count = stats["total_vector_count"]

# Export to file
vectors = index.fetch(ids=list(range(vector_count)))
output_path = os.path.join(os.environ["EXPORTS_DIR"], "vectors.json")

with open(output_path, "w") as f:
    json.dump(vectors, f)

print(f"Exported {vector_count} vectors to {output_path}")
```

### Workflow

1. **Create scripts**: Write export and import scripts for your data source
2. **Add to config**: Register scripts in `config.yaml`
3. **Test locally**: Run backup and verify exports are created
4. **Test restore**: Restore to different location and verify imports work
5. **Add to sources**: Optionally add export directory to `sources` in config

---

## 4. Platform Migration

### Overview

Move an agent from one machine to another using self-contained backups.

### Scenario

You have an agent on Machine A (local laptop) and want to move it to Machine B (cloud server).

### Step-by-Step Process

#### On Machine A (Source):

1. Ensure backup is up to date:

```bash
bulletproof backup
```

2. If using local storage, copy backup folder to Machine B:

```bash
scp -r ~/bulletproof-backups/ user@machine-b:~/bulletproof-backups/
```

3. If using cloud sync folder (Dropbox, Google Drive), backup is already synced.

4. If using git repository with remote, backup is already pushed.

#### On Machine B (Target):

1. Install bulletproof CLI

2. Initialize from backup:

```bash
bulletproof init --from-backup ~/bulletproof-backups/20250203-120000
```

Or for cloud sync folder:

```bash
bulletproof init --from-backup ~/Dropbox/bulletproof-backups/20250203-120000
```

3. Verify configuration:

```bash
bulletproof config show
```

4. Restore the latest snapshot:

```bash
bulletproof snapshots
bulletproof restore 1
```

5. Verify agent is operational

### What Gets Migrated

✅ Agent files (workspace/, SOUL.md, etc.)
✅ Bulletproof configuration
✅ Backup/restore scripts
✅ External data (via post-restore scripts)

### Notes

- Custom scripts may need platform-specific adjustments (e.g., different database paths)
- API keys and credentials should be set via environment variables on new machine
- Test restore on new machine before decommissioning old one

---

## 5. Platform-Specific Service Setup

### Overview

Set up scheduled automatic backups on each platform.

### Linux (systemd or cron)

**Option A - systemd timer**:

Create service file: `/etc/systemd/system/bulletproof-backup.service`

```ini
[Unit]
Description=Bulletproof Backup

[Service]
Type=oneshot
User=your-username
ExecStart=/usr/local/bin/bulletproof backup
```

Create timer file: `/etc/systemd/system/bulletproof-backup.timer`

```ini
[Unit]
Description=Daily bulletproof backup

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

Enable and start:

```bash
sudo systemctl enable bulletproof-backup.timer
sudo systemctl start bulletproof-backup.timer
```

**Option B - cron**:

Edit crontab:

```bash
crontab -e
```

Add entry (daily at 2 AM):

```
0 2 * * * /usr/local/bin/bulletproof backup --quiet
```

### macOS (launchd)

Create plist file: `~/Library/LaunchAgents/ai.bulletproof.backup.plist`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>ai.bulletproof.backup</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/bulletproof</string>
        <string>backup</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>2</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
</dict>
</plist>
```

Load the agent:

```bash
launchctl load ~/Library/LaunchAgents/ai.bulletproof.backup.plist
```

### Windows (Task Scheduler)

Create scheduled task:

```powershell
$action = New-ScheduledTaskAction -Execute "bulletproof.exe" -Argument "backup"
$trigger = New-ScheduledTaskTrigger -Daily -At "2:00AM"
$principal = New-ScheduledTaskPrincipal -UserId "$env:USERNAME" -RunLevel Highest
Register-ScheduledTask -TaskName "BulletproofBackup" -Action $action -Trigger $trigger -Principal $principal
```

Or use Task Scheduler GUI:

1. Open Task Scheduler
2. Create Basic Task: "Bulletproof Backup"
3. Trigger: Daily at 2:00 AM
4. Action: Start program `bulletproof.exe backup`

---

## 6. Basic Operations Reference

### List Snapshots

```bash
bulletproof snapshots
```

Shows all available snapshots with short IDs for easy reference.

### Create Backup

```bash
# Simple backup
bulletproof backup

# With message
bulletproof backup

# Dry run (see what would be backed up)
bulletproof backup --dry-run

# Skip custom scripts
bulletproof backup --no-scripts
```

### Compare Snapshots

```bash
# Compare current state vs last backup
bulletproof diff

# Compare current vs specific snapshot
bulletproof diff 5

# Compare two historical snapshots
bulletproof diff 10 5

# Use full snapshot IDs
bulletproof diff 20250101-120000 20250203-150000
```

### Restore Snapshot

```bash
# Restore using short ID
bulletproof restore 5

# Restore using full ID
bulletproof restore 20250203-120000

# Dry run (see what would be restored)
bulletproof restore 5 --dry-run

# Skip post-restore scripts
bulletproof restore 5 --no-scripts

# Skip confirmation prompts
bulletproof restore 5 --force

# Restore to different location
bulletproof restore 5 --target ~/agent-test
```

### Manage Configuration

```bash
# View current configuration
bulletproof config show

# Edit configuration
bulletproof config edit

# Show config file path
bulletproof config path
```

### Short IDs vs Full IDs

**Short IDs** (ephemeral, recalculated each time):

- `0` = Current filesystem (not a snapshot)
- `1` = Most recent snapshot
- `2, 3, 4...` = Older snapshots

**Full IDs** (permanent, timestamp-based):

- Format: `yyyyMMdd-HHmmss`
- Example: `20250203-120000` = Feb 3, 2025 at 12:00:00
- Used as folder names, git tags

Both can be used interchangeably in commands.

---

## Summary

This guide covered:

1. **Drift Diagnosis**: Use binary search with snapshots and diff to find exact drift point
2. **Script Drift**: Monitor backup/restore scripts for changes that could break compatibility
3. **Custom Data Sources**: Export/import external databases and APIs via custom scripts
4. **Platform Migration**: Move agents between machines using self-contained backups
5. **Service Setup**: Configure automatic scheduled backups on each platform
6. **Basic Operations**: Reference for common commands

For more information, see:

- `bulletproof --help` - Command reference
- `bulletproof <command> --help` - Command-specific help
- Configuration file: `~/.config/bulletproof/config.yaml`

---

**END OF SKILL GUIDE**

```

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-02-03 | Requirements Team | Initial specification |

---

**END OF REQUIREMENTS SPECIFICATION**
```
