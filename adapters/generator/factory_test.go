package generator

import (
	"strings"
	"testing"

	"github.com/hsnacar/sequelize-makemigrations/core/domain"
	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

func TestGetGenerator(t *testing.T) {
	sampleDiff := &domain.Diff{
		TablesCreated: []domain.Table{
			{
				Name: "Items",
				Columns: map[string]domain.Column{
					"name": {Name: "name", Type: domain.DataType{Name: "STRING"}},
				},
			},
		},
	}

	t.Run("Returns v6 generator by default and for v6 with Sequelize syntax", func(t *testing.T) {
		genDefault := GetGenerator("")
		resDefault, err := genDefault.Generate(sampleDiff, ports.GeneratorOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(resDefault.Content, "async up(queryInterface, Sequelize)") {
			t.Errorf("expected v6 up signature in default generator")
		}
		if strings.Contains(resDefault.Content, "@sequelize/core") {
			t.Errorf("did not expect @sequelize/core in v6 default generator")
		}

		genV6 := GetGenerator("v6")
		resV6, err := genV6.Generate(sampleDiff, ports.GeneratorOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(resV6.Content, "Sequelize.STRING") {
			t.Errorf("expected Sequelize.STRING in v6 generator")
		}
	})

	t.Run("Returns v7 generator for v7 with @sequelize/core and DataTypes syntax", func(t *testing.T) {
		genV7 := GetGenerator("v7")
		resV7, err := genV7.Generate(sampleDiff, ports.GeneratorOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(resV7.Content, "@sequelize/core") {
			t.Errorf("expected @sequelize/core import in v7 generator")
		}
		if !strings.Contains(resV7.Content, "DataTypes.STRING") {
			t.Errorf("expected DataTypes.STRING in v7 generator")
		}
		if !strings.Contains(resV7.Content, "async up(queryInterface)") {
			t.Errorf("expected v7 up(queryInterface) signature in v7 generator")
		}
	})
}
