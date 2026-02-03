import 'dart:convert';
import 'dart:io';
import 'package:crypto/crypto.dart';

/// Hash utilities for file comparison.

/// Calculate SHA-256 hash of a file
Future<String> hashFile(File file) async {
  final bytes = await file.readAsBytes();
  return sha256.convert(bytes).toString();
}

/// Calculate SHA-256 hash of a string
String hashString(String content) {
  return sha256.convert(utf8.encode(content)).toString();
}

/// Calculate SHA-256 hash of bytes
String hashBytes(List<int> bytes) {
  return sha256.convert(bytes).toString();
}

/// Quick file comparison using hash
Future<bool> filesEqual(File a, File b) async {
  final hashA = await hashFile(a);
  final hashB = await hashFile(b);
  return hashA == hashB;
}

