package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestoreDrillDocsKeepCommandSanity(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	testCases := []struct {
		name     string
		path     string
		required []string
	}{
		{
			name: "english",
			path: filepath.Join(repoRoot, "docs", "restore-drill.md"),
			required: []string{
				`psql "$DATABASE_URL" < snowpanel-postgres.dump.sql`,
				"make up",
				"make up-host-agent",
				"curl -f http://127.0.0.1:8080/health",
				"curl -f http://127.0.0.1:8080/ready",
				"POST /api/v1/backups/:id/verify-task",
				"SNOWPANEL_ENCRYPTION_KEY",
				"request id",
			},
		},
		{
			name: "zh-CN",
			path: filepath.Join(repoRoot, "docs", "restore-drill.zh-CN.md"),
			required: []string{
				`psql "$DATABASE_URL" < snowpanel-postgres.dump.sql`,
				"make up",
				"make up-host-agent",
				"curl -f http://127.0.0.1:8080/health",
				"curl -f http://127.0.0.1:8080/ready",
				"POST /api/v1/backups/:id/verify-task",
				"SNOWPANEL_ENCRYPTION_KEY",
				"request id",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("failed to read restore drill doc: %v", err)
			}
			text := string(content)
			for _, snippet := range tc.required {
				if !strings.Contains(text, snippet) {
					t.Fatalf("restore drill doc %s is missing required snippet %q", tc.path, snippet)
				}
			}
		})
	}
}
