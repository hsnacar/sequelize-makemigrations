package repair

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MigrationIssue represents a detected problem in a migration file.
type MigrationIssue struct {
	File        string `json:"file"`
	Issue       string `json:"issue"`
	Severity    string `json:"severity"` // "ERROR", "WARNING"
	CanAutoFix  bool   `json:"canAutoFix"`
}

var (
	hasUpRegex   = regexp.MustCompile(`(?i)(async\s+up\s*\(|\bup\s*\(|up\s*:\s*(async\s*)?function|up\s*:\s*(async\s*)?\(|up\s*:\s*async\s+[a-zA-Z0-9_]+\s*=>)`)
	hasDownRegex = regexp.MustCompile(`(?i)(async\s+down\s*\(|\bdown\s*\(|down\s*:\s*(async\s*)?function|down\s*:\s*(async\s*)?\(|down\s*:\s*async\s+[a-zA-Z0-9_]+\s*=>)`)
	reOuterClosing = regexp.MustCompile(`(?m)^\s*\};`)
)

// AuditMigrations inspects all migration files in the directory for structural and safety issues.
func AuditMigrations(migrationsDir string) ([]MigrationIssue, error) {
	var issues []MigrationIssue

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No migrations dir yet
		}
		return nil, err
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".js") {
			continue
		}

		fullPath := filepath.Join(migrationsDir, f.Name())
		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		content := string(contentBytes)

		// 1. Check for missing 'up' handler
		if !hasUpRegex.MatchString(content) {
			issues = append(issues, MigrationIssue{
				File:       f.Name(),
				Issue:      "Missing 'up' function. Migration cannot be applied.",
				Severity:   "ERROR",
				CanAutoFix: false,
			})
		}

		// 2. Check for missing 'down' handler (irreversible migration)
		if !hasDownRegex.MatchString(content) {
			issues = append(issues, MigrationIssue{
				File:       f.Name(),
				Issue:      "Missing 'down' rollback function. Migration cannot be safely reverted.",
				Severity:   "WARNING",
				CanAutoFix: true,
			})
		}
	}

	return issues, nil
}

// RepairMissingDown adds a stub down handler if missing.
func RepairMissingDown(filePath string) error {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(contentBytes)
	if hasDownRegex.MatchString(content) {
		return nil // Already has down
	}

	// Insert stub down before closing exports
	stub := `
  async down(queryInterface, Sequelize) {
    // TODO: Add rollback operations
  }
};`

	matches := reOuterClosing.FindAllStringIndex(content, -1)
	var insertPos int
	var endPos int
	if len(matches) > 0 {
		lastMatch := matches[len(matches)-1]
		insertPos = lastMatch[0]
		endPos = lastMatch[1]
	} else {
		lastClosingIndex := strings.LastIndex(content, "};")
		if lastClosingIndex == -1 {
			return fmt.Errorf("could not find closing '};' in %s", filePath)
		}
		insertPos = lastClosingIndex
		endPos = lastClosingIndex + 2
	}

	repaired := content[:insertPos] + stub + content[endPos:]
	return os.WriteFile(filePath, []byte(repaired), 0644)
}
