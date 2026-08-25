package api

import (
	"strings"
	"testing"
)

func TestSimpleGeneratorCompilesDefinitionAndSeeds(t *testing.T) {
	code, args, err := compileSimpleGenerator(SimpleGeneratorConfig{
		Count: 3, Seed: 41, SeedMode: "sequential",
		Fields: []SimpleGeneratorField{{Type: "number", Min: 1, Max: 9}, {Type: "array", Min: -2, Max: 2, LengthMin: 2, LengthMax: 4}},
	})
	if err != nil {
		t.Fatalf("compileSimpleGenerator: %v", err)
	}
	if code.Language != "cpp" || !strings.Contains(code.Code, "mt19937_64") {
		t.Fatalf("expected trusted C++ generator, got %+v", code)
	}
	if got := strings.Join(args, ","); got != "41,42,43" {
		t.Fatalf("unexpected seeds: %s", got)
	}
}

func TestSimpleGeneratorRejectsInvalidRanges(t *testing.T) {
	_, _, err := compileSimpleGenerator(SimpleGeneratorConfig{Count: 1, Fields: []SimpleGeneratorField{{Type: "number", Min: 9, Max: 1}}})
	if err == nil || !strings.Contains(err.Error(), "minimum cannot exceed") {
		t.Fatalf("expected range error, got %v", err)
	}
}

func TestSimpleGeneratorSupportsAllVisualFieldTypes(t *testing.T) {
	fields := []SimpleGeneratorField{
		{Type: "number", Min: 1, Max: 2}, {Type: "string", LengthMin: 1, LengthMax: 3},
		{Type: "array", Min: 1, Max: 2, LengthMin: 1, LengthMax: 3}, {Type: "matrix", Min: 1, Max: 2, Rows: 2, Columns: 2},
		{Type: "permutation", LengthMin: 2, LengthMax: 3}, {Type: "pair", Min: 1, Max: 2}, {Type: "interval", Min: 1, Max: 2},
		{Type: "repeat", Min: 1, Max: 2, LengthMin: 1, LengthMax: 3}, {Type: "tree", LengthMin: 2, LengthMax: 3},
		{Type: "graph", LengthMin: 2, LengthMax: 3, CountMin: 1, CountMax: 2},
	}
	if _, _, err := compileSimpleGenerator(SimpleGeneratorConfig{Count: 1, Fields: fields}); err != nil {
		t.Fatalf("all visual field types should compile to a generator: %v", err)
	}
}

func TestLegacyAdvancedGeneratorStaysCompatible(t *testing.T) {
	code, args, err := (GeneratorConfig{Code: "int main(){}", Language: "cpp"}).resolve()
	if err != nil || code.Code == "" || len(args) != 0 {
		t.Fatalf("legacy config must resolve: code=%+v args=%v err=%v", code, args, err)
	}
}
