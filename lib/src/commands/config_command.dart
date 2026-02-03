import 'dart:io';
import 'package:args/command_runner.dart';

import '../config/config.dart';

/// View and modify configuration.
class ConfigCommand extends Command<int> {
  @override
  final name = 'config';

  @override
  final description = 'View or modify configuration';

  ConfigCommand() {
    addSubcommand(_ConfigShowCommand());
    addSubcommand(_ConfigSetCommand());
    addSubcommand(_ConfigPathCommand());
  }
}

class _ConfigShowCommand extends Command<int> {
  @override
  final name = 'show';

  @override
  final description = 'Show current configuration';

  @override
  Future<int> run() async {
    if (!Config.exists) {
      print('No configuration found.');
      print('Run: bulletproof init');
      return 1;
    }

    final config = Config.load();
    final file = File(Config.configPath);
    
    print('Configuration file: ${Config.configPath}');
    print('');
    print(file.readAsStringSync());

    return 0;
  }
}

class _ConfigSetCommand extends Command<int> {
  @override
  final name = 'set';

  @override
  final description = 'Set a configuration value';

  @override
  String get invocation => 'bulletproof config set <key> <value>';

  @override
  Future<int> run() async {
    final rest = argResults!.rest;
    
    if (rest.length < 2) {
      print('Usage: bulletproof config set <key> <value>');
      print('');
      print('Available keys:');
      print('  openclaw_path     Path to OpenClaw installation');
      print('  destination.type  Backup type: git, local, or sync');
      print('  destination.path  Backup destination path');
      print('  schedule.enabled  Enable scheduled backups: true or false');
      print('  schedule.time     Backup time in HH:MM format');
      return 1;
    }

    final key = rest[0];
    final value = rest.sublist(1).join(' ');

    if (!Config.exists) {
      print('No configuration found. Run: bulletproof init');
      return 1;
    }

    var config = Config.load();

    switch (key) {
      case 'openclaw_path':
        config = config.copyWith(openclawPath: value);
        break;
      
      case 'destination.type':
        if (!['git', 'local', 'sync'].contains(value)) {
          print('Invalid destination type. Use: git, local, or sync');
          return 1;
        }
        final dest = config.destination ?? const DestinationConfig(type: '', path: '');
        config = config.copyWith(
          destination: DestinationConfig(type: value, path: dest.path),
        );
        break;
      
      case 'destination.path':
        final dest = config.destination ?? const DestinationConfig(type: 'local', path: '');
        config = config.copyWith(
          destination: DestinationConfig(type: dest.type, path: value),
        );
        break;
      
      case 'schedule.enabled':
        final enabled = value.toLowerCase() == 'true';
        config = config.copyWith(
          schedule: ScheduleConfig(enabled: enabled, time: config.schedule.time),
        );
        break;
      
      case 'schedule.time':
        if (!RegExp(r'^\d{2}:\d{2}$').hasMatch(value)) {
          print('Invalid time format. Use HH:MM (e.g., 03:00)');
          return 1;
        }
        config = config.copyWith(
          schedule: ScheduleConfig(enabled: config.schedule.enabled, time: value),
        );
        break;
      
      default:
        print('Unknown configuration key: $key');
        return 1;
    }

    config.save();
    print('✓ Set $key = $value');

    return 0;
  }
}

class _ConfigPathCommand extends Command<int> {
  @override
  final name = 'path';

  @override
  final description = 'Show config file path';

  @override
  Future<int> run() async {
    print(Config.configPath);
    return 0;
  }
}

