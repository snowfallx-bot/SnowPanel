package service

import "testing"

func TestValidateCronCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "allowlisted style template",
			command: "backup",
			wantErr: false,
		},
		{
			name:    "reject pipeline token",
			command: "backup|sh",
			wantErr: true,
		},
		{
			name:    "reject redirect token",
			command: "backup>tmp",
			wantErr: true,
		},
		{
			name:    "reject subcommand token",
			command: "backup$(id)",
			wantErr: true,
		},
		{
			name:    "reject whitespace separated shell command",
			command: "/bin/sh -c backup",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCronCommand(tc.command)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for command %q, got nil", tc.command)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for command %q: %v", tc.command, err)
			}
		})
	}
}
