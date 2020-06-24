package formula

import (
	"fmt"
	"math"
	"testing"

	"gonum.org/v1/gonum/floats"
)

func tokEq(a, b []*token) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if *a[i] != *b[i] {
			return false
		}
	}

	return true
}

func colSetEq(a, b *ColSet) bool {

	if len(a.Names) != len(b.Names) {
		return false
	}

	for i := range a.Names {
		if a.Names[i] != b.Names[i] {
			return false
		}
	}

	if len(a.Data) != len(b.Data) {
		return false
	}

	eq := func(x, y float64) bool { return math.Abs(x-y) < 1e-5 }
	for i, x := range a.Data {
		if !floats.EqualFunc(x, b.Data[i], eq) {
			return false
		}
	}

	return true
}

func TestLexParse(t *testing.T) {

	v, err := lex("(A + b)*c + d*f(e)")
	if err != nil {
		t.Fail()
		return
	}
	exp := []*token{&token{symbol: leftp}, &token{name: "A"},
		&token{symbol: plus}, &token{name: "b"}, &token{symbol: rightp},
		&token{symbol: times}, &token{name: "c"}, &token{symbol: plus},
		&token{name: "d"}, &token{symbol: times},
		&token{symbol: funct, name: "f(e)", funcn: "f", arg: "e"}}

	if !tokEq(v, exp) {
		t.Fail()
	}

	b, err := parse(v)
	if err != nil {
		t.Fail()
		return
	}
	exp = []*token{&token{name: "A"}, &token{name: "b"},
		&token{symbol: plus}, &token{name: "c"}, &token{symbol: times},
		&token{name: "d"},
		&token{symbol: funct, name: "f(e)", funcn: "f", arg: "e"},
		&token{symbol: times}, &token{symbol: plus},
	}

	if !tokEq(b, exp) {
		t.Fail()
	}
}

// Create some functions
func makeFuncs() map[string]Func {
	funcs := make(map[string]Func)
	funcs["square"] = func(na string, x []float64) *ColSet {
		y := make([]float64, len(x))
		for i, v := range x {
			y[i] = v * v
		}
		return &ColSet{Names: []string{na}, Data: [][]float64{y}}
	}
	funcs["pbase"] = func(na string, x []float64) *ColSet {
		y := make([]float64, len(x))
		z := make([]float64, len(x))
		for i, v := range x {
			y[i] = v * v
			z[i] = v * v * v
		}
		return &ColSet{Names: []string{na + "^2", na + "^3"}, Data: [][]float64{y, z}}
	}
	return funcs
}

// A DataSource from a map
type mapAdapter struct {
	mp map[string]interface{}
}

func (ma *mapAdapter) Names() []string {
	var names []string
	for k, _ := range ma.mp {
		names = append(names, k)
	}
	return names
}

func (ma *mapAdapter) Get(na string) interface{} {
	return ma.mp[na]
}

func simpleData() DataSource {

	mp := map[string]interface{}{
		"x1": []float64{0, 1, 2, 3, 4},
		"x2": []string{"0", "0", "0", "1", "1"},
		"x3": []string{"a", "b", "a", "b", "a"},
		"x4": []float64{-1, 0, 1, 0, -1},
	}

	return &mapAdapter{mp}
}

func TestSingle(t *testing.T) {

	rawData := simpleData()
	funcs := makeFuncs()

	for ip, pr := range []struct {
		formula   string
		reflevels map[string]string
		expected  *ColSet
		funcs     map[string]Func
	}{
		{
			formula: "x1",
			expected: &ColSet{
				Names: []string{"x1"},
				Data: [][]float64{
					{0, 1, 2, 3, 4},
				},
			},
		},
		{
			formula:   "x1 + x2 + x1*x2",
			reflevels: map[string]string{"x2": "0"},
			expected: &ColSet{
				Names: []string{"x1", "x2[1]", "x1:x2[1]"},
				Data: [][]float64{
					{0, 1, 2, 3, 4},
					{0, 0, 0, 1, 1},
					{0, 0, 0, 3, 4},
				},
			},
		},
		{
			formula:   "x1 + x2 + x1*x2",
			reflevels: map[string]string{"x2": "1"},
			expected: &ColSet{
				Names: []string{"x1", "x2[0]", "x1:x2[0]"},
				Data: [][]float64{
					{0, 1, 2, 3, 4},
					{1, 1, 1, 0, 0},
					{0, 1, 2, 0, 0},
				},
			},
		},
		{
			formula: "x1",
			expected: &ColSet{
				Names: []string{"x1"},
				Data: [][]float64{
					{0, 1, 2, 3, 4},
				},
			},
		},
		{
			formula:   "( ( x2*x3))",
			reflevels: map[string]string{"x2": "0", "x3": "a"},
			expected: &ColSet{
				Names: []string{"x2[1]:x3[b]"},
				Data: [][]float64{
					{0, 0, 0, 1, 0},
				},
			},
		},
		{
			formula:   "(x1+x2)*(x3+x4)",
			reflevels: map[string]string{"x2": "0", "x3": "a"},
			expected: &ColSet{
				Names: []string{"x1:x3[b]", "x1:x4", "x2[1]:x3[b]", "x2[1]:x4"},
				Data: [][]float64{
					{0, 1, 0, 3, 0},
					{0, 0, 2, 0, -4},
					{0, 0, 0, 1, 0},
					{0, 0, 0, 0, -1},
				},
			},
		},
		{
			formula:   "x4 + (x1+x2)*x3",
			reflevels: map[string]string{"x2": "1", "x3": "a"},
			expected: &ColSet{
				Names: []string{"x4", "x1:x3[b]", "x2[0]:x3[b]"},
				Data: [][]float64{
					{-1, 0, 1, 0, -1},
					{0, 1, 0, 3, 0},
					{0, 1, 0, 0, 0},
				},
			},
		},
		{
			formula: "1 + x1",
			expected: &ColSet{
				Names: []string{"icept", "x1"},
				Data: [][]float64{
					{1, 1, 1, 1, 1},
					{0, 1, 2, 3, 4},
				},
			},
		},
		{
			formula: "x1 + 1",
			expected: &ColSet{
				Names: []string{"x1", "icept"},
				Data: [][]float64{
					{0, 1, 2, 3, 4},
					{1, 1, 1, 1, 1},
				},
			},
		},
		{
			formula: "square(x1) + 1",
			expected: &ColSet{
				Names: []string{"square(x1)", "icept"},
				Data: [][]float64{
					{0, 1, 4, 9, 16},
					{1, 1, 1, 1, 1},
				},
			},
		},
		{
			formula: "1 + pbase(x1)",
			expected: &ColSet{
				Names: []string{"icept", "pbase(x1)^2", "pbase(x1)^3"},
				Data: [][]float64{
					{1, 1, 1, 1, 1},
					{0, 1, 4, 9, 16},
					{0, 1, 8, 27, 64},
				},
			},
		},
		{
			formula: "1 + square(x1)",
			expected: &ColSet{
				Names: []string{"icept", "square(x1)"},
				Data: [][]float64{
					{1, 1, 1, 1, 1},
					{0, 1, 4, 9, 16},
				},
			},
		},
	} {
		fp, err := New(pr.formula, rawData, &Config{pr.reflevels, funcs})
		if err != nil {
			fmt.Printf("%+v\n", err)
			t.Fail()
		}
		cols, err := fp.Parse()
		if err != nil {
			fmt.Printf("%v\n", err)
			t.Fail()
		}

		if !colSetEq(pr.expected, cols) {
			fmt.Printf("Mismatch:\nip=%d\n", ip)
			fmt.Printf("Expected: %v\n", pr.expected)
			fmt.Printf("Observed: %v\n", cols)
			t.Fail()
		}

		if fp.ErrorState != nil {
			fmt.Printf("ip=%d %v\n", ip, fp.ErrorState)
			t.Fail()
		}
	}
}

func TestError(t *testing.T) {

	rawData := simpleData()
	funcs := makeFuncs()

	for _, pr := range []struct {
		formula    string
		parseError bool
		reflevels  map[string]string
		funcs      map[string]Func
	}{
		{
			formula: "x",
		},
		{
			formula: "ff(x1)",
		},
		{
			formula:    "f()",
			parseError: true,
		},
		{
			formula:    "f(",
			parseError: true,
		},
		{
			formula:    "f)(",
			parseError: true,
		},
	} {
		fp, err := New(pr.formula, rawData, &Config{pr.reflevels, funcs})
		if pr.parseError {
			if err == nil {
				t.Fail()
			} else {
				continue
			}
		}
		_, err = fp.Parse()
		if err == nil {
			t.Fail()
		}
	}
}

func TestMulti(t *testing.T) {

	rawData := simpleData()
	funcs := makeFuncs()

	for ip, pr := range []struct {
		formulas  []string
		reflevels map[string]string
		expected  *ColSet
		funcs     map[string]Func
	}{
		{
			formulas:  []string{"x1"},
			reflevels: nil,
			expected: &ColSet{
				Names: []string{"x1"},
				Data: [][]float64{
					{0, 1, 2, 3, 4},
				},
			},
		},
		{
			formulas:  []string{"x1", "x1+x2"},
			reflevels: map[string]string{"x2": "1"},
			expected: &ColSet{
				Names: []string{"x1", "x2[0]"},
				Data: [][]float64{
					{0, 1, 2, 3, 4},
					{1, 1, 1, 0, 0},
				},
			},
		},
		{
			formulas:  []string{"x1", "square(x1) + x2"},
			reflevels: map[string]string{"x2": "1"},
			expected: &ColSet{
				Names: []string{"x1", "square(x1)", "x2[0]"},
				Data: [][]float64{
					{0, 1, 2, 3, 4},
					{0, 1, 4, 9, 16},
					{1, 1, 1, 0, 0},
				},
			},
		},
	} {
		fp, err := NewMulti(pr.formulas, rawData, &Config{pr.reflevels, funcs})
		if err != nil {
			fmt.Printf("%v\n", err)
			t.Fail()
		}
		cols, err := fp.Parse()
		if err != nil {
			fmt.Printf("%+v\n", err)
			t.Fail()
		}

		if !colSetEq(pr.expected, cols) {
			fmt.Printf("Mismatch:\nip=%d\n", ip)
			fmt.Printf("Expected: %v\n", pr.expected)
			fmt.Printf("Observed: %v\n", cols)
			t.Fail()
		}
	}
}
