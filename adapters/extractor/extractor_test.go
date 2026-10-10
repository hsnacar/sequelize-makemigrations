package extractor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNodeExtractor(t *testing.T) {
	// Root of repo to locate node_modules with sequelize
	repoRoot, _ := filepath.Abs("../..")
	nodeModulesSrc := filepath.Join(repoRoot, "examples", "ecommerce-app", "node_modules")

	tmpDir, err := os.MkdirTemp("", "seq-ext-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Symlink node_modules so require('sequelize') works
	_ = os.Symlink(nodeModulesSrc, filepath.Join(tmpDir, "node_modules"))

	t.Run("Fails gracefully when models cannot be found", func(t *testing.T) {
		emptyDir, _ := os.MkdirTemp("", "seq-empty-*")
		defer os.RemoveAll(emptyDir)

		ext := NewNodeExtractor("nonexistent/models.js")
		_, err := ext.ExtractSchema(context.Background(), emptyDir)
		if err == nil {
			t.Errorf("expected error when extracting from empty directory")
		}
	})

	t.Run("Successfully extracts schema from valid Sequelize models", func(t *testing.T) {
		modelsDir := filepath.Join(tmpDir, "models")
		_ = os.MkdirAll(modelsDir, 0755)

		modelCode := `'use strict';
const { Sequelize, DataTypes } = require('sequelize');
const sequelize = new Sequelize({ dialect: 'sqlite', storage: ':memory:', logging: false });

const User = sequelize.define('User', {
  id: { type: DataTypes.INTEGER, primaryKey: true, autoIncrement: true },
  email: { type: DataTypes.STRING(255), allowNull: false }
}, { tableName: 'Users', timestamps: false });

module.exports = { sequelize, User };`
		_ = os.WriteFile(filepath.Join(modelsDir, "index.js"), []byte(modelCode), 0644)

		ext := NewNodeExtractor("models/index.js")
		schema, err := ext.ExtractSchema(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("expected successful schema extraction, got: %v", err)
		}

		if schema == nil || len(schema.Tables) != 1 {
			t.Fatalf("expected 1 table extracted, got: %+v", schema)
		}

		userTable, ok := schema.Tables["Users"]
		if !ok {
			t.Fatalf("expected Users table in extracted schema")
		}

		if len(userTable.Columns) != 2 {
			t.Errorf("expected 2 columns (id, email), got %d", len(userTable.Columns))
		}

		if userTable.Columns["email"].Type.Name != "STRING" {
			t.Errorf("expected email type to be STRING, got %s", userTable.Columns["email"].Type.Name)
		}
	})
}
