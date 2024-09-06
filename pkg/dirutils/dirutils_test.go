package dirutils

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestGetDirectories(t *testing.T) {
	// Setup
	testDir := t.TempDir()
	subDir := filepath.Join(testDir, "subdir")
	_ = os.Mkdir(subDir, 0755)
	defer os.RemoveAll(testDir)

	// Test
	directories, err := GetDirectories(filepath.Join(testDir, "*"))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(directories) != 1 || directories[0] != subDir {
		t.Fatalf("Expected %s, got %v", subDir, directories)
	}
}

func TestExcludeDirectories(t *testing.T) {
	// Setup
	testDir := t.TempDir()
	subDir1 := filepath.Join(testDir, "subdir1")
	subDir2 := filepath.Join(testDir, "subdir2")
	_ = os.Mkdir(subDir1, 0755)
	_ = os.Mkdir(subDir2, 0755)
	defer os.RemoveAll(testDir)

	directories := []string{subDir1, subDir2}
	excludePatterns := []string{filepath.Join(testDir, "subdir1")}

	// Test
	filteredDirs := ExcludeDirectories(directories, excludePatterns, testDir)

	if len(filteredDirs) != 1 || filteredDirs[0] != subDir2 {
		t.Fatalf("Expected %s, got %v", subDir2, filteredDirs)
	}
}

func TestFilterDirectoriesWithRequirement(t *testing.T) {
	// Setup
	testDir := t.TempDir()
	subDir1 := filepath.Join(testDir, "subdir1")
	subDir2 := filepath.Join(testDir, "subdir2")
	_ = os.Mkdir(subDir1, 0755)
	_ = os.Mkdir(subDir2, 0755)
	_, _ = os.Create(filepath.Join(subDir1, ".git"))
	defer os.RemoveAll(testDir)

	directories := []string{subDir1, subDir2}

	// Test
	filteredDirs := FilterDirectoriesWithRequirement(directories, ".git")

	if len(filteredDirs) != 1 || filteredDirs[0] != subDir1 {
		t.Fatalf("Expected %s, got %v", subDir1, filteredDirs)
	}
}

func TestRunCommandInDirectoriesSequential(t *testing.T) {
	// Setup
	testDir := t.TempDir()
	subDir1 := filepath.Join(testDir, "subdir1")
	subDir2 := filepath.Join(testDir, "subdir2")
	_ = os.Mkdir(subDir1, 0755)
	_ = os.Mkdir(subDir2, 0755)
	defer os.RemoveAll(testDir)

	directories := []string{subDir1, subDir2}
	command := "echo Hello"

	interruptChan := make(chan os.Signal, 1)
	results := RunCommandInDirectoriesSequential(directories, command, false, false, interruptChan)

	for dir, err := range results {
		if err != nil {
			t.Fatalf("Expected no error in directory %s, got %v", dir, err)
		}
	}
}

func TestRunCommandInDirectoriesParallel(t *testing.T) {
	// Setup
	testDir := t.TempDir()
	subDir1 := filepath.Join(testDir, "subdir1")
	subDir2 := filepath.Join(testDir, "subdir2")
	_ = os.Mkdir(subDir1, 0755)
	_ = os.Mkdir(subDir2, 0755)
	defer os.RemoveAll(testDir)

	directories := []string{subDir1, subDir2}
	command := "echo Hello"

	interruptChan := make(chan os.Signal, 1)
	results := RunCommandInDirectoriesParallel(directories, command, false, false, interruptChan)

	for dir, err := range results {
		if err != nil {
			t.Fatalf("Expected no error in directory %s, got %v", dir, err)
		}
	}
}

func TestDryRun(t *testing.T) {
	// Setup
	testDir := t.TempDir()
	subDir1 := filepath.Join(testDir, "subdir1")
	subDir2 := filepath.Join(testDir, "subdir2")
	_ = os.Mkdir(subDir1, 0755)
	_ = os.Mkdir(subDir2, 0755)
	defer os.RemoveAll(testDir)

	directories := []string{subDir1, subDir2}
	command := "echo Hello"

	interruptChan := make(chan os.Signal, 1)
	results := RunCommandInDirectoriesSequential(directories, command, true, false, interruptChan)

	for dir, err := range results {
		if err != nil {
			t.Fatalf("Expected no error in directory %s, got %v", dir, err)
		}
	}
}

func TestHandleInterrupt(t *testing.T) {
	// Setup
	testDir := t.TempDir()
	subDir1 := filepath.Join(testDir, "subdir1")
	subDir2 := filepath.Join(testDir, "subdir2")
	_ = os.Mkdir(subDir1, 0755)
	_ = os.Mkdir(subDir2, 0755)
	defer os.RemoveAll(testDir)

	directories := []string{subDir1, subDir2}
	command := "echo Hello"

	interruptChan := make(chan os.Signal, 1)

	// Simulate an interrupt signal after a short delay
	go func() {
		time.Sleep(1 * time.Second)
		interruptChan <- syscall.SIGINT
	}()

	results := RunCommandInDirectoriesSequential(directories, command, false, false, interruptChan)

	for dir, err := range results {
		if err != nil {
			t.Fatalf("Expected no error in directory %s, got %v", dir, err)
		}
	}
}
