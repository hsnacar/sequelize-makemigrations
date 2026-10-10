package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuditDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seq-audit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	legacyMigration := filepath.Join(tmpDir, "20200101-legacy.js")
	content := `'use strict';
module.exports = {
  async up(queryInterface, Sequelize) {
    await queryInterface.createTable('Legacy', {
      amount: { type: Sequelize.NUMERIC(10, 2) },
      metadata: { type: Sequelize.JSONTYPE },
      rate: { type: Sequelize.DOUBLE PRECISION },
      name: { type: Sequelize.STRING }
    });
  }
};`
	_ = os.WriteFile(legacyMigration, []byte(content), 0644)

	t.Run("Detects breaking and warning findings", func(t *testing.T) {
		report, err := AuditDirectory(tmpDir, "v7")
		if err != nil {
			t.Fatalf("audit failed: %v", err)
		}

		if report.TotalFilesScanned != 1 {
			t.Errorf("expected 1 file scanned, got %d", report.TotalFilesScanned)
		}

		// 3 breaking findings: NUMERIC, JSONTYPE, DOUBLE PRECISION
		if report.BreakingCount != 3 {
			t.Errorf("expected 3 breaking findings, got %d", report.BreakingCount)
		}

		// 1 warning finding: Sequelize.STRING
		if report.WarningCount != 1 {
			t.Errorf("expected 1 warning finding, got %d", report.WarningCount)
		}
	})

	t.Run("Non-existent directory returns clean report with no error", func(t *testing.T) {
		report, err := AuditDirectory(filepath.Join(tmpDir, "does-not-exist"), "v7")
		if err != nil {
			t.Fatalf("expected no error on non-existent dir, got: %v", err)
		}
		if report.TotalFilesScanned != 0 || len(report.Findings) != 0 {
			t.Errorf("expected 0 files scanned on non-existent dir")
		}
	})

	t.Run("Clean directory returns 0 findings", func(t *testing.T) {
		cleanDir := filepath.Join(tmpDir, "clean-migrations")
		_ = os.Mkdir(cleanDir, 0755)
		_ = os.WriteFile(filepath.Join(cleanDir, "01-clean.js"), []byte("const { DataTypes } = require('@sequelize/core');"), 0644)

		report, err := AuditDirectory(cleanDir, "v7")
		if err != nil {
			t.Fatalf("clean audit failed: %v", err)
		}
		if report.BreakingCount != 0 || report.WarningCount != 0 {
			t.Errorf("expected 0 findings in clean dir")
		}
	})
}
