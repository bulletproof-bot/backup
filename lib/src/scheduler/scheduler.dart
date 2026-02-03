import 'dart:io';
import 'package:path/path.dart' as p;

/// Platform-specific scheduler for automatic backups.
/// 
/// Supports:
/// - Linux: cron
/// - macOS: launchd
/// - Windows: Task Scheduler
abstract class Scheduler {
  /// Create appropriate scheduler for current platform
  factory Scheduler() {
    if (Platform.isMacOS) {
      return LaunchdScheduler();
    } else if (Platform.isWindows) {
      return WindowsScheduler();
    } else {
      return CronScheduler();
    }
  }

  /// Enable scheduled backups
  Future<void> enable(int hour, int minute);

  /// Disable scheduled backups
  Future<void> disable();

  /// Check if scheduled backups are enabled
  Future<bool> isEnabled();

  /// Get current schedule status
  Future<ScheduleStatus> status();
}

/// Status of scheduled backups
class ScheduleStatus {
  final bool enabled;
  final String? schedule;
  final DateTime? lastRun;
  final DateTime? nextRun;
  final String? error;

  ScheduleStatus({
    required this.enabled,
    this.schedule,
    this.lastRun,
    this.nextRun,
    this.error,
  });
}

/// Linux cron-based scheduler
class CronScheduler implements Scheduler {
  static const _marker = '# bulletproof-backup';

  String get _bulletproofPath {
    // Find bulletproof executable
    final result = Process.runSync('which', ['bulletproof']);
    if (result.exitCode == 0) {
      return result.stdout.toString().trim();
    }
    // Fallback to common locations
    final home = Platform.environment['HOME'] ?? '/home';
    final paths = [
      '/usr/local/bin/bulletproof',
      '$home/.pub-cache/bin/bulletproof',
      '$home/.local/bin/bulletproof',
    ];
    for (final path in paths) {
      if (File(path).existsSync()) return path;
    }
    return 'bulletproof';
  }

  @override
  Future<void> enable(int hour, int minute) async {
    // Remove existing entry first
    await disable();

    // Add new cron entry
    final cronLine = '$minute $hour * * * $_bulletproofPath backup --quiet $_marker';
    
    final result = await Process.run('bash', ['-c', '''
      (crontab -l 2>/dev/null || true; echo "$cronLine") | crontab -
    ''']);

    if (result.exitCode != 0) {
      throw Exception('Failed to add cron job: ${result.stderr}');
    }

    print('✓ Scheduled daily backup at ${hour.toString().padLeft(2, '0')}:${minute.toString().padLeft(2, '0')}');
  }

  @override
  Future<void> disable() async {
    final result = await Process.run('bash', ['-c', '''
      crontab -l 2>/dev/null | grep -v "$_marker" | crontab - 2>/dev/null || true
    ''']);

    if (result.exitCode != 0 && !result.stderr.toString().contains('no crontab')) {
      throw Exception('Failed to remove cron job: ${result.stderr}');
    }
  }

  @override
  Future<bool> isEnabled() async {
    final result = await Process.run('bash', ['-c', 'crontab -l 2>/dev/null']);
    return result.stdout.toString().contains(_marker);
  }

  @override
  Future<ScheduleStatus> status() async {
    final result = await Process.run('bash', ['-c', 'crontab -l 2>/dev/null']);
    final crontab = result.stdout.toString();
    
    if (!crontab.contains(_marker)) {
      return ScheduleStatus(enabled: false);
    }

    // Parse the cron line
    final lines = crontab.split('\n');
    final cronLine = lines.firstWhere(
      (l) => l.contains(_marker),
      orElse: () => '',
    );

    if (cronLine.isEmpty) {
      return ScheduleStatus(enabled: false);
    }

    final parts = cronLine.trim().split(RegExp(r'\s+'));
    if (parts.length >= 2) {
      final minute = parts[0];
      final hour = parts[1];
      return ScheduleStatus(
        enabled: true,
        schedule: 'Daily at $hour:${minute.padLeft(2, '0')}',
      );
    }

    return ScheduleStatus(enabled: true);
  }
}

/// macOS launchd-based scheduler
class LaunchdScheduler implements Scheduler {
  static const _plistName = 'ai.bulletproof.backup';
  
  String get _plistPath {
    final home = Platform.environment['HOME'] ?? '/Users';
    return p.join(home, 'Library', 'LaunchAgents', '$_plistName.plist');
  }

  String get _bulletproofPath {
    final result = Process.runSync('which', ['bulletproof']);
    if (result.exitCode == 0) {
      return result.stdout.toString().trim();
    }
    final home = Platform.environment['HOME'] ?? '/Users';
    return '$home/.pub-cache/bin/bulletproof';
  }

  @override
  Future<void> enable(int hour, int minute) async {
    await disable();

    final plist = '''<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$_plistName</string>
    <key>ProgramArguments</key>
    <array>
        <string>$_bulletproofPath</string>
        <string>backup</string>
        <string>--quiet</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>$hour</integer>
        <key>Minute</key>
        <integer>$minute</integer>
    </dict>
    <key>StandardOutPath</key>
    <string>/tmp/bulletproof.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/bulletproof.err</string>
</dict>
</plist>
''';

    final file = File(_plistPath);
    await file.parent.create(recursive: true);
    await file.writeAsString(plist);

    // Load the agent
    final result = await Process.run('launchctl', ['load', _plistPath]);
    if (result.exitCode != 0) {
      throw Exception('Failed to load launch agent: ${result.stderr}');
    }

    print('✓ Scheduled daily backup at ${hour.toString().padLeft(2, '0')}:${minute.toString().padLeft(2, '0')}');
  }

  @override
  Future<void> disable() async {
    final file = File(_plistPath);
    if (await file.exists()) {
      // Unload first
      await Process.run('launchctl', ['unload', _plistPath]);
      await file.delete();
    }
  }

  @override
  Future<bool> isEnabled() async {
    return File(_plistPath).existsSync();
  }

  @override
  Future<ScheduleStatus> status() async {
    final file = File(_plistPath);
    if (!await file.exists()) {
      return ScheduleStatus(enabled: false);
    }

    // Check if loaded
    final result = await Process.run('launchctl', ['list', _plistName]);
    final loaded = result.exitCode == 0;

    return ScheduleStatus(
      enabled: loaded,
      schedule: 'Daily (see launchctl list $_plistName)',
    );
  }
}

/// Windows Task Scheduler
class WindowsScheduler implements Scheduler {
  static const _taskName = 'BulletproofBackup';

  String get _bulletproofPath {
    final result = Process.runSync('where', ['bulletproof.exe']);
    if (result.exitCode == 0) {
      return result.stdout.toString().trim().split('\n').first;
    }
    final appData = Platform.environment['LOCALAPPDATA'] ?? '';
    return p.join(appData, 'Pub', 'Cache', 'bin', 'bulletproof.bat');
  }

  @override
  Future<void> enable(int hour, int minute) async {
    await disable();

    final time = '${hour.toString().padLeft(2, '0')}:${minute.toString().padLeft(2, '0')}';
    
    final result = await Process.run('schtasks', [
      '/create',
      '/tn', _taskName,
      '/tr', '"$_bulletproofPath" backup --quiet',
      '/sc', 'daily',
      '/st', time,
      '/f',
    ]);

    if (result.exitCode != 0) {
      throw Exception('Failed to create scheduled task: ${result.stderr}');
    }

    print('✓ Scheduled daily backup at $time');
  }

  @override
  Future<void> disable() async {
    await Process.run('schtasks', ['/delete', '/tn', _taskName, '/f']);
  }

  @override
  Future<bool> isEnabled() async {
    final result = await Process.run('schtasks', ['/query', '/tn', _taskName]);
    return result.exitCode == 0;
  }

  @override
  Future<ScheduleStatus> status() async {
    final result = await Process.run('schtasks', [
      '/query', '/tn', _taskName, '/fo', 'list', '/v'
    ]);

    if (result.exitCode != 0) {
      return ScheduleStatus(enabled: false);
    }

    final output = result.stdout.toString();
    DateTime? nextRun;
    DateTime? lastRun;

    // Parse output for dates
    for (final line in output.split('\n')) {
      if (line.contains('Next Run Time:')) {
        final value = line.split(':').skip(1).join(':').trim();
        if (value != 'N/A') {
          try {
            nextRun = DateTime.parse(value);
          } catch (_) {}
        }
      }
      if (line.contains('Last Run Time:')) {
        final value = line.split(':').skip(1).join(':').trim();
        if (value != 'N/A') {
          try {
            lastRun = DateTime.parse(value);
          } catch (_) {}
        }
      }
    }

    return ScheduleStatus(
      enabled: true,
      schedule: 'Daily',
      nextRun: nextRun,
      lastRun: lastRun,
    );
  }
}

