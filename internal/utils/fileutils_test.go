package utils

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile_Basic(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tempDir, "source.txt")
	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Copy to destination
	dstPath := filepath.Join(tempDir, "dest.txt")
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile() failed: %v", err)
	}

	// Verify content
	gotContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination: %v", err)
	}
	if !bytes.Equal(content, gotContent) {
		t.Errorf("content mismatch: got %q, want %q", gotContent, content)
	}
}

func TestCopyFile_PreservesPermissions(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("data"), 0755); err != nil {
		t.Fatal(err)
	}

	dstPath := filepath.Join(tempDir, "dest.txt")
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatal(err)
	}

	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)

	if srcInfo.Mode().Perm() != dstInfo.Mode().Perm() {
		t.Errorf("permission mismatch: got %v, want %v", dstInfo.Mode().Perm(), srcInfo.Mode().Perm())
	}
}

func TestCopyFile_CreatesParentDirs(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "source.txt")
	os.WriteFile(srcPath, []byte("data"), 0644)

	dstPath := filepath.Join(tempDir, "nested", "dirs", "dest.txt")
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile() failed: %v", err)
	}

	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("destination file not created")
	}
}

func TestCopyFile_HandlesReadonlyDestination(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "source.txt")
	os.WriteFile(srcPath, []byte("new data"), 0644)

	dstPath := filepath.Join(tempDir, "dest.txt")
	os.WriteFile(dstPath, []byte("old data"), 0444) // readonly

	// Should succeed even though destination is readonly
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile() failed on readonly dest: %v", err)
	}

	content, _ := os.ReadFile(dstPath)
	if string(content) != "new data" {
		t.Errorf("content not updated: got %q", content)
	}
}

func TestCopyFile_LargeFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create 10MB file
	srcPath := filepath.Join(tempDir, "large.dat")
	f, _ := os.Create(srcPath)
	largeContent := make([]byte, 10*1024*1024) // 10MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	f.Write(largeContent)
	f.Close()

	dstPath := filepath.Join(tempDir, "large_copy.dat")
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile() failed on large file: %v", err)
	}

	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)

	if srcInfo.Size() != dstInfo.Size() {
		t.Errorf("size mismatch: got %d, want %d", dstInfo.Size(), srcInfo.Size())
	}

	// Verify content matches
	dstContent, _ := os.ReadFile(dstPath)
	if !bytes.Equal(largeContent, dstContent) {
		t.Error("large file content mismatch")
	}
}

func TestCopyFile_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "empty.txt")
	os.WriteFile(srcPath, []byte{}, 0644)

	dstPath := filepath.Join(tempDir, "empty_copy.txt")
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile() failed on empty file: %v", err)
	}

	dstContent, _ := os.ReadFile(dstPath)
	if len(dstContent) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(dstContent))
	}
}

func TestCopyFile_SourceNotExists(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "nonexistent.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")

	if err := CopyFile(srcPath, dstPath); err == nil {
		t.Error("CopyFile() should fail when source doesn't exist")
	}
}

func TestCopyFile_OverwritesExisting(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "source.txt")
	os.WriteFile(srcPath, []byte("new content"), 0644)

	dstPath := filepath.Join(tempDir, "dest.txt")
	os.WriteFile(dstPath, []byte("old content"), 0644)

	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile() failed: %v", err)
	}

	content, _ := os.ReadFile(dstPath)
	if string(content) != "new content" {
		t.Errorf("content not overwritten: got %q", content)
	}
}

func TestExpandPath_TildeExpansion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHome bool
	}{
		{
			name:     "tilde only",
			input:    "~",
			wantHome: true,
		},
		{
			name:     "tilde with path",
			input:    "~/documents",
			wantHome: true,
		},
		{
			name:     "absolute path",
			input:    "/usr/local/bin",
			wantHome: false,
		},
		{
			name:     "relative path",
			input:    "relative/path",
			wantHome: false,
		},
	}

	home := os.Getenv("HOME")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			if err != nil && tt.wantHome {
				t.Fatalf("ExpandPath(%q) error: %v", tt.input, err)
			}
			if tt.wantHome && result == tt.input {
				t.Errorf("ExpandPath(%q) = %q, expected expansion", tt.input, result)
			}
			if tt.wantHome && home != "" && len(result) >= len(home) && result[:len(home)] != home {
				t.Errorf("ExpandPath(%q) = %q, expected to start with %q", tt.input, result, home)
			}
			if !tt.wantHome && result != tt.input {
				t.Errorf("ExpandPath(%q) = %q, expected no change", tt.input, result)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 500, "500 B"},
		{"kilobytes", 1024, "1.0 KiB"},
		{"megabytes", 1024 * 1024, "1.0 MiB"},
		{"gigabytes", 1024 * 1024 * 1024, "1.0 GiB"},
		{"fractional KB", 1536, "1.5 KiB"},
		{"fractional MB", 1024*1024 + 512*1024, "1.5 MiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
