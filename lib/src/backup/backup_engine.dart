import 'dart:convert';
import 'dart:io';
import 'package:path/path.dart' as p;

import '../config/config.dart';
import '../config/openclaw_defaults.dart';
import '../destinations/destination.dart';
import '../destinations/git_destination.dart';
import '../destinations/local_destination.dart';
import 'snapshot.dart';

/// Main backup engine that orchestrates backups and restores.
class BackupEngine {
  final Config config;
  late final Destination destination;

  BackupEngine(this.config) {
    if (config.destination == null) {
      throw StateError('No destination configured. Run: bulletproof init');
    }

    destination = _createDestination(config.destination!);
  }

  Destination _createDestination(DestinationConfig destConfig) {
    switch (destConfig.type) {
      case 'git':
        return GitDestination(destConfig.path);
      case 'local':
        return LocalDestination(destConfig.path);
      case 'sync':
        // Sync destinations work like local - just copy files
        // The sync client (Dropbox/GDrive) handles the rest
        return LocalDestination(destConfig.path, timestamped: false);
      default:
        throw ArgumentError('Unknown destination type: ${destConfig.type}');
    }
  }

  /// Get the OpenClaw root path
  String get openclawPath {
    if (config.openclawPath != null) {
      return config.openclawPath!;
    }
    
    final detected = OpenClawDefaults.detectInstallation();
    if (detected != null) {
      return detected;
    }
    
    throw StateError(
      'OpenClaw installation not found. '
      'Run: bulletproof config set openclaw_path /path/to/.openclaw'
    );
  }

  /// Run a backup
  Future<BackupResult> backup({
    bool dryRun = false,
    String? message,
  }) async {
    print('🔍 Scanning OpenClaw installation at: $openclawPath');
    
    // Create snapshot of current state
    final snapshot = await Snapshot.fromDirectory(
      openclawPath,
      exclude: config.options.exclude,
      message: message,
    );

    print('📦 Found ${snapshot.files.length} files to back up');

    // Get last snapshot for comparison
    final lastSnapshot = await destination.getLastSnapshot();
    
    SnapshotDiff? diff;
    if (lastSnapshot != null) {
      diff = snapshot.diff(lastSnapshot);
      print('📊 Changes since last backup: $diff');
      
      if (diff.isEmpty) {
        print('✨ No changes detected. Backup skipped.');
        return BackupResult(
          snapshot: snapshot,
          diff: diff,
          skipped: true,
        );
      }
    } else {
      print('📝 First backup - no previous snapshot found');
    }

    if (dryRun) {
      print('\n🔍 Dry run - no changes made');
      diff?.printDetailed();
      return BackupResult(
        snapshot: snapshot,
        diff: diff,
        dryRun: true,
      );
    }

    // Perform the backup
    print('\n💾 Backing up to: ${config.destination!.path}');
    
    await destination.save(
      sourcePath: openclawPath,
      snapshot: snapshot,
      message: message ?? 'Backup ${snapshot.id}',
    );

    print('✅ Backup complete: ${snapshot.id}');
    
    return BackupResult(
      snapshot: snapshot,
      diff: diff,
    );
  }

  /// List all available backups
  Future<List<SnapshotInfo>> listBackups() async {
    return destination.listSnapshots();
  }

  /// Show diff between current state and last backup
  Future<SnapshotDiff?> showDiff() async {
    final current = await Snapshot.fromDirectory(
      openclawPath,
      exclude: config.options.exclude,
    );

    final last = await destination.getLastSnapshot();
    if (last == null) {
      print('No previous backup found.');
      return null;
    }

    return current.diff(last);
  }

  /// Restore from a specific backup
  Future<void> restore(String snapshotId, {bool dryRun = false}) async {
    print('🔍 Looking for backup: $snapshotId');
    
    final snapshot = await destination.getSnapshot(snapshotId);
    if (snapshot == null) {
      throw ArgumentError('Backup not found: $snapshotId');
    }

    print('📦 Found backup with ${snapshot.files.length} files');

    if (dryRun) {
      print('\n🔍 Dry run - would restore these files:');
      for (final file in snapshot.files.keys.take(20)) {
        print('  $file');
      }
      if (snapshot.files.length > 20) {
        print('  ... and ${snapshot.files.length - 20} more');
      }
      return;
    }

    // Create backup of current state before restore
    print('\n⚠️  Creating safety backup before restore...');
    final safetyBackup = await backup(
      message: 'Pre-restore safety backup',
    );
    print('📝 Safety backup created: ${safetyBackup.snapshot.id}');

    // Perform restore
    print('\n🔄 Restoring from $snapshotId...');
    await destination.restore(
      snapshotId: snapshotId,
      targetPath: openclawPath,
    );

    print('✅ Restore complete!');
    print('💡 If something went wrong, restore from: ${safetyBackup.snapshot.id}');
  }
}

/// Result of a backup operation
class BackupResult {
  final Snapshot snapshot;
  final SnapshotDiff? diff;
  final bool skipped;
  final bool dryRun;

  BackupResult({
    required this.snapshot,
    this.diff,
    this.skipped = false,
    this.dryRun = false,
  });
}

/// Basic info about a snapshot (for listing)
class SnapshotInfo {
  final String id;
  final DateTime timestamp;
  final String? message;
  final int fileCount;

  SnapshotInfo({
    required this.id,
    required this.timestamp,
    this.message,
    required this.fileCount,
  });

  @override
  String toString() {
    final msg = message != null ? ' - $message' : '';
    return '$id ($fileCount files)$msg';
  }
}

