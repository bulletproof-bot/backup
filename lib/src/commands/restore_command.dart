import 'dart:io';
import 'package:args/command_runner.dart';

import '../config/config.dart';
import '../backup/backup_engine.dart';

/// Restore from a backup.
class RestoreCommand extends Command<int> {
  @override
  final name = 'restore';

  @override
  final description = 'Restore your OpenClaw agent from a backup';

  RestoreCommand() {
    argParser
      ..addFlag(
        'dry-run',
        abbr: 'n',
        help: 'Show what would be restored without doing it',
      )
      ..addFlag(
        'force',
        abbr: 'f',
        help: 'Skip confirmation prompt',
      );
  }

  @override
  Future<int> run() async {
    final dryRun = argResults!['dry-run'] as bool;
    final force = argResults!['force'] as bool;
    final rest = argResults!.rest;

    if (rest.isEmpty) {
      print('Usage: bulletproof restore <snapshot-id>');
      print('');
      print('Run "bulletproof history" to see available snapshots.');
      return 1;
    }

    final snapshotId = rest.first;

    if (!Config.exists) {
      print('Error: Not configured. Run: bulletproof init');
      return 1;
    }

    final config = Config.load();

    if (!dryRun && !force) {
      print('');
      print('⚠️  WARNING: This will overwrite your current OpenClaw installation!');
      print('   Path: ${config.openclawPath}');
      print('');
      print('   A safety backup will be created first, but please make sure');
      print('   you understand what you\'re doing.');
      print('');
      stdout.write('Type "restore" to confirm: ');
      final confirm = stdin.readLineSync()?.trim();
      
      if (confirm != 'restore') {
        print('Aborted.');
        return 1;
      }
    }

    try {
      final engine = BackupEngine(config);
      await engine.restore(snapshotId, dryRun: dryRun);
      return 0;
    } catch (e) {
      print('Error: $e');
      return 1;
    }
  }
}

