package v7

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hsnacar/sequelize-makemigrations/core/domain"
	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

type Generator struct{}

func NewGenerator() ports.Generator {
	return &Generator{}
}

func (g *Generator) Generate(diff *domain.Diff, opts ports.GeneratorOptions) (*ports.GeneratedMigration, error) {
	timestamp := time.Now().UTC().Format("20060102150405")
	slug := sanitizeSlug(opts.MigrationName)
	if slug == "" {
		slug = "migration"
	}
	filename := fmt.Sprintf("%s-%s.js", timestamp, slug)

	var upBuf bytes.Buffer
	var downBuf bytes.Buffer

	// --- UP OPERATIONS ---

	// 1. Create Tables
	for _, table := range diff.TablesCreated {
		upBuf.WriteString(formatCreateTable(table))
		upBuf.WriteString("\n\n")
	}

	// 2. Add Columns
	for _, colDiff := range diff.ColumnsAdded {
		upBuf.WriteString(formatAddColumn(colDiff.TableName, colDiff.Column))
		upBuf.WriteString("\n\n")
	}

	// 3. Modify Columns
	for _, colDiff := range diff.ColumnsModified {
		upBuf.WriteString(formatChangeColumn(colDiff.TableName, colDiff.Column))
		upBuf.WriteString("\n\n")
	}

	// 4. Add Indexes
	for _, idxDiff := range diff.IndexesAdded {
		upBuf.WriteString(formatAddIndex(idxDiff.TableName, idxDiff.Index))
		upBuf.WriteString("\n\n")
	}

	// 5. Remove Dropped Indexes
	for _, idxDiff := range diff.IndexesDropped {
		upBuf.WriteString(formatRemoveIndex(idxDiff.TableName, idxDiff.Index.Name))
		upBuf.WriteString("\n\n")
	}

	// 6. Drop Columns
	for _, colDiff := range diff.ColumnsDropped {
		upBuf.WriteString(formatRemoveColumn(colDiff.TableName, colDiff.Column.Name))
		upBuf.WriteString("\n\n")
	}

	// 7. Drop Tables
	for _, table := range diff.TablesDropped {
		upBuf.WriteString(formatDropTable(table.Name))
		upBuf.WriteString("\n\n")
	}

	// --- DOWN OPERATIONS (Exact Reverse Order for Safe Rollback) ---

	// 1. Re-create Dropped Tables
	for i := len(diff.TablesDropped) - 1; i >= 0; i-- {
		downBuf.WriteString(formatCreateTable(diff.TablesDropped[i]))
		downBuf.WriteString("\n\n")
	}

	// 2. Re-add Dropped Columns (restoring OldColumn)
	for i := len(diff.ColumnsDropped) - 1; i >= 0; i-- {
		col := diff.ColumnsDropped[i]
		if col.OldColumn != nil {
			downBuf.WriteString(formatAddColumn(col.TableName, *col.OldColumn))
		} else {
			downBuf.WriteString(formatAddColumn(col.TableName, col.Column))
		}
		downBuf.WriteString("\n\n")
	}

	// 3. Re-add Dropped Indexes
	for i := len(diff.IndexesDropped) - 1; i >= 0; i-- {
		idxDiff := diff.IndexesDropped[i]
		downBuf.WriteString(formatAddIndex(idxDiff.TableName, idxDiff.Index))
		downBuf.WriteString("\n\n")
	}

	// 4. Remove Added Indexes
	for i := len(diff.IndexesAdded) - 1; i >= 0; i-- {
		idxDiff := diff.IndexesAdded[i]
		downBuf.WriteString(formatRemoveIndex(idxDiff.TableName, idxDiff.Index.Name))
		downBuf.WriteString("\n\n")
	}

	// 5. Revert Modified Columns (reverting to OldColumn)
	for i := len(diff.ColumnsModified) - 1; i >= 0; i-- {
		col := diff.ColumnsModified[i]
		if col.OldColumn != nil {
			downBuf.WriteString(formatChangeColumn(col.TableName, *col.OldColumn))
			downBuf.WriteString("\n\n")
		}
	}

	// 6. Remove Added Columns
	for i := len(diff.ColumnsAdded) - 1; i >= 0; i-- {
		col := diff.ColumnsAdded[i]
		downBuf.WriteString(formatRemoveColumn(col.TableName, col.Column.Name))
		downBuf.WriteString("\n\n")
	}

	// 7. Drop Created Tables
	for i := len(diff.TablesCreated) - 1; i >= 0; i-- {
		downBuf.WriteString(formatDropTable(diff.TablesCreated[i].Name))
		downBuf.WriteString("\n\n")
	}

	content := fmt.Sprintf(`'use strict';
const { DataTypes } = require('@sequelize/core');

/** @type {import('@sequelize/core').Migration} */
module.exports = {
  async up(queryInterface) {
%s  },

  async down(queryInterface) {
%s  }
};
`, indent(upBuf.String(), "    "), indent(downBuf.String(), "    "))

	return &ports.GeneratedMigration{
		Filename: filename,
		Content:  content,
	}, nil
}

func sanitizeSlug(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

func formatDataType(dt domain.DataType) string {
	name := strings.ToUpper(dt.Name)
	switch name {
	case "NUMERIC":
		name = "DECIMAL"
	case "JSONTYPE":
		name = "JSON"
	case "DOUBLE PRECISION":
		name = "DOUBLE"
	}

	switch name {
	case "STRING":
		if dt.Length > 0 {
			return fmt.Sprintf("DataTypes.STRING(%d)", dt.Length)
		}
		return "DataTypes.STRING"
	case "INTEGER":
		return "DataTypes.INTEGER"
	case "BIGINT":
		return "DataTypes.BIGINT"
	case "FLOAT":
		return "DataTypes.FLOAT"
	case "DOUBLE":
		return "DataTypes.DOUBLE"
	case "DECIMAL":
		if dt.Precision > 0 && dt.Scale > 0 {
			return fmt.Sprintf("DataTypes.DECIMAL(%d, %d)", dt.Precision, dt.Scale)
		}
		return "DataTypes.DECIMAL"
	case "BOOLEAN":
		return "DataTypes.BOOLEAN"
	case "DATE":
		return "DataTypes.DATE"
	case "DATEONLY":
		return "DataTypes.DATEONLY"
	case "TEXT":
		return "DataTypes.TEXT"
	case "JSON":
		return "DataTypes.JSON"
	case "UUID":
		return "DataTypes.UUID"
	case "ENUM":
		if len(dt.Values) > 0 {
			quoted := make([]string, len(dt.Values))
			for i, v := range dt.Values {
				quoted[i] = fmt.Sprintf("'%s'", v)
			}
			return fmt.Sprintf("DataTypes.ENUM(%s)", strings.Join(quoted, ", "))
		}
		return "DataTypes.ENUM"
	default:
		return fmt.Sprintf("DataTypes.%s", name)
	}
}

func formatColumnAttributes(col domain.Column) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString(fmt.Sprintf("      type: %s,\n", formatDataType(col.Type)))
	sb.WriteString(fmt.Sprintf("      allowNull: %t,\n", col.AllowNull))
	if col.PrimaryKey {
		sb.WriteString("      primaryKey: true,\n")
	}
	if col.AutoIncrement {
		sb.WriteString("      autoIncrement: true,\n")
	}
	if col.Unique {
		sb.WriteString("      unique: true,\n")
	}
	if col.References != nil && col.References.Model != "" {
		sb.WriteString("      references: {\n")
		sb.WriteString(fmt.Sprintf("        model: '%s',\n", col.References.Model))
		sb.WriteString(fmt.Sprintf("        key: '%s',\n", col.References.Key))
		sb.WriteString("      },\n")
	}
	if col.DefaultValue != nil {
		switch v := col.DefaultValue.(type) {
		case string:
			sb.WriteString(fmt.Sprintf("      defaultValue: '%s',\n", v))
		case bool:
			sb.WriteString(fmt.Sprintf("      defaultValue: %t,\n", v))
		default:
			sb.WriteString(fmt.Sprintf("      defaultValue: %v,\n", v))
		}
	}
	sb.WriteString("    }")
	return sb.String()
}

func formatCreateTable(table domain.Table) string {
	var primaryKeys []string
	var otherKeys []string

	for colName, col := range table.Columns {
		if col.PrimaryKey {
			primaryKeys = append(primaryKeys, colName)
		} else {
			otherKeys = append(otherKeys, colName)
		}
	}
	sort.Strings(primaryKeys)
	sort.Strings(otherKeys)
	orderedKeys := append(primaryKeys, otherKeys...)

	var cols []string
	for _, colName := range orderedKeys {
		col := table.Columns[colName]
		cols = append(cols, fmt.Sprintf("    %s: %s", colName, formatColumnAttributes(col)))
	}
	return fmt.Sprintf("await queryInterface.createTable('%s', {\n%s\n});", table.Name, strings.Join(cols, ",\n"))
}

func formatAddColumn(tableName string, col domain.Column) string {
	return fmt.Sprintf("await queryInterface.addColumn('%s', '%s', %s);",
		tableName, col.Name, formatColumnAttributes(col))
}

func formatChangeColumn(tableName string, col domain.Column) string {
	return fmt.Sprintf("await queryInterface.changeColumn('%s', '%s', %s);",
		tableName, col.Name, formatColumnAttributes(col))
}

func formatRemoveColumn(tableName, colName string) string {
	return fmt.Sprintf("await queryInterface.removeColumn('%s', '%s');", tableName, colName)
}

func formatDropTable(tableName string) string {
	return fmt.Sprintf("await queryInterface.dropTable('%s');", tableName)
}

func formatAddIndex(tableName string, idx domain.Index) string {
	quotedFields := make([]string, len(idx.Fields))
	for i, f := range idx.Fields {
		quotedFields[i] = fmt.Sprintf("'%s'", f)
	}
	var opts []string
	if idx.Name != "" {
		opts = append(opts, fmt.Sprintf("name: '%s'", idx.Name))
	}
	if idx.Unique {
		opts = append(opts, "unique: true")
	}
	optStr := ""
	if len(opts) > 0 {
		optStr = fmt.Sprintf(", {\n      %s\n    }", strings.Join(opts, ",\n      "))
	}
	return fmt.Sprintf("await queryInterface.addIndex('%s', [%s]%s);", tableName, strings.Join(quotedFields, ", "), optStr)
}

func formatRemoveIndex(tableName, indexName string) string {
	return fmt.Sprintf("await queryInterface.removeIndex('%s', '%s');", tableName, indexName)
}
