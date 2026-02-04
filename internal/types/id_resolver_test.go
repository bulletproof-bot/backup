package types

import (
	"testing"
	"time"
)

func TestIsShortID(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"0", true},
		{"1", true},
		{"42", true},
		{"999", true},
		{"20260203-120000", false},
		{"abc", false},
		{"1a", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsShortID(tt.id)
		if result != tt.expected {
			t.Errorf("IsShortID(%q) = %v, expected %v", tt.id, result, tt.expected)
		}
	}
}

func TestIsFullID(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"20260203-120000", true},
		{"20260203-235959", true},
		{"20260101-000000", true},
		{"1", false},
		{"42", false},
		{"2026020-120000", false},  // Too short
		{"202602033-120000", false}, // Too long
		{"20260203-12000", false},  // Missing second digit
		{"abc", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsFullID(tt.id)
		if result != tt.expected {
			t.Errorf("IsFullID(%q) = %v, expected %v", tt.id, result, tt.expected)
		}
	}
}

func TestResolveID(t *testing.T) {
	// Create test snapshots
	now := time.Now()
	snapshots := []*SnapshotInfo{
		{ID: "20260203-120000", Timestamp: now.Add(-3 * time.Hour)}, // Oldest
		{ID: "20260203-140000", Timestamp: now.Add(-2 * time.Hour)},
		{ID: "20260203-160000", Timestamp: now.Add(-1 * time.Hour)}, // Latest
	}

	tests := []struct {
		name      string
		id        string
		wantID    string
		wantError bool
	}{
		{
			name:      "ID 0 (current)",
			id:        "0",
			wantID:    "0",
			wantError: false,
		},
		{
			name:      "ID 1 (latest)",
			id:        "1",
			wantID:    "20260203-160000",
			wantError: false,
		},
		{
			name:      "ID 2 (second-latest)",
			id:        "2",
			wantID:    "20260203-140000",
			wantError: false,
		},
		{
			name:      "ID 3 (oldest)",
			id:        "3",
			wantID:    "20260203-120000",
			wantError: false,
		},
		{
			name:      "ID 4 (out of range)",
			id:        "4",
			wantID:    "",
			wantError: true,
		},
		{
			name:      "Full ID (passthrough)",
			id:        "20260203-140000",
			wantID:    "20260203-140000",
			wantError: false,
		},
		{
			name:      "Invalid format",
			id:        "abc",
			wantID:    "",
			wantError: true,
		},
		{
			name:      "Negative ID",
			id:        "-1",
			wantID:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := ResolveID(tt.id, snapshots)

			if tt.wantError {
				if err == nil {
					t.Errorf("ResolveID(%q) expected error, got nil", tt.id)
				}
			} else {
				if err != nil {
					t.Errorf("ResolveID(%q) unexpected error: %v", tt.id, err)
				}
				if gotID != tt.wantID {
					t.Errorf("ResolveID(%q) = %q, want %q", tt.id, gotID, tt.wantID)
				}
			}
		})
	}
}

func TestAssignShortIDs(t *testing.T) {
	now := time.Now()
	snapshots := []*SnapshotInfo{
		{ID: "20260203-120000", Timestamp: now.Add(-3 * time.Hour)}, // Oldest
		{ID: "20260203-140000", Timestamp: now.Add(-2 * time.Hour)},
		{ID: "20260203-160000", Timestamp: now.Add(-1 * time.Hour)}, // Latest
	}

	shortIDs := AssignShortIDs(snapshots)

	// Verify mappings
	expected := map[string]int{
		"20260203-160000": 1, // Latest = 1
		"20260203-140000": 2,
		"20260203-120000": 3, // Oldest = 3
	}

	if len(shortIDs) != len(expected) {
		t.Fatalf("Expected %d short IDs, got %d", len(expected), len(shortIDs))
	}

	for fullID, expectedShort := range expected {
		gotShort, exists := shortIDs[fullID]
		if !exists {
			t.Errorf("Missing short ID for full ID %q", fullID)
			continue
		}
		if gotShort != expectedShort {
			t.Errorf("Short ID for %q = %d, want %d", fullID, gotShort, expectedShort)
		}
	}
}

func TestResolveID_EmptySnapshots(t *testing.T) {
	snapshots := []*SnapshotInfo{}

	_, err := ResolveID("1", snapshots)
	if err == nil {
		t.Error("ResolveID with empty snapshots should return error")
	}
}

func TestResolveID_SingleSnapshot(t *testing.T) {
	snapshots := []*SnapshotInfo{
		{ID: "20260203-120000", Timestamp: time.Now()},
	}

	id, err := ResolveID("1", snapshots)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if id != "20260203-120000" {
		t.Errorf("ResolveID(\"1\") = %q, want %q", id, "20260203-120000")
	}

	// ID 2 should be out of range
	_, err = ResolveID("2", snapshots)
	if err == nil {
		t.Error("ResolveID(\"2\") with single snapshot should return error")
	}
}
