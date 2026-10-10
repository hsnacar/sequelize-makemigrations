package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TransformationResult summarizes the result of upgrading a migration file.
type TransformationResult struct {
	File        string `json:"file"`
	ChangesMade int    `json:"changesMade"`
	BackupPath  string `json:"backupPath,omitempty"`
	Diff        string `json:"diff,omitempty"`
}

var (
	reNumeric         = regexp.MustCompile(`Sequelize\.NUMERIC`)
	reJsonType        = regexp.MustCompile(`Sequelize\.JSONTYPE`)
	reDoublePrecision = regexp.MustCompile(`Sequelize\.DOUBLE\s+PRECISION`)
	reSequelizeTypes  = regexp.MustCompile(`Sequelize\.(STRING|TEXT|INTEGER|BIGINT|BOOLEAN|DATE|DATEONLY|FLOAT|DOUBLE|DECIMAL|JSON|UUID|ENUM)`)
)

// UpgradeFile transforms a single migration file to Sequelize v7 syntax.
func UpgradeFile(filePath string, dryRun, makeBackup bool) (*TransformationResult, error) {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	original := string(contentBytes)
	modified := original
	changes := 0

	// 1. Replace removed types
	if reNumeric.MatchString(modified) {
		modified = reNumeric.ReplaceAllString(modified, "DataTypes.DECIMAL")
		changes++
	}
	if reJsonType.MatchString(modified) {
		modified = reJsonType.ReplaceAllString(modified, "DataTypes.JSON")
		changes++
	}
	if reDoublePrecision.MatchString(modified) {
		modified = reDoublePrecision.ReplaceAllString(modified, "DataTypes.DOUBLE")
		changes++
	}

	// 2. Transform general Sequelize.TYPE to DataTypes.TYPE
	if reSequelizeTypes.MatchString(modified) {
		matches := reSequelizeTypes.FindAllString(modified, -1)
		changes += len(matches)
		modified = reSequelizeTypes.ReplaceAllString(modified, "DataTypes.$1")
	}

	// 3. Inject DataTypes import if not present
	if changes > 0 && !strings.Contains(modified, "@sequelize/core") {
		importHeader := "const { DataTypes } = require('@sequelize/core');\n"
		if strings.Contains(modified, "'use strict';") {
			modified = strings.Replace(modified, "'use strict';", "'use strict';\n\n"+importHeader, 1)
		} else {
			modified = importHeader + "\n" + modified
		}
	}

	result := &TransformationResult{
		File:        filepath.Base(filePath),
		ChangesMade: changes,
	}

	if changes == 0 {
		return result, nil // No changes needed
	}

	if dryRun {
		return result, nil
	}

	// Create backup
	if makeBackup {
		backupPath := filePath + ".bak"
		if err := os.WriteFile(backupPath, contentBytes, 0644); err != nil {
			return nil, fmt.Errorf("failed to create backup for %s: %w", filePath, err)
		}
		result.BackupPath = backupPath
	}

	// Write upgraded content
	if err := os.WriteFile(filePath, []byte(modified), 0644); err != nil {
		return nil, fmt.Errorf("failed to write upgraded file %s: %w", filePath, err)
	}

	return result, nil
}

// UpgradeDirectory processes all migration files in a directory.
func UpgradeDirectory(dir string, dryRun, makeBackup bool) ([]TransformationResult, error) {
	var results []TransformationResult

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range files {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".js") && !strings.HasSuffix(entry.Name(), ".ts")) {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		res, err := UpgradeFile(fullPath, dryRun, makeBackup)
		if err != nil {
			return nil, err
		}
		if res.ChangesMade > 0 {
			results = append(results, *res)
		}
	}

	return results, nil
}
