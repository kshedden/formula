[![Build Status](https://travis-ci.com/kshedden/formula.svg?branch=master)](https://travis-ci.com/kshedden/formula)
[![Go Report Card](https://goreportcard.com/badge/github.com/kshedden/formula)](https://goreportcard.com/report/github.com/kshedden/formula)
[![codecov](https://codecov.io/gh/kshedden/formula/branch/master/graph/badge.svg)](https://codecov.io/gh/kshedden/formula)
[![GoDoc](https://godoc.org/github.com/kshedden/formula?status.png)](https://godoc.org/github.com/kshedden/formula)

formula : Formula parser for Go
===============================

__formula__ is a library for building data sets in Go using formulas.  The most
common use-case for this library is to build a design matrix for use in
a statistical regression analysis.
The formulas in this package behave much like formulas in R, Julia, Matlab, and Python (using Patsy).
Interactions, algebraic expansion, and dummy-coding are all supported.  Compared
to these other formula packages, there are a few simplifying differences:

* Only one-sided formulas are supported.  Multiple formulas can be parsed
together to produce a single dataset.  To produce a dataset for a regression
model, parse two formulas at once -- one formula for each side of the regression
relationship.

* Main effects are not automatically included for interactions.

* Functions (transformations) must be deterministic, not "stateful"

This package does not fit any statistical models.  If you want to fit a model
to the dataset produced by the formula package, you can use one of
these packages: [GLM](http://github.com/kshedden/statmodel/glm),
[duration](http://github.com/kshedden/statmodel/duration).

See [here](https://github.com/kshedden/statmodel/blob/master/glm/examples/nhanes/nhanes.go)
for examples using this package produce datasets, and the [statmodel](http://github.com/kshedden/statmodel)
package to fit models.
