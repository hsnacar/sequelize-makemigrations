package v6

import (
	"strings"
	"testing"

	"github.com/hsnacar/sequelize-makemigrations/core/domain"
	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

func TestGeneratorV6(t *testing.T) {
	gen := NewGenerator()

	t.Run("Generates valid up and down for table creation with rich types and attributes", func(t *testing.T) {
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
						"role": {
							Name:         "role",
							Type:         domain.DataType{Name: "ENUM", Values: []string{"admin", "user", "guest"}},
							DefaultValue: "user",
							AllowNull:    false,
						},
						"balance": {
							Name:         "balance",
							Type:         domain.DataType{Name: "DECIMAL", Precision: 12, Scale: 4},
							DefaultValue: 0.0,
							AllowNull:    false,
						},
						"isActive": {
							Name:         "isActive",
							Type:         domain.DataType{Name: "BOOLEAN"},
							DefaultValue: true,
							AllowNull:    false,
						},
						"createdAt": {
							Name:      "createdAt",
							Type:      domain.DataType{Name: "DATE"},
							AllowNull: false,
						},
					},
				},
			},
		}

		res, err := gen.Generate(diff, ports.GeneratorOptions{
			MigrationName: "create users table",
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
		if !strings.Contains(res.Content, "Sequelize.ENUM('admin', 'user', 'guest')") {
			t.Errorf("expected Sequelize.ENUM in content")
		}
		if !strings.Contains(res.Content, "Sequelize.DECIMAL(12, 4)") {
			t.Errorf("expected Sequelize.DECIMAL(12, 4) in content")
		}
		if !strings.Contains(res.Content, "defaultValue: 'user'") {
			t.Errorf("expected string defaultValue formatted with quotes")
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

		if !strings.Contains(res.Content, "await queryInterface.addColumn('Users', 'avatar',") {
			t.Errorf("expected addColumn for avatar in UP")
		}
		if !strings.Contains(res.Content, "await queryInterface.removeColumn('Users', 'bio');") {
			t.Errorf("expected removeColumn for bio in UP")
		}

		if !strings.Contains(res.Content, "await queryInterface.addColumn('Users', 'bio',") {
			t.Errorf("expected addColumn for bio in DOWN rollback")
		}
		if !strings.Contains(res.Content, "await queryInterface.removeColumn('Users', 'avatar');") {
			t.Errorf("expected removeColumn for avatar in DOWN rollback")
		}
	})

	t.Run("Generates reversible changeColumn restoring OldColumn", func(t *testing.T) {
		oldCol := domain.Column{
			Name:      "title",
			Type:      domain.DataType{Name: "STRING", Length: 100},
			AllowNull: false,
		}
		newCol := domain.Column{
			Name:      "title",
			Type:      domain.DataType{Name: "STRING", Length: 250},
			AllowNull: true,
		}
		diff := &domain.Diff{
			ColumnsModified: []domain.ColumnDiff{
				{
					TableName: "Posts",
					Column:    newCol,
					OldColumn: &oldCol,
				},
			},
		}

		res, err := gen.Generate(diff, ports.GeneratorOptions{MigrationName: "widen-title"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// UP changes title to STRING(250)
		if !strings.Contains(res.Content, "await queryInterface.changeColumn('Posts', 'title',") {
			t.Errorf("expected changeColumn in UP")
		}
		if !strings.Contains(res.Content, "Sequelize.STRING(250)") {
			t.Errorf("expected STRING(250) in UP")
		}

		// DOWN reverts title to STRING(100)
		if !strings.Contains(res.Content, "Sequelize.STRING(100)") {
			t.Errorf("expected OldColumn STRING(100) restored in DOWN")
		}
	})

	t.Run("Generates reversible dropTable and createTable", func(t *testing.T) {
		droppedTable := domain.Table{
			Name: "LegacyLogs",
			Columns: map[string]domain.Column{
				"id":  {Name: "id", Type: domain.DataType{Name: "INTEGER"}, PrimaryKey: true},
				"msg": {Name: "msg", Type: domain.DataType{Name: "TEXT"}},
			},
		}
		diff := &domain.Diff{
			TablesDropped: []domain.Table{droppedTable},
		}

		res, err := gen.Generate(diff, ports.GeneratorOptions{MigrationName: "drop-logs"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// UP drops LegacyLogs
		if !strings.Contains(res.Content, "await queryInterface.dropTable('LegacyLogs');") {
			t.Errorf("expected dropTable in UP")
		}

		// DOWN re-creates LegacyLogs
		if !strings.Contains(res.Content, "await queryInterface.createTable('LegacyLogs',") {
			t.Errorf("expected createTable in DOWN to restore dropped table")
		}
	})

	t.Run("Covers all DataType branches", func(t *testing.T) {
		types := []domain.DataType{
			{Name: "BIGINT"},
			{Name: "FLOAT"},
			{Name: "DOUBLE"},
			{Name: "DATEONLY"},
			{Name: "TEXT"},
			{Name: "JSON"},
			{Name: "UUID"},
			{Name: "CUSTOM_TYPE"},
		}

		for _, dt := range types {
			str := formatDataType(dt, "Sequelize")
			if !strings.HasPrefix(str, "Sequelize.") {
				t.Errorf("expected prefix Sequelize., got: %s", str)
			}
		}
	})
}
