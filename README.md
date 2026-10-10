# sequelize-makemigrations (seq-migrate) 🚀

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg?style=flat-square&logo=go)](https://go.dev)
[![Sequelize](https://img.shields.io/badge/Sequelize-v6%20%7C%20v7-52B0E7.svg?style=flat-square&logo=sequelize)](https://sequelize.org)
[![Zero-Risk Sandbox](https://img.shields.io/badge/Zero--Risk-Shadow%20DB%20Verified-brightgreen.svg?style=flat-square)](https://github.com/hsnacar/sequelize-makemigrations)

**sequelize-makemigrations** (`seq-migrate`) brings the declarative, automated migration workflow of Django (`makemigrations`) and Prisma (`migrate dev`) to the **Sequelize (v6 & v7)** ecosystem.

Powered by a lightning-fast **Go** core with a zero-friction **npm** wrapper, it eliminates the tedious, error-prone manual creation of Sequelize migration files while guaranteeing **100% reversible rollbacks** and **Zero-Risk Shadow Database verification**.

---

## ⚡ Why sequelize-makemigrations?

- 🪄 **Declarative Workflow:** Update your Sequelize models, run `npx seq-migrate make`, and get clean, production-ready migration files automatically.
- 🛡️ **Zero-Risk Shadow DB Verification (`--verify`):** Tests the generated migration (`up` AND `down`) against an ephemeral in-memory sandbox database *before* touching your project.
- 🔄 **Guaranteed Reversible Rollbacks:** Automatically generates full `down` migrations with exact previous column types, constraints, and reverse operation ordering.
- ⚡ **Blazing Fast Go Core:** Analyzes schemas, resolves dependencies, and generates code in under 15ms.
- 🏗️ **Clean Architecture & Versioning:** Built on a decoupled Schema IR (Intermediate Representation) designed to seamlessly adapt from Sequelize v6 to v7 and future releases.
- 🔍 **Safe Dry-Run Mode (`--dry-run`):** Inspect diffs and code outputs without writing any files.

---

## 📦 Installation

Install as a development dependency in your Sequelize project:

```bash
npm install --save-dev sequelize-makemigrations
# or
yarn add -D sequelize-makemigrations
# or
pnpm add -D sequelize-makemigrations
```

You can run the CLI via `npx seq-migrate` or add it to your `package.json` scripts:

```json
{
  "scripts": {
    "makemigrations": "seq-migrate make"
  }
}
```

---

## 🚀 Usage

### 1. Generate Migrations (`make`)

Whenever you add, modify, or delete columns or tables in your Sequelize models:

```bash
npx seq-migrate make --name add-user-avatar
```

This will:
1. Inspect your current Sequelize models.
2. Calculate the difference against your last schema snapshot (`.seq-schema.json`).
3. Generate a versioned migration file: `migrations/YYYYMMDDHHMMSS-add-user-avatar.js`.
4. Update the schema snapshot baseline.

### 2. Zero-Risk Shadow DB Verification (`--verify`)

Run with `--verify` to test-drive the generated migration on an isolated sandbox database before saving:

```bash
npx seq-migrate make --name add-user-bio --verify
```

> **How Shadow DB works:** Spawns an in-memory SQLite sandbox, runs `migration.up()`, verifies schema integrity, and then executes `migration.down()`. If any SQL syntax or constraint error occurs, the process halts immediately without writing to your disk.

### 3. Safe Dry-Run (`--dry-run`)

Preview detected changes and generated migration code in your terminal without writing any files:

```bash
npx seq-migrate make --dry-run
```

### 4. Continuous Integration & Drift Detection (`check`)

Verify that all Sequelize models match the current migration schema state (returns exit code `1` if drift is detected — ideal for CI/CD pipelines):

```bash
npx seq-migrate check
```

### 5. Migration Health & Auto-Repair (`repair`)

Audit all existing migration files for missing `down` handlers or syntax irregularities, and automatically fix them:

```bash
# Audit migration files
npx seq-migrate repair

# Auto-generate and patch missing down() functions
npx seq-migrate repair --fix-down
```

### 6. Sequelize v7 Readiness Audit (`doctor`)

Diagnose legacy Sequelize patterns, deprecated DataTypes (`NUMERIC`, `JSONTYPE`, `DOUBLE PRECISION`, etc.), and incompatible migration constructs before upgrading to Sequelize v7:

```bash
npx seq-migrate doctor
```

### 7. Automated Migration Upgrade Codemod (`upgrade`)

Safely modernize existing Sequelize migrations to modern Sequelize v7 syntax (`@sequelize/core`, `DataTypes.*`), creating automated `.bak` backups before mutating files:

```bash
# Preview changes safely
npx seq-migrate upgrade --to=v7 --dry-run

# Apply codemod with automatic backup
npx seq-migrate upgrade --to=v7
```

### 8. Check Status (`status`)

Check pending model changes that have not yet been migrated:

```bash
npx seq-migrate status
```

---

## 🛠️ CLI Options & Commands

### Available Commands:
- `make` — Generate a new migration file from Sequelize model diffs.
- `check` — Assert model vs schema snapshot consistency for CI/CD.
- `doctor` — Audit migration files for Sequelize v7 breaking changes.
- `upgrade` — Automatically modernize legacy migrations to Sequelize v7.
- `repair` — Audit and auto-repair missing `down` rollback functions in migrations.
- `status` — View current schema diff against models.
- `init` — Initialize schema snapshot baseline from existing models.

### Flags:
| Option | Flag | Default | Description |
| :--- | :--- | :--- | :--- |
| `--name` | `-n` | `"migration"` | Name suffix for the generated migration file |
| `--models` | `-m` | `auto-detect` | Custom path to models index or Sequelize instance |
| `--migrations-dir` | `-d` | `"migrations"` | Target directory for generated migration files |
| `--schema-file` | `-s` | `".seq-schema.json"` | Path to the schema snapshot tracking file |
| `--target` | `-t` | `"v6"` | Target Sequelize version (`v6` or `v7`) |
| `--verify` | | `false` | Run migration UP & DOWN on an ephemeral sandbox DB |
| `--dry-run` | | `false` | Preview diff and migration code without saving |
| `--fix-down` | | `false` | Auto-generate missing down functions during repair |
| `--to` | | `"v7"` | Target version for migration upgrade codemod |

---

## 🤖 AI Agent & Contributor Friendly (`AGENTS.md`)

This project strictly adheres to **Clean Architecture**. For AI coding assistants or developers extending dialects and generators, see [AGENTS.md](file:///Users/hasanacar/Documents/Projects/open-source-projects/sequelize-makemigrations/AGENTS.md) and [docs/ARCHITECTURE.md](file:///Users/hasanacar/Documents/Projects/open-source-projects/sequelize-makemigrations/docs/ARCHITECTURE.md).

---

## 📄 License

MIT © [Hasan Acar](https://github.com/hsnacar)
