package extractor

import (
	"context"
	"os"
	"testing"
)

func TestNodeExtractor(t *testing.T) {
	ext := NewNodeExtractor("models/index.js")
	if ext == nil {
		t.Fatal("expected NewNodeExtractor to return non-nil")
	}

	t.Run("Fails gracefully when models cannot be found", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "seq-ext-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		_, err = ext.ExtractSchema(context.Background(), tmpDir)
		if err == nil {
			t.Errorf("expected error when extracting from empty directory")
		}
	})
}
