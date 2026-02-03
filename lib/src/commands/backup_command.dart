import 'dart:io';
import 'package:args/command_runner.dart';

import '../config/config.dart';
import '../backup/backup_engine.dart';

/// Run a backup of the OpenClaw installation.
class BackupCommand extends Command<int> {
  @override
  final name = 'backup';

  @override
  final description = 'Back up your OpenClaw agent';

  BackupCommand() {
    argParser
      ..addFlag(
        'dry-run',
        abbr: 'n',
        help: 'Show what would be backed up without doing it',
      )
      ..addOption(
        'message',
        abbr: 'm',
        help: 'Add a message to this backup',
      );
  }

  @override
  Future<int> run() async {
    final dryRun = argResults!['dry-run'] as bool;
    final message = argResults!['message'] as String?;

    if (!Config.exists) {
      print('Error: Not configured. Run: bulletproof init');
      return 1;
    }

    final config = Config.load();
    
    try {
      final engine = BackupEngine(config);
      final result = await engine.backup(
        dryRun: dryRun,
        message: message,
      );

      if (result.dryRun) {
        result.diff?.printDetailed();
      }

      return 0;
    } catch (e) {
      print('Error: $e');
      return 1;
    }
  }
}

