# AGENTS.md — System Prompt & Guidance for AI Agents

Welcome to `sequelize-makemigrations` (CLI: `seq-migrate`).
This document provides full context for AI agents working on this codebase.

---

## 1. Project Mission
`sequelize-makemigrations` is an enterprise-grade, zero-risk, automated migration generation tool for Sequelize (v6 & v7).
It replaces abandoned packages like `sequelize-auto-migrations` by bringing Django/Prisma-like declarative workflow (`makemigrations`) to Sequelize:
1. Inspect current Sequelize models.
2. Diff against previous schema snapshot (`.seq-schema.json`).
3. Generate safe, reversible migration files (`up` and `down`).
4. Validate changes on a **Shadow Database (Ephemeral Sandbox)** with **Zero Risk** before writing or executing.

---

## 2. Architectural Blueprint (Clean Architecture)

```
┌────────────────────────────────────────────────────────┐
│                      CLI Layer                         │
│               cmd/seq-migrate/main.go                   │
│   (Commands: make, verify, dry-run, status, apply)     │
└──────────────────────────┬─────────────────────────────┘
                           │
┌──────────────────────────▼─────────────────────────────┐
│                 Application / Core Layer               │
│  - core/domain: Schema IR, Table, Column, Index, Diff  │
│  - core/ports:  Extractor, DiffEngine, Generator, etc. │
│  - core/diff:   Pure algorithmic diff calculation      │
└────────────┬─────────────────────────────┬─────────────┘
             │                             │
┌────────────▼──────────────┐ ┌────────────▼─────────────┐
│    Adapters: Extractor    │ │   Adapters: Generator    │
│ Node.js Model Bridge      │ │ - adapters/generator/v6  │
│ (Reads Sequelize models)  │ │ - adapters/generator/v7  │
└───────────────────────────┘ └──────────────────────────┘
             │                             │
┌────────────▼─────────────────────────────▼─────────────┐
│            Adapters: Verifier / Shadow DB              │
│ - Ephemeral SQLite memory or shadow connection         │
│ - Runs generated migration UP, then DOWN, checks errors│
└────────────────────────────────────────────────────────┘
```

---

## 3. Core Domain Entities (Schema IR)

The **Schema IR (Intermediate Representation)** is completely decoupled from any specific Sequelize version or SQL dialect:

- **`Schema`**: Map of `Table` entities.
- **`Table`**: `Name`, `Columns` map, `Indexes` list, `Constraints` list.
- **`Column`**:
  - `Name` (string)
  - `Type` (standardized IR DataType: `INTEGER`, `STRING`, `BOOLEAN`, `DATE`, `JSON`, `ENUM`, etc.)
  - `AllowNull` (bool)
  - `PrimaryKey` (bool)
  - `AutoIncrement` (bool)
  - `DefaultValue` (interface{})
  - `Unique` (bool or string)
  - `Comment` (string)
- **`Index`**: `Name`, `Fields` ([]string), `Unique` (bool)
- **`Constraint`**: `Name`, `Type` (`FOREIGN_KEY`, `UNIQUE`, `CHECK`), `ForeignTable`, `ForeignField`, `OnDelete`, `OnUpdate`.
- **`Diff`**:
  - `TablesCreated` ([]Table)
  - `TablesDropped` ([]string)
  - `ColumnsAdded` ([]ColumnDiff)
  - `ColumnsModified` ([]ColumnDiff)
  - `ColumnsDropped` ([]ColumnDiff)
  - `IndexesAdded` / `IndexesDropped`
  - `ConstraintsAdded` / `ConstraintsDropped`

---

## 4. Safety & Zero-Risk Rules (MANDATORY)

1. **Reversibility Guarantee (`down`):** Every operation in `up` MUST have a deterministic inverse in `down`.
   - `createTable` -> `dropTable`
   - `addColumn` -> `removeColumn`
   - `changeColumn` -> restored to previous column state (preserved in `.seq-schema.json`)
   - `addIndex` -> `removeIndex`
2. **Foreign Key Order:** When creating tables, referenced tables MUST be created first. When dropping tables, child tables MUST be dropped first.
3. **Shadow DB Verification (`--verify`):** Before completing `makemigrations`, the CLI can execute the generated script against an ephemeral in-memory database to confirm that SQLite/Postgres accepts the SQL without syntax or constraint errors.
4. **Dry-Run Mode (`--dry-run`):** Outputs the diff and generated migration without writing to disk or changing `.seq-schema.json`.

---

## 5. Adding New Sequelize Versions
To support a new Sequelize version (e.g., v8):
1. Create `adapters/generator/v8/generator.go` implementing `ports.Generator`.
2. Register the generator in the factory based on target version flag (`--target=v8` or auto-detected).
3. Do NOT modify `core/diff` or `core/domain`.
