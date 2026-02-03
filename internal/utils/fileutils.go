package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands ~ to the user's home directory
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	return filepath.Join(homeDir, path[2:]), nil
}

// FormatSize formats a byte size as a human-readable string
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// CopyDirectory recursively copies a directory from src to dst, excluding files matching exclusion patterns
func CopyDirectory(src, dst string, exclude []string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %w", err)
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Walk the source directory
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Check exclusions
		if shouldExclude(relPath, exclude) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Destination path
		dstPath := filepath.Join(dst, relPath)

		// Copy directory or file
		if info.IsDir() {
			if err := os.MkdirAll(dstPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dstPath, err)
			}
		} else {
			if err := copyFile(path, dstPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", relPath, err)
			}
		}

		return nil
	})
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string, mode os.FileMode) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

// DirectorySize calculates the total size of all files in a directory
func DirectorySize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate directory size: %w", err)
	}

	return size, nil
}

// CountFiles counts the number of files in a directory
func CountFiles(path string) (int, error) {
	count := 0

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to count files: %w", err)
	}

	return count, nil
}

// shouldExclude checks if a path should be excluded (simplified for utils package)
func shouldExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "/") {
			if strings.HasPrefix(path, pattern) || strings.Contains(path, "/"+pattern) {
				return true
			}
		} else if strings.HasPrefix(pattern, "*.") {
			if strings.HasSuffix(path, pattern[1:]) {
				return true
			}
		} else if path == pattern || strings.HasSuffix(path, "/"+pattern) {
			return true
		}
	}
	return false
}
