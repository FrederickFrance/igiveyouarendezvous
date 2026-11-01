// Package mip describes mixed-integer linear programs independently of any
// solver, and the interface that a solver backend implements.
//
// One backend is provided: mip/gomilp (pure Go). Other solvers can be
// plugged in by implementing Solver.
package mip

import (
	"cmp"
	"errors"
	"slices"
	"time"
)

// ErrNoSolution is returned by a Solver that found no feasible solution
// within its limits.
var ErrNoSolution = errors.New("mip: no feasible solution found")

// Var is a decision variable. Use math.Inf for an absent bound.
type Var struct {
	Lower, Upper float64
	Integer      bool
	Cost         float64 // coefficient in the objective, which is minimised
}

// Term is one coefficient * variable of a linear expression.
type Term struct {
	Var  int
	Coef float64
}

// Row is the linear constraint Lower <= sum(Terms) <= Upper. Use math.Inf
// for an absent bound.
type Row struct {
	Lower, Upper float64
	Terms        []Term
}

// Model is a mixed-integer linear program: minimise sum(Cost * x) subject to
// the rows and the variable bounds.
type Model struct {
	Vars []Var
	Rows []Row
}

// AddVar adds a variable and returns its index.
func (m *Model) AddVar(lower, upper float64, integer bool) int {
	m.Vars = append(m.Vars, Var{Lower: lower, Upper: upper, Integer: integer})

	return len(m.Vars) - 1
}

// AddBinary adds a 0/1 variable and returns its index.
func (m *Model) AddBinary() int {
	return m.AddVar(0, 1, true)
}

// AddRow adds the constraint lower <= sum(terms) <= upper. The terms are
// copied, duplicate variables merged.
func (m *Model) AddRow(lower, upper float64, terms ...Term) {
	m.Rows = append(m.Rows, Row{Lower: lower, Upper: upper, Terms: Compact(terms)})
}

// SetObjective replaces the objective by sum(terms).
func (m *Model) SetObjective(terms []Term) {
	for j := range m.Vars {
		m.Vars[j].Cost = 0
	}

	for _, t := range terms {
		m.Vars[t.Var].Cost += t.Coef
	}
}

// Compact returns a copy of terms sorted by variable, with the duplicate
// variables merged and the zero coefficients removed.
func Compact(terms []Term) []Term {
	out := slices.Clone(terms)
	slices.SortFunc(out, func(a, b Term) int { return cmp.Compare(a.Var, b.Var) })

	n := 0
	for _, t := range out {
		if n > 0 && out[n-1].Var == t.Var {
			out[n-1].Coef += t.Coef

			continue
		}

		out[n] = t
		n++
	}

	return slices.DeleteFunc(out[:n], func(t Term) bool { return t.Coef == 0 })
}

// Eval returns the value of sum(terms) for the given variable values.
func Eval(terms []Term, values []float64) float64 {
	total := 0.0
	for _, t := range terms {
		total += t.Coef * values[t.Var]
	}

	return total
}

// Params tunes one call to Solver.Solve.
type Params struct {
	// TimeLimit stops the search; zero means no limit. The best solution
	// found so far is then returned, with Proven set to false.
	TimeLimit time.Duration
	// AbsGap stops the search as soon as the best solution is proven to be
	// within AbsGap of the optimum.
	AbsGap float64
	// Start is an optional feasible solution to start from. Backends that
	// cannot use it ignore it.
	Start []float64
}

// Solution is the result of Solver.Solve.
type Solution struct {
	Values []float64 // one value per variable
	Proven bool      // optimality proven, within Params.AbsGap
}

// Solver solves a Model. It returns ErrNoSolution when no feasible solution
// was found.
type Solver interface {
	Name() string
	Solve(m *Model, p Params) (*Solution, error)
}
