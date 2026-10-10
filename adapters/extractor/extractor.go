package extractor

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hsnacar/sequelize-makemigrations/core/domain"
	"github.com/hsnacar/sequelize-makemigrations/core/ports"
)

//go:embed bridge.js
var bridgeScript []byte

type NodeExtractor struct {
	ModelsPath string
}

func NewNodeExtractor(modelsPath string) ports.Extractor {
	return &NodeExtractor{
		ModelsPath: modelsPath,
	}
}

func (e *NodeExtractor) ExtractSchema(ctx context.Context, projectPath string) (*domain.Schema, error) {
	// Write bridge script to a temporary file
	tmpFile, err := os.CreateTemp("", "seq-bridge-*.js")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary bridge script: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(bridgeScript); err != nil {
		return nil, fmt.Errorf("failed to write bridge script: %w", err)
	}
	tmpFile.Close()

	cmd := exec.CommandContext(ctx, "node", tmpFile.Name())
	cmd.Dir = projectPath
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SEQ_PROJECT_ROOT=%s", projectPath),
	)
	if e.ModelsPath != "" {
		absModels := e.ModelsPath
		if !filepath.IsAbs(absModels) {
			absModels = filepath.Join(projectPath, absModels)
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("SEQ_MODELS_PATH=%s", absModels))
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("extractor failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute node bridge: %w", err)
	}

	var schema domain.Schema
	if err := json.Unmarshal(output, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema IR from bridge output: %w (raw output: %s)", err, string(output))
	}

	return &schema, nil
}
