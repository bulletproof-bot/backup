import 'dart:io';
import 'package:path/path.dart' as p;

/// Default paths for OpenClaw installations.
/// 
/// OpenClaw stores data in:
/// - ~/.openclaw/openclaw.json - main config
/// - ~/.openclaw/workspace/ - agent workspace
///   - SOUL.md - personality/identity
///   - AGENTS.md - agent definitions
///   - TOOLS.md - tool configurations
///   - skills/ - skill modules
///   - memory/ - daily logs + MEMORY.md
/// - ~/.openclaw/agents/<agentId>/ - per-agent configs
class OpenClawDefaults {
  /// Get the default OpenClaw root directory
  static String get defaultRoot {
    final home = _homeDirectory;
    return p.join(home, '.openclaw');
  }

  /// Check if OpenClaw is installed at the default location
  static bool get isInstalled {
    final root = defaultRoot;
    final configFile = File(p.join(root, 'openclaw.json'));
    return configFile.existsSync();
  }

  /// Check if running inside Docker
  static bool get isDocker {
    // Check for .dockerenv file
    if (File('/.dockerenv').existsSync()) return true;
    
    // Check cgroup for docker
    try {
      final cgroup = File('/proc/1/cgroup');
      if (cgroup.existsSync()) {
        final content = cgroup.readAsStringSync();
        if (content.contains('docker') || content.contains('kubepods')) {
          return true;
        }
      }
    } catch (_) {}
    
    return false;
  }

  /// Detect OpenClaw installation
  /// Returns the root path if found, null otherwise
  static String? detectInstallation() {
    // Check default location first
    if (isInstalled) {
      return defaultRoot;
    }
    
    // Check common Docker volume mounts
    final dockerPaths = [
      '/data/.openclaw',
      '/openclaw',
      '/app/.openclaw',
    ];
    
    for (final path in dockerPaths) {
      if (File(p.join(path, 'openclaw.json')).existsSync()) {
        return path;
      }
    }
    
    return null;
  }

  /// Get all important files/directories to back up
  static List<BackupTarget> getBackupTargets(String openclawRoot) {
    return [
      // Core config
      BackupTarget(
        path: p.join(openclawRoot, 'openclaw.json'),
        description: 'Main configuration',
        critical: true,
      ),
      
      // Soul and identity files
      BackupTarget(
        path: p.join(openclawRoot, 'workspace', 'SOUL.md'),
        description: 'Soul file (personality)',
        critical: true,
      ),
      BackupTarget(
        path: p.join(openclawRoot, 'workspace', 'AGENTS.md'),
        description: 'Agent definitions',
        critical: true,
      ),
      BackupTarget(
        path: p.join(openclawRoot, 'workspace', 'TOOLS.md'),
        description: 'Tool configurations',
        critical: true,
      ),
      
      // Skills directory
      BackupTarget(
        path: p.join(openclawRoot, 'workspace', 'skills'),
        description: 'Skills and capabilities',
        isDirectory: true,
        critical: true,
      ),
      
      // Memory and conversations
      BackupTarget(
        path: p.join(openclawRoot, 'workspace', 'memory'),
        description: 'Conversation logs',
        isDirectory: true,
        critical: true,
      ),
      BackupTarget(
        path: p.join(openclawRoot, 'workspace', 'MEMORY.md'),
        description: 'Long-term memory',
        critical: true,
      ),
      
      // Per-agent configs
      BackupTarget(
        path: p.join(openclawRoot, 'agents'),
        description: 'Per-agent configurations',
        isDirectory: true,
        critical: false,
      ),
    ];
  }

  /// Sensitive files that should be backed up with caution
  static List<String> get sensitivePatterns => [
    'auth-profiles.json',
    'oauth.json',
    '**/secrets/**',
    '**/*.key',
    '**/*.pem',
  ];

  static String get _homeDirectory {
    if (Platform.isWindows) {
      return Platform.environment['USERPROFILE'] ?? 
             Platform.environment['HOME'] ?? 
             'C:\\Users\\Default';
    }
    return Platform.environment['HOME'] ?? '/home';
  }
}

/// A target file or directory to back up
class BackupTarget {
  final String path;
  final String description;
  final bool isDirectory;
  final bool critical;

  const BackupTarget({
    required this.path,
    required this.description,
    this.isDirectory = false,
    this.critical = true,
  });

  bool get exists {
    if (isDirectory) {
      return Directory(path).existsSync();
    }
    return File(path).existsSync();
  }

  @override
  String toString() => '$description ($path)';
}

