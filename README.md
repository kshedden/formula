[![Build Status](https://travis-ci.com/kshedden/formula.svg?branch=master)](https://travis-ci.com/kshedden/formula)
[![Go Report Card](https://goreportcard.com/badge/github.com/kshedden/formula)](https://goreportcard.com/report/github.com/kshedden/formula)
[![codecov](https://codecov.io/gh/kshedden/formula/branch/master/graph/badge.svg)](https://codecov.io/gh/kshedden/formula)
[![GoDoc](https://godoc.org/github.com/kshedden/formula?status.png)](https://godoc.org/github.com/kshedden/formula)

formula : Formula parser for Go
===============================

__formula__ is a library for building data sets in Go using formulas.
The most common use-case for this library is to build design matrices
for use in statistical regression analysis.  The formulas in this
package behave much like formulas in R, Julia, Matlab, and Python
(using Patsy).  Interactions, algebraic expansion, and dummy-coding
are all supported.  Compared to these other formula packages, there
are a few simplifying differences:

* Only one-sided formulas are supported.  Multiple formulas can be
parsed together to produce a single dataset.  To produce a dataset for
a regression model, parse two formulas at once -- one formula for each
side of the regression relationship.

* Main effects are not automatically included for interactions.
Include them manually as desired.

* Functions (transformations) must be deterministic, not "stateful"

__Design:__ The data to be processed using formulas must be accesed
through a `DataSource`, which is a simple interface that allows slices
to be retrieved by name.  Parsing one or more formulas produces a
`ColSet`, which contains all the variables resulting from parsing the
formula(s).  A `ColSet` is interchangeable with a `statmodel.Dataset`,
so can be passed directly into that package for modeling.

__Modeling:__ This package does not fit any statistical models.  If
you want to fit a model to the dataset produced by the formula
package, you can use one of these packages:
[GLM](http://github.com/kshedden/statmodel/tree/master/glm),
[duration](http://github.com/kshedden/statmodel/tree/master/duration).

See
[here](https://github.com/kshedden/statmodel/blob/master/glm/examples/nhanes/nhanes.go)
for examples that use this package to produce datasets, and then use the
[statmodel](http://github.com/kshedden/statmodel) package to fit
models.
