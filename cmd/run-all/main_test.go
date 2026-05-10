package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSSHFlags(t *testing.T) {
	tests := []struct {
		name            string
		command         string
		dirPattern      string
		excludePatterns string
		requireFolder   string
		hosts           string
		hostFile        string
		wantErr         string
	}{
		{
			name:    "valid hosts flag",
			command: "hostname",
			hosts:   "web1.example.com",
		},
		{
			name:     "valid host file",
			command:  "hostname",
			hostFile: "hosts.txt",
		},
		{
			name:    "missing command",
			hosts:   "web1.example.com",
			wantErr: "command must be provided using the -command flag",
		},
		{
			name:       "dir pattern is incompatible",
			command:    "hostname",
			dirPattern: "/tmp/*",
			hosts:      "web1.example.com",
			wantErr:    "-dir-pattern cannot be used with -ssh",
		},
		{
			name:            "exclude is incompatible",
			command:         "hostname",
			excludePatterns: "backup",
			hosts:           "web1.example.com",
			wantErr:         "-exclude cannot be used with -ssh",
		},
		{
			name:          "require is incompatible",
			command:       "hostname",
			requireFolder: "go.mod",
			hosts:         "web1.example.com",
			wantErr:       "-require cannot be used with -ssh",
		},
		{
			name:    "missing hosts",
			command: "hostname",
			wantErr: "SSH mode requires -hosts or -host-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSSHFlags(
				tt.command,
				tt.dirPattern,
				tt.excludePatterns,
				tt.requireFolder,
				tt.hosts,
				tt.hostFile,
			)

			if tt.wantErr == "" {
				assert.NoError(t, err)

				return
			}
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestValidateLocalFlags(t *testing.T) {
	tests := []struct {
		name     string
		hosts    string
		hostFile string
		sshKey   string
		wantErr  string
	}{
		{
			name: "valid local flags",
		},
		{
			name:    "hosts require ssh mode",
			hosts:   "web1.example.com",
			wantErr: "-hosts can only be used with -ssh",
		},
		{
			name:     "host file requires ssh mode",
			hostFile: "hosts.txt",
			wantErr:  "-host-file can only be used with -ssh",
		},
		{
			name:    "ssh key requires ssh mode",
			sshKey:  "/tmp/test-key",
			wantErr: "-ssh-key can only be used with -ssh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLocalFlags(tt.hosts, tt.hostFile, tt.sshKey)

			if tt.wantErr == "" {
				assert.NoError(t, err)

				return
			}
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
