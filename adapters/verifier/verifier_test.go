package verifier

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

func TestSandboxVerifier(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seq-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.Mkdir(filepath.Join(tmpDir, "migrations"), 0755)

	v := NewSandboxVerifier(tmpDir)
	if v == nil {
		t.Fatal("expected NewSandboxVerifier to return non-nil")
	}

	// Test with a mock migration (if node is available)
	t.Run("Fails gracefully when migration script has runtime errors", func(t *testing.T) {
		brokenMig := &ports.GeneratedMigration{
			Filename: "20260101-broken.js",
			Content: `module.exports = {
  async up() { throw new Error("UP_CRASH"); },
  async down() {}
};`,
		}

		err := v.Verify(context.Background(), brokenMig)
		if err == nil {
			t.Errorf("expected verification to fail for crashing UP migration")
		}
	})
}
