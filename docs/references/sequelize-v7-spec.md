# Sequelize v7 (@sequelize/core) QueryInterface & Migration Specification Reference

> Source: Official Sequelize v7 Documentation (https://sequelize.org/docs/v7/)
> Scope: Architectural changes, DataTypes renaming, Dialect separation, TypeScript native support.

---

## 1. Architectural Changes in v7

In Sequelize v7:
- Core package is renamed to `@sequelize/core`.
- Dialects are isolated packages: `@sequelize/postgres`, `@sequelize/sqlite3`, `@sequelize/mysql`, etc.
- Native TypeScript is first-class citizen.
- Model definitions use new decorators or `define` with strictly typed `DataTypes`.

---

## 2. Migration File Structure (v7)

### JavaScript / ESM or CommonJS:
```javascript
import { DataTypes } from '@sequelize/core';

export async function up(queryInterface) {
  await queryInterface.createTable('Users', {
    id: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      autoIncrement: true,
      allowNull: false
    },
    name: {
      type: DataTypes.STRING,
      allowNull: false
    }
  });
}

export async function down(queryInterface) {
  await queryInterface.dropTable('Users');
}
```

---

## 3. Renamed and Deprecated DataTypes in v7
- `Sequelize.NUMERIC` -> Deprecated, use `DataTypes.DECIMAL`.
- `Sequelize.DOUBLE PRECISION` -> Deprecated, use `DataTypes.DOUBLE`.
- `Sequelize.JSONTYPE` -> Deprecated, use `DataTypes.JSON`.
- Types are directly accessed from `DataTypes.*` instead of root `Sequelize.*`.

---

## 4. QueryInterface Changes in v7
- `queryInterface.createTable` accepts typed attribute objects.
- `queryInterface.changeColumn` requires full column definition options.
- Constraints (`foreign key`, `unique`, `check`) continue to use `addConstraint` / `removeConstraint`.
