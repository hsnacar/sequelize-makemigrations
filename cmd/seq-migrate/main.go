package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hsnacar/sequelize-makemigrations/adapters/extractor"
	"github.com/hsnacar/sequelize-makemigrations/adapters/generator"
	"github.com/hsnacar/sequelize-makemigrations/adapters/verifier"
	"github.com/hsnacar/sequelize-makemigrations/core/diff"
	"github.com/hsnacar/sequelize-makemigrations/core/domain"
	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "make", "makemigrations":
		runMake(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "init":
		runInit(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("seq-migrate v%s (sequelize-makemigrations)\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`seq-migrate (sequelize-makemigrations) 🚀
Automated, zero-risk migration generator for Sequelize.

Usage:
  seq-migrate <command> [options]

Commands:
  make, makemigrations   Generate a new migration from model changes
  status                 Show pending model differences vs current schema
  init                   Snapshot current models as initial schema baseline
  version                Print CLI version
  help                   Show this help message

Options for 'make':
  -n, --name             Name of the migration (e.g. "add-user-avatar")
  -m, --models           Path to Sequelize models (default: auto-detected in models/)
  -d, --migrations-dir   Directory to save migration file (default: "migrations")
  -s, --schema-file      Path to schema snapshot (default: ".seq-schema.json")
  -t, --target           Target Sequelize version: v6 or v7 (default: "v6")
  --dry-run              Preview diff and migration code without saving files
  --verify               Verify migration on an ephemeral shadow DB sandbox before writing

Examples:
  npx seq-migrate make -n add-user-bio --verify
  npx seq-migrate make --dry-run
  npx seq-migrate status`)
}

func runMake(args []string) {
	fs := flag.NewFlagSet("make", flag.ExitOnError)
	var (
		name          string
		modelsPath    string
		migrationsDir string
		schemaFile    string
		targetVersion string
		dryRun        bool
		verify        bool
	)

	fs.StringVar(&name, "n", "migration", "Migration name")
	fs.StringVar(&name, "name", "migration", "Migration name")
	fs.StringVar(&modelsPath, "m", "", "Path to models")
	fs.StringVar(&modelsPath, "models", "", "Path to models")
	fs.StringVar(&migrationsDir, "d", "migrations", "Migrations directory")
	fs.StringVar(&migrationsDir, "migrations-dir", "migrations", "Migrations directory")
	fs.StringVar(&schemaFile, "s", ".seq-schema.json", "Schema snapshot file")
	fs.StringVar(&schemaFile, "schema-file", ".seq-schema.json", "Schema snapshot file")
	fs.StringVar(&targetVersion, "t", "v6", "Target Sequelize version")
	fs.StringVar(&targetVersion, "target", "v6", "Target Sequelize version")
	fs.BoolVar(&dryRun, "dry-run", false, "Preview without saving")
	fs.BoolVar(&verify, "verify", false, "Verify on shadow database sandbox")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cwd, _ := os.Getwd()

	// 1. Read existing schema snapshot
	oldSchema := loadSchemaSnapshot(schemaFile)

	// 2. Extract current schema from Sequelize models
	fmt.Println("🔍 Inspecting Sequelize models...")
	ext := extractor.NewNodeExtractor(modelsPath)
	newSchema, err := ext.ExtractSchema(ctx, cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error extracting models: %v\n", err)
		os.Exit(1)
	}

	// 3. Compute Diff
	engine := diff.NewEngine()
	delta := engine.Compare(oldSchema, newSchema)

	if delta.IsEmpty() {
		fmt.Println("✨ No changes detected in Sequelize models. Database schema is up to date!")
		return
	}

	// 4. Print Summary of Changes
	printDiffSummary(delta)

	// 5. Generate Migration Code
	gen := generator.GetGenerator(targetVersion)
	migration, err := gen.Generate(delta, ports.GeneratorOptions{
		SequelizeVersion: targetVersion,
		Language:         "javascript",
		MigrationName:    name,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error generating migration: %v\n", err)
		os.Exit(1)
	}

	// 6. Dry-run mode
	if dryRun {
		fmt.Printf("\n--- DRY RUN: Generated %s ---\n", migration.Filename)
		fmt.Println(migration.Content)
		fmt.Println("--------------------------------------------")
		fmt.Println("⚠️ Dry run completed. No files were written.")
		return
	}

	// 7. Sandbox Verification (Shadow DB)
	if verify {
		fmt.Println("🛡️ Running shadow database sandbox verification (UP and DOWN)...")
		v := verifier.NewSandboxVerifier(cwd)
		if err := v.Verify(ctx, migration); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Verification failed on sandbox database:\n%v\n", err)
			fmt.Fprintf(os.Stderr, "Aborting migration generation to protect database integrity.\n")
			os.Exit(2)
		}
		fmt.Println("✅ Shadow sandbox verified! Zero SQL errors, clean rollback.")
	}

	// 8. Save migration file
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create migrations directory: %v\n", err)
		os.Exit(1)
	}

	targetFilePath := filepath.Join(migrationsDir, migration.Filename)
	if err := os.WriteFile(targetFilePath, []byte(migration.Content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to write migration file: %v\n", err)
		os.Exit(1)
	}

	// 9. Update schema snapshot
	saveSchemaSnapshot(schemaFile, newSchema)

	fmt.Printf("\n🎉 Migration created: %s\n", targetFilePath)
	fmt.Printf("💾 Updated schema snapshot: %s\n", schemaFile)
}

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	var (
		modelsPath string
		schemaFile string
	)
	fs.StringVar(&modelsPath, "models", "", "Path to models")
	fs.StringVar(&schemaFile, "schema-file", ".seq-schema.json", "Schema snapshot file")
	fs.Parse(args)

	cwd, _ := os.Getwd()
	oldSchema := loadSchemaSnapshot(schemaFile)

	ext := extractor.NewNodeExtractor(modelsPath)
	newSchema, err := ext.ExtractSchema(context.Background(), cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error extracting models: %v\n", err)
		os.Exit(1)
	}

	delta := diff.NewEngine().Compare(oldSchema, newSchema)
	if delta.IsEmpty() {
		fmt.Println("✨ Schema is identical. No pending migrations.")
	} else {
		printDiffSummary(delta)
	}
}

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	var (
		modelsPath string
		schemaFile string
	)
	fs.StringVar(&modelsPath, "models", "", "Path to models")
	fs.StringVar(&schemaFile, "schema-file", ".seq-schema.json", "Schema snapshot file")
	fs.Parse(args)

	cwd, _ := os.Getwd()
	ext := extractor.NewNodeExtractor(modelsPath)
	schema, err := ext.ExtractSchema(context.Background(), cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error extracting models: %v\n", err)
		os.Exit(1)
	}

	saveSchemaSnapshot(schemaFile, schema)
	fmt.Printf("✅ Initialized schema baseline with %d tables in %s\n", len(schema.Tables), schemaFile)
}

func loadSchemaSnapshot(path string) *domain.Schema {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.NewSchema()
	}
	var schema domain.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return domain.NewSchema()
	}
	return &schema
}

func saveSchemaSnapshot(path string, schema *domain.Schema) {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to serialize schema: %v\n", err)
		return
	}
	_ = os.WriteFile(path, data, 0644)
}

func printDiffSummary(d *domain.Diff) {
	fmt.Println("\n📋 Detected Schema Changes:")
	for _, t := range d.TablesCreated {
		fmt.Printf("  [+] Table created: %s (%d columns)\n", t.Name, len(t.Columns))
	}
	for _, t := range d.TablesDropped {
		fmt.Printf("  [-] Table dropped: %s\n", t.Name)
	}
	for _, c := range d.ColumnsAdded {
		fmt.Printf("  [+] Column added: %s.%s (%s)\n", c.TableName, c.Column.Name, c.Column.Type.Name)
	}
	for _, c := range d.ColumnsModified {
		fmt.Printf("  [*] Column modified: %s.%s (%s)\n", c.TableName, c.Column.Name, c.Column.Type.Name)
	}
	for _, c := range d.ColumnsDropped {
		fmt.Printf("  [-] Column dropped: %s.%s\n", c.TableName, c.Column.Name)
	}
	for _, idx := range d.IndexesAdded {
		fmt.Printf("  [+] Index added: %s on %s\n", idx.Index.Name, idx.TableName)
	}
	for _, idx := range d.IndexesDropped {
		fmt.Printf("  [-] Index dropped: %s on %s\n", idx.Index.Name, idx.TableName)
	}
	fmt.Println()
}
