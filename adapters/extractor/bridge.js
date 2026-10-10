#!/usr/bin/env node
'use strict';

/**
 * Sequelize Model Extractor Bridge
 * Reads user's Sequelize models and outputs the standardized Schema IR (JSON) to stdout.
 */

const path = require('path');
const fs = require('fs');

async function extractSchema() {
  const projectRoot = process.env.SEQ_PROJECT_ROOT || process.cwd();
  const modelsPath = process.env.SEQ_MODELS_PATH || findModelsPath(projectRoot);

  if (!modelsPath || !fs.existsSync(modelsPath)) {
    console.error(JSON.stringify({
      error: `Could not locate Sequelize models. Looked in ${modelsPath || 'default locations'}`
    }));
    process.exit(1);
  }

  let dbOrSequelize;
  try {
    // Support TypeScript projects via ts-node if available
    if (modelsPath.endsWith('.ts')) {
      try { require('ts-node/register'); } catch (_) {}
    }
    dbOrSequelize = require(path.resolve(modelsPath));
  } catch (err) {
    console.error(JSON.stringify({
      error: `Failed to import Sequelize models from ${modelsPath}: ${err.message}`
    }));
    process.exit(1);
  }

  // Handle various export patterns: module.exports = db (where db.sequelize or db.models), or direct sequelize instance
  const sequelize = dbOrSequelize.sequelize || (dbOrSequelize.models ? dbOrSequelize : null);

  if (!sequelize || !sequelize.models) {
    console.error(JSON.stringify({
      error: 'Could not find a valid Sequelize instance with models.'
    }));
    process.exit(1);
  }

  const schema = {
    version: 1,
    tables: {}
  };

  const models = sequelize.models;
  for (const modelName of Object.keys(models)) {
    const model = models[modelName];
    const tableName = typeof model.getTableName === 'function' ? model.getTableName() : model.tableName;
    const rawAttrs = model.rawAttributes || {};

    const columns = {};
    for (const attrName of Object.keys(rawAttrs)) {
      const attr = rawAttrs[attrName];
      const colName = attr.field || attrName;
      const typeObj = attr.type || {};
      
      const typeName = typeObj.key || (typeObj.constructor && typeObj.constructor.key) || 'STRING';
      const length = typeObj._length || (typeObj.options && typeObj.options.length) || 0;
      const precision = typeObj._precision || 0;
      const scale = typeObj._scale || 0;
      const values = typeObj.values || [];

      columns[colName] = {
        name: colName,
        type: {
          name: String(typeName).toUpperCase(),
          length: Number(length) || 0,
          precision: Number(precision) || 0,
          scale: Number(scale) || 0,
          values: Array.isArray(values) ? values : []
        },
        allowNull: attr.allowNull !== false,
        primaryKey: Boolean(attr.primaryKey),
        autoIncrement: Boolean(attr.autoIncrement),
        unique: Boolean(attr.unique),
        defaultValue: attr.defaultValue !== undefined ? attr.defaultValue : null,
        comment: attr.comment || ''
      };
    }

    const indexes = [];
    if (model.options && Array.isArray(model.options.indexes)) {
      for (const idx of model.options.indexes) {
        indexes.push({
          name: idx.name || `${tableName}_${idx.fields.join('_')}_idx`,
          fields: Array.isArray(idx.fields) ? idx.fields.map(f => typeof f === 'string' ? f : f.name || f.attribute) : [],
          unique: Boolean(idx.unique),
          method: idx.using || idx.type || ''
        });
      }
    }

    schema.tables[tableName] = {
      name: tableName,
      columns,
      indexes
    };
  }

  // Output Schema IR JSON directly to stdout
  process.stdout.write(JSON.stringify(schema, null, 2));
}

function findModelsPath(root) {
  const candidates = [
    path.join(root, 'models'),
    path.join(root, 'models', 'index.js'),
    path.join(root, 'models', 'index.ts'),
    path.join(root, 'src', 'models'),
    path.join(root, 'src', 'models', 'index.js'),
    path.join(root, 'src', 'models', 'index.ts')
  ];

  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }
  return null;
}

extractSchema();
