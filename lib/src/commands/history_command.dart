import 'package:args/command_runner.dart';
import 'package:intl/intl.dart';

import '../config/config.dart';
import '../backup/backup_engine.dart';

/// List all backups.
class HistoryCommand extends Command<int> {
  @override
  final name = 'history';

  @override
  final description = 'List all backups';

  HistoryCommand() {
    argParser.addOption(
      'limit',
      abbr: 'l',
      help: 'Maximum number of backups to show',
      defaultsTo: '20',
    );
  }

  @override
  Future<int> run() async {
    final limit = int.tryParse(argResults!['limit'] as String) ?? 20;

    if (!Config.exists) {
      print('Error: Not configured. Run: bulletproof init');
      return 1;
    }

    final config = Config.load();

    try {
      final engine = BackupEngine(config);
      final snapshots = await engine.listBackups();

      if (snapshots.isEmpty) {
        print('No backups found.');
        print('Run: bulletproof backup');
        return 0;
      }

      print('Backup History');
      print('═' * 60);
      print('');

      final dateFormat = DateFormat('yyyy-MM-dd HH:mm');
      final toShow = snapshots.take(limit).toList();

      for (final snapshot in toShow) {
        final date = dateFormat.format(snapshot.timestamp);
        final msg = snapshot.message ?? '';
        final files = snapshot.fileCount > 0 ? ' (${snapshot.fileCount} files)' : '';
        
        print('  ${snapshot.id}');
        print('    $date$files');
        if (msg.isNotEmpty) {
          print('    $msg');
        }
        print('');
      }

      if (snapshots.length > limit) {
        print('  ... and ${snapshots.length - limit} more');
        print('  Use --limit to show more');
      }

      print('');
      print('To restore: bulletproof restore <snapshot-id>');

      return 0;
    } catch (e) {
      print('Error: $e');
      return 1;
    }
  }
}

