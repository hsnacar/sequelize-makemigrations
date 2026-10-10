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
				// "bio" is dropped
			},
		}

		d := engine.Compare(s1, s2)

		// Check added
		if len(d.ColumnsAdded) != 1 || d.ColumnsAdded[0].Column.Name != "email" {
			t.Errorf("expected 1 added column email, got %+v", d.ColumnsAdded)
		}

		// Check dropped
		if len(d.ColumnsDropped) != 1 || d.ColumnsDropped[0].Column.Name != "bio" {
			t.Errorf("expected 1 dropped column bio, got %+v", d.ColumnsDropped)
		}
		if d.ColumnsDropped[0].OldColumn == nil || d.ColumnsDropped[0].OldColumn.Name != "bio" {
			t.Errorf("expected OldColumn to be preserved for rollback")
		}

		// Check modified
		if len(d.ColumnsModified) != 1 || d.ColumnsModified[0].Column.Name != "name" {
			t.Errorf("expected 1 modified column name, got %+v", d.ColumnsModified)
		}
		if d.ColumnsModified[0].OldColumn == nil || d.ColumnsModified[0].OldColumn.Type.Length != 100 {
			t.Errorf("expected OldColumn length 100 to be preserved for rollback")
		}
	})
}
