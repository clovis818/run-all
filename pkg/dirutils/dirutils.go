package dirutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

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

	for _, dir := range directories {
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()
			select {
			case <-interruptChan:
				return
			default:
				output := RunCommandInDirectoryParallel(dir, command, dryRun)
				mu.Lock()
				results[dir] = output.err
				if output.err != nil {
					fmt.Printf("Error running command in directory '%s': %v\n", dir, output.out)
					if !continueOnFailure {
						mu.Unlock()
						return
					}
				}
				PrintCommandOutput(dir, output.out)
				mu.Unlock()
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
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
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
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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
	width := 80 // default width
	cmd := exec.Command("tput", "cols")
	output, err := cmd.Output()
	if err != nil {
		return width
	}

	if w, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
		width = w
	}

	return width
}
