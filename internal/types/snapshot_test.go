package types

import (
	"testing"
	"time"
)

func TestSnapshotDiff(t *testing.T) {
	now := time.Now()

	snap1 := &Snapshot{
		ID:        "20240101-120000",
		Timestamp: now,
		Files: map[string]*FileSnapshot{
			"file1.txt": {Path: "file1.txt", Hash: "abc123", Size: 100, Modified: now},
			"file2.txt": {Path: "file2.txt", Hash: "def456", Size: 200, Modified: now},
		},
	}

	snap2 := &Snapshot{
		ID:        "20240101-130000",
		Timestamp: now.Add(time.Hour),
		Files: map[string]*FileSnapshot{
			"file1.txt": {Path: "file1.txt", Hash: "abc123", Size: 100, Modified: now},
			"file2.txt": {Path: "file2.txt", Hash: "xyz789", Size: 250, Modified: now},
			"file3.txt": {Path: "file3.txt", Hash: "ghi012", Size: 150, Modified: now},
		},
	}

	diff := snap2.Diff(snap1)

	// Verify added files
	if len(diff.Added) != 1 {
		t.Errorf("expected 1 added file, got %d", len(diff.Added))
	}
	if len(diff.Added) > 0 && diff.Added[0] != "file3.txt" {
		t.Errorf("expected file3.txt to be added, got %s", diff.Added[0])
	}

	// Verify modified files
	if len(diff.Modified) != 1 {
		t.Errorf("expected 1 modified file, got %d", len(diff.Modified))
	}
	if len(diff.Modified) > 0 && diff.Modified[0] != "file2.txt" {
		t.Errorf("expected file2.txt to be modified, got %s", diff.Modified[0])
	}

	// Verify no removed files
	if len(diff.Removed) != 0 {
		t.Errorf("expected 0 removed files, got %d", len(diff.Removed))
	}

	// Verify total changes
	if diff.TotalChanges() != 2 {
		t.Errorf("expected 2 total changes, got %d", diff.TotalChanges())
	}

	// Verify not empty
	if diff.IsEmpty() {
		t.Error("diff should not be empty")
	}
}

func TestSnapshotDiff_NoChanges(t *testing.T) {
	now := time.Now()

	snap1 := &Snapshot{
		ID:        "20240101-120000",
		Timestamp: now,
		Files: map[string]*FileSnapshot{
			"file1.txt": {Path: "file1.txt", Hash: "abc123", Size: 100, Modified: now},
		},
	}

	snap2 := &Snapshot{
		ID:        "20240101-130000",
		Timestamp: now.Add(time.Hour),
		Files: map[string]*FileSnapshot{
			"file1.txt": {Path: "file1.txt", Hash: "abc123", Size: 100, Modified: now},
		},
	}

	diff := snap2.Diff(snap1)

	if !diff.IsEmpty() {
		t.Error("diff should be empty when files are identical")
	}

	if diff.TotalChanges() != 0 {
		t.Errorf("expected 0 total changes, got %d", diff.TotalChanges())
	}
}

func TestSnapshotDiff_RemovedFiles(t *testing.T) {
	now := time.Now()

	snap1 := &Snapshot{
		ID:        "20240101-120000",
		Timestamp: now,
		Files: map[string]*FileSnapshot{
			"file1.txt": {Path: "file1.txt", Hash: "abc123", Size: 100, Modified: now},
			"file2.txt": {Path: "file2.txt", Hash: "def456", Size: 200, Modified: now},
		},
	}

	snap2 := &Snapshot{
		ID:        "20240101-130000",
		Timestamp: now.Add(time.Hour),
		Files: map[string]*FileSnapshot{
			"file1.txt": {Path: "file1.txt", Hash: "abc123", Size: 100, Modified: now},
		},
	}

	diff := snap2.Diff(snap1)

	if len(diff.Removed) != 1 {
		t.Errorf("expected 1 removed file, got %d", len(diff.Removed))
	}

	if len(diff.Removed) > 0 && diff.Removed[0] != "file2.txt" {
		t.Errorf("expected file2.txt to be removed, got %s", diff.Removed[0])
	}
}

func TestGenerateID(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	id := GenerateID(testTime)

	expected := "20240115-143045"
	if id != expected {
		t.Errorf("expected ID %s, got %s", expected, id)
	}
}

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "exact match",
			path:     "file.log",
			patterns: []string{"*.log"},
			want:     true,
		},
		{
			name:     "directory pattern",
			path:     "node_modules/package/file.js",
			patterns: []string{"node_modules/"},
			want:     true,
		},
		{
			name:     "no match",
			path:     "file.txt",
			patterns: []string{"*.log"},
			want:     false,
		},
		{
			name:     ".git directory",
			path:     ".git/config",
			patterns: []string{".git/"},
			want:     true,
		},
		{
			name:     "nested .git directory",
			path:     "subdir/.git/config",
			patterns: []string{".git/"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldExclude(tt.path, tt.patterns)
			if got != tt.want {
				t.Errorf("shouldExclude(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}
