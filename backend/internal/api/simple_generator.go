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
	Count      int                    `json:"count"`
	Seed       int64                  `json:"seed"`
	SeedMode   string                 `json:"seed_mode"` // fixed or sequential
	Strategies []string               `json:"strategies,omitempty"`
	Fields     []SimpleGeneratorField `json:"fields"`
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
	// Dependencies always point to an earlier named Number field.
	SizeFrom     string `json:"size_from,omitempty"`
	RowsFrom     string `json:"rows_from,omitempty"`
	ColumnsFrom  string `json:"columns_from,omitempty"`
	CountFrom    string `json:"count_from,omitempty"`
	VerticesFrom string `json:"vertices_from,omitempty"`
	EdgesFrom    string `json:"edges_from,omitempty"`
	Pattern      string `json:"pattern,omitempty"`
	Directed     bool   `json:"directed,omitempty"`
	Connected    bool   `json:"connected,omitempty"`
	Simple       bool   `json:"simple,omitempty"`
	NoSelfLoops  bool   `json:"no_self_loops,omitempty"`
	Weighted     bool   `json:"weighted,omitempty"`
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
	strategies := normalizeStrategies(cfg.Strategies)
	variables := make(map[string]string)
	var body strings.Builder
	for i, f := range cfg.Fields {
		fieldName := strings.ToLower(strings.TrimSpace(f.Name))
		if f.Type == "number" && fieldName != "" {
			if _, exists := variables[fieldName]; exists {
				return CodeConfig{}, nil, fmt.Errorf("number field %q is defined more than once", f.Name)
			}
		}
		line, err := simpleFieldCPP(f, i, variables)
		if err != nil {
			return CodeConfig{}, nil, err
		}
		body.WriteString(line)
		if f.Type == "number" && fieldName != "" {
			variables[fieldName] = fmt.Sprintf("v%d", i)
		}
	}
	code := `#include <algorithm>
#include <iostream>
#include <numeric>
#include <random>
#include <set>
#include <string>
#include <vector>
using namespace std;
long long pick(mt19937_64& rng, long long lo, long long hi) { return uniform_int_distribution<long long>(lo, hi)(rng); }
long long choose(mt19937_64& rng, long long lo, long long hi, const string& s) { if(s=="maximum") return hi; if(s=="boundary") return pick(rng,0,1)?lo:hi; if(s=="small") return pick(rng,lo,lo+(hi-lo)/10); return pick(rng,lo,hi); }
vector<long long> values(mt19937_64& rng, int n, long long lo, long long hi, string p, const string& s) { if(p=="auto") p=s=="adversarial"?"alternating":(s=="boundary"?"alternating":"random"); vector<long long> a(n); long long same=choose(rng,lo,hi,s); for(int i=0;i<n;i++) a[i]=(p=="all_equal"?same:(p=="alternating"?(i%2?hi:lo):(p=="many_duplicates"?pick(rng,lo,min(hi,lo+min(9LL,hi-lo))):choose(rng,lo,hi,s))); if(p=="sorted"||p=="reverse") { sort(a.begin(),a.end()); if(p=="reverse") reverse(a.begin(),a.end()); } return a; }
int main(int argc, char** argv) {
  if (argc < 2) return 2;
  mt19937_64 rng(stoull(argv[1]));
  string strategy = argc > 2 ? argv[2] : "random";
` + body.String() + "  return 0;\n}\n"
	args := make([]string, cfg.Count)
	for i := range args {
		seed := cfg.Seed
		if cfg.SeedMode != "fixed" {
			seed += int64(i)
		}
		args[i] = strconv.FormatInt(seed, 10) + " " + strategies[i%len(strategies)]
	}
	return CodeConfig{Code: code, Language: "cpp"}, args, nil
}

func normalizeStrategies(strategies []string) []string {
	allowed := map[string]bool{"small": true, "random": true, "boundary": true, "maximum": true, "adversarial": true}
	result := make([]string, 0, len(strategies))
	for _, strategy := range strategies {
		if allowed[strategy] {
			result = append(result, strategy)
		}
	}
	if len(result) == 0 {
		return []string{"random"}
	}
	return result
}

func simpleFieldCPP(f SimpleGeneratorField, index int, variables map[string]string) (string, error) {
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
	value := func() string { return fmt.Sprintf("choose(rng, %d, %d, strategy)", f.Min, f.Max) }
	depends := func(ref string, fallbackMin, fallbackMax int, label string) (string, error) {
		if ref == "" {
			return fmt.Sprintf("choose(rng, %d, %d, strategy)", fallbackMin, fallbackMax), nil
		}
		variable, ok := variables[strings.ToLower(strings.TrimSpace(ref))]
		if !ok {
			return "", fmt.Errorf("%s: %s must refer to an earlier named number field", name, label)
		}
		return variable, nil
	}
	switch f.Type {
	case "number":
		return fmt.Sprintf("  long long v%d = %s; cout << v%d << '\\n';\n", index, value(), index), nil
	case "string":
		alphabet := f.Alphabet
		if alphabet == "" {
			alphabet = "abcdefghijklmnopqrstuvwxyz"
		}
		size, err := depends(f.SizeFrom, lengthMin, lengthMax, "length from")
		if err != nil {
			return "", err
		}
		pattern := f.Pattern
		if pattern == "" {
			pattern = "auto"
		}
		return fmt.Sprintf("  { int n=%s; if(n>100000) return 3; string chars = %q; string p=%q; char same=chars[pick(rng,0,(long long)chars.size()-1)]; for (int i=0;i<n;i++) { if(p==\"all_equal\") cout<<same; else if(p==\"alternating\" || (p==\"auto\" && strategy==\"adversarial\")) cout<<chars[i%%2?(chars.size()-1):0]; else cout<<chars[pick(rng,0,(long long)chars.size()-1)]; } cout << '\\n'; }\n", size, alphabet, pattern), nil
	case "array", "repeat":
		ref := f.SizeFrom
		label := "size from"
		if f.Type == "repeat" {
			ref, label = f.CountFrom, "count from"
		}
		size, err := depends(ref, lengthMin, lengthMax, label)
		if err != nil {
			return "", err
		}
		pattern := f.Pattern
		if pattern == "" {
			pattern = "auto"
		}
		return fmt.Sprintf("  { int n=%s; if(n>100000) return 3; auto a=values(rng,n,%d,%d,%q,strategy); for(int i=0;i<n;i++){if(i)cout<<' ';cout<<a[i];} cout<<'\\n'; }\n", size, f.Min, f.Max, pattern), nil
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
		rExpr, err := depends(f.RowsFrom, rows, rows, "rows from")
		if err != nil {
			return "", err
		}
		cExpr, err := depends(f.ColumnsFrom, cols, cols, "columns from")
		if err != nil {
			return "", err
		}
		pattern := f.Pattern
		if pattern == "" {
			pattern = "auto"
		}
		return fmt.Sprintf("  { int rows=%s, cols=%s; if((long long)rows*cols>100000) return 3; auto a=values(rng,rows*cols,%d,%d,%q,strategy); for(int r=0;r<rows;r++){for(int c=0;c<cols;c++){if(c)cout<<' ';cout<<a[r*cols+c];}cout<<'\\n';} }\n", rExpr, cExpr, f.Min, f.Max, pattern), nil
	case "permutation":
		size, err := depends(f.SizeFrom, lengthMin, lengthMax, "size from")
		if err != nil {
			return "", err
		}
		pattern := f.Pattern
		return fmt.Sprintf("  { int n=%s; if(n>100000) return 3; vector<int>a(n); iota(a.begin(),a.end(),1); if(%q!=\"sorted\") shuffle(a.begin(),a.end(),rng); if(%q==\"reverse\") reverse(a.begin(),a.end()); for(int i=0;i<n;i++){if(i)cout<<' ';cout<<a[i];}cout<<'\\n';}\n", size, pattern, pattern), nil
	case "pair", "interval":
		if f.Type == "interval" {
			return fmt.Sprintf("  { long long a=%s,b=%s; if(a>b) swap(a,b); cout << a << ' ' << b << '\\n'; }\n", value(), value()), nil
		}
		return fmt.Sprintf("  cout << %s << ' ' << %s << '\\n';\n", value(), value()), nil
	case "tree":
		n, err := depends(f.VerticesFrom, lengthMin, lengthMax, "vertices from")
		if err != nil {
			return "", err
		}
		header := ""
		if f.VerticesFrom == "" {
			header = "cout<<n<<'\\n';"
		}
		return fmt.Sprintf("  { int n=%s; if(n>100000) return 3; %s for(int v=2;v<=n;v++) cout<<choose(rng,1,v-1,strategy)<<' '<<v<<'\\n'; }\n", n, header), nil
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
		n, err := depends(f.VerticesFrom, lengthMin, lengthMax, "vertices from")
		if err != nil {
			return "", err
		}
		m, err := depends(f.EdgesFrom, minE, maxE, "edges from")
		if err != nil {
			return "", err
		}
		header := ""
		if f.VerticesFrom == "" && f.EdgesFrom == "" {
			header = "cout<<n<<' '<<m<<'\\n';"
		}
		maxExpr := "(long long)n*(n-1)/2"
		if f.Directed {
			maxExpr = "(long long)n*(n-1)"
		}
		connected := "false"
		if f.Connected {
			connected = "true"
		}
		directed := "false"
		if f.Directed {
			directed = "true"
		}
		weighted := "false"
		if f.Weighted {
			weighted = "true"
		}
		return fmt.Sprintf("  { int n=%s; int m=min((long long)%s,%s); if(n>10000 || m>100000 || (%s && m<n-1)) return 3; %s vector<pair<int,int>> e; set<pair<int,int>> seen; for(int v=2;%s && v<=n && (int)e.size()<m;v++){pair<int,int> q={(int)choose(rng,1,v-1,strategy),v}; if(seen.insert(q).second)e.push_back(q);} for(int a=1;a<=n && (int)e.size()<m;a++) for(int b=1;b<=n && (int)e.size()<m;b++){if(a==b)continue;if(!%s && a>b)continue;pair<int,int> q={a,b};if(seen.insert(q).second)e.push_back(q);} shuffle(e.begin(),e.end(),rng); for(auto p:e){cout<<p.first<<' '<<p.second;if(%s)cout<<' '<<choose(rng,%d,%d,strategy);cout<<'\\n';} }\n", n, m, maxExpr, connected, header, connected, directed, weighted, f.Min, f.Max), nil
	default:
		return "", fmt.Errorf("%s: unsupported input field type", name)
	}
}
