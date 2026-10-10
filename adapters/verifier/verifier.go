package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
	// 1. Collect and sort all existing migrations in project
	absMigrationsDir := filepath.Join(v.ProjectPath, v.MigrationsDir)
	existingFiles := []string{}

	if entries, err := os.ReadDir(absMigrationsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".js") {
				existingFiles = append(existingFiles, filepath.Join(absMigrationsDir, entry.Name()))
			}
		}
		sort.Strings(existingFiles)
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

	existingFilesJSON, _ := json.Marshal(existingFiles)

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
  const existingFiles = %s;
  const newMigration = require(%q);

  // 1. Replay all existing migrations in order
  for (const file of existingFiles) {
    try {
      const mig = require(file);
      if (typeof mig.up === 'function') {
        await mig.up(queryInterface, Sequelize);
      }
    } catch (err) {
      console.error("FAIL_PREVIOUS_MIGRATION (" + path.basename(file) + "): " + err.message);
      process.exit(9);
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
`, string(existingFilesJSON), migrationPath)

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
