package ports

import (
	"context"
	"github.com/hsnacar/sequelize-makemigrations/core/domain"
)

// Extractor extracts a Schema IR from a user's Sequelize application.
type Extractor interface {
	ExtractSchema(ctx context.Context, projectPath string) (*domain.Schema, error)
}

// DiffEngine compares two Schemas and outputs a structured Diff.
type DiffEngine interface {
	Compare(oldSchema, newSchema *domain.Schema) *domain.Diff
}

// GeneratorOptions contains configuration for code generation.
type GeneratorOptions struct {
	SequelizeVersion string // "v6" or "v7"
	Language         string // "javascript" or "typescript"
	MigrationName    string // e.g. "add-user-avatar"
}

// GeneratedMigration contains the source code and metadata of the migration.
type GeneratedMigration struct {
	Filename string
	Content  string
}

// Generator translates a Diff into version-specific Sequelize migration code.
type Generator interface {
	Generate(diff *domain.Diff, opts GeneratorOptions) (*GeneratedMigration, error)
}

// Verifier executes generated migrations against an isolated sandbox database.
type Verifier interface {
	Verify(ctx context.Context, migration *GeneratedMigration) error
}
