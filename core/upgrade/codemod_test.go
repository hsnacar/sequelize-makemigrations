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
    await queryInterface.addColumn('Orders', 'weight', {
      type: Sequelize.DOUBLE PRECISION
    });
  }
};`
	_ = os.WriteFile(legacyFile, []byte(originalContent), 0644)

	t.Run("Dry run does not mutate file", func(t *testing.T) {
		res, err := UpgradeFile(legacyFile, true, false)
		if err != nil {
			t.Fatalf("dry run failed: %v", err)
		}
		if res.ChangesMade < 3 {
			t.Errorf("expected at least 3 changes, got %d", res.ChangesMade)
		}

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

		if res.BackupPath == "" {
			t.Errorf("expected backup path to be set")
		}
		backupContent, err := os.ReadFile(res.BackupPath)
		if err != nil || string(backupContent) != originalContent {
			t.Errorf("backup does not match original content")
		}

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
		if !strings.Contains(str, "DataTypes.DOUBLE") {
			t.Errorf("expected DataTypes.DOUBLE in upgraded file")
		}
		if !strings.Contains(str, "@sequelize/core") {
			t.Errorf("expected @sequelize/core import header in upgraded file")
		}
	})

	t.Run("File requiring no changes returns 0 changes and no backup", func(t *testing.T) {
		cleanFile := filepath.Join(tmpDir, "20220101-clean.js")
		_ = os.WriteFile(cleanFile, []byte("const { DataTypes } = require('@sequelize/core'); module.exports = {};"), 0644)

		res, err := UpgradeFile(cleanFile, false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ChangesMade != 0 {
			t.Errorf("expected 0 changes, got %d", res.ChangesMade)
		}
		if res.BackupPath != "" {
			t.Errorf("expected no backup path for untouched file")
		}
	})

	t.Run("Handles file without use strict header", func(t *testing.T) {
		noHeaderFile := filepath.Join(tmpDir, "20230101-noheader.js")
		_ = os.WriteFile(noHeaderFile, []byte("module.exports = { type: Sequelize.NUMERIC };"), 0644)

		res, err := UpgradeFile(noHeaderFile, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content, _ := os.ReadFile(noHeaderFile)
		if !strings.Contains(string(content), "const { DataTypes } = require('@sequelize/core');") {
			t.Errorf("expected DataTypes import header prepended")
		}
		if res.ChangesMade == 0 {
			t.Errorf("expected changes to be made")
		}
	})

	t.Run("Non-existent file returns error", func(t *testing.T) {
		_, err := UpgradeFile(filepath.Join(tmpDir, "non-existent.js"), false, false)
		if err == nil {
			t.Errorf("expected error for non-existent file")
		}
	})

	t.Run("UpgradeDirectory processes files correctly", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "mig-dir")
		_ = os.Mkdir(dir, 0755)

		f1 := filepath.Join(dir, "01-test.js")
		_ = os.WriteFile(f1, []byte("module.exports = { val: Sequelize.NUMERIC };"), 0644)

		f2 := filepath.Join(dir, "02-test.ts")
		_ = os.WriteFile(f2, []byte("module.exports = { val: Sequelize.JSONTYPE };"), 0644)

		// Non-js file should be ignored
		f3 := filepath.Join(dir, "readme.txt")
		_ = os.WriteFile(f3, []byte("Sequelize.NUMERIC"), 0644)

		results, err := UpgradeDirectory(dir, false, true)
		if err != nil {
			t.Fatalf("UpgradeDirectory failed: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 files transformed, got %d", len(results))
		}

		// Invalid directory returns error
		_, err = UpgradeDirectory(filepath.Join(tmpDir, "invalid-dir"), false, false)
		if err == nil {
			t.Errorf("expected error for invalid directory")
		}
	})
}
