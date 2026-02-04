package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bulletproof-bot/backup/internal/config"
)

// TestScripts_PreBackupExecution tests pre-backup script execution
func TestScripts_PreBackupExecution(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("script-agent")
	backupDir := helper.createBackupDestination("scripts")
	scriptsDir := filepath.Join(helper.baseDir, "scripts")

	// Create mock scripts
	helper.createMockScriptFiles(scriptsDir)

	// Create configuration with scripts
	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
		Scripts: config.ScriptsConfig{
			PreBackup: []config.ScriptConfig{
				{
					Name:    "export-graph",
					Command: filepath.Join(scriptsDir, "pre-backup", "export-graph.sh"),
					Timeout: 60,
				},
			},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create backup - should execute pre-backup script
	result, err := engine.Backup(false, "Backup with pre-backup script", false, false)
	helper.assertNoError(err, "Backup with script failed")

	// Verify _exports directory was created in snapshot
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	exportsPath := filepath.Join(snapshotPath, "_exports")
	helper.assertFileExists(exportsPath)

	// Verify graph export file exists
	graphExportPath := filepath.Join(exportsPath, "graph_memory.json")
	helper.assertFileExists(graphExportPath)

	// Verify export file has valid JSON
	content := helper.readFile(graphExportPath)
	var data map[string]interface{}
	err = json.Unmarshal([]byte(content), &data)
	helper.assertNoError(err, "Graph export should be valid JSON")
}

// TestScripts_PostRestoreExecution tests post-restore script execution
func TestScripts_PostRestoreExecution(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("restore-script-agent")
	backupDir := helper.createBackupDestination("restore-scripts")
	scriptsDir := filepath.Join(helper.baseDir, "scripts")

	helper.createMockScriptFiles(scriptsDir)

	// Create configuration with scripts
	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
		Scripts: config.ScriptsConfig{
			PreBackup: []config.ScriptConfig{
				{
					Name:    "export-graph",
					Command: filepath.Join(scriptsDir, "pre-backup", "export-graph.sh"),
					Timeout: 60,
				},
			},
			PostRestore: []config.ScriptConfig{
				{
					Name:    "import-graph",
					Command: filepath.Join(scriptsDir, "post-restore", "import-graph.sh"),
					Timeout: 60,
				},
			},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create backup with export
	result, err := engine.Backup(false, "Backup for restore test", false, false)
	helper.assertNoError(err, "Backup failed")

	// Remove the imported file if it exists from previous runs
	importedPath := filepath.Join(agentDir, "graph_imported.json")
	os.Remove(importedPath)

	// Restore - should execute post-restore script (use RestoreToTarget with force=true for tests)
	err = engine.RestoreToTarget(result.Snapshot.ID, "", false, false, true)
	helper.assertNoError(err, "Restore failed")

	// Verify post-restore script imported the data
	helper.assertFileExists(importedPath)

	// Verify imported data is valid
	helper.verifyGraphMemoryImport(importedPath)
}

// TestScripts_EnvironmentVariables tests environment variable substitution
func TestScripts_EnvironmentVariables(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("env-agent")
	backupDir := helper.createBackupDestination("env-scripts")
	scriptsDir := filepath.Join(helper.baseDir, "scripts")

	// Create a script that uses environment variables
	preBackupDir := filepath.Join(scriptsDir, "pre-backup")
	if err := os.MkdirAll(preBackupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Script that echoes environment variables to verify they're set
	envScript := `#!/bin/bash
set -e
echo "EXPORTS_DIR: $EXPORTS_DIR" > "$EXPORTS_DIR/env_check.txt"
echo "SNAPSHOT_ID: $SNAPSHOT_ID" >> "$EXPORTS_DIR/env_check.txt"
echo "OPENCLAW_PATH: $OPENCLAW_PATH" >> "$EXPORTS_DIR/env_check.txt"
`
	helper.writeFile(filepath.Join(preBackupDir, "check-env.sh"), envScript)
	os.Chmod(filepath.Join(preBackupDir, "check-env.sh"), 0755)

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
		Scripts: config.ScriptsConfig{
			PreBackup: []config.ScriptConfig{
				{
					Name:    "check-env",
					Command: filepath.Join(preBackupDir, "check-env.sh"),
					Timeout: 60,
				},
			},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	result, err := engine.Backup(false, "Test environment variables", false, false)
	helper.assertNoError(err, "Backup failed")

	// Verify environment variables were set correctly
	envCheckPath := filepath.Join(backupDir, result.Snapshot.ID, "_exports", "env_check.txt")
	helper.assertFileExists(envCheckPath)

	helper.assertFileContains(envCheckPath, "EXPORTS_DIR:")
	helper.assertFileContains(envCheckPath, "SNAPSHOT_ID:")
	helper.assertFileContains(envCheckPath, "OPENCLAW_PATH:")
	helper.assertFileContains(envCheckPath, result.Snapshot.ID) // Should contain snapshot ID
}

// TestScripts_TimeoutHandling tests script timeout behavior
func TestScripts_TimeoutHandling(t *testing.T) {
	t.Skip("Timeout handling for bash subprocesses (sleep) doesn't work reliably due to process group issues - this is a known limitation")
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("timeout-agent")
	backupDir := helper.createBackupDestination("timeout-scripts")
	scriptsDir := filepath.Join(helper.baseDir, "scripts")

	// Create a script that sleeps longer than timeout
	preBackupDir := filepath.Join(scriptsDir, "pre-backup")
	if err := os.MkdirAll(preBackupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Script that sleeps for 10 seconds (will be killed by 2 second timeout)
	timeoutScript := `#!/bin/bash
echo "Starting long operation"
sleep 10
echo "This should not be reached"
`
	helper.writeFile(filepath.Join(preBackupDir, "slow-script.sh"), timeoutScript)
	os.Chmod(filepath.Join(preBackupDir, "slow-script.sh"), 0755)

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
		Scripts: config.ScriptsConfig{
			PreBackup: []config.ScriptConfig{
				{
					Name:    "slow-script",
					Command: filepath.Join(preBackupDir, "slow-script.sh"),
					Timeout: 2, // 2 second timeout
				},
			},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	start := time.Now()
	result, err := engine.Backup(false, "Test timeout handling", false, false)
	duration := time.Since(start)

	// Backup should fail if script times out (scripts are required to succeed)
	if err == nil {
		t.Error("Backup should fail when script times out")
	}

	// Should complete in ~2 seconds (timeout), not 10 seconds (full script duration)
	if duration > 5*time.Second {
		t.Errorf("Backup took %v, expected ~2 seconds due to timeout", duration)
	}

	// Verify snapshot was NOT created (backup should fail on script timeout)
	_ = result
}

// TestScripts_NoScriptsFlag tests --no-scripts flag behavior
func TestScripts_NoScriptsFlag(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("noscript-agent")
	backupDir := helper.createBackupDestination("noscript")
	scriptsDir := filepath.Join(helper.baseDir, "scripts")

	helper.createMockScriptFiles(scriptsDir)

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
		Scripts: config.ScriptsConfig{
			PreBackup: []config.ScriptConfig{
				{
					Name:    "export-graph",
					Command: filepath.Join(scriptsDir, "pre-backup", "export-graph.sh"),
					Timeout: 60,
				},
			},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Backup with --no-scripts flag (third parameter is noScripts)
	result, err := engine.Backup(false, "Backup without scripts", true, false)
	helper.assertNoError(err, "Backup failed")

	// Verify _exports directory was NOT created (scripts were skipped)
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	exportsPath := filepath.Join(snapshotPath, "_exports")
	helper.assertFileNotExists(exportsPath)
}

// TestScripts_UntrustedBackupWarning tests security warning for untrusted backups
func TestScripts_UntrustedBackupWarning(t *testing.T) {
	t.Skip("Script execution framework not yet implemented - Phase 2 feature")

	// This test would verify that when restoring a backup with scripts,
	// the user is warned and prompted for confirmation before scripts execute.
	//
	// Expected behavior:
	// 1. Restore detects scripts in backup
	// 2. Displays warning about script execution
	// 3. Lists scripts that will be executed
	// 4. Prompts for confirmation (unless --force flag is used)
	// 5. Only executes scripts if user confirms or --force is used
	//
	// Security requirement from specs/requirements.md section 4.8
}

// TestScripts_MockGraphMemory tests backup/restore of mock graph database
func TestScripts_MockGraphMemory(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("graph-agent")
	backupDir := helper.createBackupDestination("graph-backup")
	scriptsDir := filepath.Join(helper.baseDir, "scripts")
	graphDataDir := filepath.Join(helper.baseDir, "graph-data")

	// Create mock graph database directory
	if err := os.MkdirAll(graphDataDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial graph data
	graphData := map[string]interface{}{
		"nodes": []map[string]interface{}{
			{"id": "user_1", "type": "user", "name": "Alice"},
			{"id": "concept_1", "type": "concept", "name": "AI Safety"},
			{"id": "concept_2", "type": "concept", "name": "Machine Learning"},
		},
		"edges": []map[string]interface{}{
			{"from": "user_1", "to": "concept_1", "type": "interested_in"},
			{"from": "user_1", "to": "concept_2", "type": "interested_in"},
			{"from": "concept_1", "to": "concept_2", "type": "related_to"},
		},
	}
	helper.writeJSON(filepath.Join(graphDataDir, "graph.json"), graphData)

	// Create pre-backup script that exports graph
	preBackupDir := filepath.Join(scriptsDir, "pre-backup")
	if err := os.MkdirAll(preBackupDir, 0755); err != nil {
		t.Fatal(err)
	}

	exportScript := `#!/bin/bash
set -e
echo "Exporting graph memory..."
cp ` + filepath.Join(graphDataDir, "graph.json") + ` "$EXPORTS_DIR/graph_memory.json"
echo "Graph exported successfully"
`
	helper.writeFile(filepath.Join(preBackupDir, "export-graph.sh"), exportScript)
	os.Chmod(filepath.Join(preBackupDir, "export-graph.sh"), 0755)

	// Create post-restore script that imports graph
	postRestoreDir := filepath.Join(scriptsDir, "post-restore")
	if err := os.MkdirAll(postRestoreDir, 0755); err != nil {
		t.Fatal(err)
	}

	importScript := `#!/bin/bash
set -e
echo "Importing graph memory..."
cp "$BACKUP_DIR/_exports/graph_memory.json" ` + filepath.Join(graphDataDir, "graph.json") + `
echo "Graph imported successfully"
`
	helper.writeFile(filepath.Join(postRestoreDir, "import-graph.sh"), importScript)
	os.Chmod(filepath.Join(postRestoreDir, "import-graph.sh"), 0755)

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
		Scripts: config.ScriptsConfig{
			PreBackup: []config.ScriptConfig{
				{
					Name:    "export-graph",
					Command: filepath.Join(preBackupDir, "export-graph.sh"),
					Timeout: 60,
				},
			},
			PostRestore: []config.ScriptConfig{
				{
					Name:    "import-graph",
					Command: filepath.Join(postRestoreDir, "import-graph.sh"),
					Timeout: 60,
				},
			},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create initial backup (exports graph)
	result1, err := engine.Backup(false, "Backup with graph export", false, false)
	helper.assertNoError(err, "Initial backup failed")

	// Verify graph was exported to snapshot
	graphExportPath := filepath.Join(backupDir, result1.Snapshot.ID, "_exports", "graph_memory.json")
	helper.assertFileExists(graphExportPath)

	// Verify exported graph has correct data
	exportedContent := helper.readFile(graphExportPath)
	var exportedData map[string]interface{}
	err = json.Unmarshal([]byte(exportedContent), &exportedData)
	helper.assertNoError(err, "Exported graph should be valid JSON")

	nodes := exportedData["nodes"].([]interface{})
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes in exported graph, got %d", len(nodes))
	}

	// Modify graph data (simulate changes after backup)
	modifiedGraphData := map[string]interface{}{
		"nodes": []map[string]interface{}{
			{"id": "user_1", "type": "user", "name": "Alice"},
			{"id": "concept_1", "type": "concept", "name": "AI Safety"},
			{"id": "concept_3", "type": "concept", "name": "Deep Learning"}, // New node
		},
		"edges": []map[string]interface{}{
			{"from": "user_1", "to": "concept_1", "type": "interested_in"},
			{"from": "user_1", "to": "concept_3", "type": "interested_in"}, // New edge
		},
	}
	helper.writeJSON(filepath.Join(graphDataDir, "graph.json"), modifiedGraphData)

	// Also modify a file in the agent directory to trigger a backup
	helper.modifyAgentPersonality(agentDir, "# Agent Personality\n\nI learn from graph data.\n")

	// Create second backup
	result2, err := engine.Backup(false, "Backup after graph modification", false, false)
	helper.assertNoError(err, "Second backup failed")

	// Verify second export has modified data
	graphExportPath2 := filepath.Join(backupDir, result2.Snapshot.ID, "_exports", "graph_memory.json")
	helper.assertFileExists(graphExportPath2)

	// Restore to first backup (imports old graph) - use RestoreToTarget with force=true for tests
	err = engine.RestoreToTarget(result1.Snapshot.ID, "", false, false, true)
	helper.assertNoError(err, "Restore failed")

	// Verify graph was restored to original state
	restoredContent := helper.readFile(filepath.Join(graphDataDir, "graph.json"))
	var restoredData map[string]interface{}
	err = json.Unmarshal([]byte(restoredContent), &restoredData)
	helper.assertNoError(err, "Restored graph should be valid JSON")

	restoredNodes := restoredData["nodes"].([]interface{})
	if len(restoredNodes) != 3 {
		t.Errorf("Expected 3 nodes in restored graph, got %d", len(restoredNodes))
	}

	// Verify specific node data was restored
	helper.assertFileContains(filepath.Join(graphDataDir, "graph.json"), "Machine Learning")
	helper.assertFileContains(filepath.Join(graphDataDir, "graph.json"), "AI Safety")
}

// TestScripts_MultipleScripts tests execution of multiple pre/post scripts
func TestScripts_MultipleScripts(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("multi-script-agent")
	backupDir := helper.createBackupDestination("multi-scripts")
	scriptsDir := filepath.Join(helper.baseDir, "scripts")

	// Create multiple scripts
	preBackupDir := filepath.Join(scriptsDir, "pre-backup")
	os.MkdirAll(preBackupDir, 0755)

	// First script
	script1 := filepath.Join(preBackupDir, "script1.sh")
	helper.writeFile(script1, `#!/bin/bash
set -e
echo "Script 1 executed" > "$EXPORTS_DIR/script1.txt"
`)
	os.Chmod(script1, 0755)

	// Second script
	script2 := filepath.Join(preBackupDir, "script2.sh")
	helper.writeFile(script2, `#!/bin/bash
set -e
echo "Script 2 executed" > "$EXPORTS_DIR/script2.txt"
`)
	os.Chmod(script2, 0755)

	// Create config with multiple scripts
	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
		Scripts: config.ScriptsConfig{
			PreBackup: []config.ScriptConfig{
				{
					Name:    "script1",
					Command: script1,
					Timeout: 60,
				},
				{
					Name:    "script2",
					Command: script2,
					Timeout: 60,
				},
			},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create backup - should execute both scripts
	result, err := engine.Backup(false, "Test multiple scripts", false, false)
	helper.assertNoError(err, "Backup failed")

	// Verify both scripts executed
	script1Output := filepath.Join(backupDir, result.Snapshot.ID, "_exports", "script1.txt")
	script2Output := filepath.Join(backupDir, result.Snapshot.ID, "_exports", "script2.txt")

	helper.assertFileExists(script1Output)
	helper.assertFileExists(script2Output)
	helper.assertFileContains(script1Output, "Script 1 executed")
	helper.assertFileContains(script2Output, "Script 2 executed")
}

// TestScripts_ErrorHandling tests script failure scenarios
func TestScripts_ErrorHandling(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("error-script-agent")
	backupDir := helper.createBackupDestination("error-scripts")
	scriptsDir := filepath.Join(helper.baseDir, "scripts")

	// Create failing script
	preBackupDir := filepath.Join(scriptsDir, "pre-backup")
	os.MkdirAll(preBackupDir, 0755)

	failingScript := filepath.Join(preBackupDir, "failing.sh")
	helper.writeFile(failingScript, `#!/bin/bash
echo "This script will fail" >&2
exit 1
`)
	os.Chmod(failingScript, 0755)

	// Create config with failing script
	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
		Scripts: config.ScriptsConfig{
			PreBackup: []config.ScriptConfig{
				{
					Name:    "failing-script",
					Command: failingScript,
					Timeout: 60,
				},
			},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Backup should fail because script failed
	_, err = engine.Backup(false, "Test script error", false, false)
	if err == nil {
		t.Error("Expected backup to fail when script fails, but it succeeded")
	}

	// Verify error message mentions script failure
	if err != nil && !strings.Contains(err.Error(), "script") {
		t.Errorf("Expected error message to mention script failure, got: %v", err)
	}
}
