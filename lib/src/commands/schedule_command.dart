import 'dart:io';
import 'package:args/command_runner.dart';

import '../config/config.dart';
import '../scheduler/scheduler.dart';

/// Manage scheduled backups.
class ScheduleCommand extends Command<int> {
  @override
  final name = 'schedule';

  @override
  final description = 'Manage scheduled backups';

  ScheduleCommand() {
    addSubcommand(_ScheduleEnableCommand());
    addSubcommand(_ScheduleDisableCommand());
    addSubcommand(_ScheduleStatusCommand());
  }
}

class _ScheduleEnableCommand extends Command<int> {
  @override
  final name = 'enable';

  @override
  final description = 'Enable scheduled backups';

  _ScheduleEnableCommand() {
    argParser.addOption(
      'time',
      abbr: 't',
      help: 'Time to run backup (HH:MM)',
      defaultsTo: '03:00',
    );
  }

  @override
  Future<int> run() async {
    final time = argResults!['time'] as String;

    if (!RegExp(r'^\d{2}:\d{2}$').hasMatch(time)) {
      print('Invalid time format. Use HH:MM (e.g., 03:00)');
      return 1;
    }

    if (!Config.exists) {
      print('Error: Not configured. Run: bulletproof init');
      return 1;
    }

    final parts = time.split(':');
    final hour = int.parse(parts[0]);
    final minute = int.parse(parts[1]);

    if (hour < 0 || hour > 23 || minute < 0 || minute > 59) {
      print('Invalid time: $time');
      return 1;
    }

    try {
      final scheduler = Scheduler();
      await scheduler.enable(hour, minute);

      // Update config
      var config = Config.load();
      config = config.copyWith(
        schedule: ScheduleConfig(enabled: true, time: time),
      );
      config.save();

      _printPlatformInfo();
      return 0;
    } catch (e) {
      print('Error: $e');
      return 1;
    }
  }

  void _printPlatformInfo() {
    print('');
    if (Platform.isMacOS) {
      print('ℹ️  Using macOS launchd');
      print('   View logs: cat /tmp/bulletproof.log');
      print('   Status: launchctl list ai.bulletproof.backup');
    } else if (Platform.isWindows) {
      print('ℹ️  Using Windows Task Scheduler');
      print('   Status: schtasks /query /tn BulletproofBackup');
    } else {
      print('ℹ️  Using cron');
      print('   View schedule: crontab -l | grep bulletproof');
    }
  }
}

class _ScheduleDisableCommand extends Command<int> {
  @override
  final name = 'disable';

  @override
  final description = 'Disable scheduled backups';

  @override
  Future<int> run() async {
    try {
      final scheduler = Scheduler();
      await scheduler.disable();

      if (Config.exists) {
        var config = Config.load();
        config = config.copyWith(
          schedule: ScheduleConfig(enabled: false, time: config.schedule.time),
        );
        config.save();
      }

      print('✓ Scheduled backups disabled');
      return 0;
    } catch (e) {
      print('Error: $e');
      return 1;
    }
  }
}

class _ScheduleStatusCommand extends Command<int> {
  @override
  final name = 'status';

  @override
  final description = 'Show scheduled backup status';

  @override
  Future<int> run() async {
    try {
      final scheduler = Scheduler();
      final status = await scheduler.status();

      print('Scheduled Backups');
      print('─' * 40);

      if (status.enabled) {
        print('  Status: ✓ Enabled');
        if (status.schedule != null) {
          print('  Schedule: ${status.schedule}');
        }
        if (status.lastRun != null) {
          print('  Last run: ${status.lastRun}');
        }
        if (status.nextRun != null) {
          print('  Next run: ${status.nextRun}');
        }
      } else {
        print('  Status: ✗ Disabled');
        print('');
        print('  Enable with: bulletproof schedule enable');
      }

      if (status.error != null) {
        print('  Error: ${status.error}');
      }

      return 0;
    } catch (e) {
      print('Error: $e');
      return 1;
    }
  }
}

