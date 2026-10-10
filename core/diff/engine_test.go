package diff

import (
	"testing"
	"github.com/hsnacar/sequelize-makemigrations/core/domain"
)

func TestDiffEngine(t *testing.T) {
	engine := NewEngine()

	t.Run("Empty schemas produce empty diff", func(t *testing.T) {
		s1 := domain.NewSchema()
		s2 := domain.NewSchema()
		d := engine.Compare(s1, s2)
		if !d.IsEmpty() {
			t.Errorf("expected diff to be empty, got %+v", d)
		}
		if d.Summary() != "No schema changes detected." {
			t.Errorf("expected no changes summary, got: %s", d.Summary())
		}
	})

	t.Run("Handles nil schemas gracefully", func(t *testing.T) {
		d := engine.Compare(nil, nil)
		if !d.IsEmpty() {
			t.Errorf("expected diff from nil schemas to be empty")
		}
	})

	t.Run("Detects created and dropped tables", func(t *testing.T) {
		s1 := domain.NewSchema()
		s1.Tables["OldTable"] = domain.Table{Name: "OldTable"}

		s2 := domain.NewSchema()
		s2.Tables["NewTable"] = domain.Table{Name: "NewTable"}

		d := engine.Compare(s1, s2)
		if len(d.TablesCreated) != 1 || d.TablesCreated[0].Name != "NewTable" {
			t.Errorf("expected 1 created table NewTable, got %+v", d.TablesCreated)
		}
		if len(d.TablesDropped) != 1 || d.TablesDropped[0].Name != "OldTable" {
			t.Errorf("expected 1 dropped table OldTable, got %+v", d.TablesDropped)
		}
	})

	t.Run("Detects added, dropped and modified columns", func(t *testing.T) {
		s1 := domain.NewSchema()
		s1.Tables["Users"] = domain.Table{
			Name: "Users",
			Columns: map[string]domain.Column{
				"id":   {Name: "id", Type: domain.DataType{Name: "INTEGER"}, PrimaryKey: true},
				"name": {Name: "name", Type: domain.DataType{Name: "STRING", Length: 100}, AllowNull: false},
				"bio":  {Name: "bio", Type: domain.DataType{Name: "STRING", Length: 255}},
			},
		}

		s2 := domain.NewSchema()
		s2.Tables["Users"] = domain.Table{
			Name: "Users",
			Columns: map[string]domain.Column{
				"id":    {Name: "id", Type: domain.DataType{Name: "INTEGER"}, PrimaryKey: true},
				"name":  {Name: "name", Type: domain.DataType{Name: "STRING", Length: 200}, AllowNull: true}, // Modified length & allowNull
				"email": {Name: "email", Type: domain.DataType{Name: "STRING", Length: 255}, Unique: true},   // Added
			},
		}

		d := engine.Compare(s1, s2)

		if len(d.ColumnsAdded) != 1 || d.ColumnsAdded[0].Column.Name != "email" {
			t.Errorf("expected 1 added column email, got %+v", d.ColumnsAdded)
		}
		if len(d.ColumnsDropped) != 1 || d.ColumnsDropped[0].Column.Name != "bio" {
			t.Errorf("expected 1 dropped column bio, got %+v", d.ColumnsDropped)
		}
		if d.ColumnsDropped[0].OldColumn == nil || d.ColumnsDropped[0].OldColumn.Name != "bio" {
			t.Errorf("expected OldColumn to be preserved for rollback")
		}
		if len(d.ColumnsModified) != 1 || d.ColumnsModified[0].Column.Name != "name" {
			t.Errorf("expected 1 modified column name, got %+v", d.ColumnsModified)
		}
		if d.ColumnsModified[0].OldColumn == nil || d.ColumnsModified[0].OldColumn.Type.Length != 100 {
			t.Errorf("expected OldColumn length 100 to be preserved for rollback")
		}
	})

	t.Run("Detects column attribute changes (defaultValue, primaryKey, autoIncrement, unique, type)", func(t *testing.T) {
		colBase := domain.Column{Name: "col", Type: domain.DataType{Name: "INTEGER"}, DefaultValue: 10}

		// Type changed
		colTypeChanged := colBase
		colTypeChanged.Type = domain.DataType{Name: "BIGINT"}
		if !hasColumnChanged(colBase, colTypeChanged) {
			t.Errorf("expected type change to be detected")
		}

		// Default value changed
		colDefChanged := colBase
		colDefChanged.DefaultValue = 20
		if !hasColumnChanged(colBase, colDefChanged) {
			t.Errorf("expected default value change to be detected")
		}

		// Primary key changed
		colPkChanged := colBase
		colPkChanged.PrimaryKey = true
		if !hasColumnChanged(colBase, colPkChanged) {
			t.Errorf("expected primary key change to be detected")
		}

		// AutoIncrement changed
		colAiChanged := colBase
		colAiChanged.AutoIncrement = true
		if !hasColumnChanged(colBase, colAiChanged) {
			t.Errorf("expected autoIncrement change to be detected")
		}

		// Unique changed
		colUniqChanged := colBase
		colUniqChanged.Unique = true
		if !hasColumnChanged(colBase, colUniqChanged) {
			t.Errorf("expected unique change to be detected")
		}

		// References changed
		colRefChanged := colBase
		colRefChanged.References = &domain.Reference{Model: "Users", Key: "id"}
		if !hasColumnChanged(colBase, colRefChanged) {
			t.Errorf("expected foreign key reference change to be detected")
		}
	})

	t.Run("Detects added and dropped indexes", func(t *testing.T) {
		s1 := domain.NewSchema()
		s1.Tables["Orders"] = domain.Table{
			Name: "Orders",
			Indexes: []domain.Index{
				{Name: "idx_old_status", Fields: []string{"status"}},
			},
		}

		s2 := domain.NewSchema()
		s2.Tables["Orders"] = domain.Table{
			Name: "Orders",
			Indexes: []domain.Index{
				{Name: "idx_user_created", Fields: []string{"user_id", "created_at"}, Unique: true},
			},
		}

		d := engine.Compare(s1, s2)
		if len(d.IndexesAdded) != 1 || d.IndexesAdded[0].Index.Name != "idx_user_created" {
			t.Errorf("expected 1 added index idx_user_created, got %+v", d.IndexesAdded)
		}
		if len(d.IndexesDropped) != 1 || d.IndexesDropped[0].Index.Name != "idx_old_status" {
			t.Errorf("expected 1 dropped index idx_old_status, got %+v", d.IndexesDropped)
		}
	})

	t.Run("Detects added and dropped constraints", func(t *testing.T) {
		s1 := domain.NewSchema()
		s1.Tables["Orders"] = domain.Table{
			Name: "Orders",
			Constraints: []domain.Constraint{
				{Name: "fk_old", Type: "FOREIGN_KEY"},
			},
		}

		s2 := domain.NewSchema()
		s2.Tables["Orders"] = domain.Table{
			Name: "Orders",
			Constraints: []domain.Constraint{
				{Name: "fk_user_id", Type: "FOREIGN_KEY", ForeignTable: "Users", ForeignField: "id"},
			},
		}

		d := engine.Compare(s1, s2)
		if len(d.ConstraintsAdded) != 1 || d.ConstraintsAdded[0].Constraint.Name != "fk_user_id" {
			t.Errorf("expected 1 added constraint fk_user_id, got %+v", d.ConstraintsAdded)
		}
		if len(d.ConstraintsDropped) != 1 || d.ConstraintsDropped[0].Constraint.Name != "fk_old" {
			t.Errorf("expected 1 dropped constraint fk_old, got %+v", d.ConstraintsDropped)
		}
	})
}
