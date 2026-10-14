package repair

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepair(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seq-repair-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Valid migration
	validFile := filepath.Join(tmpDir, "20260101-valid.js")
	_ = os.WriteFile(validFile, []byte(`
module.exports = {
  async up() {},
  async down() {}
};`), 0644)

	// Incomplete migration (missing down)
	brokenFile := filepath.Join(tmpDir, "20260102-missing-down.js")
	_ = os.WriteFile(brokenFile, []byte(`
module.exports = {
  async up() {}
};`), 0644)

	// Incomplete migration (missing up)
	noUpFile := filepath.Join(tmpDir, "20260103-missing-up.js")
	_ = os.WriteFile(noUpFile, []byte(`
module.exports = {
  async down() {}
};`), 0644)

	t.Run("Audits migration files correctly", func(t *testing.T) {
		issues, err := AuditMigrations(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 2 {
			t.Fatalf("expected 2 issues, got %d", len(issues))
		}
	})

	t.Run("Auto-repairs missing down handler", func(t *testing.T) {
		err := RepairMissingDown(brokenFile)
		if err != nil {
			t.Fatalf("failed to repair: %v", err)
		}

		// Re-repairing an already valid file should be a no-op
		err = RepairMissingDown(brokenFile)
		if err != nil {
			t.Fatalf("re-repairing should succeed as no-op: %v", err)
		}
	})

	t.Run("Fails gracefully when closing bracket is missing", func(t *testing.T) {
		corruptFile := filepath.Join(tmpDir, "corrupt.js")
		_ = os.WriteFile(corruptFile, []byte("module.exports = {"), 0644)

		err := RepairMissingDown(corruptFile)
		if err == nil {
			t.Errorf("expected error when closing bracket is missing")
		}
	})

	t.Run("Non-existing directory returns empty with no error", func(t *testing.T) {
		issues, err := AuditMigrations(filepath.Join(tmpDir, "non-existent"))
		if err != nil || len(issues) != 0 {
			t.Errorf("expected nil error and 0 issues for non-existent directory")
		}
	})

	t.Run("Non-existing file returns error on repair", func(t *testing.T) {
		err := RepairMissingDown(filepath.Join(tmpDir, "does-not-exist.js"))
		if err == nil {
			t.Errorf("expected error when repairing non-existent file")
		}
	})

	t.Run("Recognizes various valid up and down syntax without false positives", func(t *testing.T) {
		syntaxDir, err := os.MkdirTemp("", "seq-syntax-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(syntaxDir)

		// 1. Arrow functions
		arrowFile := filepath.Join(syntaxDir, "20260101-arrow.js")
		_ = os.WriteFile(arrowFile, []byte(`
module.exports = {
  up: async (queryInterface) => {},
  down: (queryInterface) => {}
};`), 0644)

		// 2. Standard ES6 methods without async
		syncFile := filepath.Join(syntaxDir, "20260102-sync.js")
		_ = os.WriteFile(syncFile, []byte(`
module.exports = {
  up(queryInterface, Sequelize) {},
  down(queryInterface, Sequelize) {}
};`), 0644)

		issues, err := AuditMigrations(syntaxDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for valid syntax variants, got: %+v", issues)
		}
	})

	t.Run("Repair safely handles raw query strings containing closing brace", func(t *testing.T) {
		queryFile := filepath.Join(tmpDir, "20260104-raw-query.js")
		content := `module.exports = {
  async up(queryInterface, Sequelize) {
    await queryInterface.sequelize.query("SELECT * FROM users WHERE status LIKE '};'");
  }
};`
		_ = os.WriteFile(queryFile, []byte(content), 0644)

		err := RepairMissingDown(queryFile)
		if err != nil {
			t.Fatalf("failed to repair file with string containing closing brace: %v", err)
		}

		repairedBytes, _ := os.ReadFile(queryFile)
		repairedStr := string(repairedBytes)
		if !strings.Contains(repairedStr, "WHERE status LIKE '};'") {
			t.Errorf("raw query string was corrupted during repair: %s", repairedStr)
		}
		if !strings.Contains(repairedStr, "async down(queryInterface, Sequelize)") {
			t.Errorf("down handler was not appended properly: %s", repairedStr)
		}
	})
}
