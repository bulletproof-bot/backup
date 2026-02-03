import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:path/path.dart' as p;

/// Service management (start/stop/status/logs).
/// 
/// For most users, `bulletproof schedule` is sufficient.
/// This command is for advanced users who want to run
/// bulletproof as a persistent service with file watching.
class ServiceCommand extends Command<int> {
  @override
  final name = 'service';

  @override
  final description = 'Manage bulletproof as a service (advanced)';

  ServiceCommand() {
    addSubcommand(_ServiceStartCommand());
    addSubcommand(_ServiceStopCommand());
    addSubcommand(_ServiceStatusCommand());
    addSubcommand(_ServiceLogsCommand());
    addSubcommand(_ServiceInstallCommand());
    addSubcommand(_ServiceUninstallCommand());
  }
}

String get _serviceName => 'bulletproof';

String get _logPath {
  if (Platform.isMacOS || Platform.isLinux) {
    return '/tmp/bulletproof.log';
  }
  final temp = Platform.environment['TEMP'] ?? 'C:\\Temp';
  return p.join(temp, 'bulletproof.log');
}

class _ServiceStartCommand extends Command<int> {
  @override
  final name = 'start';

  @override
  final description = 'Start the bulletproof service';

  @override
  Future<int> run() async {
    if (Platform.isMacOS) {
      final result = await Process.run('launchctl', [
        'start', 'ai.bulletproof.backup'
      ]);
      if (result.exitCode != 0) {
        print('Error: ${result.stderr}');
        print('');
        print('Service may not be installed. Run:');
        print('  bulletproof schedule enable');
        return 1;
      }
      print('✓ Service started');
    } else if (Platform.isWindows) {
      print('On Windows, use Task Scheduler to manage the service.');
      print('  bulletproof schedule enable');
      return 1;
    } else {
      // Linux - try systemd first, then fall back to manual
      final result = await Process.run('systemctl', [
        '--user', 'start', _serviceName
      ]);
      if (result.exitCode != 0) {
        print('Error: ${result.stderr}');
        print('');
        print('Service may not be installed. Run:');
        print('  bulletproof service install');
        return 1;
      }
      print('✓ Service started');
    }
    return 0;
  }
}

class _ServiceStopCommand extends Command<int> {
  @override
  final name = 'stop';

  @override
  final description = 'Stop the bulletproof service';

  @override
  Future<int> run() async {
    if (Platform.isMacOS) {
      await Process.run('launchctl', ['stop', 'ai.bulletproof.backup']);
      print('✓ Service stopped');
    } else if (Platform.isWindows) {
      print('On Windows, use Task Scheduler to manage the service.');
      return 1;
    } else {
      await Process.run('systemctl', ['--user', 'stop', _serviceName]);
      print('✓ Service stopped');
    }
    return 0;
  }
}

class _ServiceStatusCommand extends Command<int> {
  @override
  final name = 'status';

  @override
  final description = 'Show service status';

  @override
  Future<int> run() async {
    if (Platform.isMacOS) {
      final result = await Process.run('launchctl', [
        'list', 'ai.bulletproof.backup'
      ]);
      if (result.exitCode != 0) {
        print('Service not running or not installed.');
        return 0;
      }
      print(result.stdout);
    } else if (Platform.isWindows) {
      final result = await Process.run('schtasks', [
        '/query', '/tn', 'BulletproofBackup', '/v'
      ]);
      if (result.exitCode != 0) {
        print('Service not installed.');
        return 0;
      }
      print(result.stdout);
    } else {
      final result = await Process.run('systemctl', [
        '--user', 'status', _serviceName
      ]);
      print(result.stdout);
      if (result.stderr.toString().isNotEmpty) {
        print(result.stderr);
      }
    }
    return 0;
  }
}

class _ServiceLogsCommand extends Command<int> {
  @override
  final name = 'logs';

  @override
  final description = 'Show service logs';

  _ServiceLogsCommand() {
    argParser
      ..addFlag(
        'follow',
        abbr: 'f',
        help: 'Follow log output',
      )
      ..addOption(
        'lines',
        abbr: 'n',
        help: 'Number of lines to show',
        defaultsTo: '50',
      );
  }

  @override
  Future<int> run() async {
    final follow = argResults!['follow'] as bool;
    final lines = argResults!['lines'] as String;

    if (Platform.isLinux) {
      // Use journalctl for systemd
      final args = ['--user', '-u', _serviceName, '-n', lines];
      if (follow) args.add('-f');
      
      final result = await Process.start('journalctl', args);
      await stdout.addStream(result.stdout);
      await stderr.addStream(result.stderr);
      return await result.exitCode;
    }

    // For macOS and Windows, check log file
    final logFile = File(_logPath);
    if (!logFile.existsSync()) {
      print('No logs found at: $_logPath');
      return 0;
    }

    if (follow) {
      // Tail -f equivalent
      final result = await Process.start('tail', ['-f', '-n', lines, _logPath]);
      await stdout.addStream(result.stdout);
      return await result.exitCode;
    } else {
      final result = await Process.run('tail', ['-n', lines, _logPath]);
      print(result.stdout);
      return 0;
    }
  }
}

class _ServiceInstallCommand extends Command<int> {
  @override
  final name = 'install';

  @override
  final description = 'Install systemd service (Linux only)';

  @override
  Future<int> run() async {
    if (!Platform.isLinux) {
      print('This command is only for Linux with systemd.');
      print('');
      print('On macOS/Windows, use:');
      print('  bulletproof schedule enable');
      return 1;
    }

    final home = Platform.environment['HOME'] ?? '';
    final serviceDir = p.join(home, '.config', 'systemd', 'user');
    final servicePath = p.join(serviceDir, 'bulletproof.service');

    // Find bulletproof binary
    final whichResult = await Process.run('which', ['bulletproof']);
    final bulletproofPath = whichResult.exitCode == 0
        ? whichResult.stdout.toString().trim()
        : p.join(home, '.pub-cache', 'bin', 'bulletproof');

    final serviceContent = '''[Unit]
Description=Bulletproof - OpenClaw Agent Backup
After=network.target

[Service]
Type=oneshot
ExecStart=$bulletproofPath backup --quiet
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
''';

    final timerContent = '''[Unit]
Description=Daily bulletproof backup

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true

[Install]
WantedBy=timers.target
''';

    // Create service files
    await Directory(serviceDir).create(recursive: true);
    await File(servicePath).writeAsString(serviceContent);
    await File(p.join(serviceDir, 'bulletproof.timer')).writeAsString(timerContent);

    // Reload and enable
    await Process.run('systemctl', ['--user', 'daemon-reload']);
    await Process.run('systemctl', ['--user', 'enable', 'bulletproof.timer']);
    await Process.run('systemctl', ['--user', 'start', 'bulletproof.timer']);

    print('✓ Systemd service installed and enabled');
    print('');
    print('Commands:');
    print('  bulletproof service status');
    print('  bulletproof service logs');
    print('  systemctl --user status bulletproof.timer');

    return 0;
  }
}

class _ServiceUninstallCommand extends Command<int> {
  @override
  final name = 'uninstall';

  @override
  final description = 'Uninstall systemd service (Linux only)';

  @override
  Future<int> run() async {
    if (!Platform.isLinux) {
      print('This command is only for Linux with systemd.');
      return 1;
    }

    await Process.run('systemctl', ['--user', 'stop', 'bulletproof.timer']);
    await Process.run('systemctl', ['--user', 'disable', 'bulletproof.timer']);

    final home = Platform.environment['HOME'] ?? '';
    final serviceDir = p.join(home, '.config', 'systemd', 'user');
    
    final serviceFile = File(p.join(serviceDir, 'bulletproof.service'));
    final timerFile = File(p.join(serviceDir, 'bulletproof.timer'));

    if (await serviceFile.exists()) await serviceFile.delete();
    if (await timerFile.exists()) await timerFile.delete();

    await Process.run('systemctl', ['--user', 'daemon-reload']);

    print('✓ Systemd service uninstalled');
    return 0;
  }
}

