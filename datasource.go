package formula

// DataSource defines a dataset that will be processed through a formula.
type DataSource interface {

	// Names defines the names of the variables in the dataset.
	Names() []string

	// Get returns the data corresponding to one variable.  It should
	// only return []float64 or []string
	Get(string) interface{}
}

type basicSource struct {
	names []string
	colix map[string]int
	data  []interface{}
}

// Names returns a slice containing all the names of variables in the
// source.
func (b *basicSource) Names() []string {
	return b.names
}

// Get returns the data corresponding to a given variable name.
func (b *basicSource) Get(col string) interface{} {
	ix, ok := b.colix[col]
	if !ok {
		return nil
	}
	return b.data[ix]
}

// NewSource returns a DataSource for the given variables
// and data values.
func NewSource(data []interface{}, names []string) DataSource {
	colix := make(map[string]int)
	for k, c := range names {
		colix[c] = k
	}
	return &basicSource{
		names: names,
		colix: colix,
		data:  data,
	}
}
