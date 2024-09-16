package dirutils

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetDirectories(t *testing.T) {
	// Setup
	testDir := t.TempDir()
	subDir := filepath.Join(testDir, "subdir")
	_ = os.Mkdir(subDir, 0755)
	defer os.RemoveAll(testDir)

	// Test
	directories, err := GetDirectories(filepath.Join(testDir, "*"))

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, len(directories))
	assert.Equal(t, subDir, directories[0])
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

	// Assert
	assert.Equal(t, 1, len(filteredDirs))
	assert.Equal(t, subDir2, filteredDirs[0])
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

	// Assert
	assert.Equal(t, 1, len(filteredDirs))
	assert.Equal(t, subDir1, filteredDirs[0])
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

	// Assert
	for dir, err := range results {
		assert.NoError(t, err, "Expected no error in directory %s", dir)
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

	// Assert
	for dir, err := range results {
		assert.NoError(t, err, "Expected no error in directory %s", dir)
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

	// Assert
	for dir, err := range results {
		assert.NoError(t, err, "Expected no error in directory %s", dir)
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

	// Assert
	for dir, err := range results {
		assert.NoError(t, err, "Expected no error in directory %s", dir)
	}
}
