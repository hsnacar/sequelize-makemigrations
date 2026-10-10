package v7

import (
	"strings"
	"testing"

	"github.com/hsnacar/sequelize-makemigrations/core/domain"
	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

func TestGeneratorV7(t *testing.T) {
	gen := NewGenerator()

	t.Run("Generates valid up and down with @sequelize/core import and DataTypes", func(t *testing.T) {
		diff := &domain.Diff{
			TablesCreated: []domain.Table{
				{
					Name: "Users",
					Columns: map[string]domain.Column{
						"id": {
							Name:          "id",
							Type:          domain.DataType{Name: "INTEGER"},
							PrimaryKey:    true,
							AutoIncrement: true,
							AllowNull:     false,
						},
						"email": {
							Name:      "email",
							Type:      domain.DataType{Name: "STRING", Length: 255},
							AllowNull: false,
							Unique:    true,
						},
						"balance": {
							Name:      "balance",
							Type:      domain.DataType{Name: "NUMERIC", Precision: 10, Scale: 2}, // Should map to DECIMAL in v7
							AllowNull: false,
						},
						"role": {
							Name:         "role",
							Type:         domain.DataType{Name: "ENUM", Values: []string{"admin", "member"}},
							DefaultValue: "member",
							AllowNull:    false,
						},
						"isActive": {
							Name:         "isActive",
							Type:         domain.DataType{Name: "BOOLEAN"},
							DefaultValue: true,
							AllowNull:    false,
						},
					},
				},
			},
		}

		res, err := gen.Generate(diff, ports.GeneratorOptions{MigrationName: "create-users"})
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if !strings.Contains(res.Content, "require('@sequelize/core')") {
			t.Errorf("Expected @sequelize/core import in v7 migration, got:\n%s", res.Content)
		}

		if !strings.Contains(res.Content, "DataTypes.DECIMAL(10, 2)") {
			t.Errorf("Expected NUMERIC to map to DataTypes.DECIMAL(10, 2) in v7, got:\n%s", res.Content)
		}

		if !strings.Contains(res.Content, "DataTypes.ENUM('admin', 'member')") {
			t.Errorf("Expected DataTypes.ENUM in v7, got:\n%s", res.Content)
		}

		if !strings.Contains(res.Content, "DataTypes.STRING(255)") {
			t.Errorf("Expected DataTypes.STRING(255), got:\n%s", res.Content)
		}

		if !strings.Contains(res.Content, "async up(queryInterface)") {
			t.Errorf("Expected v7 up(queryInterface) signature, got:\n%s", res.Content)
		}

		if !strings.Contains(res.Content, "async down(queryInterface)") {
			t.Errorf("Expected v7 down(queryInterface) signature, got:\n%s", res.Content)
		}

		if !strings.Contains(res.Content, "await queryInterface.dropTable('Users');") {
			t.Errorf("Expected down to drop Users table, got:\n%s", res.Content)
		}
	})

	t.Run("Generates reversible changeColumn and dropTable in v7", func(t *testing.T) {
		oldCol := domain.Column{
			Name: "note",
			Type: domain.DataType{Name: "STRING", Length: 50},
		}
		newCol := domain.Column{
			Name: "note",
			Type: domain.DataType{Name: "TEXT"},
		}

		droppedTable := domain.Table{
			Name: "AuditLogs",
			Columns: map[string]domain.Column{
				"id": {Name: "id", Type: domain.DataType{Name: "INTEGER"}, PrimaryKey: true},
			},
		}

		diff := &domain.Diff{
			ColumnsModified: []domain.ColumnDiff{
				{TableName: "Tasks", Column: newCol, OldColumn: &oldCol},
			},
			TablesDropped: []domain.Table{droppedTable},
		}

		res, err := gen.Generate(diff, ports.GeneratorOptions{MigrationName: "modify-tasks"})
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// UP operations
		if !strings.Contains(res.Content, "await queryInterface.changeColumn('Tasks', 'note',") {
			t.Errorf("Expected changeColumn in UP")
		}
		if !strings.Contains(res.Content, "await queryInterface.dropTable('AuditLogs');") {
			t.Errorf("Expected dropTable in UP")
		}

		// DOWN operations
		if !strings.Contains(res.Content, "await queryInterface.createTable('AuditLogs',") {
			t.Errorf("Expected createTable in DOWN to restore dropped table")
		}
		if !strings.Contains(res.Content, "DataTypes.STRING(50)") {
			t.Errorf("Expected OldColumn restored in DOWN")
		}
	})

	t.Run("Covers all DataType branches in v7", func(t *testing.T) {
		types := []domain.DataType{
			{Name: "BIGINT"},
			{Name: "FLOAT"},
			{Name: "DOUBLE PRECISION"}, // should become DOUBLE
			{Name: "DATE"},
			{Name: "DATEONLY"},
			{Name: "JSONTYPE"}, // should become JSON
			{Name: "UUID"},
			{Name: "CUSTOM_V7"},
		}

		for _, dt := range types {
			str := formatDataType(dt)
			if !strings.HasPrefix(str, "DataTypes.") {
				t.Errorf("expected prefix DataTypes., got: %s", str)
			}
		}
	})

	t.Run("Generates reversible addIndex and removeIndex in v7", func(t *testing.T) {
		diff := &domain.Diff{
			IndexesAdded: []domain.IndexDiff{
				{
					TableName: "Users",
					Index: domain.Index{
						Name:   "users_email_idx",
						Fields: []string{"email"},
						Unique: true,
					},
				},
			},
			IndexesDropped: []domain.IndexDiff{
				{
					TableName: "Orders",
					Index: domain.Index{
						Name:   "orders_status_idx",
						Fields: []string{"status"},
					},
				},
			},
		}

		res, err := gen.Generate(diff, ports.GeneratorOptions{MigrationName: "v7-indexes"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(res.Content, "await queryInterface.addIndex('Users', ['email']") {
			t.Errorf("expected addIndex in v7 UP, got: %s", res.Content)
		}
		if !strings.Contains(res.Content, "await queryInterface.removeIndex('Orders', 'orders_status_idx');") {
			t.Errorf("expected removeIndex in v7 UP, got: %s", res.Content)
		}
		if !strings.Contains(res.Content, "await queryInterface.addIndex('Orders', ['status']") {
			t.Errorf("expected addIndex in v7 DOWN, got: %s", res.Content)
		}
		if !strings.Contains(res.Content, "await queryInterface.removeIndex('Users', 'users_email_idx');") {
			t.Errorf("expected removeIndex in v7 DOWN, got: %s", res.Content)
		}
	})

	t.Run("Generates foreign key references in v7 column attributes", func(t *testing.T) {
		diff := &domain.Diff{
			ColumnsAdded: []domain.ColumnDiff{
				{
					TableName: "Orders",
					Column: domain.Column{
						Name: "customerId",
						Type: domain.DataType{Name: "INTEGER"},
						References: &domain.Reference{
							Model: "Customers",
							Key:   "id",
						},
					},
				},
			},
		}

		res, err := gen.Generate(diff, ports.GeneratorOptions{MigrationName: "v7-fk"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(res.Content, "model: 'Customers'") || !strings.Contains(res.Content, "key: 'id'") {
			t.Errorf("expected references block with Customers.id in generated code, got: %s", res.Content)
		}
	})
}
