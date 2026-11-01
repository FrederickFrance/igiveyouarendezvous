// Package gomilp solves mip models with github.com/daniel-sullivan/go-milp,
// a pure Go branch-and-bound solver (no cgo). It is convenient for small
// models; use planning.Search for larger ones.
//
// go-milp cannot start from a given solution: Params.Start is ignored.
package gomilp

import (
	"errors"
	"math"

	milp "github.com/daniel-sullivan/go-milp"

	"github.com/FrederickFrance/igiveyouarendezvous/mip"
)

// Solver is the go-milp backend.
type Solver struct{}

// New returns the go-milp backend.
func New() Solver {
	return Solver{}
}

// Name implements mip.Solver.
func (Solver) Name() string {
	return "go-milp"
}

// Solve implements mip.Solver.
func (Solver) Solve(m *mip.Model, p mip.Params) (*mip.Solution, error) {
	mdl := milp.NewModel()
	vars := make([]milp.Var, len(m.Vars))

	var objective []milp.Term

	for j, v := range m.Vars {
		kind := milp.Continuous

		switch {
		case v.Integer && v.Lower == 0 && v.Upper == 1:
			kind = milp.Binary
		case v.Integer:
			kind = milp.Integer
		}

		vars[j] = mdl.AddVar(v.Lower, v.Upper, kind)
		if v.Cost != 0 {
			objective = append(objective, milp.Term{Var: vars[j], Coef: v.Cost})
		}
	}

	if len(objective) == 0 { // go-milp requires an objective: use a constant one
		objective = []milp.Term{{Var: mdl.AddContinuous(0, 0), Coef: 1}}
	}

	mdl.SetObjective(milp.Minimize, objective)

	for _, r := range m.Rows {
		if len(r.Terms) == 0 {
			continue
		}

		terms := make([]milp.Term, len(r.Terms))
		for i, t := range r.Terms {
			terms[i] = milp.Term{Var: vars[t.Var], Coef: t.Coef}
		}

		if r.Lower == r.Upper {
			mdl.AddConstraint(terms, milp.EqualTo, r.Upper)

			continue
		}

		if !math.IsInf(r.Lower, -1) {
			mdl.AddConstraint(terms, milp.GreaterEq, r.Lower)
		}

		if !math.IsInf(r.Upper, 1) {
			mdl.AddConstraint(terms, milp.LessEq, r.Upper)
		}
	}

	opts := milp.DefaultOptions()
	opts.TimeLimit = p.TimeLimit
	opts.AbsoluteGap = p.AbsGap

	sol, err := mdl.SolveWith(opts)
	limited := errors.Is(err, milp.ErrTimeLimit) || errors.Is(err, milp.ErrNodeLimit)

	if err != nil && !limited {
		return nil, err
	}

	if sol == nil || sol.Values == nil || sol.Status != milp.Optimal {
		return nil, mip.ErrNoSolution
	}

	return &mip.Solution{
		Values: sol.Values[:len(m.Vars)],
		Proven: !limited,
	}, nil
}
