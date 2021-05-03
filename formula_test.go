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

func TestColSet(t *testing.T) {

	cs := ColSet{
		names: []string{"a", "b", "c"},
		data:  [][]float64{{1, 3}, {-1, 2}, {5, 6}},
	}

	b, err := cs.Get("b")
	if err != nil || floats.Sum(b) != 1 {
		t.Fail()
	}
}

func colSetEq(a, b *ColSet) bool {

	if len(a.names) != len(b.names) {
		return false
	}

	for i := range a.names {
		if a.names[i] != b.names[i] {
			return false
		}
	}

	if len(a.data) != len(b.data) {
		return false
	}

	eq := func(x, y float64) bool { return math.Abs(x-y) < 1e-5 }
	for i, x := range a.data {
		if !floats.EqualFunc(x, b.data[i], eq) {
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
	exp := []*token{
		{symbol: leftp}, {name: "A"},
		{symbol: plus}, {name: "b"},
		{symbol: rightp}, {symbol: times},
		{name: "c"}, {symbol: plus},
		{name: "d"}, {symbol: times},
		{symbol: funct, name: "f(e)", funcn: "f", arg: "e"},
	}

	if !tokEq(v, exp) {
		t.Fail()
	}

	b, err := parse(v)
	if err != nil {
		t.Fail()
		return
	}
	exp = []*token{
		{name: "A"}, {name: "b"},
		{symbol: plus}, {name: "c"},
		{symbol: times}, {name: "d"},
		{symbol: funct, name: "f(e)", funcn: "f", arg: "e"},
		{symbol: times}, {symbol: plus},
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
		return &ColSet{names: []string{na}, data: [][]float64{y}}
	}
	funcs["pbase"] = func(na string, x []float64) *ColSet {
		y := make([]float64, len(x))
		z := make([]float64, len(x))
		for i, v := range x {
			y[i] = v * v
			z[i] = v * v * v
		}
		return &ColSet{names: []string{na + "^2", na + "^3"}, data: [][]float64{y, z}}
	}
	return funcs
}

func simpleData() DataSource {

	names := []string{"x1", "x2", "x3", "x4"}
	data := []interface{}{
		[]float64{0, 1, 2, 3, 4},
		[]string{"0", "0", "0", "1", "1"},
		[]string{"a", "b", "a", "b", "a"},
		[]float64{-1, 0, 1, 0, -1},
	}

	return NewSource(data, names)
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
				names: []string{"x1"},
				data: [][]float64{
					{0, 1, 2, 3, 4},
				},
			},
		},
		{
			formula:   "x1 + x2 + x1*x2",
			reflevels: map[string]string{"x2": "0"},
			expected: &ColSet{
				names: []string{"x1", "x2[1]", "x1:x2[1]"},
				data: [][]float64{
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
				names: []string{"x1", "x2[0]", "x1:x2[0]"},
				data: [][]float64{
					{0, 1, 2, 3, 4},
					{1, 1, 1, 0, 0},
					{0, 1, 2, 0, 0},
				},
			},
		},
		{
			formula: "x1",
			expected: &ColSet{
				names: []string{"x1"},
				data: [][]float64{
					{0, 1, 2, 3, 4},
				},
			},
		},
		{
			formula:   "( ( x2*x3))",
			reflevels: map[string]string{"x2": "0", "x3": "a"},
			expected: &ColSet{
				names: []string{"x2[1]:x3[b]"},
				data: [][]float64{
					{0, 0, 0, 1, 0},
				},
			},
		},
		{
			formula:   "(x1+x2)*(x3+x4)",
			reflevels: map[string]string{"x2": "0", "x3": "a"},
			expected: &ColSet{
				names: []string{"x1:x3[b]", "x1:x4", "x2[1]:x3[b]", "x2[1]:x4"},
				data: [][]float64{
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
				names: []string{"x4", "x1:x3[b]", "x2[0]:x3[b]"},
				data: [][]float64{
					{-1, 0, 1, 0, -1},
					{0, 1, 0, 3, 0},
					{0, 1, 0, 0, 0},
				},
			},
		},
		{
			formula: "1 + x1",
			expected: &ColSet{
				names: []string{"icept", "x1"},
				data: [][]float64{
					{1, 1, 1, 1, 1},
					{0, 1, 2, 3, 4},
				},
			},
		},
		{
			formula: "x1 + 1",
			expected: &ColSet{
				names: []string{"x1", "icept"},
				data: [][]float64{
					{0, 1, 2, 3, 4},
					{1, 1, 1, 1, 1},
				},
			},
		},
		{
			formula: "square(x1) + 1",
			expected: &ColSet{
				names: []string{"square(x1)", "icept"},
				data: [][]float64{
					{0, 1, 4, 9, 16},
					{1, 1, 1, 1, 1},
				},
			},
		},
		{
			formula: "1 + pbase(x1)",
			expected: &ColSet{
				names: []string{"icept", "pbase(x1)^2", "pbase(x1)^3"},
				data: [][]float64{
					{1, 1, 1, 1, 1},
					{0, 1, 4, 9, 16},
					{0, 1, 8, 27, 64},
				},
			},
		},
		{
			formula: "1 + square(x1)",
			expected: &ColSet{
				names: []string{"icept", "square(x1)"},
				data: [][]float64{
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
				names: []string{"x1"},
				data: [][]float64{
					{0, 1, 2, 3, 4},
				},
			},
		},
		{
			formulas:  []string{"x1", "x1+x2"},
			reflevels: map[string]string{"x2": "1"},
			expected: &ColSet{
				names: []string{"x1", "x2[0]"},
				data: [][]float64{
					{0, 1, 2, 3, 4},
					{1, 1, 1, 0, 0},
				},
			},
		},
		{
			formulas:  []string{"x1", "square(x1) + x2"},
			reflevels: map[string]string{"x2": "1"},
			expected: &ColSet{
				names: []string{"x1", "square(x1)", "x2[0]"},
				data: [][]float64{
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
