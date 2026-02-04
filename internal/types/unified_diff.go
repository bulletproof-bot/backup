package types

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
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
	// Read file contents
	fromContent, err := readFileContent(filepath.Join(fromPath, relPath))
	if err != nil {
		return fmt.Errorf("failed to read from file: %w", err)
	}

	toContent, err := readFileContent(filepath.Join(toPath, relPath))
	if err != nil {
		return fmt.Errorf("failed to read to file: %w", err)
	}

	// Check if files are binary
	if isBinary(fromContent) || isBinary(toContent) {
		fmt.Printf("diff --git a/%s b/%s\n", relPath, relPath)
		fmt.Printf("--- a/%s\n", relPath)
		fmt.Printf("+++ b/%s\n", relPath)
		fmt.Println("Binary files differ")
		return nil
	}

	// Generate and print unified diff
	diff := generateUnifiedDiff(fromContent, toContent, relPath)
	fmt.Print(diff)
	return nil
}

// readFileContent reads file content as a string
func readFileContent(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// isBinary checks if content appears to be binary (contains null bytes or invalid UTF-8)
func isBinary(content string) bool {
	// Check for null bytes
	if strings.Contains(content, "\x00") {
		return true
	}

	// Check if valid UTF-8 (simple heuristic)
	return !utf8.ValidString(content)
}

// generateUnifiedDiff generates a proper unified diff between two text contents
func generateUnifiedDiff(fromContent, toContent, path string) string {
	fromLines := splitLines(fromContent)
	toLines := splitLines(toContent)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", path, path))
	result.WriteString(fmt.Sprintf("--- a/%s\n", path))
	result.WriteString(fmt.Sprintf("+++ b/%s\n", path))

	// Generate hunks using simple line-by-line comparison
	hunks := generateHunks(fromLines, toLines)

	for _, hunk := range hunks {
		result.WriteString(hunk)
	}

	return result.String()
}

// generateHunks generates unified diff hunks with context
func generateHunks(fromLines, toLines []string) []string {
	const contextLines = 3 // Number of context lines to show

	// Simple diff: find matching and non-matching sections
	type change struct {
		fromStart, fromCount int
		toStart, toCount     int
		lines                []string // Lines with +/- prefixes
	}

	var hunks []string
	var currentHunk []string
	var hunkFromStart, hunkToStart int
	var hunkFromCount, hunkToCount int

	// Simple algorithm: compare line by line
	i, j := 0, 0
	inHunk := false
	contextAfter := 0

	for i < len(fromLines) || j < len(toLines) {
		if i < len(fromLines) && j < len(toLines) && fromLines[i] == toLines[j] {
			// Lines match - add as context
			if inHunk {
				currentHunk = append(currentHunk, " "+fromLines[i])
				hunkFromCount++
				hunkToCount++
				contextAfter++

				// End hunk if we have enough context
				if contextAfter >= contextLines {
					hunks = append(hunks, formatHunk(hunkFromStart, hunkFromCount, hunkToStart, hunkToCount, currentHunk))
					currentHunk = nil
					inHunk = false
					contextAfter = 0
				}
			}
			i++
			j++
		} else {
			// Lines differ - start or continue hunk
			if !inHunk {
				// Start new hunk with context
				hunkFromStart = max(0, i-contextLines) + 1
				hunkToStart = max(0, j-contextLines) + 1
				hunkFromCount = 0
				hunkToCount = 0
				currentHunk = nil

				// Add leading context
				for k := max(0, i-contextLines); k < i && k < len(fromLines); k++ {
					currentHunk = append(currentHunk, " "+fromLines[k])
					hunkFromCount++
					hunkToCount++
				}

				inHunk = true
				contextAfter = 0
			}

			// Add changed lines
			if i < len(fromLines) && (j >= len(toLines) || fromLines[i] != toLines[j]) {
				currentHunk = append(currentHunk, "-"+fromLines[i])
				hunkFromCount++
				i++
				contextAfter = 0
			}
			if j < len(toLines) && (i >= len(fromLines) || (i > 0 && fromLines[i-1] != toLines[j])) {
				currentHunk = append(currentHunk, "+"+toLines[j])
				hunkToCount++
				j++
				contextAfter = 0
			}
		}
	}

	// Finalize last hunk
	if inHunk {
		hunks = append(hunks, formatHunk(hunkFromStart, hunkFromCount, hunkToStart, hunkToCount, currentHunk))
	}

	return hunks
}

// formatHunk formats a single hunk with header
func formatHunk(fromStart, fromCount, toStart, toCount int, lines []string) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", fromStart, fromCount, toStart, toCount))
	for _, line := range lines {
		result.WriteString(line + "\n")
	}
	return result.String()
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
