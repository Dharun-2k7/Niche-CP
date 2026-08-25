package api

import (
	"testing"
)

func TestCompileSimpleGenerator_Structures(t *testing.T) {
	tests := []struct {
		name    string
		config  SimpleGeneratorConfig
		wantErr bool
	}{
		{
			name: "n + array of n",
			config: SimpleGeneratorConfig{
				Count:    3,
				Seed:     42,
				SeedMode: "sequential",
				Fields: []SimpleGeneratorField{
					{Type: "number", Name: "n", Min: 1, Max: 100},
					{Type: "array", Name: "a", SizeFrom: "n", Min: -1000, Max: 1000},
				},
			},
			wantErr: false,
		},
		{
			name: "n m + m graph edges",
			config: SimpleGeneratorConfig{
				Count:    3,
				Seed:     42,
				SeedMode: "sequential",
				Fields: []SimpleGeneratorField{
					{Type: "number", Name: "n", Min: 2, Max: 50},
					{Type: "number", Name: "m", Min: 1, Max: 100},
					{Type: "graph", Name: "g", VerticesFrom: "n", EdgesFrom: "m", Connected: true, Simple: true, NoSelfLoops: true},
				},
			},
			wantErr: false,
		},
		{
			name: "n x m matrix",
			config: SimpleGeneratorConfig{
				Count:    3,
				Seed:     42,
				SeedMode: "sequential",
				Fields: []SimpleGeneratorField{
					{Type: "number", Name: "n", Min: 1, Max: 10},
					{Type: "number", Name: "m", Min: 1, Max: 10},
					{Type: "matrix", Name: "mat", RowsFrom: "n", ColumnsFrom: "m", Min: 0, Max: 1},
				},
			},
			wantErr: false,
		},
		{
			name: "permutation of n",
			config: SimpleGeneratorConfig{
				Count:    3,
				Seed:     42,
				SeedMode: "sequential",
				Fields: []SimpleGeneratorField{
					{Type: "number", Name: "n", Min: 1, Max: 100},
					{Type: "permutation", Name: "p", SizeFrom: "n"},
				},
			},
			wantErr: false,
		},
		{
			name: "tree with n vertices",
			config: SimpleGeneratorConfig{
				Count:    3,
				Seed:     42,
				SeedMode: "sequential",
				Fields: []SimpleGeneratorField{
					{Type: "number", Name: "n", Min: 2, Max: 100},
					{Type: "tree", Name: "t", VerticesFrom: "n"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codeConfig, args, err := compileSimpleGenerator(tt.config)
			if (err != nil) != tt.wantErr {
				t.Fatalf("compileSimpleGenerator() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if codeConfig.Code == "" || codeConfig.Language != "cpp" {
					t.Errorf("Expected non-empty C++ code, got lang=%s, code len=%d", codeConfig.Language, len(codeConfig.Code))
				}
				if len(args) != tt.config.Count {
					t.Errorf("Expected %d arg lines, got %d", tt.config.Count, len(args))
				}
			}
		})
	}
}
