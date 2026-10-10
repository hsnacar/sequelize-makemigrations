# Sequelize v6 QueryInterface & Migration Specification Reference

> Source: Official Sequelize v6 Documentation (https://sequelize.org/v6/)
> Scope: QueryInterface, DataTypes, Migrations, Constraints, and Indexes.

---

## 1. Migration File Structure (v6)

A Sequelize v6 migration file exports an object with `up` and `down` async functions:

```javascript
'use strict';

module.exports = {
  async up(queryInterface, Sequelize) {
    // Migration operations
  },

  async down(queryInterface, Sequelize) {
    // Rollback operations in reverse order
  }
};
```

---

## 2. Core QueryInterface Operations

### A. Table Management
```javascript
// Create Table
await queryInterface.createTable('Users', {
  id: {
    type: Sequelize.INTEGER,
    primaryKey: true,
    autoIncrement: true,
    allowNull: false
  },
  name: {
    type: Sequelize.STRING(255),
    allowNull: false
  },
  email: {
    type: Sequelize.STRING,
    allowNull: false,
    unique: true
  },
  createdAt: {
    type: Sequelize.DATE,
    allowNull: false
  },
  updatedAt: {
    type: Sequelize.DATE,
    allowNull: false
  }
}, {
  // Table options (charset, schema, engine)
});

// Drop Table
await queryInterface.dropTable('Users');

// Rename Table
await queryInterface.renameTable('OldName', 'NewName');
```

### B. Column Management
```javascript
// Add Column
await queryInterface.addColumn('Users', 'age', {
  type: Sequelize.INTEGER,
  allowNull: true,
  defaultValue: 18
});

// Remove Column
await queryInterface.removeColumn('Users', 'age');

// Change / Alter Column
await queryInterface.changeColumn('Users', 'age', {
  type: Sequelize.SMALLINT,
  allowNull: false,
  defaultValue: 21
});

// Rename Column
await queryInterface.renameColumn('Users', 'age', 'user_age');
```

### C. Indexes & Constraints
```javascript
// Add Index
await queryInterface.addIndex('Users', ['email'], {
  name: 'users_email_idx',
  unique: true
});

// Remove Index
await queryInterface.removeIndex('Users', 'users_email_idx');

// Add Constraint (Foreign Key, Unique, Check)
await queryInterface.addConstraint('Posts', {
  fields: ['userId'],
  type: 'foreign key',
  name: 'posts_user_id_fk',
  references: {
    table: 'Users',
    field: 'id'
  },
  onDelete: 'CASCADE',
  onUpdate: 'CASCADE'
});

// Remove Constraint
await queryInterface.removeConstraint('Posts', 'posts_user_id_fk');
```

---

## 3. Supported DataTypes in v6
- `Sequelize.STRING` / `Sequelize.STRING(length)`
- `Sequelize.TEXT` / `Sequelize.TEXT('tiny' | 'medium' | 'long')`
- `Sequelize.INTEGER`
- `Sequelize.BIGINT`
- `Sequelize.FLOAT` / `Sequelize.DOUBLE`
- `Sequelize.DECIMAL` / `Sequelize.DECIMAL(precision, scale)`
- `Sequelize.BOOLEAN`
- `Sequelize.DATE` / `Sequelize.DATEONLY`
- `Sequelize.JSON` / `Sequelize.JSONB` (PostgreSQL)
- `Sequelize.UUID` / `Sequelize.UUIDV4`
- `Sequelize.ENUM('val1', 'val2')`
- `Sequelize.ARRAY(Sequelize.STRING)` (PostgreSQL)
