package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/clovis818/run-all/pkg/dirutils"
)

func main() {
	// Define and parse the flags
	dryRun := flag.Bool("dry-run", false, "Perform a dry run without executing commands")
	dirPattern := flag.String("dir-pattern", "", "Directory pattern to match")
	command := flag.String("command", "", "Command(s) to run in each matched directory")
	excludePatterns := flag.String(
		"exclude",
		"",
		"Comma-separated list of patterns of directories to exclude (relative to dir-pattern or full path)",
	)
	requireFolder := flag.String(
		"require",
		"",
		"Required folder or file for a directory to be included",
	)
	continueOnFailure := flag.Bool(
		"continue-on-failure",
		false,
		"Continue executing commands in other directories even if one fails",
	)
	parallel := flag.Bool("parallel", false, "Run commands in parallel")
	flag.Parse()

	if *dirPattern == "" {
		// Get the directory pattern from stdin if not provided as an argument
		*dirPattern = getDirectoryPattern()
	}

	if *command == "" {
		fmt.Println("Error: Command(s) must be provided using the -command flag.")
		return
	}

	// Set up channel to catch interrupt signal (Ctrl+C)
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	// Get the list of directories matching the pattern
	directories, err := dirutils.GetDirectories(*dirPattern)
	if err != nil {
		fmt.Printf("Error getting directories: %v\n", err)
		return
	}

	if len(directories) == 0 {
		fmt.Println("No directories matched the pattern.")
		return
	}

	// Exclude directories matching the exclude patterns
	if *excludePatterns != "" {
		excludePatternsList := strings.Split(*excludePatterns, ",")
		directories = dirutils.ExcludeDirectories(directories, excludePatternsList, *dirPattern)
	}

	// Filter directories to only include those with the required folder or file
	if *requireFolder != "" {
		directories = dirutils.FilterDirectoriesWithRequirement(directories, *requireFolder)
	}

	if len(directories) == 0 {
		fmt.Println("No directories with the required folder or file matched the pattern.")
		return
	}

	fmt.Println("Matched directories:")
	for _, dir := range directories {
		fmt.Println(" - " + dir)
	}

	// Confirm with the user before proceeding
	fmt.Println("Do you want to proceed with these directories? (yes/no)")
	confirmation := getUserInput()
	if strings.ToLower(confirmation) != "yes" {
		fmt.Println("Operation aborted.")
		return
	}

	var results map[string]error
	if *parallel {
		results = dirutils.RunCommandInDirectoriesParallel(
			directories,
			*command,
			*dryRun,
			*continueOnFailure,
			interruptChan,
		)
	} else {
		results = dirutils.RunCommandInDirectoriesSequential(directories, *command, *dryRun, *continueOnFailure, interruptChan)
	}

	// Print the results summary
	dirutils.PrintResultsSummary(results)
}

// getDirectoryPattern gets the directory pattern from the user
func getDirectoryPattern() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter the directory pattern (e.g., /tmp/hello-*/world/run-here):")
	fmt.Print("> ")
	pattern, _ := reader.ReadString('\n')
	return strings.TrimSpace(pattern)
}

// getUserInput reads a line of input from the user
func getUserInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
