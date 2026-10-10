package doctor

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DeprecatedRule defines a pattern that is deprecated or removed in a newer Sequelize version.
type DeprecatedRule struct {
	Name        string
	Pattern     *regexp.Regexp
	Replacement string
	Severity    string // "BREAKING", "WARNING"
	Description string
}

// DiagnosticFinding represents a specific issue found in a migration or model file.
type DiagnosticFinding struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	MatchedText string `json:"matchedText"`
	RuleName    string `json:"ruleName"`
	Severity    string `json:"severity"`
	Suggestion  string `json:"suggestion"`
	Description string `json:"description"`
}

// AuditReport holds the full diagnostic results.
type AuditReport struct {
	TotalFilesScanned int                 `json:"totalFilesScanned"`
	TargetVersion     string              `json:"targetVersion"`
	BreakingCount     int                 `json:"breakingCount"`
	WarningCount      int                 `json:"warningCount"`
	Findings          []DiagnosticFinding `json:"findings"`
}

var v7Rules = []DeprecatedRule{
	{
		Name:        "NUMERIC_REMOVED",
		Pattern:     regexp.MustCompile(`Sequelize\.NUMERIC`),
		Replacement: "Sequelize.DECIMAL (or DataTypes.DECIMAL)",
		Severity:    "BREAKING",
		Description: "Sequelize.NUMERIC is removed in v7; use DECIMAL instead.",
	},
	{
		Name:        "JSONTYPE_REMOVED",
		Pattern:     regexp.MustCompile(`Sequelize\.JSONTYPE`),
		Replacement: "Sequelize.JSON (or DataTypes.JSON)",
		Severity:    "BREAKING",
		Description: "Sequelize.JSONTYPE is removed in v7; use JSON instead.",
	},
	{
		Name:        "DOUBLE_PRECISION_REMOVED",
		Pattern:     regexp.MustCompile(`Sequelize\.DOUBLE\s+PRECISION`),
		Replacement: "Sequelize.DOUBLE (or DataTypes.DOUBLE)",
		Severity:    "BREAKING",
		Description: "Sequelize.DOUBLE PRECISION is removed in v7; use DOUBLE instead.",
	},
	{
		Name:        "V6_DIRECT_SEQUELIZE_TYPE",
		Pattern:     regexp.MustCompile(`Sequelize\.(STRING|INTEGER|BIGINT|BOOLEAN|DATE|FLOAT|TEXT)`),
		Replacement: "DataTypes.$1",
		Severity:    "WARNING",
		Description: "In v7, accessing data types from DataTypes is strongly recommended over Sequelize root.",
	},
}

// AuditDirectory scans all JavaScript and TypeScript migration files for Sequelize version incompatibilities.
func AuditDirectory(dir string, targetVersion string) (*AuditReport, error) {
	report := &AuditReport{
		TargetVersion: targetVersion,
		Findings:      []DiagnosticFinding{},
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return nil, err
	}

	for _, entry := range files {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".js" && ext != ".ts" {
			continue
		}

		report.TotalFilesScanned++
		fullPath := filepath.Join(dir, entry.Name())
		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(contentBytes), "\n")
		for lineNum, line := range lines {
			for _, rule := range v7Rules {
				matches := rule.Pattern.FindAllString(line, -1)
				for _, match := range matches {
					finding := DiagnosticFinding{
						File:        entry.Name(),
						Line:        lineNum + 1,
						MatchedText: match,
						RuleName:    rule.Name,
						Severity:    rule.Severity,
						Suggestion:  rule.Replacement,
						Description: rule.Description,
					}
					report.Findings = append(report.Findings, finding)
					if rule.Severity == "BREAKING" {
						report.BreakingCount++
					} else {
						report.WarningCount++
					}
				}
			}
		}
	}

	return report, nil
}
