package v6

import (
	"strings"
	"testing"

	"github.com/hsnacar/sequelize-makemigrations/core/domain"
	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

func TestGeneratorV6(t *testing.T) {
	gen := NewGenerator()

	t.Run("Generates valid up and down for table creation", func(t *testing.T) {
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
					},
				},
			},
		}

		res, err := gen.Generate(diff, ports.GeneratorOptions{
			MigrationName: "create-users-table",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(res.Filename, "create-users-table.js") {
			t.Errorf("filename should contain slug, got %s", res.Filename)
		}

		// UP must have createTable
		if !strings.Contains(res.Content, "await queryInterface.createTable('Users',") {
			t.Errorf("expected createTable in UP block")
		}
		if !strings.Contains(res.Content, "Sequelize.INTEGER") {
			t.Errorf("expected Sequelize.INTEGER in content")
		}
		if !strings.Contains(res.Content, "Sequelize.STRING(255)") {
			t.Errorf("expected Sequelize.STRING(255) in content")
		}

		// DOWN must have exact reverse dropTable
		if !strings.Contains(res.Content, "await queryInterface.dropTable('Users');") {
			t.Errorf("expected dropTable in DOWN block for rollback")
		}
	})

	t.Run("Generates reversible addColumn and removeColumn", func(t *testing.T) {
		oldBio := domain.Column{
			Name: "bio",
			Type: domain.DataType{Name: "TEXT"},
		}
		diff := &domain.Diff{
			ColumnsAdded: []domain.ColumnDiff{
				{
					TableName: "Users",
					Column: domain.Column{
						Name:      "avatar",
						Type:      domain.DataType{Name: "STRING", Length: 500},
						AllowNull: true,
					},
				},
			},
			ColumnsDropped: []domain.ColumnDiff{
				{
					TableName: "Users",
					Column:    oldBio,
					OldColumn: &oldBio,
				},
			},
		}

		res, err := gen.Generate(diff, ports.GeneratorOptions{
			MigrationName: "alter-user-columns",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// In UP: avatar is added, bio is removed
		if !strings.Contains(res.Content, "await queryInterface.addColumn('Users', 'avatar',") {
			t.Errorf("expected addColumn for avatar in UP")
		}
		if !strings.Contains(res.Content, "await queryInterface.removeColumn('Users', 'bio');") {
			t.Errorf("expected removeColumn for bio in UP")
		}

		// In DOWN (Reversed!): avatar is removed, bio is restored with its exact TEXT type!
		if !strings.Contains(res.Content, "await queryInterface.addColumn('Users', 'bio',") {
			t.Errorf("expected addColumn for bio in DOWN rollback")
		}
		if !strings.Contains(res.Content, "await queryInterface.removeColumn('Users', 'avatar');") {
			t.Errorf("expected removeColumn for avatar in DOWN rollback")
		}
	})
}
