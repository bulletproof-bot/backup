package types

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Snapshot represents a point-in-time backup snapshot
type Snapshot struct {
	ID        string                   `json:"id"`
	Timestamp time.Time                `json:"timestamp"`
	Files     map[string]*FileSnapshot `json:"files"`
	Message   string                   `json:"message,omitempty"`
}

// FileSnapshot represents a single file in a snapshot
type FileSnapshot struct {
	Path     string    `json:"path"`
	Hash     string    `json:"hash"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// SnapshotDiff represents changes between two snapshots
type SnapshotDiff struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}

// GenerateID generates a snapshot ID from a timestamp
// Format: yyyyMMdd-HHmmss-SSS (includes milliseconds to prevent collisions)
func GenerateID(t time.Time) string {
	// Format with milliseconds: 20060102-150405-000
	ms := t.UnixMilli() % 1000
	return fmt.Sprintf("%s-%03d", t.Format("20060102-150405"), ms)
}

// FromDirectory creates a snapshot from a directory
func FromDirectory(path string, exclude []string, message string) (*Snapshot, error) {
	return FromDirectoryWithTimestamp(path, exclude, message, time.Now())
}

// FromDirectoryWithTimestamp creates a snapshot from a directory with a specific timestamp
func FromDirectoryWithTimestamp(path string, exclude []string, message string, timestamp time.Time) (*Snapshot, error) {
	id := GenerateID(timestamp)
	files := make(map[string]*FileSnapshot)

	// Check if directory exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("directory does not exist: %s: %w", path, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	// Walk the directory tree
	err = filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if fileInfo.IsDir() {
			return nil
		}

		// Get relative path
		relativePath, err := filepath.Rel(path, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Check exclusions
		if shouldExclude(relativePath, exclude) {
			return nil
		}

		// Create file snapshot
		fileSnapshot, err := fromFile(filePath, relativePath)
		if err != nil {
			return fmt.Errorf("failed to snapshot file %s: %w", relativePath, err)
		}

		files[relativePath] = fileSnapshot
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return &Snapshot{
		ID:        id,
		Timestamp: timestamp,
		Files:     files,
		Message:   message,
	}, nil
}

// fromFile creates a FileSnapshot from an actual file
func fromFile(filePath string, relativePath string) (*FileSnapshot, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Calculate SHA-256 hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to hash file: %w", err)
	}
	hashString := fmt.Sprintf("%x", hash.Sum(nil))

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	return &FileSnapshot{
		Path:     relativePath,
		Hash:     hashString,
		Size:     fileInfo.Size(),
		Modified: fileInfo.ModTime(),
	}, nil
}

// Diff calculates the difference between this snapshot and another
func (s *Snapshot) Diff(other *Snapshot) *SnapshotDiff {
	diff := &SnapshotDiff{
		From:     other.ID,
		To:       s.ID,
		Added:    []string{},
		Removed:  []string{},
		Modified: []string{},
	}

	// Find added and modified files
	for path, file := range s.Files {
		if otherFile, exists := other.Files[path]; !exists {
			diff.Added = append(diff.Added, path)
		} else if file.Hash != otherFile.Hash {
			diff.Modified = append(diff.Modified, path)
		}
	}

	// Find removed files
	for path := range other.Files {
		if _, exists := s.Files[path]; !exists {
			diff.Removed = append(diff.Removed, path)
		}
	}

	return diff
}

// IsEmpty returns true if the diff has no changes
func (d *SnapshotDiff) IsEmpty() bool {
	return len(d.Added) == 0 && len(d.Removed) == 0 && len(d.Modified) == 0
}

// TotalChanges returns the total number of changes
func (d *SnapshotDiff) TotalChanges() int {
	return len(d.Added) + len(d.Removed) + len(d.Modified)
}

// String returns a string representation of the diff
func (d *SnapshotDiff) String() string {
	if d.IsEmpty() {
		return "No changes"
	}

	parts := []string{}
	if len(d.Added) > 0 {
		parts = append(parts, fmt.Sprintf("+%d added", len(d.Added)))
	}
	if len(d.Modified) > 0 {
		parts = append(parts, fmt.Sprintf("~%d modified", len(d.Modified)))
	}
	if len(d.Removed) > 0 {
		parts = append(parts, fmt.Sprintf("-%d removed", len(d.Removed)))
	}
	return strings.Join(parts, ", ")
}

// PrintDetailed prints a detailed view of the diff
func (d *SnapshotDiff) PrintDetailed() {
	if d.IsEmpty() {
		fmt.Println("No changes detected.")
		return
	}

	if len(d.Added) > 0 {
		fmt.Println("\n  Added:")
		for _, f := range d.Added {
			fmt.Printf("    + %s\n", f)
		}
	}
	if len(d.Modified) > 0 {
		fmt.Println("\n  Modified:")
		for _, f := range d.Modified {
			fmt.Printf("    ~ %s\n", f)
		}
	}
	if len(d.Removed) > 0 {
		fmt.Println("\n  Removed:")
		for _, f := range d.Removed {
			fmt.Printf("    - %s\n", f)
		}
	}
}

// shouldExclude checks if a path should be excluded based on patterns
func shouldExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "/") {
			// Directory pattern
			if strings.HasPrefix(path, pattern) || strings.Contains(path, "/"+pattern) {
				return true
			}
		} else if strings.HasPrefix(pattern, "*.") {
			// Extension pattern
			if strings.HasSuffix(path, pattern[1:]) {
				return true
			}
		} else if strings.Contains(pattern, "**") {
			// Glob pattern - simplified matching
			regexPattern := strings.ReplaceAll(pattern, "**", ".*")
			regexPattern = strings.ReplaceAll(regexPattern, "*", "[^/]*")
			regex := regexp.MustCompile(regexPattern)
			if regex.MatchString(path) {
				return true
			}
		} else if path == pattern || strings.HasSuffix(path, "/"+pattern) {
			return true
		}
	}
	return false
}

// String returns a string representation of the snapshot
func (s *Snapshot) String() string {
	return fmt.Sprintf("Snapshot(%s, %d files)", s.ID, len(s.Files))
}

// ToJSON serializes the snapshot to JSON
func (s *Snapshot) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON deserializes a snapshot from JSON
func FromJSON(data []byte) (*Snapshot, error) {
	var snapshot Snapshot
	err := json.Unmarshal(data, &snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}
	return &snapshot, nil
}

// MergeWithSources combines multiple snapshots into a single snapshot
// Each snapshot's files are prefixed with their source base name to avoid conflicts
// For example, files from ~/.openclaw become ".openclaw/file.txt"
func MergeWithSources(snapshots []*Snapshot, sourcePaths []string, message string, timestamp time.Time) (*Snapshot, error) {
	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots to merge")
	}
	if len(snapshots) != len(sourcePaths) {
		return nil, fmt.Errorf("number of snapshots (%d) does not match number of source paths (%d)", len(snapshots), len(sourcePaths))
	}

	// Generate a single ID for the merged snapshot
	id := GenerateID(timestamp)

	merged := &Snapshot{
		ID:        id,
		Timestamp: timestamp,
		Files:     make(map[string]*FileSnapshot),
		Message:   message,
	}

	// Merge all files from all snapshots
	for i, snapshot := range snapshots {
		// Use the base name of the source path as the prefix
		sourceBase := filepath.Base(sourcePaths[i])

		for relPath, fileSnapshot := range snapshot.Files {
			// Prefix the path with source base name
			prefixedPath := filepath.Join(sourceBase, relPath)
			merged.Files[prefixedPath] = &FileSnapshot{
				Path:     prefixedPath,
				Hash:     fileSnapshot.Hash,
				Size:     fileSnapshot.Size,
				Modified: fileSnapshot.Modified,
			}
		}
	}

	return merged, nil
}
