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
}
