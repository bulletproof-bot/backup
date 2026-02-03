package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// HashFile computes the SHA-256 hash of a file
func HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// HashBytes computes the SHA-256 hash of a byte slice
func HashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:])
}

// HashString computes the SHA-256 hash of a string
func HashString(s string) string {
	return HashBytes([]byte(s))
}

// FilesEqual compares two files by their SHA-256 hashes
func FilesEqual(path1, path2 string) (bool, error) {
	hash1, err := HashFile(path1)
	if err != nil {
		return false, err
	}

	hash2, err := HashFile(path2)
	if err != nil {
		return false, err
	}

	return hash1 == hash2, nil
}
