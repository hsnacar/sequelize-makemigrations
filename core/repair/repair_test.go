package repair

import (
	"os"
	"path/filepath"
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
}
