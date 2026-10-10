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
      rate: { type: Sequelize.DOUBLE PRECISION }
    });
  }
};`
	_ = os.WriteFile(legacyMigration, []byte(content), 0644)

	report, err := AuditDirectory(tmpDir, "v7")
	if err != nil {
		t.Fatalf("audit failed: %v", err)
	}

	if report.TotalFilesScanned != 1 {
		t.Errorf("expected 1 file scanned, got %d", report.TotalFilesScanned)
	}

	// 3 breaking findings: NUMERIC, JSONTYPE, DOUBLE PRECISION
	if report.BreakingCount != 3 {
		t.Errorf("expected 3 breaking findings, got %d (findings: %+v)", report.BreakingCount, report.Findings)
	}
}
