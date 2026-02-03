import '../backup/snapshot.dart';
import '../backup/backup_engine.dart';

/// Abstract interface for backup destinations.
/// 
/// Implementations handle storing and retrieving backups
/// in different formats (git, local folders, cloud sync folders).
abstract class Destination {
  /// Save a backup to this destination
  Future<void> save({
    required String sourcePath,
    required Snapshot snapshot,
    required String message,
  });

  /// Get the most recent snapshot
  Future<Snapshot?> getLastSnapshot();

  /// Get a specific snapshot by ID
  Future<Snapshot?> getSnapshot(String id);

  /// List all available snapshots
  Future<List<SnapshotInfo>> listSnapshots();

  /// Restore files from a snapshot
  Future<void> restore({
    required String snapshotId,
    required String targetPath,
  });

  /// Check if this destination is properly configured
  Future<bool> validate();
}

