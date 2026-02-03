#!/usr/bin/env dart

import 'dart:io';
import 'package:args/command_runner.dart';
import 'package:bulletproof/bulletproof.dart';

void main(List<String> arguments) async {
  final runner = CommandRunner<int>(
    'bulletproof',
    'Back up your OpenClaw agent. Track changes. Rollback anytime.',
  )
    ..addCommand(InitCommand())
    ..addCommand(BackupCommand())
    ..addCommand(RestoreCommand())
    ..addCommand(DiffCommand())
    ..addCommand(HistoryCommand())
    ..addCommand(ConfigCommand())
    ..addCommand(ScheduleCommand())
    ..addCommand(ServiceCommand());

  try {
    final exitCode = await runner.run(arguments) ?? 0;
    exit(exitCode);
  } on UsageException catch (e) {
    print(e);
    exit(64);
  } catch (e, stack) {
    print('Error: $e');
    if (Platform.environment['BULLETPROOF_DEBUG'] == '1') {
      print(stack);
    }
    exit(1);
  }
}

