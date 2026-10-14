package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

type SandboxVerifier struct {
	ProjectPath   string
	MigrationsDir string
}

func NewSandboxVerifier(projectPath string) ports.Verifier {
	return &SandboxVerifier{
		ProjectPath:   projectPath,
		MigrationsDir: "migrations",
	}
}

func (v *SandboxVerifier) Verify(ctx context.Context, migration *ports.GeneratedMigration) error {
	// 1. Read schema snapshot if available to reconstruct base schema
	var snapshotJSON string = "null"
	schemaPath := filepath.Join(v.ProjectPath, ".seq-schema.json")
	if data, err := os.ReadFile(schemaPath); err == nil && len(data) > 0 {
		var jsTest map[string]interface{}
		if json.Unmarshal(data, &jsTest) == nil {
			snapshotJSON = string(data)
		}
	}

	// 2. Create a temporary migration file in a sandbox directory
	tmpDir, err := os.MkdirTemp("", "seq-verify-*")
	if err != nil {
		return fmt.Errorf("failed to create sandbox directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	migrationPath := filepath.Join(tmpDir, migration.Filename)
	if err := os.WriteFile(migrationPath, []byte(migration.Content), 0644); err != nil {
		return fmt.Errorf("failed to write sandbox migration: %w", err)
	}

	// 3. Prepare verification runner script using an in-memory SQLite sandbox
	runnerScript := fmt.Sprintf(`'use strict';
const path = require('path');
const { Sequelize } = require('sequelize');

async function testMigration() {
  // Ephemeral in-memory SQLite sandbox
  const sequelize = new Sequelize({
    dialect: 'sqlite',
    storage: ':memory:',
    logging: false
  });

  const queryInterface = sequelize.getQueryInterface();
  const snapshotData = %s;
  const newMigration = require(%q);

  // 1. If schema snapshot exists, reconstruct base tables instead of executing past migrations
  if (snapshotData && snapshotData.tables) {
    for (const [tableName, table] of Object.entries(snapshotData.tables)) {
      const columns = {};
      for (const [colName, col] of Object.entries(table.columns || {})) {
        let type = Sequelize.STRING;
        const typeName = (col.type && col.type.name) || 'STRING';
        if (Sequelize[typeName]) {
          if (col.type && col.type.length) {
            type = Sequelize[typeName](col.type.length);
          } else if (col.type && col.type.precision && col.type.scale) {
            type = Sequelize[typeName](col.type.precision, col.type.scale);
          } else if (col.type && Array.isArray(col.type.values) && col.type.values.length) {
            type = Sequelize[typeName](...col.type.values);
          } else {
            type = Sequelize[typeName];
          }
        }
        columns[colName] = {
          type,
          allowNull: col.allowNull !== false,
          primaryKey: Boolean(col.primaryKey),
          autoIncrement: Boolean(col.autoIncrement)
        };
      }
      await queryInterface.createTable(tableName, columns);
    }
  }

  // 2. Test new migration UP
  try {
    await newMigration.up(queryInterface, Sequelize);
  } catch (err) {
    console.error("FAIL_UP: " + err.message);
    process.exit(10);
  }

  // 3. Test new migration DOWN (Reversibility check)
  try {
    await newMigration.down(queryInterface, Sequelize);
  } catch (err) {
    console.error("FAIL_DOWN: " + err.message);
    process.exit(11);
  }

  await sequelize.close();
  process.exit(0);
}

testMigration().catch(err => {
  console.error("FAIL_SANDBOX: " + err.message);
  process.exit(12);
});
`, snapshotJSON, migrationPath)

	runnerPath := filepath.Join(tmpDir, "verify-runner.js")
	if err := os.WriteFile(runnerPath, []byte(runnerScript), 0644); err != nil {
		return fmt.Errorf("failed to write verification runner script: %w", err)
	}

	// 4. Run sandbox test
	cmd := exec.CommandContext(ctx, "node", runnerPath)
	cmd.Dir = v.ProjectPath
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("NODE_PATH=%s", filepath.Join(v.ProjectPath, "node_modules")),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sandbox verification failed:\n%s", string(output))
	}

	return nil
}
