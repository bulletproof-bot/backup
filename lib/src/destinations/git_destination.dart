import 'dart:convert';
import 'dart:io';
import 'package:path/path.dart' as p;

import '../backup/snapshot.dart';
import '../backup/backup_engine.dart';
import 'destination.dart';

/// Git repository destination.
/// 
/// Stores backups as commits in a git repository.
/// Each backup is a commit with all files copied to the repo.
/// Automatically handles:
/// - Detecting if destination is a git repo
/// - Initializing repo if needed
/// - Committing changes
/// - Pushing to remote (if configured)
class GitDestination implements Destination {
  final String repoPath;
  bool _validated = false;
  bool _isRemote = false;

  GitDestination(this.repoPath) {
    _isRemote = repoPath.startsWith('git@') || 
                repoPath.startsWith('https://') ||
                repoPath.startsWith('ssh://');
  }

  String get _localPath {
    if (_isRemote) {
      // Clone to a local cache directory
      final home = Platform.environment['HOME'] ?? 
                   Platform.environment['USERPROFILE'] ?? '.';
      final repoName = p.basenameWithoutExtension(
        repoPath.split('/').last.replaceAll('.git', '')
      );
      return p.join(home, '.cache', 'bulletproof', 'repos', repoName);
    }
    return repoPath;
  }

  @override
  Future<bool> validate() async {
    if (_validated) return true;

    if (_isRemote) {
      // Clone or pull the remote repo
      await _ensureCloned();
    } else {
      // Check if local path exists and is a git repo
      final gitDir = Directory(p.join(_localPath, '.git'));
      if (!gitDir.existsSync()) {
        // Initialize new repo
        await _initRepo();
      }
    }

    _validated = true;
    return true;
  }

  Future<void> _ensureCloned() async {
    final localDir = Directory(_localPath);
    
    if (localDir.existsSync()) {
      // Pull latest
      print('  Pulling latest from remote...');
      await _runGit(['pull', '--rebase'], workingDir: _localPath);
    } else {
      // Clone
      print('  Cloning repository...');
      await localDir.create(recursive: true);
      await _runGit(['clone', repoPath, _localPath]);
    }
  }

  Future<void> _initRepo() async {
    final dir = Directory(_localPath);
    if (!dir.existsSync()) {
      await dir.create(recursive: true);
    }
    
    print('  Initializing git repository...');
    await _runGit(['init'], workingDir: _localPath);
    
    // Create initial .gitignore
    final gitignore = File(p.join(_localPath, '.gitignore'));
    await gitignore.writeAsString('.DS_Store\n*.log\n');
    
    await _runGit(['add', '.gitignore'], workingDir: _localPath);
    await _runGit(['commit', '-m', 'Initial commit'], workingDir: _localPath);
  }

  @override
  Future<void> save({
    required String sourcePath,
    required Snapshot snapshot,
    required String message,
  }) async {
    await validate();

    // Copy all files from source to repo
    print('  Copying files to backup repository...');
    await _syncFiles(sourcePath, _localPath, snapshot);

    // Save snapshot metadata
    final metaFile = File(p.join(_localPath, '.bulletproof', 'snapshot.json'));
    await metaFile.parent.create(recursive: true);
    await metaFile.writeAsString(
      const JsonEncoder.withIndent('  ').convert(snapshot.toJson())
    );

    // Stage all changes
    await _runGit(['add', '-A'], workingDir: _localPath);

    // Check if there are changes to commit
    final status = await _runGit(['status', '--porcelain'], workingDir: _localPath);
    if (status.trim().isEmpty) {
      print('  No changes to commit.');
      return;
    }

    // Commit
    await _runGit(
      ['commit', '-m', message],
      workingDir: _localPath,
    );

    // Tag with snapshot ID
    await _runGit(
      ['tag', '-a', snapshot.id, '-m', message],
      workingDir: _localPath,
    );

    // Push if remote
    if (_isRemote) {
      print('  Pushing to remote...');
      await _runGit(['push', '--follow-tags'], workingDir: _localPath);
    }
  }

  Future<void> _syncFiles(
    String sourcePath, 
    String destPath, 
    Snapshot snapshot,
  ) async {
    // Clear existing files (except .git and .bulletproof)
    final destDir = Directory(destPath);
    await for (final entity in destDir.list()) {
      final name = p.basename(entity.path);
      if (name == '.git' || name == '.bulletproof' || name == '.gitignore') {
        continue;
      }
      if (entity is Directory) {
        await entity.delete(recursive: true);
      } else {
        await entity.delete();
      }
    }

    // Copy files from snapshot
    for (final filePath in snapshot.files.keys) {
      final sourceFile = File(p.join(sourcePath, filePath));
      final destFile = File(p.join(destPath, filePath));
      
      if (await sourceFile.exists()) {
        await destFile.parent.create(recursive: true);
        await sourceFile.copy(destFile.path);
      }
    }
  }

  @override
  Future<Snapshot?> getLastSnapshot() async {
    await validate();
    
    final metaFile = File(p.join(_localPath, '.bulletproof', 'snapshot.json'));
    if (!await metaFile.exists()) {
      return null;
    }

    try {
      final content = await metaFile.readAsString();
      final json = jsonDecode(content) as Map<String, dynamic>;
      return Snapshot.fromJson(json);
    } catch (e) {
      print('Warning: Failed to read snapshot metadata: $e');
      return null;
    }
  }

  @override
  Future<Snapshot?> getSnapshot(String id) async {
    await validate();

    // Check out the tag
    try {
      await _runGit(['checkout', id], workingDir: _localPath);
      final snapshot = await getLastSnapshot();
      // Return to main branch
      await _runGit(['checkout', 'main'], workingDir: _localPath);
      return snapshot;
    } catch (e) {
      // Try master branch if main doesn't exist
      try {
        await _runGit(['checkout', 'master'], workingDir: _localPath);
      } catch (_) {}
      return null;
    }
  }

  @override
  Future<List<SnapshotInfo>> listSnapshots() async {
    await validate();

    final result = await _runGit(
      ['tag', '-l', '--sort=-version:refname', '--format=%(refname:short)|%(creatordate:iso)|%(subject)'],
      workingDir: _localPath,
    );

    final snapshots = <SnapshotInfo>[];
    for (final line in result.split('\n')) {
      if (line.trim().isEmpty) continue;
      
      final parts = line.split('|');
      if (parts.isEmpty) continue;

      final id = parts[0];
      DateTime? timestamp;
      String? message;

      if (parts.length > 1) {
        try {
          timestamp = DateTime.parse(parts[1].trim());
        } catch (_) {}
      }
      if (parts.length > 2) {
        message = parts[2];
      }

      snapshots.add(SnapshotInfo(
        id: id,
        timestamp: timestamp ?? DateTime.now(),
        message: message,
        fileCount: 0, // Would need to check out tag to count
      ));
    }

    return snapshots;
  }

  @override
  Future<void> restore({
    required String snapshotId,
    required String targetPath,
  }) async {
    await validate();

    // Check out the specific tag
    await _runGit(['checkout', snapshotId], workingDir: _localPath);

    // Copy files to target
    final sourceDir = Directory(_localPath);
    await for (final entity in sourceDir.list(recursive: true)) {
      if (entity is! File) continue;
      
      final relativePath = entity.path.substring(_localPath.length + 1);
      
      // Skip git internals and metadata
      if (relativePath.startsWith('.git') || 
          relativePath.startsWith('.bulletproof') ||
          relativePath == '.gitignore') {
        continue;
      }

      final targetFile = File(p.join(targetPath, relativePath));
      await targetFile.parent.create(recursive: true);
      await entity.copy(targetFile.path);
    }

    // Return to main branch
    try {
      await _runGit(['checkout', 'main'], workingDir: _localPath);
    } catch (_) {
      await _runGit(['checkout', 'master'], workingDir: _localPath);
    }
  }

  Future<String> _runGit(List<String> args, {String? workingDir}) async {
    final result = await Process.run(
      'git',
      args,
      workingDirectory: workingDir,
    );

    if (result.exitCode != 0) {
      final stderr = result.stderr.toString();
      // Ignore "nothing to commit" messages
      if (!stderr.contains('nothing to commit')) {
        throw ProcessException('git', args, stderr, result.exitCode);
      }
    }

    return result.stdout.toString();
  }
}

