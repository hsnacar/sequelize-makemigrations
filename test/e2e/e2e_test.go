package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEndToEndSequelizeWorkflow(t *testing.T) {
	// 1. Locate repository root and build binary
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to determine repo root: %v", err)
	}
	binPath := filepath.Join(repoRoot, "bin", "seq-migrate")

	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/seq-migrate")
	buildCmd.Dir = repoRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %s", string(out))
	}

	// 2. Prepare isolated workspace directory
	workDir, err := os.MkdirTemp("", "seq-e2e-workspace-*")
	if err != nil {
		t.Fatalf("failed to create temp workspace: %v", err)
	}
	defer os.RemoveAll(workDir)

	modelsDir := filepath.Join(workDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatalf("failed to create models dir: %v", err)
	}

	// Link node_modules from example app to avoid re-downloading in test
	exampleNodeModules := filepath.Join(repoRoot, "examples", "ecommerce-app", "node_modules")
	targetNodeModules := filepath.Join(workDir, "node_modules")
	if err := os.Symlink(exampleNodeModules, targetNodeModules); err != nil {
		t.Fatalf("failed to symlink node_modules: %v", err)
	}

	dbFile := filepath.Join(workDir, "real_database.sqlite")

	// Helper to write models/index.js
	writeModels := func(userFields, productFields string) {
		content := fmt.Sprintf(`'use strict';
const { Sequelize, DataTypes } = require('sequelize');

const sequelize = new Sequelize({
  dialect: 'sqlite',
  storage: %q,
  logging: false
});

const User = sequelize.define('User', {
  id: { type: DataTypes.INTEGER, primaryKey: true, autoIncrement: true },
  name: { type: DataTypes.STRING, allowNull: false },
%s
}, { tableName: 'Users', timestamps: false });

const Product = sequelize.define('Product', {
  id: { type: DataTypes.INTEGER, primaryKey: true, autoIncrement: true },
  title: { type: DataTypes.STRING, allowNull: false },
%s
}, { tableName: 'Products', timestamps: false });

module.exports = { sequelize, User, Product };
`, dbFile, userFields, productFields)

		err := os.WriteFile(filepath.Join(modelsDir, "index.js"), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write models: %v", err)
		}
	}

	// Helper to execute generated migrations on the real SQLite DB
	applyMigration := func(migrationFile, direction string) error {
		script := fmt.Sprintf(`'use strict';
const { Sequelize } = require('sequelize');
const sequelize = new Sequelize({ dialect: 'sqlite', storage: %q, logging: false });
const queryInterface = sequelize.getQueryInterface();
const migration = require(%q);

async function run() {
  if (%q === 'down') {
    await migration.down(queryInterface, Sequelize);
  } else {
    await migration.up(queryInterface, Sequelize);
  }
  await sequelize.close();
}
run().catch(err => { console.error(err); process.exit(1); });
`, dbFile, migrationFile, direction)

		cmd := exec.Command("node", "-e", script)
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("migration run failed: %s", string(out))
		}
		return nil
	}

	// Helper to query actual database table columns from SQLite with strict error checking
	getActualColumns := func(tableName string) ([]string, error) {
		script := fmt.Sprintf(`'use strict';
const { Sequelize } = require('sequelize');
const sequelize = new Sequelize({ dialect: 'sqlite', storage: %q, logging: false });

async function inspect() {
  const [results] = await sequelize.query("PRAGMA table_info(" + %q + ");");
  console.log(JSON.stringify(results.map(r => r.name)));
  await sequelize.close();
}
inspect().catch(err => { console.error(err); process.exit(1); });
`, dbFile, tableName)

		cmd := exec.Command("node", "-e", script)
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("inspection failed: %s", string(out))
		}
		var cols []string
		if err := json.Unmarshal(out, &cols); err != nil {
			return nil, fmt.Errorf("parse error: %w (out: %s)", err, string(out))
		}
		return cols, nil
	}

	// ==========================================
	// STAGE 1: Initial Migration Creation
	// ==========================================
	t.Run("Stage 1: Generate initial schema migration with verify", func(t *testing.T) {
		// Define baseline models with proper attributes
		writeModels(
			"  email: { type: DataTypes.STRING, allowNull: false, unique: true }",
			"  price: { type: DataTypes.DECIMAL(10, 2), allowNull: false }",
		)

		// Run: seq-migrate make --name=init-schema --verify
		cmd := exec.Command(binPath, "make", "--name=init-schema", "--verify")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("seq-migrate make failed: %s, err: %v", string(out), err)
		}

		if !strings.Contains(string(out), "Migration created:") {
			t.Errorf("expected success message in output, got: %s", string(out))
		}
		if !strings.Contains(string(out), "Shadow sandbox verified!") {
			t.Errorf("expected verify message in output, got: %s", string(out))
		}

		// 1. Verify that migration file exists in migrations/
		migrationsDir := filepath.Join(workDir, "migrations")
		entries, err := os.ReadDir(migrationsDir)
		if err != nil || len(entries) != 1 {
			t.Fatalf("expected 1 migration file, found %d", len(entries))
		}

		migrationFile := filepath.Join(migrationsDir, entries[0].Name())
		if !strings.HasSuffix(migrationFile, "-init-schema.js") {
			t.Errorf("unexpected migration name: %s", migrationFile)
		}

		// 2. Verify that .seq-schema.json snapshot was created
		snapshotPath := filepath.Join(workDir, ".seq-schema.json")
		if _, err := os.Stat(snapshotPath); err != nil {
			t.Fatalf("expected .seq-schema.json to exist: %v", err)
		}

		// 3. Physically execute the migration UP on the real SQLite DB!
		if err := applyMigration(migrationFile, "up"); err != nil {
			t.Fatalf("failed to apply initial migration to real DB: %v", err)
		}

		// 4. Verify physical database schema in SQLite!
		userCols, err := getActualColumns("Users")
		if err != nil {
			t.Fatalf("error checking Users: %v", err)
		}
		expectedUserCols := []string{"id", "email", "name"}
		if strings.Join(userCols, ",") != strings.Join(expectedUserCols, ",") {
			t.Errorf("Users columns mismatch! Expected %v, got %v", expectedUserCols, userCols)
		}

		prodCols, err := getActualColumns("Products")
		if err != nil {
			t.Fatalf("error checking Products: %v", err)
		}
		expectedProdCols := []string{"id", "price", "title"}
		if strings.Join(prodCols, ",") != strings.Join(expectedProdCols, ",") {
			t.Errorf("Products columns mismatch! Expected %v, got %v", expectedProdCols, prodCols)
		}
	})

	// ==========================================
	// STAGE 2: CI Check (No Drift)
	// ==========================================
	t.Run("Stage 2: Check command confirms 100% sync", func(t *testing.T) {
		cmd := exec.Command(binPath, "check")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("seq-migrate check failed unexpectedly: %s", string(out))
		}

		if !strings.Contains(string(out), "100% in sync") {
			t.Errorf("expected clean schema message, got: %s", string(out))
		}
	})

	// ==========================================
	// STAGE 3: Incremental Column Additions & Drift Detection
	// ==========================================
	t.Run("Stage 3: Add new column, detect drift, and generate migration", func(t *testing.T) {
		// Update models: add User.bio and Product.sku
		writeModels(
			"  email: { type: DataTypes.STRING, allowNull: false, unique: true },\n  bio: { type: DataTypes.STRING(500), allowNull: true }",
			"  price: { type: DataTypes.DECIMAL(10, 2), allowNull: false },\n  sku: { type: DataTypes.STRING(50), allowNull: true }",
		)

		// 1. Run check -> Should fail with drift!
		checkCmd := exec.Command(binPath, "check")
		checkCmd.Dir = workDir
		checkCmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		checkOut, checkErr := checkCmd.CombinedOutput()
		if checkErr == nil {
			t.Fatalf("expected check command to exit with code 1 due to drift, but succeeded!")
		}
		if !strings.Contains(string(checkOut), "Schema Drift Detected") {
			t.Errorf("expected Schema Drift Detected in check output, got: %s", string(checkOut))
		}

		// 2. Generate incremental migration with --verify
		makeCmd := exec.Command(binPath, "make", "--name=add-bio-and-sku", "--verify")
		makeCmd.Dir = workDir
		makeCmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		makeOut, makeErr := makeCmd.CombinedOutput()
		if makeErr != nil {
			t.Fatalf("incremental make failed: %s, err: %v", string(makeOut), makeErr)
		}

		// Check that second migration file exists
		migrationsDir := filepath.Join(workDir, "migrations")
		entries, _ := os.ReadDir(migrationsDir)
		var migrations []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".js") {
				migrations = append(migrations, filepath.Join(migrationsDir, e.Name()))
			}
		}
		if len(migrations) != 2 {
			t.Fatalf("expected 2 migration files, found %d", len(migrations))
		}

		// 3. Apply incremental migration to real database
		secondMigration := migrations[1]
		if err := applyMigration(secondMigration, "up"); err != nil {
			t.Fatalf("failed to apply incremental migration: %v", err)
		}

		// 4. Verify that "bio" and "sku" now physically exist in real SQLite DB with strict error checking!
		userCols, err := getActualColumns("Users")
		if err != nil {
			t.Fatalf("error checking Users columns: %v", err)
		}
		expectedUserCols := []string{"id", "email", "name", "bio"}
		if strings.Join(userCols, ",") != strings.Join(expectedUserCols, ",") {
			t.Errorf("Users columns mismatch after incremental migration! Expected %v, got %v", expectedUserCols, userCols)
		}

		prodCols, err := getActualColumns("Products")
		if err != nil {
			t.Fatalf("error checking Products columns: %v", err)
		}
		expectedProdCols := []string{"id", "price", "title", "sku"}
		if strings.Join(prodCols, ",") != strings.Join(expectedProdCols, ",") {
			t.Errorf("Products columns mismatch after incremental migration! Expected %v, got %v", expectedProdCols, prodCols)
		}

		// ==========================================
		// STAGE 4: Rollback Test (Reversibility Guarantee)
		// ==========================================
		// Roll back the second migration on the real DB!
		if err := applyMigration(secondMigration, "down"); err != nil {
			t.Fatalf("failed to rollback migration: %v", err)
		}

		// Verify that "bio" and "sku" were cleanly removed and schema returned to initial state!
		revertedUserCols, err := getActualColumns("Users")
		if err != nil {
			t.Fatalf("error checking reverted Users columns: %v", err)
		}
		initialUserCols := []string{"id", "email", "name"}
		if strings.Join(revertedUserCols, ",") != strings.Join(initialUserCols, ",") {
			t.Errorf("Rollback failed! Expected %v, got %v", initialUserCols, revertedUserCols)
		}
	})

	// ==========================================
	// STAGE 5: Doctor & Upgrade (Version Modernization)
	// ==========================================
	t.Run("Stage 5: Doctor diagnoses legacy migration and upgrade modernizes it", func(t *testing.T) {
		legacyFile := filepath.Join(workDir, "migrations", "20200101000000-legacy-v6.js")
		legacyContent := `'use strict';
module.exports = {
  async up(queryInterface, Sequelize) {
    await queryInterface.createTable('OldData', {
      amount: { type: Sequelize.NUMERIC(10, 2) },
      payload: { type: Sequelize.JSONTYPE }
    });
  },
  async down(queryInterface, Sequelize) {
    await queryInterface.dropTable('OldData');
  }
};`
		if err := os.WriteFile(legacyFile, []byte(legacyContent), 0644); err != nil {
			t.Fatalf("failed to create legacy migration: %v", err)
		}

		// 1. Run Doctor -> Expect breaking findings
		docCmd := exec.Command(binPath, "doctor", "--target=v7")
		docCmd.Dir = workDir
		docOut, docErr := docCmd.CombinedOutput()
		if docErr == nil {
			t.Fatalf("expected doctor to fail with exit status 1 due to breaking types, but passed! Output: %s", string(docOut))
		}
		if !strings.Contains(string(docOut), "Sequelize.NUMERIC") || !strings.Contains(string(docOut), "Sequelize.JSONTYPE") {
			t.Errorf("expected NUMERIC and JSONTYPE breaking findings, got: %s", string(docOut))
		}

		// 2. Run Upgrade -> Modernize migration
		upgCmd := exec.Command(binPath, "upgrade", "--to=v7")
		upgCmd.Dir = workDir
		upgOut, upgErr := upgCmd.CombinedOutput()
		if upgErr != nil {
			t.Fatalf("upgrade failed: %s", string(upgOut))
		}
		if !strings.Contains(string(upgOut), "Successfully modernized") {
			t.Errorf("expected success message, got: %s", string(upgOut))
		}

		// Check that backup file was created
		if _, err := os.Stat(legacyFile + ".bak"); err != nil {
			t.Errorf("expected backup file to exist: %v", err)
		}

		// Check that file now has DataTypes.DECIMAL, DataTypes.JSON, and @sequelize/core import
		upgraded, _ := os.ReadFile(legacyFile)
		upStr := string(upgraded)
		if !strings.Contains(upStr, "DataTypes.DECIMAL(10, 2)") {
			t.Errorf("expected DataTypes.DECIMAL in modernized file, got: %s", upStr)
		}
		if !strings.Contains(upStr, "DataTypes.JSON") {
			t.Errorf("expected DataTypes.JSON in modernized file, got: %s", upStr)
		}
		if !strings.Contains(upStr, "@sequelize/core") {
			t.Errorf("expected @sequelize/core import in modernized file, got: %s", upStr)
		}

		// 3. Re-run Doctor -> Now should pass clean!
		cleanDocCmd := exec.Command(binPath, "doctor", "--target=v7")
		cleanDocCmd.Dir = workDir
		cleanDocOut, cleanDocErr := cleanDocCmd.CombinedOutput()
		if cleanDocErr != nil {
			t.Fatalf("expected doctor to pass after upgrade, got err: %v, out: %s", cleanDocErr, string(cleanDocOut))
		}
		if !strings.Contains(string(cleanDocOut), "Clean bill of health") {
			t.Errorf("expected clean bill of health, got: %s", string(cleanDocOut))
		}
	})

	// ==========================================
	// STAGE 6: CLI Subcommands & Edge Cases (status, dry-run, init, invalid args)
	// ==========================================
	t.Run("Stage 6: CLI subcommands and edge cases behave correctly", func(t *testing.T) {
		// 1. Test status command
		statusCmd := exec.Command(binPath, "status")
		statusCmd.Dir = workDir
		statusCmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		statusOut, statusErr := statusCmd.CombinedOutput()
		if statusErr != nil {
			t.Fatalf("status command failed: %v, out: %s", statusErr, string(statusOut))
		}

		// 2. Test make --dry-run
		dryCmd := exec.Command(binPath, "make", "--dry-run")
		dryCmd.Dir = workDir
		dryCmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		dryOut, dryErr := dryCmd.CombinedOutput()
		if dryErr != nil {
			t.Fatalf("dry-run make failed: %v, out: %s", dryErr, string(dryOut))
		}
		if !strings.Contains(string(dryOut), "No changes detected") && !strings.Contains(string(dryOut), "DRY RUN") {
			t.Errorf("expected dry run or no changes message, got: %s", string(dryOut))
		}

		// 3. Test invalid command -> Exits with code 1
		invalidCmd := exec.Command(binPath, "nonexistent-cmd")
		invalidCmd.Dir = workDir
		_, invalidErr := invalidCmd.CombinedOutput()
		if invalidErr == nil {
			t.Errorf("expected nonexistent command to fail with non-zero exit code")
		}

		// 4. Test repair command
		repairCmd := exec.Command(binPath, "repair", "--fix-down")
		repairCmd.Dir = workDir
		repairOut, repairErr := repairCmd.CombinedOutput()
		if repairErr != nil {
			t.Fatalf("repair command failed: %v, out: %s", repairErr, string(repairOut))
		}
		if !strings.Contains(string(repairOut), "Auditing and repairing migration files") {
			t.Errorf("expected repair auditing message, got: %s", string(repairOut))
		}
	})

	// ==========================================
	// STAGE 7: Sequelize v7 Generator Integration
	// ==========================================
	t.Run("Stage 7: Generates valid v7 migration with target=v7", func(t *testing.T) {
		// Make changes to trigger a diff
		writeModels(
			"  email: { type: DataTypes.STRING, allowNull: false, unique: true },\n  bio: { type: DataTypes.STRING(500), allowNull: true },\n  avatarUrl: { type: DataTypes.STRING(255), allowNull: true }",
			"  price: { type: DataTypes.DECIMAL(10, 2), allowNull: false },\n  sku: { type: DataTypes.STRING(50), allowNull: true }",
		)

		v7Cmd := exec.Command(binPath, "make", "--name=add-v7-avatar", "--target=v7", "--dry-run")
		v7Cmd.Dir = workDir
		v7Cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		v7Out, v7Err := v7Cmd.CombinedOutput()
		if v7Err != nil {
			t.Fatalf("v7 generation failed: %v, out: %s", v7Err, string(v7Out))
		}

		v7Str := string(v7Out)
		if !strings.Contains(v7Str, "@sequelize/core") {
			t.Errorf("expected @sequelize/core import in v7 generated code, got: %s", v7Str)
		}
		if !strings.Contains(v7Str, "DataTypes.") {
			t.Errorf("expected DataTypes.* in v7 generated code, got: %s", v7Str)
		}
	})
}
