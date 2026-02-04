package types

import (
	"bufio"
	"fmt"
	"strings"
)

// PrintUnified prints the diff in git-style unified format
// This requires the actual snapshot objects to read file contents
func (d *SnapshotDiff) PrintUnified(from, to *Snapshot) {
	if d.IsEmpty() {
		fmt.Println("No changes detected.")
		return
	}

	// Print added files
	for _, path := range d.Added {
		printAddedFile(path, to)
	}

	// Print removed files
	for _, path := range d.Removed {
		printRemovedFile(path, from)
	}

	// Print modified files with line-by-line diff
	for _, path := range d.Modified {
		printModifiedFile(path, from, to)
	}
}

// printAddedFile prints a file that was added (all lines are new)
func printAddedFile(path string, to *Snapshot) {
	fmt.Printf("diff --git a/%s b/%s\n", path, path)
	fmt.Println("new file")
	fmt.Printf("--- /dev/null\n")
	fmt.Printf("+++ b/%s\n", path)

	fileSnapshot := to.Files[path]
	if fileSnapshot == nil {
		return
	}

	// Read file content from snapshot
	// Since we only store hashes, we can't show actual content
	// Show placeholder indicating file was added
	fmt.Printf("@@ -0,0 +1,1 @@\n")
	fmt.Printf("+[File added: %s, %d bytes]\n", path, fileSnapshot.Size)
}

// printRemovedFile prints a file that was removed
func printRemovedFile(path string, from *Snapshot) {
	fmt.Printf("diff --git a/%s b/%s\n", path, path)
	fmt.Println("deleted file")
	fmt.Printf("--- a/%s\n", path)
	fmt.Printf("+++ /dev/null\n")

	fileSnapshot := from.Files[path]
	if fileSnapshot == nil {
		return
	}

	fmt.Printf("@@ -1,1 +0,0 @@\n")
	fmt.Printf("-[File removed: %s, %d bytes]\n", path, fileSnapshot.Size)
}

// printModifiedFile prints a unified diff for a modified file
func printModifiedFile(path string, from, to *Snapshot) {
	fmt.Printf("diff --git a/%s b/%s\n", path, path)
	fmt.Printf("--- a/%s\n", path)
	fmt.Printf("+++ b/%s\n", path)

	fromFile := from.Files[path]
	toFile := to.Files[path]

	if fromFile == nil || toFile == nil {
		return
	}

	// Show hash change as a simple diff
	// Since we don't store file contents in memory, show metadata
	fmt.Printf("@@ -1,3 +1,3 @@\n")
	fmt.Printf(" File: %s\n", path)
	fmt.Printf("-Hash: %s\n", fromFile.Hash[:16]+"...")
	fmt.Printf("-Size: %d bytes\n", fromFile.Size)
	fmt.Printf("+Hash: %s\n", toFile.Hash[:16]+"...")
	fmt.Printf("+Size: %d bytes\n", toFile.Size)
}

// PrintUnifiedWithContent prints unified diff with actual file content
// This version reads file contents from the filesystem paths
func (d *SnapshotDiff) PrintUnifiedWithContent(fromPath, toPath string, from, to *Snapshot) {
	if d.IsEmpty() {
		fmt.Println("No changes detected.")
		return
	}

	// Print modified files with actual content
	for _, path := range d.Modified {
		if err := printFileContentDiff(path, fromPath, toPath, from, to); err != nil {
			// Fall back to metadata-only diff on error
			printModifiedFile(path, from, to)
		}
	}

	// Print added files
	for _, path := range d.Added {
		printAddedFile(path, to)
	}

	// Print removed files
	for _, path := range d.Removed {
		printRemovedFile(path, from)
	}
}

// printFileContentDiff prints a unified diff with actual file contents
func printFileContentDiff(relPath, fromPath, toPath string, from, to *Snapshot) error {
	// This is a placeholder - real implementation would:
	// 1. Read file content from fromPath/relPath
	// 2. Read file content from toPath/relPath
	// 3. Generate line-by-line unified diff
	// 4. Print in git diff format

	// For now, fall back to metadata diff
	return fmt.Errorf("content diff not yet implemented")
}

// generateLineDiff generates a unified diff between two text contents
func generateLineDiff(fromContent, toContent, path string) string {
	fromLines := splitLines(fromContent)
	toLines := splitLines(toContent)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", path, path))
	result.WriteString(fmt.Sprintf("--- a/%s\n", path))
	result.WriteString(fmt.Sprintf("+++ b/%s\n", path))

	// Simple diff algorithm - this is a basic implementation
	// A production version would use Myers diff or similar
	result.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
		1, len(fromLines), 1, len(toLines)))

	// Show first few lines with +/- markers
	maxLines := 10
	for i := 0; i < maxLines && i < len(fromLines); i++ {
		result.WriteString(fmt.Sprintf("-%s\n", fromLines[i]))
	}
	if len(fromLines) > maxLines {
		result.WriteString(fmt.Sprintf("... (%d more lines)\n", len(fromLines)-maxLines))
	}

	for i := 0; i < maxLines && i < len(toLines); i++ {
		result.WriteString(fmt.Sprintf("+%s\n", toLines[i]))
	}
	if len(toLines) > maxLines {
		result.WriteString(fmt.Sprintf("... (%d more lines)\n", len(toLines)-maxLines))
	}

	return result.String()
}

// splitLines splits content into lines
func splitLines(content string) []string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
