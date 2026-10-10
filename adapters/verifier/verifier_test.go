package verifier

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

func TestSandboxVerifier(t *testing.T) {
	// Root of repo to locate node_modules with sequelize
	repoRoot, _ := filepath.Abs("../..")
	nodeModulesSrc := filepath.Join(repoRoot, "examples", "ecommerce-app", "node_modules")

	tmpDir, err := os.MkdirTemp("", "seq-verify-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.Mkdir(filepath.Join(tmpDir, "migrations"), 0755)

	// Symlink node_modules so require('sequelize') works in sandbox
	_ = os.Symlink(nodeModulesSrc, filepath.Join(tmpDir, "node_modules"))

	v := NewSandboxVerifier(tmpDir)
	if v == nil {
		t.Fatal("expected NewSandboxVerifier to return non-nil")
	}

	t.Run("Successfully verifies clean reversible migration (UP and DOWN pass)", func(t *testing.T) {
		cleanMig := &ports.GeneratedMigration{
			Filename: "20260101-clean.js",
			Content: `'use strict';
module.exports = {
  async up(queryInterface, Sequelize) {
    await queryInterface.createTable('TestTable', {
      id: { type: Sequelize.INTEGER, primaryKey: true }
    });
  },
  async down(queryInterface, Sequelize) {
    await queryInterface.dropTable('TestTable');
  }
};`,
		}

		err := v.Verify(context.Background(), cleanMig)
		if err != nil {
			t.Fatalf("expected verification to pass for clean migration, got: %v", err)
		}
	})

	t.Run("Fails gracefully when UP fails in sandbox", func(t *testing.T) {
		upCrashMig := &ports.GeneratedMigration{
			Filename: "20260102-up-crash.js",
			Content: `'use strict';
module.exports = {
  async up() { throw new Error("CRASH_IN_UP"); },
  async down() {}
};`,
		}

		err := v.Verify(context.Background(), upCrashMig)
		if err == nil {
			t.Errorf("expected verification to fail when UP crashes")
		}
	})

	t.Run("Fails gracefully when DOWN rollback fails in sandbox", func(t *testing.T) {
		downCrashMig := &ports.GeneratedMigration{
			Filename: "20260103-down-crash.js",
			Content: `'use strict';
module.exports = {
  async up() {},
  async down() { throw new Error("CRASH_IN_DOWN"); }
};`,
		}

		err := v.Verify(context.Background(), downCrashMig)
		if err == nil {
			t.Errorf("expected verification to fail when DOWN crashes")
		}
	})
}
