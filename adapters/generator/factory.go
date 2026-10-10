package generator

import (
	"strings"

	"github.com/hsnacar/sequelize-makemigrations/adapters/generator/v6"
	"github.com/hsnacar/sequelize-makemigrations/adapters/generator/v7"
	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

// GetGenerator returns the appropriate generator adapter for the targeted Sequelize version.
func GetGenerator(version string) ports.Generator {
	v := strings.ToLower(strings.TrimSpace(version))
	switch v {
	case "v7", "7":
		return v7.NewGenerator()
	default:
		return v6.NewGenerator()
	}
}
