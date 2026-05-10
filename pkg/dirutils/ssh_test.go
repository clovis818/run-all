package dirutils

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type sshCall struct {
	host    string
	command string
	options SSHOptions
}

type fakeSSHRunner struct {
	mu      sync.Mutex
	calls   []sshCall
	outputs map[string]string
	errs    map[string]error
}

func (f *fakeSSHRunner) Run(host, command string, options SSHOptions) error {
	f.record(host, command, options)

	return f.errs[host]
}

func (f *fakeSSHRunner) RunOutput(host, command string, options SSHOptions) (string, error) {
	f.record(host, command, options)

	return f.outputs[host], f.errs[host]
}

func (f *fakeSSHRunner) record(host, command string, options SSHOptions) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, sshCall{host: host, command: command, options: options})
}

func (f *fakeSSHRunner) getCalls() []sshCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	calls := make([]sshCall, len(f.calls))
	copy(calls, f.calls)

	return calls
}

func TestNewSSHCommandUsesFixedExecutableAndPath(t *testing.T) {
	unsafePath := filepath.Join(t.TempDir(), "bin")
	t.Setenv("PATH", unsafePath)
	config := currentSSHPlatformConfig()

	cmd := newSSHCommand(
		"web1.example.com",
		"hostname",
		SSHOptions{KeyPath: "/tmp/test-key"},
	)

	assert.Equal(t, config.executablePath, cmd.Path)
	assert.Equal(
		t,
		[]string{config.executablePath, "-i", "/tmp/test-key", "web1.example.com", "hostname"},
		cmd.Args,
	)
	assert.Equal(t, config.fixedPath, envValue(cmd.Env, "PATH"))
}

func TestSSHPlatformConfigForSupportedOSes(t *testing.T) {
	tests := []struct {
		name string
		goos string
		want sshPlatformConfig
	}{
		{
			name: "linux",
			goos: "linux",
			want: sshPlatformConfig{
				executablePath: "/usr/bin/ssh",
				fixedPath:      "/usr/bin:/bin:/usr/sbin:/sbin",
			},
		},
		{
			name: "macos",
			goos: "darwin",
			want: sshPlatformConfig{
				executablePath: "/usr/bin/ssh",
				fixedPath:      "/usr/bin:/bin:/usr/sbin:/sbin",
			},
		},
		{
			name: "windows",
			goos: "windows",
			want: sshPlatformConfig{
				executablePath: `C:\Windows\System32\OpenSSH\ssh.exe`,
				fixedPath:      `C:\Windows\System32\OpenSSH;C:\Windows\System32;C:\Windows`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sshPlatformConfigFor(tt.goos))
		})
	}
}

func TestSSHEnvironmentReplacesPathCaseInsensitively(t *testing.T) {
	env := sshEnvironment(
		[]string{
			"HOME=/home/test",
			"PATH=/tmp/bin",
			"Path=C:\\Users\\test\\bin",
		},
		"/usr/bin:/bin",
	)

	assert.Equal(
		t,
		[]string{"HOME=/home/test", "PATH=/usr/bin:/bin"},
		env,
	)
}

func envValue(env []string, key string) string {
	for _, entry := range env {
		envKey, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(envKey, key) {
			return value
		}
	}

	return ""
}

func TestParseHosts(t *testing.T) {
	hosts := ParseHosts(" web1.example.com,web2.example.com, , user@db1 ")

	assert.Equal(t, []string{"web1.example.com", "web2.example.com", "user@db1"}, hosts)
}

func TestParseHostFile(t *testing.T) {
	testDir := t.TempDir()
	hostFile := filepath.Join(testDir, "hosts.txt")
	err := os.WriteFile(
		hostFile,
		[]byte("\n# web hosts\nweb1.example.com\n  web2.example.com  \n\n# db hosts\nuser@db1\n"),
		0644,
	)
	assert.NoError(t, err)

	hosts, err := ParseHostFile(hostFile)

	assert.NoError(t, err)
	assert.Equal(t, []string{"web1.example.com", "web2.example.com", "user@db1"}, hosts)
}

func TestMergeHosts(t *testing.T) {
	hosts := MergeHosts(
		[]string{"web1.example.com", "web2.example.com"},
		[]string{"web2.example.com", "user@db1"},
	)

	assert.Equal(t, []string{"web1.example.com", "web2.example.com", "user@db1"}, hosts)
}

func TestGetSSHHosts(t *testing.T) {
	testDir := t.TempDir()
	hostFile := filepath.Join(testDir, "hosts.txt")
	err := os.WriteFile(hostFile, []byte("web2.example.com\nuser@db1\n"), 0644)
	assert.NoError(t, err)

	hosts, err := GetSSHHosts("web1.example.com,web2.example.com", hostFile)

	assert.NoError(t, err)
	assert.Equal(t, []string{"web1.example.com", "web2.example.com", "user@db1"}, hosts)
}

func TestFormatSSHCommand(t *testing.T) {
	command := FormatSSHCommand("web1.example.com", "hostname; uptime")

	assert.Equal(t, `ssh web1.example.com "hostname; uptime"`, command)
}

func TestFormatSSHCommandWithOptions(t *testing.T) {
	command := FormatSSHCommandWithOptions(
		"web1.example.com",
		"hostname; uptime",
		SSHOptions{KeyPath: "/tmp/test key"},
	)

	assert.Equal(t, `ssh -i "/tmp/test key" web1.example.com "hostname; uptime"`, command)
}

func TestRunCommandOnHostsSequentialUsesSSHRunner(t *testing.T) {
	runner := &fakeSSHRunner{
		outputs: map[string]string{},
		errs:    map[string]error{},
	}
	originalRunner := sshRunner
	sshRunner = runner
	t.Cleanup(func() {
		sshRunner = originalRunner
	})

	interruptChan := make(chan os.Signal, 1)
	results := RunCommandOnHostsSequential(
		[]string{"web1.example.com", "web2.example.com"},
		"hostname",
		false,
		true,
		interruptChan,
	)

	assert.NoError(t, results["web1.example.com"])
	assert.NoError(t, results["web2.example.com"])
	assert.Equal(
		t,
		[]sshCall{
			{host: "web1.example.com", command: "hostname"},
			{host: "web2.example.com", command: "hostname"},
		},
		runner.getCalls(),
	)
}

func TestRunCommandOnHostsSequentialWithOptionsUsesSSHRunner(t *testing.T) {
	runner := &fakeSSHRunner{
		outputs: map[string]string{},
		errs:    map[string]error{},
	}
	originalRunner := sshRunner
	sshRunner = runner
	t.Cleanup(func() {
		sshRunner = originalRunner
	})

	interruptChan := make(chan os.Signal, 1)
	results := RunCommandOnHostsSequentialWithOptions(
		[]string{"web1.example.com"},
		"hostname",
		SSHOptions{KeyPath: "/tmp/test-key"},
		false,
		true,
		interruptChan,
	)

	assert.NoError(t, results["web1.example.com"])
	assert.Equal(
		t,
		[]sshCall{
			{host: "web1.example.com", command: "hostname", options: SSHOptions{KeyPath: "/tmp/test-key"}},
		},
		runner.getCalls(),
	)
}

func TestRunCommandOnHostsParallelUsesSSHRunner(t *testing.T) {
	runner := &fakeSSHRunner{
		outputs: map[string]string{
			"web1.example.com": "web1 output",
			"web2.example.com": "web2 output",
		},
		errs: map[string]error{},
	}
	originalRunner := sshRunner
	sshRunner = runner
	t.Cleanup(func() {
		sshRunner = originalRunner
	})

	interruptChan := make(chan os.Signal, 1)
	results := RunCommandOnHostsParallel(
		[]string{"web1.example.com", "web2.example.com"},
		"hostname",
		false,
		true,
		interruptChan,
	)

	assert.NoError(t, results["web1.example.com"])
	assert.NoError(t, results["web2.example.com"])
	assert.ElementsMatch(
		t,
		[]sshCall{
			{host: "web1.example.com", command: "hostname"},
			{host: "web2.example.com", command: "hostname"},
		},
		runner.getCalls(),
	)
}

func TestRunCommandOnHostsParallelWithOptionsUsesSSHRunner(t *testing.T) {
	runner := &fakeSSHRunner{
		outputs: map[string]string{"web1.example.com": "web1 output"},
		errs:    map[string]error{},
	}
	originalRunner := sshRunner
	sshRunner = runner
	t.Cleanup(func() {
		sshRunner = originalRunner
	})

	interruptChan := make(chan os.Signal, 1)
	results := RunCommandOnHostsParallelWithOptions(
		[]string{"web1.example.com"},
		"hostname",
		SSHOptions{KeyPath: "/tmp/test-key"},
		false,
		true,
		interruptChan,
	)

	assert.NoError(t, results["web1.example.com"])
	assert.Equal(
		t,
		[]sshCall{
			{host: "web1.example.com", command: "hostname", options: SSHOptions{KeyPath: "/tmp/test-key"}},
		},
		runner.getCalls(),
	)
}
