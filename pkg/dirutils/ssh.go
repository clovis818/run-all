package dirutils

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type sshCommandRunner interface {
	Run(host, command string, options SSHOptions) error
	RunOutput(host, command string, options SSHOptions) (string, error)
}

type SSHOptions struct {
	KeyPath string
}

type openSSHRunner struct{}

var sshRunner sshCommandRunner = openSSHRunner{}

func (openSSHRunner) Run(host, command string, options SSHOptions) error {
	cmd := exec.Command("ssh", buildSSHArgs(host, command, options)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (openSSHRunner) RunOutput(host, command string, options SSHOptions) (string, error) {
	cmd := exec.Command("ssh", buildSSHArgs(host, command, options)...)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()

	return out.String(), err
}

func buildSSHArgs(host, command string, options SSHOptions) []string {
	args := make([]string, 0, 4)
	if options.KeyPath != "" {
		args = append(args, "-i", options.KeyPath)
	}
	args = append(args, host, command)

	return args
}

func ParseHosts(hosts string) []string {
	parts := strings.Split(hosts, ",")
	parsed := make([]string, 0, len(parts))
	for _, part := range parts {
		host := strings.TrimSpace(part)
		if host != "" {
			parsed = append(parsed, host)
		}
	}

	return parsed
}

func ParseHostFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var hosts []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		hosts = append(hosts, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return hosts, nil
}

func MergeHosts(hostGroups ...[]string) []string {
	seen := make(map[string]bool)
	hosts := make([]string, 0)
	for _, group := range hostGroups {
		for _, host := range group {
			if host == "" || seen[host] {
				continue
			}
			seen[host] = true
			hosts = append(hosts, host)
		}
	}

	return hosts
}

func GetSSHHosts(hosts, hostFile string) ([]string, error) {
	inlineHosts := ParseHosts(hosts)
	if hostFile == "" {
		return MergeHosts(inlineHosts), nil
	}

	fileHosts, err := ParseHostFile(hostFile)
	if err != nil {
		return nil, err
	}

	return MergeHosts(inlineHosts, fileHosts), nil
}

func FormatSSHCommand(host, command string) string {
	return FormatSSHCommandWithOptions(host, command, SSHOptions{})
}

func FormatSSHCommandWithOptions(host, command string, options SSHOptions) string {
	parts := []string{"ssh"}
	if options.KeyPath != "" {
		parts = append(parts, "-i", strconv.Quote(options.KeyPath))
	}
	parts = append(parts, host, strconv.Quote(command))

	return strings.Join(parts, " ")
}

func RunCommandOnHostsSequential(
	hosts []string,
	command string,
	dryRun, continueOnFailure bool,
	interruptChan chan os.Signal,
) map[string]error {
	return RunCommandOnHostsSequentialWithOptions(
		hosts,
		command,
		SSHOptions{},
		dryRun,
		continueOnFailure,
		interruptChan,
	)
}

func RunCommandOnHostsSequentialWithOptions(
	hosts []string,
	command string,
	options SSHOptions,
	dryRun, continueOnFailure bool,
	interruptChan chan os.Signal,
) map[string]error {
	results := make(map[string]error)

	for _, host := range hosts {
		select {
		case <-interruptChan:
			fmt.Println("\nExecution interrupted by user.")
			return results
		default:
			fmt.Println(strings.Repeat("#", GetTerminalWidth()))
			fmt.Printf("Running command on host: %s\n", host)
			fmt.Println(strings.Repeat("#", GetTerminalWidth()))

			if dryRun {
				fmt.Printf(
					"[Dry Run] Command to be run on host '%s': %s\n",
					host,
					FormatSSHCommandWithOptions(host, command, options),
				)
				results[host] = nil
			} else {
				err := RunCommandOnHostWithOptions(host, command, options)
				results[host] = err
				if err != nil && !continueOnFailure {
					fmt.Printf("Error running command on host '%s': %v\n", host, err)
					return results
				}
			}

			fmt.Println(strings.Repeat("#", GetTerminalWidth()))
			fmt.Printf("Finished command on host: %s\n", host)
			fmt.Println(strings.Repeat("#", GetTerminalWidth()))
		}
	}

	return results
}

func RunCommandOnHostsParallel(
	hosts []string,
	command string,
	dryRun, continueOnFailure bool,
	interruptChan chan os.Signal,
) map[string]error {
	return RunCommandOnHostsParallelWithOptions(
		hosts,
		command,
		SSHOptions{},
		dryRun,
		continueOnFailure,
		interruptChan,
	)
}

func RunCommandOnHostsParallelWithOptions(
	hosts []string,
	command string,
	options SSHOptions,
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

	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			select {
			case <-interruptChan:
				return
			case <-stopChan:
				return
			default:
			}
			output := RunCommandOnHostParallelWithOptions(host, command, options, dryRun)
			mu.Lock()
			results[host] = output.err
			if output.err != nil {
				fmt.Printf("Error running command on host '%s': %v\n", host, output.err)
			}
			PrintHostCommandOutput(host, output.out)
			shouldStop := output.err != nil && !continueOnFailure
			mu.Unlock()

			if shouldStop {
				stop()
			}
		}(host)
	}

	wg.Wait()
	return results
}

func RunCommandOnHostParallel(host, command string, dryRun bool) (output struct {
	out string
	err error
}) {
	return RunCommandOnHostParallelWithOptions(host, command, SSHOptions{}, dryRun)
}

func RunCommandOnHostParallelWithOptions(
	host, command string,
	options SSHOptions,
	dryRun bool,
) (output struct {
	out string
	err error
}) {
	if dryRun {
		output.out = fmt.Sprintf(
			"[Dry Run] Command to be run on host '%s': %s\n",
			host,
			FormatSSHCommandWithOptions(host, command, options),
		)
		return
	}

	output.out, output.err = sshRunner.RunOutput(host, command, options)

	return
}

func PrintHostCommandOutput(host, output string) {
	fmt.Println(strings.Repeat("#", GetTerminalWidth()))
	fmt.Printf("Output for host: %s\n", host)
	fmt.Println(strings.Repeat("#", GetTerminalWidth()))
	fmt.Println(output)
	fmt.Println(strings.Repeat("#", GetTerminalWidth()))
	fmt.Printf("Finished command on host: %s\n", host)
	fmt.Println(strings.Repeat("#", GetTerminalWidth()))
}

func RunCommandOnHost(host, command string) error {
	return RunCommandOnHostWithOptions(host, command, SSHOptions{})
}

func RunCommandOnHostWithOptions(host, command string, options SSHOptions) error {
	return sshRunner.Run(host, command, options)
}

func PrintHostResultsSummary(results map[string]error) {
	fmt.Println("Command execution summary:")
	for host, err := range results {
		if err != nil {
			fmt.Printf("Host: %s - Error: %v\n", host, err)
		} else {
			fmt.Printf("Host: %s - Success\n", host)
		}
	}
}
