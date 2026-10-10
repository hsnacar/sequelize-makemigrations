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

	t.Run("Audits migration files correctly", func(t *testing.T) {
		issues, err := AuditMigrations(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		if issues[0].File != "20260102-missing-down.js" {
			t.Errorf("expected issue on broken file, got %s", issues[0].File)
		}
	})

	t.Run("Auto-repairs missing down handler", func(t *testing.T) {
		err := RepairMissingDown(brokenFile)
		if err != nil {
			t.Fatalf("failed to repair: %v", err)
		}

		issues, err := AuditMigrations(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 0 {
			t.Errorf("expected 0 issues after repair, got %d", len(issues))
		}
	})
}
