package domain

import (
	"fmt"
	"strings"
)

// DataType represents standardized, dialect-agnostic column types.
type DataType struct {
	Name      string        `json:"name"`                // e.g. "STRING", "INTEGER", "BOOLEAN", "ENUM"
	Length    int           `json:"length,omitempty"`    // e.g. 255 for STRING(255)
	Precision int           `json:"precision,omitempty"` // for DECIMAL
	Scale     int           `json:"scale,omitempty"`     // for DECIMAL
	Values    []string      `json:"values,omitempty"`    // for ENUM
}

// Reference represents a foreign key target.
type Reference struct {
	Model string `json:"model"`
	Key   string `json:"key"`
}

// Column represents a single database column in the Schema IR.
type Column struct {
	Name          string      `json:"name"`
	Type          DataType    `json:"type"`
	AllowNull     bool        `json:"allowNull"`
	PrimaryKey    bool        `json:"primaryKey"`
	AutoIncrement bool        `json:"autoIncrement"`
	DefaultValue  interface{} `json:"defaultValue,omitempty"`
	Unique        bool        `json:"unique,omitempty"`
	Comment       string      `json:"comment,omitempty"`
	References    *Reference  `json:"references,omitempty"`
}

// Index represents a table index.
type Index struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
	Unique bool     `json:"unique"`
	Method string   `json:"method,omitempty"` // e.g. "BTREE", "GIN"
}

// Constraint represents table constraints (Foreign Keys, etc.).
type Constraint struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"` // "FOREIGN_KEY", "UNIQUE", "CHECK"
	Fields       []string `json:"fields"`
	ForeignTable string   `json:"foreignTable,omitempty"`
	ForeignField string   `json:"foreignField,omitempty"`
	OnDelete     string   `json:"onDelete,omitempty"`
	OnUpdate     string   `json:"onUpdate,omitempty"`
}

// Table represents a database table definition.
type Table struct {
	Name        string                `json:"name"`
	Columns     map[string]Column     `json:"columns"`
	Indexes     []Index               `json:"indexes,omitempty"`
	Constraints []Constraint          `json:"constraints,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// Schema represents the complete database schema snapshot (Schema IR).
type Schema struct {
	Version int              `json:"version"` // Schema format version
	Tables  map[string]Table `json:"tables"`
}

// NewSchema creates a blank initialized Schema.
func NewSchema() *Schema {
	return &Schema{
		Version: 1,
		Tables:  make(map[string]Table),
	}
}

// ColumnDiff represents a change to a single column.
type ColumnDiff struct {
	TableName string  `json:"tableName"`
	Column    Column  `json:"column"`
	OldColumn *Column `json:"oldColumn,omitempty"` // Populated for changes/drops for rollback
}

// IndexDiff represents index changes.
type IndexDiff struct {
	TableName string `json:"tableName"`
	Index     Index  `json:"index"`
}

// ConstraintDiff represents constraint changes.
type ConstraintDiff struct {
	TableName  string     `json:"tableName"`
	Constraint Constraint `json:"constraint"`
}

// Diff holds the full delta between two schemas.
type Diff struct {
	TablesCreated      []Table          `json:"tablesCreated"`
	TablesDropped      []Table          `json:"tablesDropped"`
	ColumnsAdded       []ColumnDiff     `json:"columnsAdded"`
	ColumnsDropped     []ColumnDiff     `json:"columnsDropped"`
	ColumnsModified    []ColumnDiff     `json:"columnsModified"`
	IndexesAdded       []IndexDiff      `json:"indexesAdded"`
	IndexesDropped     []IndexDiff      `json:"indexesDropped"`
	ConstraintsAdded   []ConstraintDiff `json:"constraintsAdded"`
	ConstraintsDropped []ConstraintDiff `json:"constraintsDropped"`
}

// IsEmpty returns true if there are no schema differences.
func (d *Diff) IsEmpty() bool {
	return len(d.TablesCreated) == 0 &&
		len(d.TablesDropped) == 0 &&
		len(d.ColumnsAdded) == 0 &&
		len(d.ColumnsDropped) == 0 &&
		len(d.ColumnsModified) == 0 &&
		len(d.IndexesAdded) == 0 &&
		len(d.IndexesDropped) == 0 &&
		len(d.ConstraintsAdded) == 0 &&
		len(d.ConstraintsDropped) == 0
}

// Summary returns a human-readable summary of the diff.
func (d *Diff) Summary() string {
	if d.IsEmpty() {
		return "No schema changes detected."
	}
	var lines []string
	if len(d.TablesCreated) > 0 {
		var names []string
		for _, t := range d.TablesCreated {
			names = append(names, t.Name)
		}
		lines = append(lines, fmt.Sprintf("Created tables (%d): %s", len(d.TablesCreated), strings.Join(names, ", ")))
	}
	if len(d.TablesDropped) > 0 {
		var names []string
		for _, t := range d.TablesDropped {
			names = append(names, t.Name)
		}
		lines = append(lines, fmt.Sprintf("Dropped tables (%d): %s", len(d.TablesDropped), strings.Join(names, ", ")))
	}
	if len(d.ColumnsAdded) > 0 {
		var names []string
		for _, c := range d.ColumnsAdded {
			names = append(names, fmt.Sprintf("%s.%s", c.TableName, c.Column.Name))
		}
		lines = append(lines, fmt.Sprintf("Added columns (%d): %s", len(d.ColumnsAdded), strings.Join(names, ", ")))
	}
	if len(d.ColumnsModified) > 0 {
		var names []string
		for _, c := range d.ColumnsModified {
			names = append(names, fmt.Sprintf("%s.%s", c.TableName, c.Column.Name))
		}
		lines = append(lines, fmt.Sprintf("Modified columns (%d): %s", len(d.ColumnsModified), strings.Join(names, ", ")))
	}
	if len(d.ColumnsDropped) > 0 {
		var names []string
		for _, c := range d.ColumnsDropped {
			names = append(names, fmt.Sprintf("%s.%s", c.TableName, c.Column.Name))
		}
		lines = append(lines, fmt.Sprintf("Dropped columns (%d): %s", len(d.ColumnsDropped), strings.Join(names, ", ")))
	}
	return strings.Join(lines, "\n")
}

