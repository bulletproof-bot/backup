import 'dart:io';
import 'package:path/path.dart' as p;
import 'package:yaml/yaml.dart';

/// Bulletproof configuration manager.
/// 
/// Config file location: ~/.config/bulletproof/config.yaml
class Config {
  static const String configVersion = '1';
  
  final String? openclawPath;
  final DestinationConfig? destination;
  final ScheduleConfig schedule;
  final BackupOptions options;

  Config({
    this.openclawPath,
    this.destination,
    this.schedule = const ScheduleConfig(),
    this.options = const BackupOptions(),
  });

  /// Get the config file path
  static String get configPath {
    final home = Platform.environment['HOME'] ?? 
                 Platform.environment['USERPROFILE'] ?? 
                 '.';
    return p.join(home, '.config', 'bulletproof', 'config.yaml');
  }

  /// Get the config directory
  static String get configDir => p.dirname(configPath);

  /// Check if config exists
  static bool get exists => File(configPath).existsSync();

  /// Load config from file
  static Config load() {
    final file = File(configPath);
    if (!file.existsSync()) {
      return Config();
    }

    try {
      final content = file.readAsStringSync();
      final yaml = loadYaml(content) as YamlMap?;
      
      if (yaml == null) return Config();

      return Config(
        openclawPath: yaml['openclaw_path'] as String?,
        destination: yaml['destination'] != null
            ? DestinationConfig.fromYaml(yaml['destination'] as YamlMap)
            : null,
        schedule: yaml['schedule'] != null
            ? ScheduleConfig.fromYaml(yaml['schedule'] as YamlMap)
            : const ScheduleConfig(),
        options: yaml['options'] != null
            ? BackupOptions.fromYaml(yaml['options'] as YamlMap)
            : const BackupOptions(),
      );
    } catch (e) {
      print('Warning: Failed to parse config: $e');
      return Config();
    }
  }

  /// Save config to file
  void save() {
    final dir = Directory(configDir);
    if (!dir.existsSync()) {
      dir.createSync(recursive: true);
    }

    final buffer = StringBuffer();
    buffer.writeln('# Bulletproof configuration');
    buffer.writeln('# https://github.com/bulletproof-bot/backup');
    buffer.writeln('version: "$configVersion"');
    buffer.writeln();

    if (openclawPath != null) {
      buffer.writeln('# Path to OpenClaw installation');
      buffer.writeln('openclaw_path: "$openclawPath"');
      buffer.writeln();
    }

    if (destination != null) {
      buffer.writeln('# Backup destination');
      buffer.writeln('destination:');
      buffer.writeln('  type: "${destination!.type}"');
      buffer.writeln('  path: "${destination!.path}"');
      buffer.writeln();
    }

    buffer.writeln('# Backup schedule');
    buffer.writeln('schedule:');
    buffer.writeln('  enabled: ${schedule.enabled}');
    buffer.writeln('  time: "${schedule.time}"');
    buffer.writeln();

    buffer.writeln('# Backup options');
    buffer.writeln('options:');
    buffer.writeln('  include_auth: ${options.includeAuth}');
    if (options.exclude.isNotEmpty) {
      buffer.writeln('  exclude:');
      for (final pattern in options.exclude) {
        buffer.writeln('    - "$pattern"');
      }
    }

    File(configPath).writeAsStringSync(buffer.toString());
  }

  /// Create a copy with updated values
  Config copyWith({
    String? openclawPath,
    DestinationConfig? destination,
    ScheduleConfig? schedule,
    BackupOptions? options,
  }) {
    return Config(
      openclawPath: openclawPath ?? this.openclawPath,
      destination: destination ?? this.destination,
      schedule: schedule ?? this.schedule,
      options: options ?? this.options,
    );
  }

  @override
  String toString() {
    return '''Config(
  openclawPath: $openclawPath,
  destination: $destination,
  schedule: $schedule,
  options: $options,
)''';
  }
}

/// Destination configuration
class DestinationConfig {
  final String type; // 'git', 'local', or 'sync'
  final String path;

  const DestinationConfig({
    required this.type,
    required this.path,
  });

  factory DestinationConfig.fromYaml(YamlMap yaml) {
    return DestinationConfig(
      type: yaml['type'] as String? ?? 'local',
      path: yaml['path'] as String? ?? '',
    );
  }

  bool get isGit => type == 'git';
  bool get isLocal => type == 'local';
  bool get isSync => type == 'sync';

  @override
  String toString() => 'DestinationConfig(type: $type, path: $path)';
}

/// Schedule configuration
class ScheduleConfig {
  final bool enabled;
  final String time; // HH:MM format

  const ScheduleConfig({
    this.enabled = false,
    this.time = '03:00',
  });

  factory ScheduleConfig.fromYaml(YamlMap yaml) {
    return ScheduleConfig(
      enabled: yaml['enabled'] as bool? ?? false,
      time: yaml['time'] as String? ?? '03:00',
    );
  }

  int get hour => int.parse(time.split(':')[0]);
  int get minute => int.parse(time.split(':')[1]);

  @override
  String toString() => 'ScheduleConfig(enabled: $enabled, time: $time)';
}

/// Backup options
class BackupOptions {
  final bool includeAuth;
  final List<String> exclude;

  const BackupOptions({
    this.includeAuth = false,
    this.exclude = const ['*.log', 'node_modules/', '.git/'],
  });

  factory BackupOptions.fromYaml(YamlMap yaml) {
    final excludeList = yaml['exclude'] as YamlList?;
    return BackupOptions(
      includeAuth: yaml['include_auth'] as bool? ?? false,
      exclude: excludeList?.cast<String>().toList() ?? 
               const ['*.log', 'node_modules/', '.git/'],
    );
  }

  @override
  String toString() => 'BackupOptions(includeAuth: $includeAuth, exclude: $exclude)';
}

