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
	autoYes := flag.Bool("yes", false, "Skip confirmation prompt and proceed immediately")
	sshMode := flag.Bool("ssh", false, "Run commands on SSH hosts")
	hosts := flag.String("hosts", "", "Comma-separated list of SSH hosts")
	hostFile := flag.String("host-file", "", "Path to a newline-delimited SSH host file")
	sshKey := flag.String("ssh-key", "", "Path to the SSH private key to use")
	flag.Parse()

	if *sshMode {
		if err := validateSSHFlags(*command, *dirPattern, *excludePatterns, *requireFolder, *hosts, *hostFile); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		runSSHMode(*hosts, *hostFile, *sshKey, *command, *dryRun, *continueOnFailure, *parallel, *autoYes)
		return
	}

	if err := validateLocalFlags(*hosts, *hostFile, *sshKey); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

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
	if !*autoYes {
		fmt.Println("Do you want to proceed with these directories? (yes/no)")
		confirmation := strings.ToLower(getUserInput())
		if confirmation != "yes" && confirmation != "y" {
			fmt.Println("Operation aborted.")
			return
		}
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

func validateSSHFlags(command, dirPattern, excludePatterns, requireFolder, hosts, hostFile string) error {
	if command == "" {
		return fmt.Errorf("command must be provided using the -command flag")
	}
	if dirPattern != "" {
		return fmt.Errorf("-dir-pattern cannot be used with -ssh")
	}
	if excludePatterns != "" {
		return fmt.Errorf("-exclude cannot be used with -ssh")
	}
	if requireFolder != "" {
		return fmt.Errorf("-require cannot be used with -ssh")
	}
	if hosts == "" && hostFile == "" {
		return fmt.Errorf("SSH mode requires -hosts or -host-file")
	}

	return nil
}

func validateLocalFlags(hosts, hostFile, sshKey string) error {
	if hosts != "" {
		return fmt.Errorf("-hosts can only be used with -ssh")
	}
	if hostFile != "" {
		return fmt.Errorf("-host-file can only be used with -ssh")
	}
	if sshKey != "" {
		return fmt.Errorf("-ssh-key can only be used with -ssh")
	}

	return nil
}

func runSSHMode(
	hostsFlag, hostFile, sshKey, command string,
	dryRun, continueOnFailure, parallel, autoYes bool,
) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	hosts, err := dirutils.GetSSHHosts(hostsFlag, hostFile)
	if err != nil {
		fmt.Printf("Error getting SSH hosts: %v\n", err)
		return
	}

	if len(hosts) == 0 {
		fmt.Println("No SSH hosts matched the provided inputs.")
		return
	}

	fmt.Println("Matched hosts:")
	for _, host := range hosts {
		fmt.Println(" - " + host)
	}

	if !autoYes {
		fmt.Println("Do you want to proceed with these hosts? (yes/no)")
		confirmation := strings.ToLower(getUserInput())
		if confirmation != "yes" && confirmation != "y" {
			fmt.Println("Operation aborted.")
			return
		}
	}

	sshOptions := dirutils.SSHOptions{KeyPath: sshKey}
	var results map[string]error
	if parallel {
		results = dirutils.RunCommandOnHostsParallelWithOptions(
			hosts,
			command,
			sshOptions,
			dryRun,
			continueOnFailure,
			interruptChan,
		)
	} else {
		results = dirutils.RunCommandOnHostsSequentialWithOptions(
			hosts,
			command,
			sshOptions,
			dryRun,
			continueOnFailure,
			interruptChan,
		)
	}

	dirutils.PrintHostResultsSummary(results)
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
