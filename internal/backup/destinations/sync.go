package destinations

// SyncDestination is a destination for cloud sync services (Dropbox, Google Drive, iCloud).
// It's essentially a LocalDestination without timestamping - the sync client handles versioning.
type SyncDestination struct {
	*LocalDestination
}

// NewSyncDestination creates a new sync destination
func NewSyncDestination(basePath string) *SyncDestination {
	return &SyncDestination{
		LocalDestination: NewLocalDestination(basePath, false),
	}
}
