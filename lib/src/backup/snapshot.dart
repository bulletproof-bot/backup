import 'dart:convert';
import 'dart:io';
import 'package:crypto/crypto.dart';
import 'package:intl/intl.dart';

/// Represents a point-in-time backup snapshot
class Snapshot {
  final String id;
  final DateTime timestamp;
  final Map<String, FileSnapshot> files;
  final String? message;

  Snapshot({
    required this.id,
    required this.timestamp,
    required this.files,
    this.message,
  });

  /// Generate a snapshot ID from timestamp
  static String generateId(DateTime dt) {
    return DateFormat('yyyyMMdd-HHmmss').format(dt);
  }

  /// Create a snapshot from a directory
  static Future<Snapshot> fromDirectory(
    String path, {
    List<String> exclude = const [],
    String? message,
  }) async {
    final timestamp = DateTime.now();
    final id = generateId(timestamp);
    final files = <String, FileSnapshot>{};

    final dir = Directory(path);
    if (!dir.existsSync()) {
      throw ArgumentError('Directory does not exist: $path');
    }

    await for (final entity in dir.list(recursive: true, followLinks: false)) {
      if (entity is File) {
        final relativePath = entity.path.substring(path.length + 1);
        
        // Check exclusions
        if (_shouldExclude(relativePath, exclude)) continue;

        final fileSnapshot = await FileSnapshot.fromFile(entity, relativePath);
        files[relativePath] = fileSnapshot;
      }
    }

    return Snapshot(
      id: id,
      timestamp: timestamp,
      files: files,
      message: message,
    );
  }

  /// Calculate diff between this snapshot and another
  SnapshotDiff diff(Snapshot other) {
    final added = <String>[];
    final removed = <String>[];
    final modified = <String>[];

    // Find added and modified files
    for (final path in files.keys) {
      if (!other.files.containsKey(path)) {
        added.add(path);
      } else if (files[path]!.hash != other.files[path]!.hash) {
        modified.add(path);
      }
    }

    // Find removed files
    for (final path in other.files.keys) {
      if (!files.containsKey(path)) {
        removed.add(path);
      }
    }

    return SnapshotDiff(
      from: other.id,
      to: id,
      added: added,
      removed: removed,
      modified: modified,
    );
  }

  /// Serialize to JSON
  Map<String, dynamic> toJson() => {
    'id': id,
    'timestamp': timestamp.toIso8601String(),
    'message': message,
    'files': files.map((k, v) => MapEntry(k, v.toJson())),
  };

  /// Deserialize from JSON
  factory Snapshot.fromJson(Map<String, dynamic> json) {
    final filesJson = json['files'] as Map<String, dynamic>;
    return Snapshot(
      id: json['id'] as String,
      timestamp: DateTime.parse(json['timestamp'] as String),
      message: json['message'] as String?,
      files: filesJson.map(
        (k, v) => MapEntry(k, FileSnapshot.fromJson(v as Map<String, dynamic>)),
      ),
    );
  }

  static bool _shouldExclude(String path, List<String> patterns) {
    for (final pattern in patterns) {
      if (pattern.endsWith('/')) {
        // Directory pattern
        if (path.startsWith(pattern) || path.contains('/$pattern')) {
          return true;
        }
      } else if (pattern.startsWith('*.')) {
        // Extension pattern
        if (path.endsWith(pattern.substring(1))) {
          return true;
        }
      } else if (pattern.contains('**')) {
        // Glob pattern - simplified matching
        final regex = RegExp(
          pattern.replaceAll('**/', '.*').replaceAll('*', '[^/]*'),
        );
        if (regex.hasMatch(path)) {
          return true;
        }
      } else if (path == pattern || path.endsWith('/$pattern')) {
        return true;
      }
    }
    return false;
  }

  @override
  String toString() => 'Snapshot($id, ${files.length} files)';
}

/// Represents a single file in a snapshot
class FileSnapshot {
  final String path;
  final String hash;
  final int size;
  final DateTime modified;

  FileSnapshot({
    required this.path,
    required this.hash,
    required this.size,
    required this.modified,
  });

  /// Create from an actual file
  static Future<FileSnapshot> fromFile(File file, String relativePath) async {
    final bytes = await file.readAsBytes();
    final hash = sha256.convert(bytes).toString();
    final stat = await file.stat();

    return FileSnapshot(
      path: relativePath,
      hash: hash,
      size: stat.size,
      modified: stat.modified,
    );
  }

  Map<String, dynamic> toJson() => {
    'path': path,
    'hash': hash,
    'size': size,
    'modified': modified.toIso8601String(),
  };

  factory FileSnapshot.fromJson(Map<String, dynamic> json) => FileSnapshot(
    path: json['path'] as String,
    hash: json['hash'] as String,
    size: json['size'] as int,
    modified: DateTime.parse(json['modified'] as String),
  );
}

/// Represents changes between two snapshots
class SnapshotDiff {
  final String from;
  final String to;
  final List<String> added;
  final List<String> removed;
  final List<String> modified;

  SnapshotDiff({
    required this.from,
    required this.to,
    required this.added,
    required this.removed,
    required this.modified,
  });

  bool get isEmpty => added.isEmpty && removed.isEmpty && modified.isEmpty;
  int get totalChanges => added.length + removed.length + modified.length;

  @override
  String toString() {
    if (isEmpty) return 'No changes';
    
    final parts = <String>[];
    if (added.isNotEmpty) parts.add('+${added.length} added');
    if (modified.isNotEmpty) parts.add('~${modified.length} modified');
    if (removed.isNotEmpty) parts.add('-${removed.length} removed');
    return parts.join(', ');
  }

  void printDetailed() {
    if (isEmpty) {
      print('No changes detected.');
      return;
    }

    if (added.isNotEmpty) {
      print('\n  Added:');
      for (final f in added) {
        print('    + $f');
      }
    }
    if (modified.isNotEmpty) {
      print('\n  Modified:');
      for (final f in modified) {
        print('    ~ $f');
      }
    }
    if (removed.isNotEmpty) {
      print('\n  Removed:');
      for (final f in removed) {
        print('    - $f');
      }
    }
  }
}

