import 'dart:io';
import 'package:path/path.dart' as p;

/// File system utilities.

/// Expand ~ to home directory
String expandPath(String path) {
  if (path.startsWith('~/')) {
    final home = Platform.environment['HOME'] ?? 
                 Platform.environment['USERPROFILE'] ?? '.';
    return path.replaceFirst('~', home);
  }
  return path;
}

/// Get file size in human readable format
String formatSize(int bytes) {
  if (bytes < 1024) return '$bytes B';
  if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
  if (bytes < 1024 * 1024 * 1024) {
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
  }
  return '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(1)} GB';
}

/// Copy directory recursively
Future<void> copyDirectory(String from, String to, {
  List<String> exclude = const [],
}) async {
  final fromDir = Directory(from);
  
  await for (final entity in fromDir.list(recursive: true, followLinks: false)) {
    final relativePath = entity.path.substring(from.length + 1);
    
    // Check exclusions
    if (_shouldExclude(relativePath, exclude)) continue;
    
    final targetPath = p.join(to, relativePath);
    
    if (entity is Directory) {
      await Directory(targetPath).create(recursive: true);
    } else if (entity is File) {
      await Directory(p.dirname(targetPath)).create(recursive: true);
      await entity.copy(targetPath);
    }
  }
}

/// Check if path matches any exclusion pattern
bool _shouldExclude(String path, List<String> patterns) {
  for (final pattern in patterns) {
    if (pattern.endsWith('/')) {
      // Directory pattern
      final dirName = pattern.substring(0, pattern.length - 1);
      if (path == dirName || 
          path.startsWith('$dirName/') || 
          path.contains('/$dirName/')) {
        return true;
      }
    } else if (pattern.startsWith('*.')) {
      // Extension pattern
      if (path.endsWith(pattern.substring(1))) {
        return true;
      }
    } else if (pattern.contains('**')) {
      // Glob pattern
      final regex = RegExp(
        pattern
          .replaceAll('.', r'\.')
          .replaceAll('**/', '.*')
          .replaceAll('*', '[^/]*'),
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

/// Calculate total size of directory
Future<int> directorySize(String path) async {
  int size = 0;
  final dir = Directory(path);
  
  await for (final entity in dir.list(recursive: true, followLinks: false)) {
    if (entity is File) {
      size += await entity.length();
    }
  }
  
  return size;
}

/// Count files in directory
Future<int> countFiles(String path) async {
  int count = 0;
  final dir = Directory(path);
  
  await for (final entity in dir.list(recursive: true, followLinks: false)) {
    if (entity is File) {
      count++;
    }
  }
  
  return count;
}

