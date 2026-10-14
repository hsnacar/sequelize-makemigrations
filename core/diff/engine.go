package diff

import (
	"reflect"
	"github.com/hsnacar/sequelize-makemigrations/core/domain"
	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

type Engine struct{}

func NewEngine() ports.DiffEngine {
	return &Engine{}
}

// Compare calculates the complete delta between old and new schema snapshots.
func (e *Engine) Compare(oldSchema, newSchema *domain.Schema) *domain.Diff {
	diff := &domain.Diff{
		TablesCreated:      []domain.Table{},
		TablesDropped:      []domain.Table{},
		ColumnsAdded:       []domain.ColumnDiff{},
		ColumnsDropped:     []domain.ColumnDiff{},
		ColumnsModified:    []domain.ColumnDiff{},
		IndexesAdded:       []domain.IndexDiff{},
		IndexesDropped:     []domain.IndexDiff{},
		ConstraintsAdded:   []domain.ConstraintDiff{},
		ConstraintsDropped: []domain.ConstraintDiff{},
	}

	if oldSchema == nil {
		oldSchema = domain.NewSchema()
	}
	if newSchema == nil {
		newSchema = domain.NewSchema()
	}

	// 1. Detect New Tables
	for tableName, newTable := range newSchema.Tables {
		if _, exists := oldSchema.Tables[tableName]; !exists {
			diff.TablesCreated = append(diff.TablesCreated, newTable)
		}
	}

	// 2. Detect Dropped Tables
	for tableName, oldTable := range oldSchema.Tables {
		if _, exists := newSchema.Tables[tableName]; !exists {
			diff.TablesDropped = append(diff.TablesDropped, oldTable)
		}
	}

	// 3. Inspect Existing Tables for Column / Index / Constraint changes
	for tableName, newTable := range newSchema.Tables {
		oldTable, exists := oldSchema.Tables[tableName]
		if !exists {
			continue // Already captured as created table
		}

		// Columns Added
		for colName, newCol := range newTable.Columns {
			if _, colExists := oldTable.Columns[colName]; !colExists {
				diff.ColumnsAdded = append(diff.ColumnsAdded, domain.ColumnDiff{
					TableName: tableName,
					Column:    newCol,
				})
			}
		}

		// Columns Dropped
		for colName, oldCol := range oldTable.Columns {
			if _, colExists := newTable.Columns[colName]; !colExists {
				diff.ColumnsDropped = append(diff.ColumnsDropped, domain.ColumnDiff{
					TableName: tableName,
					Column:    oldCol,
					OldColumn: &oldCol,
				})
			}
		}

		// Columns Modified
		for colName, newCol := range newTable.Columns {
			if oldCol, colExists := oldTable.Columns[colName]; colExists {
				if hasColumnChanged(oldCol, newCol) {
					oldCopy := oldCol
					diff.ColumnsModified = append(diff.ColumnsModified, domain.ColumnDiff{
						TableName: tableName,
						Column:    newCol,
						OldColumn: &oldCopy,
					})
				}
			}
		}

		// Indexes Added & Dropped
		e.diffIndexes(tableName, oldTable.Indexes, newTable.Indexes, diff)

		// Constraints Added & Dropped
		e.diffConstraints(tableName, oldTable.Constraints, newTable.Constraints, diff)
	}

	return diff
}

func hasColumnChanged(oldCol, newCol domain.Column) bool {
	if oldCol.Type.Name != newCol.Type.Name {
		return true
	}
	if oldCol.Type.Length != newCol.Type.Length {
		return true
	}
	if oldCol.Type.Precision != newCol.Type.Precision {
		return true
	}
	if oldCol.Type.Scale != newCol.Type.Scale {
		return true
	}
	if !slicesEqual(oldCol.Type.Values, newCol.Type.Values) {
		return true
	}
	if oldCol.AllowNull != newCol.AllowNull {
		return true
	}
	if oldCol.PrimaryKey != newCol.PrimaryKey {
		return true
	}
	if oldCol.AutoIncrement != newCol.AutoIncrement {
		return true
	}
	if !reflect.DeepEqual(oldCol.DefaultValue, newCol.DefaultValue) {
		return true
	}
	if !uniqueEqual(oldCol.Unique, newCol.Unique) {
		return true
	}
	if oldCol.Comment != newCol.Comment {
		return true
	}
	if !reflect.DeepEqual(oldCol.References, newCol.References) {
		return true
	}
	return false
}

func slicesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func uniqueEqual(a, b interface{}) bool {
	hasA, nameA := parseUnique(a)
	hasB, nameB := parseUnique(b)
	return hasA == hasB && nameA == nameB
}

func parseUnique(u interface{}) (bool, string) {
	if u == nil {
		return false, ""
	}
	switch v := u.(type) {
	case bool:
		return v, ""
	case string:
		return v != "", v
	default:
		return false, ""
	}
}

func (e *Engine) diffIndexes(tableName string, oldIndexes, newIndexes []domain.Index, diff *domain.Diff) {
	oldMap := make(map[string]domain.Index)
	for _, idx := range oldIndexes {
		oldMap[idx.Name] = idx
	}

	newMap := make(map[string]domain.Index)
	for _, idx := range newIndexes {
		newMap[idx.Name] = idx
		oldIdx, exists := oldMap[idx.Name]
		if !exists {
			diff.IndexesAdded = append(diff.IndexesAdded, domain.IndexDiff{
				TableName: tableName,
				Index:     idx,
			})
		} else if !reflect.DeepEqual(oldIdx, idx) {
			diff.IndexesDropped = append(diff.IndexesDropped, domain.IndexDiff{
				TableName: tableName,
				Index:     oldIdx,
			})
			diff.IndexesAdded = append(diff.IndexesAdded, domain.IndexDiff{
				TableName: tableName,
				Index:     idx,
			})
		}
	}

	for _, idx := range oldIndexes {
		if _, exists := newMap[idx.Name]; !exists {
			diff.IndexesDropped = append(diff.IndexesDropped, domain.IndexDiff{
				TableName: tableName,
				Index:     idx,
			})
		}
	}
}

func (e *Engine) diffConstraints(tableName string, oldConstraints, newConstraints []domain.Constraint, diff *domain.Diff) {
	oldMap := make(map[string]domain.Constraint)
	for _, c := range oldConstraints {
		oldMap[c.Name] = c
	}

	newMap := make(map[string]domain.Constraint)
	for _, c := range newConstraints {
		newMap[c.Name] = c
		oldC, exists := oldMap[c.Name]
		if !exists {
			diff.ConstraintsAdded = append(diff.ConstraintsAdded, domain.ConstraintDiff{
				TableName:  tableName,
				Constraint: c,
			})
		} else if !reflect.DeepEqual(oldC, c) {
			diff.ConstraintsDropped = append(diff.ConstraintsDropped, domain.ConstraintDiff{
				TableName:  tableName,
				Constraint: oldC,
			})
			diff.ConstraintsAdded = append(diff.ConstraintsAdded, domain.ConstraintDiff{
				TableName:  tableName,
				Constraint: c,
			})
		}
	}

	for _, c := range oldConstraints {
		if _, exists := newMap[c.Name]; !exists {
			diff.ConstraintsDropped = append(diff.ConstraintsDropped, domain.ConstraintDiff{
				TableName:  tableName,
				Constraint: c,
			})
		}
	}
}
