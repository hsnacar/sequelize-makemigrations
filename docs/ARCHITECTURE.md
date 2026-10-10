# System Architecture & Design Specification

`sequelize-makemigrations` (CLI: `seq-migrate`)

---

## 1. Architectural Philosophy: Clean Architecture

To ensure adaptability as Sequelize evolves from v6 to v7 and beyond, the system is strictly decoupled into 4 layers:

```
┌────────────────────────────────────────────────────────┐
│                   CLI / Presentation                   │
│                 cmd/seq-migrate/main.go                │
└──────────────────────────┬─────────────────────────────┘
                           │
┌──────────────────────────▼─────────────────────────────┐
│                 Core Domain & Engine                   │
│   - core/domain: Schema IR, Tables, Columns, Diff      │
│   - core/ports:  Contracts (Extractor, Generator, etc.)│
│   - core/diff:   Pure deterministic diff algorithm     │
└────────────┬─────────────────────────────┬─────────────┘
             │                             │
┌────────────▼──────────────┐ ┌────────────▼─────────────┐
│    Adapters: Extractor    │ │   Adapters: Generator    │
│  adapters/extractor       │ │  adapters/generator/v6   │
│  (Node.js Model Bridge)   │ │  adapters/generator/v7   │
└───────────────────────────┘ └──────────────────────────┘
             │                             │
┌────────────▼─────────────────────────────▼─────────────┐
│             Adapters: Verifier / Sandbox               │
│  adapters/verifier: Shadow DB Execution & Rollback     │
└────────────────────────────────────────────────────────┘
```

---

## 2. The Zero-Risk Strategy: Shadow DB & Dry Run

Database schema operations carry extreme risk in production environments. `sequelize-makemigrations` protects data through a 3-tier validation protocol:

### Tier 1: Schema IR Diff Verification
The diff engine is a pure, side-effect-free algorithm with 100% deterministic outputs. It preserves `OldColumn` definitions so every change or drop has an exact inverse operation.

### Tier 2: Reversible Rollback Contract
Every generated migration contains both `up` and `down` handlers. The `down` method is automatically generated in the exact reverse sequence of `up`.

### Tier 3: Shadow Sandbox Verification (`--verify`)
Before saving the migration to disk, the CLI invokes the `verifier` adapter:
1. Spawns an isolated in-memory SQLite sandbox database.
2. Applies the generated migration `up`.
3. Verifies table/column integrity.
4. Executes the migration `down` to test rollback.
5. If any SQL syntax error, constraint failure, or rollback issue occurs, the process halts with exit code 2 and aborts immediately. Real files and databases remain untouched.

---

## 3. Versioning & Extensibility (v6, v7, v8)

Because the core diff engine operates solely on the dialect-neutral **Schema IR**, adding support for a new Sequelize release or custom dialect requires only:
1. Implementing the `ports.Generator` interface in `adapters/generator/<version>/`.
2. Registering the new generator in `adapters/generator/factory.go`.
Zero changes to core domain or diffing logic.
