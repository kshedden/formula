package formula

import (
	"fmt"
	"strings"
	"unicode"
)

// DataSource defines a dataset that will be processed through a formula.
type DataSource interface {

	// Names defines the names of the variables in the dataset.
	Names() []string

	// Get returns the data corresponding to one variable.  It should
	// only return []float64 or []string
	Get(string) interface{}
}

// Tokens that can appear in a formula.
type tokType int

// Allowed token types types.
const (
	vname = iota
	leftp
	rightp
	times
	plus
	icept
	funct
)

// Func is a transformation of a numeric column to a column set.
type Func func(string, []float64) *ColSet

// Operator precedence values; lower number is higher precedence.
var precedence = map[tokType]int{times: 0, plus: 1}

// The token is either a symbol (operator or parentheses), a variable
// name, or a function
type token struct {
	symbol tokType
	name   string // only used if symbol == vname

	// Below are only used for functions
	funcn string
	arg   string
}

// pop removes the last token from the slice, and returns it along
// with the shortened slice.  nil is returned if the slice has length
// zero.
func pop(tokens []*token) ([]*token, *token) {
	if len(tokens) == 0 {
		return nil, nil
	}
	n := len(tokens)
	tok := tokens[n-1]
	tokens = tokens[0 : n-1]
	return tokens, tok
}

// peek returns the last token from the slice.  nil is returned if the
// slice has length zero.
func peek(tokens []*token) *token {
	if len(tokens) == 0 {
		return nil
	}
	n := len(tokens)
	tok := tokens[n-1]
	return tok
}

// push appends the token to the end of the slice and returns the new slice.
func push(tokens []*token, tok *token) []*token {
	return append(tokens, tok)
}

// lex takes a formula and lexes it to obtain an array of tokens.
func lex(input string) ([]*token, error) {

	var tokens []*token
	rdr := strings.NewReader(input)

	isValidContinuation := func(r rune) bool {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return true
		}
		return false
	}

	for rdr.Len() > 0 {
		r, _, err := rdr.ReadRune()
		if err != nil {
			return nil, err
		}
		switch {
		case r == '(':
			tokens = append(tokens, &token{symbol: leftp})
		case r == ')':
			tokens = append(tokens, &token{symbol: rightp})
		case r == '+':
			tokens = append(tokens, &token{symbol: plus})
		case r == '*':
			tokens = append(tokens, &token{symbol: times})
		case r == '1':
			tokens = append(tokens, &token{symbol: icept})
		case r == ' ':
			// skip whitespace
		case unicode.IsLetter(r) || r == '_':
			name := []rune{r}
			for rdr.Len() > 0 {
				q, _, err := rdr.ReadRune()
				if err != nil {
					panic(err)
				}
				if !isValidContinuation(q) {
					_ = rdr.UnreadRune()
					break
				}
				name = append(name, q)
			}
			tokens = append(tokens, &token{symbol: vname, name: string(name)})
		default:
			return nil, fmt.Errorf("Invalid formula, symbol '%s' is not known.", string(r))
		}
	}

	tokens, err := lexFuncs(tokens)
	return tokens, err
}

func lexFuncs(input []*token) ([]*token, error) {

	output := make([]*token, 0, len(input))
	i := 0
	m := len(input)
	for i < m {
		if i+1 < m && input[i].symbol == vname && input[i+1].symbol == leftp {
			if i+3 < m && input[i+3].symbol == rightp {
				// A function
				name := fmt.Sprintf("%s(%s)", input[i].name, input[i+2].name)
				newtok := &token{symbol: funct, name: name, arg: input[i+2].name, funcn: input[i].name}
				output = append(output, newtok)
				i = i + 4
			} else {
				return nil, fmt.Errorf("Malformed function call")
			}
		} else {
			// Not a function
			output = append(output, input[i])
			i++
		}
	}

	return output, nil
}

// isOperator returns true if the token is an opertor (times or plus)
func isOperator(tok *token) bool {
	if tok.symbol == times || tok.symbol == plus {
		return true
	}
	return false
}

// parse converts the formula to RPN
// https://en.wikipedia.org/wiki/Shunting-yard_algorithm
func parse(input []*token) ([]*token, error) {

	var stack, output []*token
	var last *token

	for _, tok := range input {

		switch {
		case tok.symbol == vname || tok.symbol == funct || tok.symbol == icept:
			output = append(output, tok)
		case isOperator(tok):
			for {
				last := peek(stack)
				if last == nil || !isOperator(last) {
					break
				}
				if precedence[tok.symbol] > precedence[last.symbol] {
					stack, last = pop(stack)
					output = append(output, last)
				} else {
					break
				}
			}
			stack = push(stack, tok)
		case tok.symbol == leftp:
			stack = push(stack, tok)
		case tok.symbol == rightp:
			for {
				stack, last = pop(stack)
				if last == nil {
					return nil, fmt.Errorf("Unbalanced parentheses")
				}
				if last.symbol == leftp {
					break
				} else {
					output = append(output, last)
				}
			}
		}
	}

	for {
		stack, last = pop(stack)
		if last == nil {
			break
		}
		if last.symbol == leftp || last.symbol == rightp {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		output = append(output, last)
	}

	return output, nil
}

// Parser takes a formula and dataset, and produces a design
// matrix from them.
type Parser struct {

	// The formula defining the design matrix
	Formulas []string

	// Produces data in chunks
	RawData DataSource

	// Reference levels for string variables are omitted when
	// forming indicators
	refLevels map[string]string

	// Codes is a map from variable names to maps from variable
	// values to integer codes.  The distinct values of a
	// variable, excluding the reference level, are mapped to the
	// integers 0, 1, ...  Can be set manually, but if it is not
	// will be computed from data.  Not used if all variables are
	// of float64 type.
	codes map[string]map[string]int

	// Map from function name to function.
	funcs map[string]Func

	// The final data produced by parsing the formula
	data *ColSet

	ErrorState error

	// Intermediate data
	workData map[string]*ColSet

	facNames map[string][]string
	rpn      [][]*token // separate RPN for each formula
	rawNames []string
	names    []string
}

// New creates a Parser from a formula and a data stream.
func New(formula string, rawdata DataSource, config *Config) (*Parser, error) {

	fp := &Parser{
		Formulas: []string{formula},
		RawData:  rawdata,
	}

	if config != nil && config.Funcs != nil {
		fp.funcs = config.Funcs
	}

	if config != nil && config.RefLevels != nil {
		fp.refLevels = config.RefLevels
	}

	if err := fp.init(); err != nil {
		return nil, err
	}

	return fp, nil
}

// NewMulti accepts several formulas and includes all their parsed
// terms in the resulting data set.
func NewMulti(formulas []string, rawdata DataSource, config *Config) (*Parser, error) {

	fp := &Parser{
		Formulas: formulas,
		RawData:  rawdata,
	}

	if config != nil && config.Funcs != nil {
		fp.funcs = config.Funcs
	}

	if config != nil && config.RefLevels != nil {
		fp.refLevels = config.RefLevels
	}

	if err := fp.init(); err != nil {
		return nil, err
	}

	return fp, nil
}

// ColSet represents a design matrix.  It is an ordered set of named
// numeric data columns.
type ColSet struct {
	Names []string
	Data  [][]float64
}

// Extend a ColSet with the data of another ColSet.
func (c *ColSet) Extend(o *ColSet) {

	// Don't add duplicate terms (which may arise when parsing
	// multiple formulas together or when using Keep).
	mp := make(map[string]bool)
	for _, na := range c.Names {
		mp[na] = true
	}

	for j, na := range o.Names {
		if !mp[na] {
			c.Names = append(c.Names, na)
			c.Data = append(c.Data, o.Data[j])
		}
	}
}

type Config struct {
	RefLevels map[string]string
	Funcs     map[string]Func
}

// checkConv ensures that the variables with the given names have been
// converted from raw to ColSet form.
func (fp *Parser) checkConv(v ...string) error {
	for _, x := range v {
		if err := fp.convertColumn(x); err != nil {
			return err
		}
	}
	return nil
}

// setCodes inspects the data to determine integer codes for the
// distinct, non-reference levels of each categorical (string type)
// variable.
func (fp *Parser) setCodes() {

	fp.codes = make(map[string]map[string]int)
	fp.facNames = make(map[string][]string)

	for _, na := range fp.RawData.Names() {
		v := fp.RawData.Get(na)
		if v == nil {
			break
		}
		switch v := v.(type) {
		case []string:
			// Get the category codes for this
			// variable.  If this is the first
			// chunk, start from scratch.
			codes, ok := fp.codes[na]
			if !ok {
				codes = make(map[string]int)
				fp.codes[na] = codes
			}

			ref := fp.refLevels[na]
			for _, x := range v {
				if x == ref {
					continue
				}
				_, ok := codes[x]
				if !ok {
					// New code
					fm := fmt.Sprintf("%s[%s]", na, x)
					fp.facNames[na] = append(fp.facNames[na], fm)
					codes[x] = len(codes)
				}
			}
		}
	}
}

// codeStrings creates a ColSet from a string array, creating
// indicator variables for each distinct value in the string array,
// except for ref (the reference level).
func (fp *Parser) codeStrings(na, ref string, s []string) {

	// Get the category codes for this variable
	codes := fp.codes[na]

	var dat [][]float64
	for range codes {
		dat = append(dat, make([]float64, len(s)))
	}

	for i, x := range s {
		if x == ref {
			continue
		}
		c := codes[x]
		dat[c][i] = 1
	}

	fp.workData[na] = &ColSet{Names: fp.facNames[na], Data: dat}
}

// convertColumn converts the raw data column with the given name to a
// ColSet object.
func (fp *Parser) convertColumn(na string) error {

	// Only need to convert once
	_, ok := fp.workData[na]
	if ok {
		return nil
	}

	s := fp.RawData.Get(na)
	switch s := s.(type) {
	case nil:
		return fmt.Errorf("Variable '%s' not found.\n", na)
	case []string:
		ref := fp.refLevels[na]
		fp.codeStrings(na, ref, s)
	case []float64:
		fp.workData[na] = &ColSet{
			Names: []string{na},
			Data:  [][]float64{s},
		}
	default:
		return fmt.Errorf("unknown type %T for variable '%s' in convertColumn", s, na)
	}

	return nil
}

// doPlus creates a new ColSet by adding the columnsets named 'a' and
// 'b'.  Addition of two ColSet objects produces a new ColSet with
// columns comprising the union of the two arguments.
func (fp *Parser) doPlus(a, b string) *ColSet {

	ds1 := fp.workData[a]
	ds2 := fp.workData[b]

	var names []string
	var dat [][]float64

	names = append(names, ds1.Names...)
	names = append(names, ds2.Names...)
	dat = append(dat, ds1.Data...)
	dat = append(dat, ds2.Data...)

	return &ColSet{Names: names, Data: dat}
}

// doTimes creates a new ColSet by multiplying the columnsets named
// 'a' and 'b'.  Multiplication produces a new ColSet with columns
// comprising all pairwise product of the two arguments.
func (fp *Parser) doTimes(a, b string) *ColSet {

	ds1 := fp.workData[a]
	ds2 := fp.workData[b]

	var names []string
	var dat [][]float64

	for j1, na1 := range ds1.Names {
		for j2, na2 := range ds2.Names {
			d1 := ds1.Data[j1]
			d2 := ds2.Data[j2]
			x := make([]float64, len(d1))
			for i := range x {
				x[i] = d1[i] * d2[i]
			}
			names = append(names, na1+":"+na2)
			dat = append(dat, x)
		}
	}

	return &ColSet{names, dat}
}

// createIcept inserts an intercept (array of 1's) into the dataset
// being constructed and returns true if an intercept is not already
// included, otherwise returns false.
func (fp *Parser) createIcept() bool {

	if _, ok := fp.workData["icept"]; ok {
		return false
	}

	// Get the length of the data set.
	var nobs int
	{
		na0 := fp.RawData.Names()[0]
		x := fp.RawData.Get(na0)
		switch x := x.(type) {
		case []float64:
			nobs = len(x)
		case []string:
			nobs = len(x)
		default:
			panic("unknown type")
		}
	}

	x := make([]float64, nobs)
	for i := range x {
		x[i] = 1
	}
	fp.workData["icept"] = &ColSet{Names: []string{"icept"}, Data: [][]float64{x}}

	return true
}

// Names returns the names of the variables.
func (fp *Parser) Names() []string {
	return fp.names
}

func checkParens(fml string) bool {

	l, r := 0, 0
	for _, c := range fml {
		if c == '(' {
			l++
		} else if c == ')' {
			r++
		}
	}

	return l == r
}

// init performs lexing and parsing of the formula, only done once.
func (fp *Parser) init() error {

	for _, fml := range fp.Formulas {

		if !checkParens(fml) {
			return fmt.Errorf("Unbalanced parentheses in '%s'", fml)
		}

		fmx, err := lex(fml)
		if err != nil {
			return err
		}
		rpn, err := parse(fmx)
		if err != nil {
			return err
		}
		fp.rpn = append(fp.rpn, rpn)
	}

	if fp.codes == nil {
		fp.setCodes()
	}

	return nil
}

func (fp *Parser) doFormula(rpn []*token) error {

	if err := fp.runFuncs(rpn); err != nil {
		return err
	}

	// Special case a single variable with no operators
	if len(rpn) == 1 {
		na := rpn[0].name
		if err := fp.checkConv(na); err != nil {
			return err
		}
		fp.data.Extend(fp.workData[na])
		fp.workData = nil
		return nil
	}

	var stack []string

	for ix, tok := range rpn {
		last := ix == len(rpn)-1
		switch {
		case isOperator(tok):
			if len(stack) < 2 {
				return fmt.Errorf("not enough arguments")
			}

			// Pop the last two arguments off the stack
			arg2 := stack[len(stack)-1]
			arg1 := stack[len(stack)-2]
			stack = stack[0 : len(stack)-2]

			fp.checkConv(arg1, arg2)
			var rslt *ColSet
			switch tok.symbol {
			case plus:
				rslt = fp.doPlus(arg1, arg2)
			case times:
				rslt = fp.doTimes(arg1, arg2)
			default:
				return fmt.Errorf("Invalid symbol: %v", tok.symbol)
			}
			if last {
				// The last thing computed is the result
				fp.data.Extend(rslt)
			}
			nm := fmt.Sprintf("tmp%d", ix)
			fp.workData[nm] = rslt
			stack = append(stack, nm)
		case tok.symbol == icept:
			q := fp.createIcept()
			if q {
				stack = append(stack, "icept")
			}
		case tok.symbol == vname:
			fp.checkConv(tok.name)
			stack = append(stack, tok.name)
		case tok.symbol == funct:
			stack = append(stack, tok.name)
		}
	}

	if len(stack) != 1 {
		return fmt.Errorf("invalid formula")
	}

	return nil
}

func (fp *Parser) Parse() (*ColSet, error) {

	fp.data = new(ColSet)

	fp.rawNames = fp.RawData.Names()

	for _, rpn := range fp.rpn {
		fp.workData = make(map[string]*ColSet)
		if err := fp.doFormula(rpn); err != nil {
			return nil, err
		}
	}

	fp.workData = nil

	return fp.data, nil
}

func (fp *Parser) runFuncs(rpn []*token) error {

	for _, tok := range rpn {
		if tok.symbol != funct {
			continue
		}

		f, ok := fp.funcs[tok.funcn]
		if !ok {
			return fmt.Errorf("Function '%s' not found", tok.funcn)
		}
		x := fp.RawData.Get(tok.arg)
		switch x := x.(type) {
		case []float64:
			fp.workData[tok.name] = f(tok.name, x)
		default:
			panic("funtions can only be applied to numeric data")
		}
	}

	return nil
}

func find(s []string, x string) int {
	for i, v := range s {
		if v == x {
			return i
		}
	}
	return -1
}
