import 'dart:io';
import 'package:args/command_runner.dart';

import '../config/config.dart';
import '../config/openclaw_defaults.dart';

/// Interactive setup wizard for bulletproof.
class InitCommand extends Command<int> {
  @override
  final name = 'init';

  @override
  final description = 'Set up bulletproof backup for your OpenClaw agent';

  InitCommand() {
    argParser.addFlag(
      'force',
      abbr: 'f',
      help: 'Overwrite existing configuration',
    );
  }

  @override
  Future<int> run() async {
    final force = argResults!['force'] as bool;

    print('');
    print('🛡️  bulletproof backup setup');
    print('   Back up your OpenClaw agent. Track changes. Rollback anytime.');
    print('');

    // Check for existing config
    if (Config.exists && !force) {
      print('⚠️  Configuration already exists at: ${Config.configPath}');
      print('   Use --force to overwrite.');
      return 1;
    }

    // Step 1: Find OpenClaw installation
    print('Step 1: Locate OpenClaw installation');
    print('─' * 50);
    
    String? openclawPath;
    final detected = OpenClawDefaults.detectInstallation();
    
    if (detected != null) {
      print('✓ Found OpenClaw at: $detected');
      stdout.write('Use this location? [Y/n] ');
      final answer = stdin.readLineSync()?.toLowerCase() ?? 'y';
      
      if (answer.isEmpty || answer == 'y' || answer == 'yes') {
        openclawPath = detected;
      }
    } else {
      print('✗ OpenClaw installation not found at default location');
      print('  (Expected: ${OpenClawDefaults.defaultRoot})');
    }

    if (openclawPath == null) {
      stdout.write('Enter OpenClaw path: ');
      openclawPath = stdin.readLineSync()?.trim();
      
      if (openclawPath == null || openclawPath.isEmpty) {
        print('Error: OpenClaw path is required');
        return 1;
      }

      // Validate path
      final configFile = File('$openclawPath/openclaw.json');
      if (!configFile.existsSync()) {
        print('⚠️  Warning: openclaw.json not found at $openclawPath');
        stdout.write('Continue anyway? [y/N] ');
        final answer = stdin.readLineSync()?.toLowerCase() ?? 'n';
        if (answer != 'y' && answer != 'yes') {
          return 1;
        }
      }
    }

    // Show what will be backed up
    print('');
    print('Files to back up:');
    final targets = OpenClawDefaults.getBackupTargets(openclawPath);
    for (final target in targets) {
      final exists = target.exists ? '✓' : '✗';
      final critical = target.critical ? '' : ' (optional)';
      print('  $exists ${target.description}$critical');
    }

    // Step 2: Configure destination
    print('');
    print('Step 2: Configure backup destination');
    print('─' * 50);
    print('');
    print('Where should backups be stored?');
    print('  1. Git repository (recommended - version history + remote backup)');
    print('  2. Local folder (timestamped backups)');
    print('  3. Sync folder (Dropbox/Google Drive/iCloud)');
    print('');
    
    stdout.write('Choice [1-3]: ');
    final destChoice = stdin.readLineSync()?.trim() ?? '1';

    String destType;
    String destPath;

    switch (destChoice) {
      case '1':
        destType = 'git';
        print('');
        print('Enter git repository URL or local path:');
        print('  Examples:');
        print('    git@github.com:username/agent-backup.git');
        print('    https://github.com/username/agent-backup.git');
        print('    ~/backups/openclaw');
        stdout.write('Repository: ');
        destPath = stdin.readLineSync()?.trim() ?? '';
        break;
      
      case '2':
        destType = 'local';
        print('');
        stdout.write('Backup folder path [~/openclaw-backups]: ');
        destPath = stdin.readLineSync()?.trim() ?? '';
        if (destPath.isEmpty) {
          final home = Platform.environment['HOME'] ?? 
                       Platform.environment['USERPROFILE'] ?? '.';
          destPath = '$home/openclaw-backups';
        }
        break;
      
      case '3':
        destType = 'sync';
        print('');
        print('Enter path to your sync folder:');
        print('  Examples:');
        print('    ~/Dropbox/openclaw-backup');
        print('    ~/Google Drive/openclaw-backup');
        print('    ~/Library/Mobile Documents/com~apple~CloudDocs/openclaw-backup');
        stdout.write('Sync folder: ');
        destPath = stdin.readLineSync()?.trim() ?? '';
        break;
      
      default:
        print('Invalid choice');
        return 1;
    }

    if (destPath.isEmpty) {
      print('Error: Destination path is required');
      return 1;
    }

    // Expand ~ to home directory
    if (destPath.startsWith('~/')) {
      final home = Platform.environment['HOME'] ?? 
                   Platform.environment['USERPROFILE'] ?? '.';
      destPath = destPath.replaceFirst('~', home);
    }

    // Step 3: Schedule (optional)
    print('');
    print('Step 3: Automatic backups (optional)');
    print('─' * 50);
    stdout.write('Enable daily automatic backups at 3:00 AM? [Y/n] ');
    final scheduleAnswer = stdin.readLineSync()?.toLowerCase() ?? 'y';
    final enableSchedule = scheduleAnswer.isEmpty || 
                           scheduleAnswer == 'y' || 
                           scheduleAnswer == 'yes';

    // Save configuration
    print('');
    print('Saving configuration...');
    
    final config = Config(
      openclawPath: openclawPath,
      destination: DestinationConfig(type: destType, path: destPath),
      schedule: ScheduleConfig(enabled: enableSchedule, time: '03:00'),
      options: const BackupOptions(),
    );

    config.save();
    print('✓ Configuration saved to: ${Config.configPath}');

    // Run first backup?
    print('');
    stdout.write('Run first backup now? [Y/n] ');
    final backupAnswer = stdin.readLineSync()?.toLowerCase() ?? 'y';
    
    if (backupAnswer.isEmpty || backupAnswer == 'y' || backupAnswer == 'yes') {
      print('');
      print('Running first backup...');
      print('─' * 50);
      print('');
      print('Run: bulletproof backup');
    }

    // Summary
    print('');
    print('✅ Setup complete!');
    print('');
    print('Commands:');
    print('  bulletproof backup      Run a backup now');
    print('  bulletproof diff        Show changes since last backup');
    print('  bulletproof history     List all backups');
    print('  bulletproof restore ID  Restore from a backup');
    
    if (enableSchedule) {
      print('');
      print('📅 To enable scheduled backups, run:');
      print('   bulletproof schedule enable');
    }

    print('');
    return 0;
  }
}

