package persistence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPasskeyCreatedAtMigrationExists(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", "000002_add_passkey_credentials_created_at.up.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := string(content)
	for _, required := range []string{
		"ALTER TABLE passkey_credentials",
		"ADD COLUMN IF NOT EXISTS created_at",
		"TIMESTAMPTZ NOT NULL",
		"DEFAULT NOW()",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration must contain %q", required)
		}
	}
}
