package types

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

// IsShortID returns true if the given ID is a short numeric ID
func IsShortID(id string) bool {
	matched, _ := regexp.MatchString(`^\d+$`, id)
	return matched
}

// IsFullID returns true if the given ID is a full timestamp ID
func IsFullID(id string) bool {
	matched, _ := regexp.MatchString(`^\d{8}-\d{6}$`, id)
	return matched
}

// ResolveID converts a short numeric ID to a full timestamp ID
// ID 0 = "current" (special case, no snapshot)
// ID 1 = latest snapshot
// ID 2 = second-latest snapshot
// ...
// Returns the original ID if it's already a full ID
func ResolveID(id string, snapshots []*SnapshotInfo) (string, error) {
	// If it's already a full ID, return as-is
	if IsFullID(id) {
		return id, nil
	}

	// Special case: ID 0 means current filesystem state
	if id == "0" {
		return "0", nil // Will be handled specially by callers
	}

	// Must be a short numeric ID
	if !IsShortID(id) {
		return "", fmt.Errorf("invalid snapshot ID format: %s (expected numeric ID or yyyyMMdd-HHmmss)", id)
	}

	// Convert to number
	num, err := strconv.Atoi(id)
	if err != nil {
		return "", fmt.Errorf("invalid numeric ID: %s", id)
	}

	if num < 0 {
		return "", fmt.Errorf("snapshot ID must be >= 0, got: %d", num)
	}

	// Sort snapshots by timestamp (newest first)
	sorted := make([]*SnapshotInfo, len(snapshots))
	copy(sorted, snapshots)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.After(sorted[j].Timestamp)
	})

	// ID 1 = latest (index 0), ID 2 = second-latest (index 1), etc.
	index := num - 1
	if index < 0 || index >= len(sorted) {
		return "", fmt.Errorf("snapshot ID %d out of range (have %d snapshots)", num, len(sorted))
	}

	return sorted[index].ID, nil
}

// AssignShortIDs assigns short numeric IDs to snapshots (sorted newest to oldest)
// Returns a map from full ID to short ID
func AssignShortIDs(snapshots []*SnapshotInfo) map[string]int {
	// Sort by timestamp (newest first)
	sorted := make([]*SnapshotInfo, len(snapshots))
	copy(sorted, snapshots)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.After(sorted[j].Timestamp)
	})

	// Assign short IDs: 1 = newest, 2 = second-newest, etc.
	shortIDs := make(map[string]int)
	for i, snapshot := range sorted {
		shortIDs[snapshot.ID] = i + 1
	}

	return shortIDs
}
