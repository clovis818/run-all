package dirutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const defaultTerminalWidth = 80

type shellConfig struct {
	executablePath string
	args           []string
}

func GetDirectories(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func ExcludeDirectories(directories, excludePatterns []string, dirPattern string) []string {
	var filteredDirs []string
	excludeMap := make(map[string]bool)
	excludeRegexps := make([]*regexp.Regexp, 0, len(excludePatterns))

	// Store exact match patterns in a map for quick lookup
	for _, pattern := range excludePatterns {
		excludeMap[pattern] = true
	}

	// Convert glob patterns to regex and compile them
	for _, pattern := range excludePatterns {
		if strings.Contains(pattern, "*") {
			regexPattern := "^" + regexp.QuoteMeta(pattern)
			regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
			re, err := regexp.Compile(regexPattern)
			if err == nil {
				excludeRegexps = append(excludeRegexps, re)
			}
		}
	}

	// Check each directory against exact matches and regex patterns
	for _, dir := range directories {
		if excludeMap[filepath.Base(dir)] {
			continue
		}
		if excludeMap[dir] {
			continue
		}

		shouldExclude := false
		for _, re := range excludeRegexps {
			if re.MatchString(dir) {
				shouldExclude = true
				break
			}
		}

		if !shouldExclude {
			filteredDirs = append(filteredDirs, dir)
		}
	}

	return filteredDirs
}

func FilterDirectoriesWithRequirement(directories []string, requirement string) []string {
	var filteredDirs []string
	for _, dir := range directories {
		if _, err := os.Stat(filepath.Join(dir, requirement)); err == nil {
			filteredDirs = append(filteredDirs, dir)
		}
	}
	return filteredDirs
}

func RunCommandInDirectoriesSequential(
	directories []string,
	command string,
	dryRun, continueOnFailure bool,
	interruptChan chan os.Signal,
) map[string]error {
	results := make(map[string]error)

	for _, dir := range directories {
		select {
		case <-interruptChan:
			fmt.Println("\nExecution interrupted by user.")
			return results
		default:
			fmt.Println(strings.Repeat("#", GetTerminalWidth()))
			fmt.Printf("Running command in directory: %s\n", dir)
			fmt.Println(strings.Repeat("#", GetTerminalWidth()))

			if dryRun {
				fmt.Printf("[Dry Run] Command to be run in directory '%s': %s\n", dir, command)
				results[dir] = nil
			} else {
				err := RunCommandInDirectory(dir, command)
				results[dir] = err
				if err != nil && !continueOnFailure {
					fmt.Printf("Error running command in directory '%s': %v\n", dir, err)
					return results
				}
			}

			fmt.Println(strings.Repeat("#", GetTerminalWidth()))
			fmt.Printf("Finished command in directory: %s\n", dir)
			fmt.Println(strings.Repeat("#", GetTerminalWidth()))
		}
	}

	return results
}

func RunCommandInDirectoriesParallel(
	directories []string,
	command string,
	dryRun, continueOnFailure bool,
	interruptChan chan os.Signal,
) map[string]error {
	results := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup
	stopChan := make(chan struct{})
	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			close(stopChan)
		})
	}

	for _, dir := range directories {
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()
			select {
			case <-interruptChan:
				return
			case <-stopChan:
				return
			default:
			}
			output := RunCommandInDirectoryParallel(dir, command, dryRun)
			mu.Lock()
			results[dir] = output.err
			if output.err != nil {
				fmt.Printf("Error running command in directory '%s': %v\n", dir, output.err)
			}
			PrintCommandOutput(dir, output.out)
			shouldStop := output.err != nil && !continueOnFailure
			mu.Unlock()

			if shouldStop {
				stop()
			}
		}(dir)
	}

	wg.Wait()
	return results
}

func RunCommandInDirectoryParallel(dir, command string, dryRun bool) (output struct {
	out string
	err error
}) {
	if dryRun {
		output.out = fmt.Sprintf(
			"[Dry Run] Command to be run in directory '%s': %s\n",
			dir,
			command,
		)
		return
	}
	cmd := newLocalCommand(dir, command)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	output.err = cmd.Run()
	output.out = out.String()
	return
}

func PrintCommandOutput(dir, output string) {
	fmt.Println(strings.Repeat("#", GetTerminalWidth()))
	fmt.Printf("Output for directory: %s\n", dir)
	fmt.Println(strings.Repeat("#", GetTerminalWidth()))
	fmt.Println(output)
	fmt.Println(strings.Repeat("#", GetTerminalWidth()))
	fmt.Printf("Finished command in directory: %s\n", dir)
	fmt.Println(strings.Repeat("#", GetTerminalWidth()))
}

func RunCommandInDirectory(dir, command string) error {
	cmd := newLocalCommand(dir, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func newLocalCommand(dir, command string) *exec.Cmd {
	config := currentShellConfig()
	args := make([]string, 0, len(config.args)+1)
	args = append(args, config.args...)
	args = append(args, command)

	cmd := exec.Command(config.executablePath, args...)
	cmd.Dir = dir

	return cmd
}

func currentShellConfig() shellConfig {
	return shellConfigFor(runtime.GOOS)
}

func shellConfigFor(goos string) shellConfig {
	switch goos {
	case "windows":
		return shellConfig{
			executablePath: `C:\Windows\System32\cmd.exe`,
			args:           []string{"/D", "/S", "/C"},
		}
	default:
		return shellConfig{
			executablePath: "/bin/sh",
			args:           []string{"-c"},
		}
	}
}

func PrintResultsSummary(results map[string]error) {
	fmt.Println("Command execution summary:")
	for dir, err := range results {
		if err != nil {
			fmt.Printf("Directory: %s - Error: %v\n", dir, err)
		} else {
			fmt.Printf("Directory: %s - Success\n", dir)
		}
	}
}

func GetTerminalWidth() int {
	if width := terminalWidthFromColumns(os.Getenv("COLUMNS")); width > 0 {
		return width
	}
	if runtime.GOOS == "windows" {
		return defaultTerminalWidth
	}

	return terminalWidthFromTput()
}

func terminalWidthFromColumns(columns string) int {
	width, err := strconv.Atoi(strings.TrimSpace(columns))
	if err != nil || width <= 0 {
		return 0
	}

	return width
}

func terminalWidthFromTput() int {
	cmd := exec.Command("tput", "cols")
	output, err := cmd.Output()
	if err != nil {
		return defaultTerminalWidth
	}

	if w, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil && w > 0 {
		return w
	}

	return defaultTerminalWidth
}
