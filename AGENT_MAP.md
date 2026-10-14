# 🧭 Code Atlas: Architectural Map for AI Agents

> 📊 **Original Codebase:** ~35.183 tokens (30 files, 4.543 lines)
> ⚡ **Atlas Map:** ~1.785 tokens (**94.9% token reduction**)
> 🏷️ **Primary Stack:** Go
> 🕒 **Generated:** 2026-10-14 16:49:36

---

## 🗺️ Architectural Layer Map

### 🌐 API & Entrypoints (2 files)

- **`cmd/seq-migrate/main.go`** (535 lines)
  - 🧩 *Exports:* `func main() `, `func printUsage() `, `func runMake(args []string) `, `func runCheck(args []string) `, `func runRepair(args []string) ` *(+7 more)*
  - 🔗 *Uses:* `context, encoding/json, flag, fmt` (+12)
- **`examples/ecommerce-app/models/index.js`** (68 lines)
  - 🧩 *Exports:* `const sequelize`, `const User`, `const Product`
  - 🔗 *Uses:* `sequelize`

### ⚙️ Business Logic & Services (1 files)

- **`core/domain/schema.go`** (167 lines)
  - 🧩 *Exports:* `type DataType struct`, `type Reference struct`, `type Column struct`, `type Index struct`, `type Constraint struct` *(+9 more)*
  - 🔗 *Uses:* `fmt, strings`

### 🧪 Test Suites (12 files)

- **`adapters/extractor/extractor_test.go`** (75 lines)
  - 🧩 *Exports:* `func TestNodeExtractor(t *testing.T) `
  - 🔗 *Uses:* `context, os, path/filepath, testing`
- **`adapters/generator/factory_test.go`** (63 lines)
  - 🧩 *Exports:* `func TestGetGenerator(t *testing.T) `
  - 🔗 *Uses:* `strings, testing, github.com/hsnacar/sequelize-makemigrations/core/domain, github.com/hsnacar/sequelize-makemigrations/core/ports`
- **`adapters/generator/v6/generator_test.go`** (305 lines)
  - 🧩 *Exports:* `func TestGeneratorV6(t *testing.T) `
  - 🔗 *Uses:* `strings, testing, github.com/hsnacar/sequelize-makemigrations/core/domain, github.com/hsnacar/sequelize-makemigrations/core/ports`
- **`adapters/generator/v7/generator_test.go`** (224 lines)
  - 🧩 *Exports:* `func TestGeneratorV7(t *testing.T) `
  - 🔗 *Uses:* `strings, testing, github.com/hsnacar/sequelize-makemigrations/core/domain, github.com/hsnacar/sequelize-makemigrations/core/ports`
- **`adapters/verifier/verifier_test.go`** (87 lines)
  - 🧩 *Exports:* `func TestSandboxVerifier(t *testing.T) `
  - 🔗 *Uses:* `context, os, path/filepath, testing` (+1)
- **`core/diff/engine_test.go`** (184 lines)
  - 🧩 *Exports:* `func TestDiffEngine(t *testing.T) `
  - 🔗 *Uses:* `testing, github.com/hsnacar/sequelize-makemigrations/core/domain`
- **`core/doctor/audit_test.go`** (75 lines)
  - 🧩 *Exports:* `func TestAuditDirectory(t *testing.T) `
  - 🔗 *Uses:* `os, path/filepath, testing`
- **`core/domain/schema_test.go`** (41 lines)
  - 🧩 *Exports:* `func TestDomainSchema(t *testing.T) `
  - 🔗 *Uses:* `strings, testing`
- **`core/repair/repair_test.go`** (85 lines)
  - 🧩 *Exports:* `func TestRepair(t *testing.T) `
  - 🔗 *Uses:* `os, path/filepath, testing`
- **`core/upgrade/codemod_test.go`** (153 lines)
  - 🧩 *Exports:* `func TestUpgradeFile(t *testing.T) `
  - 🔗 *Uses:* `os, path/filepath, strings, testing`
- **`test/e2e/e2e_test.go`** (725 lines)
  - 🧩 *Exports:* `func TestEndToEndSequelizeWorkflow(t *testing.T) `
  - 🔗 *Uses:* `encoding/json, fmt, os, os/exec` (+3)
- **`test/wrapper.test.js`** (44 lines)
  - 🔗 *Uses:* `assert, child_process, path, fs` (+1)

### 📁 Miscellaneous Modules (15 files)

- **`adapters/extractor/bridge.js`** (134 lines)
  - 🧩 *Exports:* `extractSchema()`, `findModelsPath(root)`
  - 🔗 *Uses:* `path, fs, ts-node/register`
- **`adapters/extractor/extractor.go`** (70 lines)
  - 🧩 *Exports:* `type NodeExtractor struct`, `func NewNodeExtractor(modelsPath string) ports.Extractor`, `func ExtractSchema(ctx context.Context, projectPath string) (*domain.Schema, error)`
  - 🔗 *Uses:* `context, embed, encoding/json, fmt` (+5)
- **`adapters/generator/factory.go`** (21 lines)
  - 🧩 *Exports:* `func GetGenerator(version string) ports.Generator`
  - 🔗 *Uses:* `strings, github.com/hsnacar/sequelize-makemigrations/adapters/generator/v6, github.com/hsnacar/sequelize-makemigrations/adapters/generator/v7, github.com/hsnacar/sequelize-makemigrations/core/ports`
- **`adapters/generator/v6/generator.go`** (307 lines)
  - 🧩 *Exports:* `type Generator struct`, `func NewGenerator() ports.Generator`, `func Generate(diff *domain.Diff, opts ports.GeneratorOptions) (*ports.GeneratedMigration, error)`, `func sanitizeSlug(name string) string`, `func indent(text, prefix string) string` *(+9 more)*
  - 🔗 *Uses:* `bytes, fmt, sort, strings` (+3)
- **`adapters/generator/v7/generator.go`** (319 lines)
  - 🧩 *Exports:* `type Generator struct`, `func NewGenerator() ports.Generator`, `func Generate(diff *domain.Diff, opts ports.GeneratorOptions) (*ports.GeneratedMigration, error)`, `func sanitizeSlug(name string) string`, `func indent(text, prefix string) string` *(+9 more)*
  - 🔗 *Uses:* `bytes, fmt, sort, strings` (+3)
- **`adapters/verifier/verifier.go`** (130 lines)
  - 🧩 *Exports:* `type SandboxVerifier struct`, `func NewSandboxVerifier(projectPath string) ports.Verifier`, `func Verify(ctx context.Context, migration *ports.GeneratedMigration) error`
  - 🔗 *Uses:* `context, encoding/json, fmt, os` (+5)
- **`bin/run.js`** (48 lines)
  - 🧩 *Exports:* `getBinaryPath()`
  - 🔗 *Uses:* `child_process, path, os, fs`
- **`core/diff/engine.go`** (183 lines)
  - 🧩 *Exports:* `type Engine struct`, `func NewEngine() ports.DiffEngine`, `func Compare(oldSchema, newSchema *domain.Schema) *domain.Diff`, `func hasColumnChanged(oldCol, newCol domain.Column) bool`, `func diffIndexes(tableName string, oldIndexes, newIndexes []domain.Index, diff *domain.Diff) ` *(+1 more)*
  - 🔗 *Uses:* `reflect, github.com/hsnacar/sequelize-makemigrations/core/domain, github.com/hsnacar/sequelize-makemigrations/core/ports`
- **`core/doctor/audit.go`** (128 lines)
  - 🧩 *Exports:* `type DeprecatedRule struct`, `type DiagnosticFinding struct`, `type AuditReport struct`, `func AuditDirectory(dir string, targetVersion string) (*AuditReport, error)`
  - 🔗 *Uses:* `os, path/filepath, regexp, strings`
- **`core/ports/ports.go`** (40 lines)
  - 🧩 *Exports:* `type Extractor interface`, `type DiffEngine interface`, `type GeneratorOptions struct`, `type GeneratedMigration struct`, `type Generator interface` *(+1 more)*
  - 🔗 *Uses:* `context, github.com/hsnacar/sequelize-makemigrations/core/domain`
- **`core/repair/repair.go`** (98 lines)
  - 🧩 *Exports:* `type MigrationIssue struct`, `func AuditMigrations(migrationsDir string) ([]MigrationIssue, error)`, `func RepairMissingDown(filePath string) error`
  - 🔗 *Uses:* `fmt, os, path/filepath, regexp` (+1)
- **`core/upgrade/codemod.go`** (127 lines)
  - 🧩 *Exports:* `type TransformationResult struct`, `func UpgradeFile(filePath string, dryRun, makeBackup bool) (*TransformationResult, error)`, `func UpgradeDirectory(dir string, dryRun, makeBackup bool) ([]TransformationResult, error)`
  - 🔗 *Uses:* `fmt, os, path/filepath, regexp` (+1)
- **`examples/ecommerce-app/package.json`** (10 lines)
- **`package.json`** (43 lines)
- **`scripts/build.js`** (54 lines)
  - 🔗 *Uses:* `child_process, path, fs`
