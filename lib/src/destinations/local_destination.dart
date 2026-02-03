import 'dart:convert';
import 'dart:io';
import 'package:path/path.dart' as p;

import '../backup/snapshot.dart';
import '../backup/backup_engine.dart';
import 'destination.dart';

/// Local filesystem destination.
/// 
/// Stores backups as folders on the local filesystem.
/// Can operate in two modes:
/// - timestamped: Each backup creates a new folder (default)
/// - overwrite: Overwrites the same folder (for sync services)
class LocalDestination implements Destination {
  final String basePath;
  final bool timestamped;

  LocalDestination(this.basePath, {this.timestamped = true});

  String _snapshotPath(String id) => p.join(basePath, id);
  String get _latestPath => p.join(basePath, 'latest');
  String get _metadataPath => p.join(basePath, '.bulletproof');

  @override
  Future<bool> validate() async {
    final dir = Directory(basePath);
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    return true;
  }

  @override
  Future<void> save({
    required String sourcePath,
    required Snapshot snapshot,
    required String message,
  }) async {
    await validate();

    final targetPath = timestamped 
        ? _snapshotPath(snapshot.id)
        : basePath;

    final targetDir = Directory(targetPath);
    
    if (timestamped) {
      // Create new snapshot folder
      await targetDir.create(recursive: true);
    } else {
      // Clear existing files for sync mode
      if (await targetDir.exists()) {
        await for (final entity in targetDir.list()) {
          final name = p.basename(entity.path);
          if (name == '.bulletproof') continue;
          
          if (entity is Directory) {
            await entity.delete(recursive: true);
          } else {
            await entity.delete();
          }
        }
      }
    }

    // Copy files
    print('  Copying ${snapshot.files.length} files...');
    for (final filePath in snapshot.files.keys) {
      final sourceFile = File(p.join(sourcePath, filePath));
      final destFile = File(p.join(targetPath, filePath));
      
      if (await sourceFile.exists()) {
        await destFile.parent.create(recursive: true);
        await sourceFile.copy(destFile.path);
      }
    }

    // Save metadata
    final metaDir = Directory(timestamped ? _metadataPath : p.join(basePath, '.bulletproof'));
    await metaDir.create(recursive: true);

    // Save snapshot info
    final snapshotFile = File(p.join(metaDir.path, '${snapshot.id}.json'));
    await snapshotFile.writeAsString(
      const JsonEncoder.withIndent('  ').convert(snapshot.toJson())
    );

    // Update latest pointer
    final latestFile = File(p.join(metaDir.path, 'latest'));
    await latestFile.writeAsString(snapshot.id);

    // Update index
    await _updateIndex(snapshot, message);

    print('  Backup saved to: $targetPath');
  }

  Future<void> _updateIndex(Snapshot snapshot, String message) async {
    final indexFile = File(p.join(_metadataPath, 'index.json'));
    
    List<Map<String, dynamic>> index = [];
    if (await indexFile.exists()) {
      try {
        final content = await indexFile.readAsString();
        index = (jsonDecode(content) as List).cast<Map<String, dynamic>>();
      } catch (_) {}
    }

    index.insert(0, {
      'id': snapshot.id,
      'timestamp': snapshot.timestamp.toIso8601String(),
      'message': message,
      'fileCount': snapshot.files.length,
    });

    // Keep last 100 entries
    if (index.length > 100) {
      index = index.sublist(0, 100);
    }

    await indexFile.parent.create(recursive: true);
    await indexFile.writeAsString(
      const JsonEncoder.withIndent('  ').convert(index)
    );
  }

  @override
  Future<Snapshot?> getLastSnapshot() async {
    final latestFile = File(p.join(_metadataPath, 'latest'));
    if (!await latestFile.exists()) {
      return null;
    }

    final latestId = (await latestFile.readAsString()).trim();
    return getSnapshot(latestId);
  }

  @override
  Future<Snapshot?> getSnapshot(String id) async {
    final snapshotFile = File(p.join(_metadataPath, '$id.json'));
    if (!await snapshotFile.exists()) {
      return null;
    }

    try {
      final content = await snapshotFile.readAsString();
      return Snapshot.fromJson(jsonDecode(content) as Map<String, dynamic>);
    } catch (e) {
      print('Warning: Failed to read snapshot $id: $e');
      return null;
    }
  }

  @override
  Future<List<SnapshotInfo>> listSnapshots() async {
    final indexFile = File(p.join(_metadataPath, 'index.json'));
    if (!await indexFile.exists()) {
      return [];
    }

    try {
      final content = await indexFile.readAsString();
      final index = (jsonDecode(content) as List).cast<Map<String, dynamic>>();
      
      return index.map((entry) => SnapshotInfo(
        id: entry['id'] as String,
        timestamp: DateTime.parse(entry['timestamp'] as String),
        message: entry['message'] as String?,
        fileCount: entry['fileCount'] as int? ?? 0,
      )).toList();
    } catch (e) {
      print('Warning: Failed to read index: $e');
      return [];
    }
  }

  @override
  Future<void> restore({
    required String snapshotId,
    required String targetPath,
  }) async {
    final snapshotPath = timestamped 
        ? _snapshotPath(snapshotId)
        : basePath;

    final sourceDir = Directory(snapshotPath);
    if (!await sourceDir.exists()) {
      throw ArgumentError('Snapshot not found: $snapshotId');
    }

    // Copy files
    await for (final entity in sourceDir.list(recursive: true)) {
      if (entity is! File) continue;
      
      final relativePath = entity.path.substring(snapshotPath.length + 1);
      
      // Skip metadata
      if (relativePath.startsWith('.bulletproof')) continue;

      final targetFile = File(p.join(targetPath, relativePath));
      await targetFile.parent.create(recursive: true);
      await entity.copy(targetFile.path);
    }
  }
}

