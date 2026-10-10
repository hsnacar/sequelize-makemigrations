package generator

import (
	"testing"
)

func TestGetGenerator(t *testing.T) {
	t.Run("Returns v6 generator by default and for v6", func(t *testing.T) {
		genDefault := GetGenerator("")
		if genDefault == nil {
			t.Fatal("expected generator for default")
		}

		genV6 := GetGenerator("v6")
		if genV6 == nil {
			t.Fatal("expected generator for v6")
		}
	})

	t.Run("Returns v7 generator for v7 and 7", func(t *testing.T) {
		genV7 := GetGenerator("v7")
		if genV7 == nil {
			t.Fatal("expected generator for v7")
		}

		gen7 := GetGenerator("7")
		if gen7 == nil {
			t.Fatal("expected generator for 7")
		}
	})
}
