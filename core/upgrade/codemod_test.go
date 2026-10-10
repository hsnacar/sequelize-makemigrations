package upgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpgradeFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seq-codemod-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	legacyFile := filepath.Join(tmpDir, "20210101-legacy.js")
	originalContent := `'use strict';
module.exports = {
  async up(queryInterface, Sequelize) {
    await queryInterface.addColumn('Orders', 'amount', {
      type: Sequelize.NUMERIC(10, 2),
      allowNull: false
    });
    await queryInterface.addColumn('Orders', 'extra', {
      type: Sequelize.JSONTYPE
    });
  }
};`
	_ = os.WriteFile(legacyFile, []byte(originalContent), 0644)

	t.Run("Dry run does not mutate file", func(t *testing.T) {
		res, err := UpgradeFile(legacyFile, true, false)
		if err != nil {
			t.Fatalf("dry run failed: %v", err)
		}
		if res.ChangesMade != 2 {
			t.Errorf("expected 2 changes, got %d", res.ChangesMade)
		}

		// File content should be untouched
		curr, _ := os.ReadFile(legacyFile)
		if string(curr) != originalContent {
			t.Errorf("file was mutated during dry run!")
		}
	})

	t.Run("Applies codemod and creates backup", func(t *testing.T) {
		res, err := UpgradeFile(legacyFile, false, true)
		if err != nil {
			t.Fatalf("upgrade failed: %v", err)
		}
		if res.ChangesMade != 2 {
			t.Errorf("expected 2 changes, got %d", res.ChangesMade)
		}

		// Check backup
		if res.BackupPath == "" {
			t.Errorf("expected backup path to be set")
		}
		backupContent, err := os.ReadFile(res.BackupPath)
		if err != nil || string(backupContent) != originalContent {
			t.Errorf("backup does not match original content")
		}

		// Check transformed file
		upgradedContent, _ := os.ReadFile(legacyFile)
		str := string(upgradedContent)

		if strings.Contains(str, "Sequelize.NUMERIC") {
			t.Errorf("Sequelize.NUMERIC should have been replaced")
		}
		if !strings.Contains(str, "DataTypes.DECIMAL(10, 2)") {
			t.Errorf("expected DataTypes.DECIMAL(10, 2) in upgraded file")
		}
		if !strings.Contains(str, "DataTypes.JSON") {
			t.Errorf("expected DataTypes.JSON in upgraded file")
		}
		if !strings.Contains(str, "@sequelize/core") {
			t.Errorf("expected @sequelize/core import header in upgraded file")
		}
	})
}
