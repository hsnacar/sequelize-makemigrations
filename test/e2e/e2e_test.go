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
	// 1. Prepare isolated workspace directory
	workDir, err := os.MkdirTemp("", "seq-e2e-workspace-*")
	if err != nil {
		t.Fatalf("failed to create temp workspace: %v", err)
	}
	defer os.RemoveAll(workDir)

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to determine repo root: %v", err)
	}
	binPath := filepath.Join(workDir, "seq-migrate")

	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/seq-migrate")
	buildCmd.Dir = repoRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %s", string(out))
	}

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

	// Helper to write models/index.js with optional extra table options (e.g. indexes)
	writeModelsWithOptions := func(userFields, productFields, userExtraOpts, productExtraOpts string) {
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
}, { tableName: 'Users', timestamps: false %s });

const Product = sequelize.define('Product', {
  id: { type: DataTypes.INTEGER, primaryKey: true, autoIncrement: true },
  title: { type: DataTypes.STRING, allowNull: false },
%s
}, { tableName: 'Products', timestamps: false %s });

module.exports = { sequelize, User, Product };
`, dbFile, userFields, userExtraOpts, productFields, productExtraOpts)

		err := os.WriteFile(filepath.Join(modelsDir, "index.js"), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write models: %v", err)
		}
	}

	writeModels := func(userFields, productFields string) {
		writeModelsWithOptions(userFields, productFields, "", "")
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

	// Helper to query actual database table indexes from SQLite with strict error checking
	getActualIndexes := func(tableName string) ([]string, error) {
		script := fmt.Sprintf(`'use strict';
const { Sequelize } = require('sequelize');
const sequelize = new Sequelize({ dialect: 'sqlite', storage: %q, logging: false });

async function inspect() {
  const [results] = await sequelize.query("PRAGMA index_list(" + %q + ");");
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
			return nil, fmt.Errorf("index inspection failed: %s", string(out))
		}
		var indexes []string
		if err := json.Unmarshal(out, &indexes); err != nil {
			return nil, fmt.Errorf("parse error: %w (out: %s)", err, string(out))
		}
		return indexes, nil
	}

	// Helper to query actual database table foreign keys from SQLite with strict error checking
	getActualForeignKeys := func(tableName string) ([]map[string]interface{}, error) {
		script := fmt.Sprintf(`'use strict';
const { Sequelize } = require('sequelize');
const sequelize = new Sequelize({ dialect: 'sqlite', storage: %q, logging: false });

async function inspect() {
  const [results] = await sequelize.query("PRAGMA foreign_key_list(" + %q + ");");
  const list = Array.isArray(results) ? results : (results ? [results] : []);
  console.log(JSON.stringify(list));
  await sequelize.close();
}
inspect().catch(err => { console.error(err); process.exit(1); });
`, dbFile, tableName)

		cmd := exec.Command("node", "-e", script)
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("foreign key inspection failed: %s", string(out))
		}
		var fks []map[string]interface{}
		if err := json.Unmarshal(out, &fks); err != nil {
			return nil, fmt.Errorf("parse error: %w (out: %s)", err, string(out))
		}
		return fks, nil
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
	// STAGE 4: Real Database Index and Foreign Key Migration & Rollback
	// ==========================================
	t.Run("Stage 4: Real database index and foreign key creation and rollback", func(t *testing.T) {
		// Update models with an index on Users and a Foreign Key on Products
		writeModelsWithOptions(
			"  email: { type: DataTypes.STRING, allowNull: false, unique: true },\n  bio: { type: DataTypes.STRING(500), allowNull: true }",
			"  price: { type: DataTypes.DECIMAL(10, 2), allowNull: false },\n  sku: { type: DataTypes.STRING(50), allowNull: true },\n  ownerId: { type: DataTypes.INTEGER, references: { model: 'Users', key: 'id' } }",
			", indexes: [{ name: 'idx_users_email_custom', fields: ['email'] }]",
			"",
		)

		// 1. Generate migration with --verify
		makeCmd := exec.Command(binPath, "make", "--name=add-index-and-fk", "--verify")
		makeCmd.Dir = workDir
		makeCmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		makeOut, makeErr := makeCmd.CombinedOutput()
		if makeErr != nil {
			t.Fatalf("make with index and fk failed: %s, err: %v", string(makeOut), makeErr)
		}

		// Find the third migration file
		migrationsDir := filepath.Join(workDir, "migrations")
		entries, _ := os.ReadDir(migrationsDir)
		var migrations []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".js") {
				migrations = append(migrations, filepath.Join(migrationsDir, e.Name()))
			}
		}
		if len(migrations) < 3 {
			t.Fatalf("expected at least 3 migration files, found %d", len(migrations))
		}
		thirdMigration := migrations[len(migrations)-1]

		// 2. Check generated code contains addIndex and references
		content, _ := os.ReadFile(thirdMigration)
		migStr := string(content)
		if !strings.Contains(migStr, "addIndex('Users'") || !strings.Contains(migStr, "idx_users_email_custom") {
			t.Errorf("expected addIndex for idx_users_email_custom in generated migration, got: %s", migStr)
		}
		if !strings.Contains(migStr, "model: 'Users'") || !strings.Contains(migStr, "key: 'id'") {
			t.Errorf("expected foreign key references block in generated migration, got: %s", migStr)
		}

		// 3. Apply UP to real SQLite DB
		if err := applyMigration(thirdMigration, "up"); err != nil {
			t.Fatalf("failed to apply index and fk migration: %v", err)
		}

		// 4. Verify index physically exists in SQLite!
		indexes, err := getActualIndexes("Users")
		if err != nil {
			t.Fatalf("error checking Users indexes: %v", err)
		}
		hasIndex := false
		for _, idx := range indexes {
			if idx == "idx_users_email_custom" {
				hasIndex = true
				break
			}
		}
		if !hasIndex {
			t.Errorf("expected idx_users_email_custom physically in SQLite indexes, got: %v", indexes)
		}

		// Verify ownerId column physically exists in SQLite Products table
		prodCols, err := getActualColumns("Products")
		if err != nil {
			t.Fatalf("error checking Products columns: %v", err)
		}
		hasOwnerId := false
		for _, col := range prodCols {
			if col == "ownerId" {
				hasOwnerId = true
				break
			}
		}
		if !hasOwnerId {
			t.Errorf("expected ownerId physically in SQLite Products table, got: %v", prodCols)
		}

		// Verify foreign key relationship is physically recorded in SQLite PRAGMA foreign_key_list
		fks, err := getActualForeignKeys("Products")
		if err != nil {
			t.Fatalf("error checking Products foreign keys: %v", err)
		}
		hasFK := false
		for _, fk := range fks {
			if fk["table"] == "Users" && fk["from"] == "ownerId" && fk["to"] == "id" {
				hasFK = true
				break
			}
		}
		if !hasFK {
			t.Errorf("expected physical foreign key pointing to Users(id) in PRAGMA foreign_key_list, got: %v", fks)
		}

		// 5. Apply DOWN (Rollback) on real SQLite DB
		if err := applyMigration(thirdMigration, "down"); err != nil {
			t.Fatalf("failed to rollback index and fk migration: %v", err)
		}

		// 6. Verify index physically removed from SQLite!
		revertedIndexes, err := getActualIndexes("Users")
		if err != nil {
			t.Fatalf("error checking reverted Users indexes: %v", err)
		}
		for _, idx := range revertedIndexes {
			if idx == "idx_users_email_custom" {
				t.Errorf("idx_users_email_custom was not removed on rollback! Found: %v", revertedIndexes)
			}
		}

		// Verify ownerId physically removed from SQLite!
		revertedProdCols, err := getActualColumns("Products")
		if err != nil {
			t.Fatalf("error checking reverted Products columns: %v", err)
		}
		for _, col := range revertedProdCols {
			if col == "ownerId" {
				t.Errorf("ownerId was not removed on rollback! Found: %v", revertedProdCols)
			}
		}

		// Verify foreign key relationship physically removed from SQLite on rollback!
		revertedFks, err := getActualForeignKeys("Products")
		if err != nil {
			t.Fatalf("error checking reverted foreign keys: %v", err)
		}
		for _, fk := range revertedFks {
			if fk["from"] == "ownerId" {
				t.Errorf("foreign key for ownerId was not removed on rollback! Found: %v", revertedFks)
			}
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

		// 3. Test init command generates schema snapshot
		initSchemaPath := filepath.Join(workDir, "init-test-schema.json")
		initCmd := exec.Command(binPath, "init", "--schema-file", initSchemaPath)
		initCmd.Dir = workDir
		initCmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		initOut, initErr := initCmd.CombinedOutput()
		if initErr != nil {
			t.Fatalf("init command failed: %v, out: %s", initErr, string(initOut))
		}
		if _, err := os.Stat(initSchemaPath); err != nil {
			t.Fatalf("expected init command to create schema file %s", initSchemaPath)
		}

		// 4. Test invalid command -> Exits with exact code 1
		invalidCmd := exec.Command(binPath, "nonexistent-cmd")
		invalidCmd.Dir = workDir
		invalidErr := invalidCmd.Run()
		if exitError, ok := invalidErr.(*exec.ExitError); !ok || exitError.ExitCode() != 1 {
			t.Errorf("expected exact exit code 1 for nonexistent command, got: %v", invalidErr)
		}

		// 5. Test repair command
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
	// STAGE 7: Sequelize v7 Runtime Execution & Migration Test
	// ==========================================
	t.Run("Stage 7: Generates and executes v7 migration at runtime with UP and DOWN", func(t *testing.T) {
		// 1. Generate v7 migration
		v7MigDir := filepath.Join(workDir, "v7_migrations")
		_ = os.MkdirAll(v7MigDir, 0755)
		v7Cmd := exec.Command(binPath, "make", "--name=v7-runtime-test", "--target=v7", "--migrations-dir", v7MigDir, "--schema-file", filepath.Join(workDir, "v7-schema.json"))
		v7Cmd.Dir = workDir
		v7Cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		v7Out, v7Err := v7Cmd.CombinedOutput()
		if v7Err != nil {
			t.Fatalf("v7 migration generation failed: %v, out: %s", v7Err, string(v7Out))
		}

		// Locate generated v7 migration file
		entries, _ := os.ReadDir(v7MigDir)
		if len(entries) == 0 {
			t.Fatalf("expected generated v7 migration file")
		}
		v7FilePath := filepath.Join(v7MigDir, entries[0].Name())

		// Verify syntax has @sequelize/core and DataTypes
		v7Content, _ := os.ReadFile(v7FilePath)
		v7Str := string(v7Content)
		if !strings.Contains(v7Str, "@sequelize/core") || !strings.Contains(v7Str, "DataTypes.") {
			t.Errorf("v7 migration missing @sequelize/core or DataTypes syntax: %s", v7Str)
		}

		// 2. Real Runtime Execution Test of v7 migration (UP and DOWN) in Node.js
		runnerScript := fmt.Sprintf(`'use strict';
const Module = require('module');
const { Sequelize, DataTypes } = require('sequelize');

// Alias @sequelize/core to local DataTypes for test runtime execution
const origRequire = Module.prototype.require;
Module.prototype.require = function(id) {
  if (id === '@sequelize/core') {
    return { DataTypes };
  }
  return origRequire.apply(this, arguments);
};

const sequelize = new Sequelize({ dialect: 'sqlite', storage: ':memory:', logging: false });
const queryInterface = sequelize.getQueryInterface();
const migration = require(%q);

async function testRuntime() {
  // Test UP
  await migration.up(queryInterface);
  const [tablesUp] = await sequelize.query("SELECT name FROM sqlite_master WHERE type='table' AND name='Users';");
  if (!tablesUp.length) {
    throw new Error("Users table was not created by v7 migration.up()");
  }

  // Test DOWN
  await migration.down(queryInterface);
  const [tablesDown] = await sequelize.query("SELECT name FROM sqlite_master WHERE type='table' AND name='Users';");
  if (tablesDown.length) {
    throw new Error("Users table was not dropped by v7 migration.down()");
  }

  console.log("V7_RUNTIME_SUCCESS");
  await sequelize.close();
}

testRuntime().catch(err => {
  console.error("V7_RUNTIME_ERROR:", err.message);
  process.exit(1);
});
`, v7FilePath)

		nodeCmd := exec.Command("node", "-e", runnerScript)
		nodeCmd.Dir = workDir
		nodeCmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", targetNodeModules))
		nodeOut, nodeErr := nodeCmd.CombinedOutput()
		if nodeErr != nil {
			t.Fatalf("v7 runtime execution failed: %v, out: %s", nodeErr, string(nodeOut))
		}
		if !strings.Contains(string(nodeOut), "V7_RUNTIME_SUCCESS") {
			t.Errorf("expected V7_RUNTIME_SUCCESS, got: %s", string(nodeOut))
		}
	})
}
