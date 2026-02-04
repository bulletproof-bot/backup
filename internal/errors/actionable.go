package errors

import (
	"fmt"
	"strings"
)

// ActionableError represents an error with context and suggested remediation
type ActionableError struct {
	Operation   string   // What failed (e.g., "create backup", "restore snapshot")
	Cause       error    // Root cause error
	Reasons     []string // Likely reasons why this happened
	Remediation string   // Specific command or action to try
	Related     string   // Related helpful command (optional)
}

// Error implements the error interface
func (e *ActionableError) Error() string {
	var sb strings.Builder

	// Main error message
	sb.WriteString(fmt.Sprintf("Error: failed to %s: %v.\n", e.Operation, e.Cause))

	// Likely reasons
	if len(e.Reasons) > 0 {
		sb.WriteString("\nThis usually means:\n")
		for _, reason := range e.Reasons {
			sb.WriteString(fmt.Sprintf("- %s\n", reason))
		}
	}

	// Remediation
	if e.Remediation != "" {
		sb.WriteString(fmt.Sprintf("\nTry:\n%s\n", e.Remediation))
	}

	// Related command
	if e.Related != "" {
		sb.WriteString(fmt.Sprintf("\nRelated: %s\n", e.Related))
	}

	return sb.String()
}

// Unwrap returns the underlying error
func (e *ActionableError) Unwrap() error {
	return e.Cause
}

// NewActionableError creates a new actionable error
func NewActionableError(operation string, cause error, reasons []string, remediation string, related string) *ActionableError {
	return &ActionableError{
		Operation:   operation,
		Cause:       cause,
		Reasons:     reasons,
		Remediation: remediation,
		Related:     related,
	}
}

// ConfigNotFound returns an error for missing config file
func ConfigNotFound(cause error) *ActionableError {
	return &ActionableError{
		Operation: "load configuration",
		Cause:     cause,
		Reasons: []string{
			"Bulletproof has not been initialized yet",
			"Config file was deleted or moved",
		},
		Remediation: "bulletproof init",
		Related:     "bulletproof config show",
	}
}

// OpenClawNotFound returns an error for missing OpenClaw installation
func OpenClawNotFound() *ActionableError {
	return &ActionableError{
		Operation: "locate OpenClaw installation",
		Cause:     fmt.Errorf("OpenClaw installation not found"),
		Reasons: []string{
			"OpenClaw is not installed in the default location (~/.openclaw/)",
			"Custom path not configured",
		},
		Remediation: "bulletproof config set openclaw_path /path/to/.openclaw",
		Related:     "bulletproof config show",
	}
}

// SnapshotNotFound returns an error for missing snapshot
func SnapshotNotFound(snapshotID string) *ActionableError {
	return &ActionableError{
		Operation: "find snapshot",
		Cause:     fmt.Errorf("snapshot not found: %s", snapshotID),
		Reasons: []string{
			"Snapshot ID is incorrect",
			"Snapshot was deleted",
			"Wrong backup destination",
		},
		Remediation: "bulletproof snapshots",
		Related:     "bulletproof config show",
	}
}

// PermissionDenied returns an error for permission issues
func PermissionDenied(operation string, path string, cause error) *ActionableError {
	return &ActionableError{
		Operation: operation,
		Cause:     cause,
		Reasons: []string{
			"File permissions too restrictive",
			"Directory owned by different user",
			"Parent directory not writable",
		},
		Remediation: fmt.Sprintf("chmod -R u+rw %s\n\nOr run as correct user:\nsudo -u owner-user bulletproof %s", path, operation),
		Related:     "ls -la " + path,
	}
}

// GitError returns an error for git operations
func GitError(operation string, cause error) *ActionableError {
	return &ActionableError{
		Operation: operation,
		Cause:     cause,
		Reasons: []string{
			"Git repository not initialized",
			"Remote not configured or unreachable",
			"Authentication failed",
			"Network connectivity issue",
		},
		Remediation: "Check git configuration:\ngit remote -v\ngit status",
		Related:     "bulletproof config show",
	}
}

// ScriptExecutionError returns an error for script failures
func ScriptExecutionError(scriptName string, cause error) *ActionableError {
	return &ActionableError{
		Operation: fmt.Sprintf("execute script '%s'", scriptName),
		Cause:     cause,
		Reasons: []string{
			"Script file not found or not executable",
			"Script returned non-zero exit code",
			"Script timeout exceeded",
			"Missing dependencies or environment variables",
		},
		Remediation: fmt.Sprintf("Check script exists and is executable:\nls -la ~/.config/bulletproof/scripts/\nchmod +x ~/.config/bulletproof/scripts/%s", scriptName),
		Related:     "bulletproof backup --no-scripts",
	}
}

// BackupDestinationError returns an error for backup destination issues
func BackupDestinationError(operation string, path string, cause error) *ActionableError {
	return &ActionableError{
		Operation: operation,
		Cause:     cause,
		Reasons: []string{
			"Destination path does not exist",
			"Insufficient disk space",
			"Path is not a directory",
			"No write permissions",
		},
		Remediation: fmt.Sprintf("Verify destination exists and is writable:\nls -ld %s\ndf -h %s", path, path),
		Related:     "bulletproof config set destination /path/to/backups",
	}
}
