package domain

import (
	"strings"
	"testing"
)

func TestDomainSchema(t *testing.T) {
	s := NewSchema()
	if s.Version != 1 || len(s.Tables) != 0 {
		t.Errorf("NewSchema initialized incorrectly: %+v", s)
	}

	diff := &Diff{
		TablesCreated: []Table{{Name: "Users"}},
		ColumnsAdded: []ColumnDiff{
			{TableName: "Users", Column: Column{Name: "bio"}},
		},
	}

	if diff.IsEmpty() {
		t.Errorf("expected diff not to be empty")
	}

	summary := diff.Summary()
	if !strings.Contains(summary, "Created tables (1): Users") {
		t.Errorf("expected Users in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "Added columns (1): Users.bio") {
		t.Errorf("expected Users.bio in summary, got: %s", summary)
	}

	emptyDiff := &Diff{}
	if !emptyDiff.IsEmpty() {
		t.Errorf("expected empty diff to return true for IsEmpty")
	}
	if emptyDiff.Summary() != "No schema changes detected." {
		t.Errorf("expected no changes summary, got: %s", emptyDiff.Summary())
	}
}
