package semanticmatcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if len(config.VectorFilePaths) != 0 {
		t.Errorf("Expected empty VectorFilePaths, got %v", config.VectorFilePaths)
	}

	if config.MaxSequenceLen != DefaultMaxSequenceLen {
		t.Errorf("Expected MaxSequenceLen %d, got %d", DefaultMaxSequenceLen, config.MaxSequenceLen)
	}

	if config.MemoryLimit != DefaultMemoryLimit {
		t.Errorf("Expected MemoryLimit %d, got %d", DefaultMemoryLimit, config.MemoryLimit)
	}

	if !config.EnableStats {
		t.Error("Expected EnableStats to be true")
	}

	if len(config.SupportedLanguages) != 2 {
		t.Errorf("Expected 2 supported languages, got %d", len(config.SupportedLanguages))
	}
}

func TestValidate_EmptyVectorFilePaths(t *testing.T) {
	config := DefaultConfig()
	config.VectorFilePaths = []string{}

	err := Validate(config)
	if err != ErrNoVectorFiles {
		t.Errorf("Expected ErrNoVectorFiles, got %v", err)
	}
}

func TestValidate_NilConfig(t *testing.T) {
	err := Validate(nil)
	if err != ErrInvalidConfiguration {
		t.Errorf("Expected ErrInvalidConfiguration, got %v", err)
	}
}

func TestValidate_EmptyStringInVectorFilePaths(t *testing.T) {
	config := DefaultConfig()
	config.VectorFilePaths = []string{""}

	err := Validate(config)
	if err != ErrInvalidConfiguration {
		t.Errorf("Expected ErrInvalidConfiguration for empty string, got %v", err)
	}
}

func TestValidate_SingleFileConfig(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.vec")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := DefaultConfig()
	config.VectorFilePaths = []string{testFile}

	err := Validate(config)
	if err != nil {
		t.Errorf("Expected no error for valid single file config, got %v", err)
	}
}

func TestValidate_MultipleFilesConfig(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.vec")
	testFile2 := filepath.Join(tmpDir, "test2.vec")

	if err := os.WriteFile(testFile1, []byte("test content 1"), 0o644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("test content 2"), 0o644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	config := DefaultConfig()
	config.VectorFilePaths = []string{testFile1, testFile2}

	err := Validate(config)
	if err != nil {
		t.Errorf("Expected no error for valid multiple files config, got %v", err)
	}
}

func TestValidate_FileNotExists(t *testing.T) {
	config := DefaultConfig()
	config.VectorFilePaths = []string{"/nonexistent/path/to/file.vec"}

	err := Validate(config)
	if err != ErrInvalidConfiguration {
		t.Errorf("Expected ErrInvalidConfiguration for non-existent file, got %v", err)
	}
}

func TestValidate_MixedExistingAndNonExistingFiles(t *testing.T) {
	// Create one valid file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.vec")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := DefaultConfig()
	config.VectorFilePaths = []string{testFile, "/nonexistent/file.vec"}

	err := Validate(config)
	if err != ErrInvalidConfiguration {
		t.Errorf("Expected ErrInvalidConfiguration when one file doesn't exist, got %v", err)
	}
}

func TestValidate_InvalidMaxSequenceLen(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.vec")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := DefaultConfig()
	config.VectorFilePaths = []string{testFile}
	config.MaxSequenceLen = 0

	err := Validate(config)
	if err != ErrInvalidConfiguration {
		t.Errorf("Expected ErrInvalidConfiguration for invalid MaxSequenceLen, got %v", err)
	}
}

func TestValidate_InvalidMemoryLimit(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.vec")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := DefaultConfig()
	config.VectorFilePaths = []string{testFile}
	config.MemoryLimit = -1

	err := Validate(config)
	if err != ErrInvalidConfiguration {
		t.Errorf("Expected ErrInvalidConfiguration for invalid MemoryLimit, got %v", err)
	}
}

func TestValidate_EmptySupportedLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.vec")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := DefaultConfig()
	config.VectorFilePaths = []string{testFile}
	config.SupportedLanguages = []string{}

	err := Validate(config)
	if err != ErrInvalidConfiguration {
		t.Errorf("Expected ErrInvalidConfiguration for empty SupportedLanguages, got %v", err)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.vec")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := DefaultConfig()
	config.VectorFilePaths = []string{testFile}

	err := Validate(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}
}
