package api

import (
	"fmt"
	"strconv"
	"strings"
)

// SimpleGeneratorConfig is the code-free testcase definition stored with a problem.
// It is intentionally data only: the API translates it to a trusted C++ generator
// and the existing worker still compiles/runs it inside the Docker sandbox.
type SimpleGeneratorConfig struct {
	Count    int                    `json:"count"`
	Seed     int64                  `json:"seed"`
	SeedMode string                 `json:"seed_mode"` // fixed or sequential
	Fields   []SimpleGeneratorField `json:"fields"`
}

type SimpleGeneratorField struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Min       int    `json:"min"`
	Max       int    `json:"max"`
	LengthMin int    `json:"length_min"`
	LengthMax int    `json:"length_max"`
	Rows      int    `json:"rows"`
	Columns   int    `json:"columns"`
	CountMin  int    `json:"count_min"`
	CountMax  int    `json:"count_max"`
	Alphabet  string `json:"alphabet"`
}

// GeneratorConfig is versioned while retaining code/language at the top level so
// old JSONB configurations continue to deserialize and execute without migration.
type GeneratorConfig struct {
	Version  int                    `json:"version,omitempty"`
	Mode     string                 `json:"mode,omitempty"`
	Code     string                 `json:"code,omitempty"`
	Language string                 `json:"language,omitempty"`
	Advanced *CodeConfig            `json:"advanced,omitempty"`
	Simple   *SimpleGeneratorConfig `json:"simple,omitempty"`
}

func (g GeneratorConfig) advancedCode() CodeConfig {
	if g.Advanced != nil && g.Advanced.Code != "" {
		return *g.Advanced
	}
	return CodeConfig{Code: g.Code, Language: g.Language}
}

func (g GeneratorConfig) resolve() (CodeConfig, []string, error) {
	if g.Mode == "simple" {
		if g.Simple == nil {
			return CodeConfig{}, nil, fmt.Errorf("add at least one input field before generating test cases")
		}
		return compileSimpleGenerator(*g.Simple)
	}
	code := g.advancedCode()
	if code.Code == "" || code.Language == "" {
		return CodeConfig{}, nil, fmt.Errorf("advanced generator code and language are required")
	}
	return code, nil, nil
}

func compileSimpleGenerator(cfg SimpleGeneratorConfig) (CodeConfig, []string, error) {
	if cfg.Count < 1 || cfg.Count > 100 {
		return CodeConfig{}, nil, fmt.Errorf("choose between 1 and 100 test cases")
	}
	if len(cfg.Fields) == 0 {
		return CodeConfig{}, nil, fmt.Errorf("add at least one input field")
	}
	var body strings.Builder
	for i, f := range cfg.Fields {
		line, err := simpleFieldCPP(f, i)
		if err != nil {
			return CodeConfig{}, nil, err
		}
		body.WriteString(line)
	}
	code := `#include <algorithm>
#include <iostream>
#include <numeric>
#include <random>
#include <string>
#include <vector>
using namespace std;
long long pick(mt19937_64& rng, long long lo, long long hi) { return uniform_int_distribution<long long>(lo, hi)(rng); }
int main(int argc, char** argv) {
  if (argc < 2) return 2;
  mt19937_64 rng(stoull(argv[1]));
` + body.String() + "  return 0;\n}\n"
	args := make([]string, cfg.Count)
	for i := range args {
		seed := cfg.Seed
		if cfg.SeedMode != "fixed" {
			seed += int64(i)
		}
		args[i] = strconv.FormatInt(seed, 10)
	}
	return CodeConfig{Code: code, Language: "cpp"}, args, nil
}

func simpleFieldCPP(f SimpleGeneratorField, index int) (string, error) {
	name := strings.TrimSpace(f.Name)
	if name == "" {
		name = fmt.Sprintf("field %d", index+1)
	}
	if f.Min > f.Max {
		return "", fmt.Errorf("%s: minimum cannot exceed maximum", name)
	}
	if f.Min < -1000000000 || f.Max > 1000000000 {
		return "", fmt.Errorf("%s: values must stay between -1,000,000,000 and 1,000,000,000", name)
	}
	lengthMin, lengthMax := f.LengthMin, f.LengthMax
	if lengthMin > lengthMax {
		return "", fmt.Errorf("%s: minimum length cannot exceed maximum length", name)
	}
	if lengthMin < 1 {
		lengthMin = 1
	}
	if lengthMax < lengthMin {
		lengthMax = lengthMin
	}
	if lengthMax > 100000 {
		return "", fmt.Errorf("%s: length or node count must not exceed 100,000", name)
	}
	value := func() string { return fmt.Sprintf("pick(rng, %d, %d)", f.Min, f.Max) }
	switch f.Type {
	case "number":
		return "  cout << " + value() + " << '\\n';\n", nil
	case "string":
		alphabet := f.Alphabet
		if alphabet == "" {
			alphabet = "abcdefghijklmnopqrstuvwxyz"
		}
		return fmt.Sprintf("  { string chars = %q; int n = pick(rng, %d, %d); for (int i=0;i<n;i++) cout << chars[pick(rng,0,(long long)chars.size()-1)]; cout << '\\n'; }\n", alphabet, lengthMin, lengthMax), nil
	case "array", "repeat":
		return fmt.Sprintf("  { int n = pick(rng, %d, %d); for (int i=0;i<n;i++) { if(i) cout << ' '; cout << %s; } cout << '\\n'; }\n", lengthMin, lengthMax, value()), nil
	case "matrix":
		rows, cols := f.Rows, f.Columns
		if rows < 1 {
			rows = 1
		}
		if cols < 1 {
			cols = 1
		}
		if rows > 1000 || cols > 1000 || rows*cols > 100000 {
			return "", fmt.Errorf("%s: keep a grid to 100,000 values or fewer", name)
		}
		return fmt.Sprintf("  for (int r=0;r<%d;r++) { for (int c=0;c<%d;c++) { if(c) cout << ' '; cout << %s; } cout << '\\n'; }\n", rows, cols, value()), nil
	case "permutation":
		return fmt.Sprintf("  { int n = pick(rng, %d, %d); vector<int> a(n); iota(a.begin(),a.end(),1); shuffle(a.begin(),a.end(),rng); for(int i=0;i<n;i++) { if(i) cout << ' '; cout << a[i]; } cout << '\\n'; }\n", lengthMin, lengthMax), nil
	case "pair", "interval":
		if f.Type == "interval" {
			return fmt.Sprintf("  { long long a=%s,b=%s; if(a>b) swap(a,b); cout << a << ' ' << b << '\\n'; }\n", value(), value()), nil
		}
		return fmt.Sprintf("  cout << %s << ' ' << %s << '\\n';\n", value(), value()), nil
	case "tree":
		return fmt.Sprintf("  { int n=pick(rng,%d,%d); cout << n << '\\n'; for(int v=2;v<=n;v++) cout << pick(rng,1,v-1) << ' ' << v << '\\n'; }\n", lengthMin, lengthMax), nil
	case "graph":
		minE, maxE := f.CountMin, f.CountMax
		if minE < 0 {
			minE = 0
		}
		if maxE < minE {
			maxE = minE
		}
		if lengthMax > 10000 || maxE > 100000 {
			return "", fmt.Errorf("%s: graphs support up to 10,000 nodes and 100,000 edges", name)
		}
		return fmt.Sprintf("  { int n=pick(rng,%d,%d); int m=min((long long)pick(rng,%d,%d),(long long)n*(n-1)/2); cout << n << ' ' << m << '\\n'; vector<pair<int,int>> e; for(int a=1;a<=n && (int)e.size()<m;a++) for(int b=a+1;b<=n && (int)e.size()<m;b++) e.push_back({a,b}); shuffle(e.begin(),e.end(),rng); for(auto p:e) cout<<p.first<<' '<<p.second<<'\\n'; }\n", lengthMin, lengthMax, minE, maxE), nil
	default:
		return "", fmt.Errorf("%s: unsupported input field type", name)
	}
}
