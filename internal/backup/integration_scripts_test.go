package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bulletproof-bot/backup/internal/config"
)

// Note: These tests are currently SKIPPED because the script execution framework
// is not yet implemented. They document the expected behavior for when scripts
// are implemented (Phase 2 of the implementation plan).

// TestScripts_PreBackupExecution tests pre-backup script execution
func TestScripts_PreBackupExecution(t *testing.T) {
	t.Skip("Script execution framework not yet implemented - Phase 2 feature")

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
		// TODO: Add Scripts field when implemented
		// Scripts: config.ScriptsConfig{
		//     PreBackup: []config.ScriptConfig{
		//         {
		//             Name:    "export-graph",
		//             Command: filepath.Join(scriptsDir, "pre-backup", "export-graph.sh"),
		//             Timeout: 60,
		//         },
		//     },
		// },
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create backup - should execute pre-backup script
	result, err := engine.Backup(false, "Backup with pre-backup script")
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
	t.Skip("Script execution framework not yet implemented - Phase 2 feature")

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
		// TODO: Add Scripts field when implemented
		// Scripts: config.ScriptsConfig{
		//     PreBackup: []config.ScriptConfig{
		//         {
		//             Name:    "export-graph",
		//             Command: filepath.Join(scriptsDir, "pre-backup", "export-graph.sh"),
		//             Timeout: 60,
		//         },
		//     },
		//     PostRestore: []config.ScriptConfig{
		//         {
		//             Name:    "import-graph",
		//             Command: filepath.Join(scriptsDir, "post-restore", "import-graph.sh"),
		//             Timeout: 60,
		//         },
		//     },
		// },
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create backup with export
	result, err := engine.Backup(false, "Backup for restore test")
	helper.assertNoError(err, "Backup failed")

	// Remove the imported file if it exists from previous runs
	importedPath := filepath.Join(agentDir, "graph_imported.json")
	os.Remove(importedPath)

	// Restore - should execute post-restore script
	err = engine.Restore(result.Snapshot.ID, false)
	helper.assertNoError(err, "Restore failed")

	// Verify post-restore script imported the data
	helper.assertFileExists(importedPath)

	// Verify imported data is valid
	helper.verifyGraphMemoryImport(importedPath)
}

// TestScripts_EnvironmentVariables tests environment variable substitution
func TestScripts_EnvironmentVariables(t *testing.T) {
	t.Skip("Script execution framework not yet implemented - Phase 2 feature")

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
		// TODO: Add Scripts field
		// Scripts: config.ScriptsConfig{
		//     PreBackup: []config.ScriptConfig{
		//         {
		//             Name:    "check-env",
		//             Command: filepath.Join(preBackupDir, "check-env.sh"),
		//             Timeout: 60,
		//         },
		//     },
		// },
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	result, err := engine.Backup(false, "Test environment variables")
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
	t.Skip("Script execution framework not yet implemented - Phase 2 feature")

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
		// TODO: Add Scripts field with 2 second timeout
		// Scripts: config.ScriptsConfig{
		//     PreBackup: []config.ScriptConfig{
		//         {
		//             Name:    "slow-script",
		//             Command: filepath.Join(preBackupDir, "slow-script.sh"),
		//             Timeout: 2, // 2 second timeout
		//         },
		//     },
		// },
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	start := time.Now()
	result, err := engine.Backup(false, "Test timeout handling")
	duration := time.Since(start)

	// Backup should succeed even if script times out
	helper.assertNoError(err, "Backup should succeed even with script timeout")

	// Should complete in ~2 seconds (timeout), not 10 seconds (full script duration)
	if duration > 5*time.Second {
		t.Errorf("Backup took %v, expected ~2 seconds due to timeout", duration)
	}

	// Verify snapshot was still created
	if result.Snapshot == nil {
		t.Error("Snapshot should be created even if script times out")
	}
}

// TestScripts_NoScriptsFlag tests --no-scripts flag behavior
func TestScripts_NoScriptsFlag(t *testing.T) {
	t.Skip("Script execution framework not yet implemented - Phase 2 feature")

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
		// TODO: Add Scripts field
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Backup with --no-scripts flag (would be passed to Backup method)
	// TODO: Add noScripts parameter to Backup method
	result, err := engine.Backup(false, "Backup without scripts")
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
	t.Skip("Script execution framework not yet implemented - Phase 2 feature")

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
		// TODO: Add Scripts field
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create initial backup (exports graph)
	result1, err := engine.Backup(false, "Backup with graph export")
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

	// Create second backup
	result2, err := engine.Backup(false, "Backup after graph modification")
	helper.assertNoError(err, "Second backup failed")

	// Verify second export has modified data
	graphExportPath2 := filepath.Join(backupDir, result2.Snapshot.ID, "_exports", "graph_memory.json")
	helper.assertFileExists(graphExportPath2)

	// Restore to first backup (imports old graph)
	err = engine.Restore(result1.Snapshot.ID, false)
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
	t.Skip("Script execution framework not yet implemented - Phase 2 feature")

	// This test would verify that multiple scripts execute in sequence
	// and that all scripts complete successfully before backup continues
	//
	// Expected behavior:
	// 1. Execute first pre-backup script
	// 2. Wait for completion
	// 3. Execute second pre-backup script
	// 4. Wait for completion
	// 5. Continue with backup
	//
	// Same for post-restore scripts
}

// TestScripts_ErrorHandling tests script failure scenarios
func TestScripts_ErrorHandling(t *testing.T) {
	t.Skip("Script execution framework not yet implemented - Phase 2 feature")

	// This test would verify error handling when scripts fail
	//
	// Expected behaviors to test:
	// 1. Script exits with non-zero status code
	// 2. Script produces stderr output
	// 3. Configuration option: abort backup on script failure vs continue
	// 4. Logging of script failures
	// 5. User notification of failures
}
