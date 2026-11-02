package planning

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/FrederickFrance/igiveyouarendezvous/mip"
)

// absGap is the optimality tolerance given to the solver: every objective
// takes integer values, so a gap below 1 proves optimality.
const absGap = 0.999

// ErrUnsupported is returned by Solve for a problem beyond the exact model:
// a period of several days, gaps other than "back to back" or "no limit",
// or ranked requests. Search handles them.
var ErrUnsupported = errors.New("planning: the exact model only plans one day, with interviews back to back or with any wait, in any order")

// Solve plans one day, with an exact model (see ErrUnsupported).
//
// The objectives are optimised one after the other: each one is optimised,
// then frozen ("never do worse than this") before the next one is
// considered. The deadline of ctx, if any, is the total time limit, shared
// between the objectives; when it is reached, the best schedule found so far
// is returned and its scores are not proven.
//
// Requests that cannot all be held are listed in Schedule.Unscheduled;
// Schedule.Remaining gives the problem to plan in the next period.
func Solve(ctx context.Context, solver mip.Solver, p *Problem, o Options) (*Schedule, error) {
	if solver == nil {
		return nil, errors.New("planning: no solver")
	}

	in, err := newInstance(p, o)
	if err != nil {
		return nil, err
	}

	if !in.exact() {
		return nil, ErrUnsupported
	}

	f := in.formulate()

	values, proven, err := f.optimize(ctx, solver)
	if err != nil {
		return nil, err
	}

	s := f.schedule(values)
	if err := Check(p, o, s); err != nil {
		return nil, fmt.Errorf("planning: the solver returned an invalid schedule: %w", err)
	}

	if s.Scores, err = Evaluate(p, o, s); err != nil {
		return nil, err
	}

	for i := range s.Scores {
		s.Scores[i].Proven = proven[i]
	}

	return s, nil
}

// exact tells whether the exact model handles the instance.
func (in *instance) exact() bool {
	o := in.o
	if o.days() > 1 || !o.MinGap.IsZero() || !o.MaxGap.IsZero() && o.MaxGap != (Gap{Duration: NoLimit}) {
		return false
	}

	for _, r := range in.p.Requests {
		if r.Rank > 0 {
			return false
		}
	}

	return true
}

// optimize runs the lexicographic optimisation and returns the values of the
// variables, and for each objective whether its optimum was proven.
func (f *formulation) optimize(ctx context.Context, solver mip.Solver) ([]float64, []bool, error) {
	objectives := f.in.crit
	proven := make([]bool, len(objectives))
	deadline, limited := ctx.Deadline()

	var values []float64

	for level, obj := range objectives {
		terms := f.objective(obj)
		if len(terms) == 0 { // constant objective: nothing to optimise
			proven[level] = true

			continue
		}

		params := mip.Params{AbsGap: absGap, Start: values}

		if limited {
			left := time.Until(deadline)
			if left <= 0 || ctx.Err() != nil {
				break
			}

			params.TimeLimit = left / time.Duration(len(objectives)-level)
		} else if err := ctx.Err(); err != nil {
			break
		}

		f.m.SetObjective(terms)

		sol, err := solver.Solve(&f.m, params)
		if errors.Is(err, mip.ErrNoSolution) && values != nil {
			continue // keep the previous solution, which is still feasible
		}

		if err != nil {
			return nil, nil, fmt.Errorf("planning: objective %s: %w", obj.obj, err)
		}

		values, proven[level] = sol.Values, sol.Proven

		// Freeze the level. The value is an integer; the margin of 0.5 only
		// absorbs rounding errors.
		f.m.AddRow(-inf, math.Round(mip.Eval(terms, values))+0.5, terms...)
	}

	if values == nil {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("planning: %w", err)
		}

		// Only constant objectives: any schedule is optimal, the empty one too.
		values = make([]float64, len(f.m.Vars))
	}

	return values, proven, nil
}
