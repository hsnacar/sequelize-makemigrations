package v6

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
		upBuf.WriteString(formatCreateTable(table, "Sequelize"))
		upBuf.WriteString("\n\n")
	}

	// 2. Add Columns
	for _, colDiff := range diff.ColumnsAdded {
		upBuf.WriteString(formatAddColumn(colDiff.TableName, colDiff.Column, "Sequelize"))
		upBuf.WriteString("\n\n")
	}

	// 3. Modify Columns
	for _, colDiff := range diff.ColumnsModified {
		upBuf.WriteString(formatChangeColumn(colDiff.TableName, colDiff.Column, "Sequelize"))
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
		downBuf.WriteString(formatCreateTable(diff.TablesDropped[i], "Sequelize"))
		downBuf.WriteString("\n\n")
	}

	// 2. Re-add Dropped Columns (using preserved OldColumn)
	for i := len(diff.ColumnsDropped) - 1; i >= 0; i-- {
		col := diff.ColumnsDropped[i]
		if col.OldColumn != nil {
			downBuf.WriteString(formatAddColumn(col.TableName, *col.OldColumn, "Sequelize"))
		} else {
			downBuf.WriteString(formatAddColumn(col.TableName, col.Column, "Sequelize"))
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
			downBuf.WriteString(formatChangeColumn(col.TableName, *col.OldColumn, "Sequelize"))
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

/** @type {import('sequelize-cli').Migration} */
module.exports = {
  async up(queryInterface, Sequelize) {
%s  },

  async down(queryInterface, Sequelize) {
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

func formatDataType(dt domain.DataType, seqVar string) string {
	name := strings.ToUpper(dt.Name)
	switch name {
	case "STRING":
		if dt.Length > 0 {
			return fmt.Sprintf("%s.STRING(%d)", seqVar, dt.Length)
		}
		return fmt.Sprintf("%s.STRING", seqVar)
	case "INTEGER":
		return fmt.Sprintf("%s.INTEGER", seqVar)
	case "BIGINT":
		return fmt.Sprintf("%s.BIGINT", seqVar)
	case "FLOAT":
		return fmt.Sprintf("%s.FLOAT", seqVar)
	case "DOUBLE":
		return fmt.Sprintf("%s.DOUBLE", seqVar)
	case "DECIMAL":
		if dt.Precision > 0 && dt.Scale > 0 {
			return fmt.Sprintf("%s.DECIMAL(%d, %d)", seqVar, dt.Precision, dt.Scale)
		}
		return fmt.Sprintf("%s.DECIMAL", seqVar)
	case "BOOLEAN":
		return fmt.Sprintf("%s.BOOLEAN", seqVar)
	case "DATE":
		return fmt.Sprintf("%s.DATE", seqVar)
	case "DATEONLY":
		return fmt.Sprintf("%s.DATEONLY", seqVar)
	case "TEXT":
		return fmt.Sprintf("%s.TEXT", seqVar)
	case "JSON":
		return fmt.Sprintf("%s.JSON", seqVar)
	case "UUID":
		return fmt.Sprintf("%s.UUID", seqVar)
	case "ENUM":
		if len(dt.Values) > 0 {
			quoted := make([]string, len(dt.Values))
			for i, v := range dt.Values {
				quoted[i] = fmt.Sprintf("'%s'", v)
			}
			return fmt.Sprintf("%s.ENUM(%s)", seqVar, strings.Join(quoted, ", "))
		}
		return fmt.Sprintf("%s.ENUM", seqVar)
	default:
		return fmt.Sprintf("%s.%s", seqVar, name)
	}
}

func formatColumnAttributes(col domain.Column, seqVar string) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString(fmt.Sprintf("      type: %s,\n", formatDataType(col.Type, seqVar)))
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

func formatCreateTable(table domain.Table, seqVar string) string {
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
		cols = append(cols, fmt.Sprintf("    %s: %s", colName, formatColumnAttributes(col, seqVar)))
	}
	return fmt.Sprintf("await queryInterface.createTable('%s', {\n%s\n});", table.Name, strings.Join(cols, ",\n"))
}

func formatAddColumn(tableName string, col domain.Column, seqVar string) string {
	return fmt.Sprintf("await queryInterface.addColumn('%s', '%s', %s);", tableName, col.Name, formatColumnAttributes(col, seqVar))
}

func formatChangeColumn(tableName string, col domain.Column, seqVar string) string {
	return fmt.Sprintf("await queryInterface.changeColumn('%s', '%s', %s);", tableName, col.Name, formatColumnAttributes(col, seqVar))
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
