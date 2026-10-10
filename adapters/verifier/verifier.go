package verifier

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

type SandboxVerifier struct {
	ProjectPath string
}

func NewSandboxVerifier(projectPath string) ports.Verifier {
	return &SandboxVerifier{
		ProjectPath: projectPath,
	}
}

func (v *SandboxVerifier) Verify(ctx context.Context, migration *ports.GeneratedMigration) error {
	// 1. Create a temporary migration file in a sandbox directory
	tmpDir, err := os.MkdirTemp("", "seq-verify-*")
	if err != nil {
		return fmt.Errorf("failed to create sandbox directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	migrationPath := filepath.Join(tmpDir, migration.Filename)
	if err := os.WriteFile(migrationPath, []byte(migration.Content), 0644); err != nil {
		return fmt.Errorf("failed to write sandbox migration: %w", err)
	}

	// 2. Prepare verification runner script using an in-memory SQLite sandbox
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
  const migration = require(%q);

  try {
    // 1. Test UP
    await migration.up(queryInterface, Sequelize);
  } catch (err) {
    console.error("FAIL_UP:" + err.message);
    process.exit(10);
  }

  try {
    // 2. Test DOWN (Reversibility check)
    await migration.down(queryInterface, Sequelize);
  } catch (err) {
    console.error("FAIL_DOWN:" + err.message);
    process.exit(11);
  }

  await sequelize.close();
  process.exit(0);
}

testMigration().catch(err => {
  console.error("FAIL_SANDBOX:" + err.message);
  process.exit(12);
});
`, migrationPath)

	runnerPath := filepath.Join(tmpDir, "verify-runner.js")
	if err := os.WriteFile(runnerPath, []byte(runnerScript), 0644); err != nil {
		return fmt.Errorf("failed to write verification runner script: %w", err)
	}

	// 3. Run sandbox test
	cmd := exec.CommandContext(ctx, "node", runnerPath)
	cmd.Dir = v.ProjectPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sandbox verification failed:\n%s", string(output))
	}

	return nil
}
