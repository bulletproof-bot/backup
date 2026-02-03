import 'package:args/command_runner.dart';

import '../config/config.dart';
import '../backup/backup_engine.dart';

/// Show changes since last backup.
class DiffCommand extends Command<int> {
  @override
  final name = 'diff';

  @override
  final description = 'Show changes since last backup';

  @override
  Future<int> run() async {
    if (!Config.exists) {
      print('Error: Not configured. Run: bulletproof init');
      return 1;
    }

    final config = Config.load();

    try {
      final engine = BackupEngine(config);
      final diff = await engine.showDiff();
      
      if (diff == null) {
        print('No previous backup to compare against.');
        print('Run: bulletproof backup');
        return 0;
      }

      print('Changes since last backup:');
      print('');
      
      if (diff.isEmpty) {
        print('  No changes detected.');
      } else {
        diff.printDetailed();
      }

      return 0;
    } catch (e) {
      print('Error: $e');
      return 1;
    }
  }
}

