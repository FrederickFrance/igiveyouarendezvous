package mip_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/mip"
	"github.com/FrederickFrance/igiveyouarendezvous/mip/gomilp"
)

func TestCompact(t *testing.T) {
	t.Parallel()

	got := mip.Compact([]mip.Term{{Var: 2, Coef: 1}, {Var: 0, Coef: 3}, {Var: 2, Coef: -1}, {Var: 0, Coef: 1}, {Var: 1, Coef: 0}})
	require.Equal(t, []mip.Term{{Var: 0, Coef: 4}}, got)
}

// knapsack: maximise 5a + 4b + 3c with 2a + 3b + c <= 4, a, b, c binary,
// and d continuous in [0, 1.5] with a + d <= 2 (optimum: a = c = 1, value 8).
func knapsack() *mip.Model {
	m := &mip.Model{}
	a, b, c := m.AddBinary(), m.AddBinary(), m.AddBinary()
	d := m.AddVar(0, 1.5, false)
	m.AddRow(math.Inf(-1), 4, mip.Term{Var: a, Coef: 2}, mip.Term{Var: b, Coef: 3}, mip.Term{Var: c, Coef: 1})
	m.AddRow(math.Inf(-1), 2, mip.Term{Var: a, Coef: 1}, mip.Term{Var: d, Coef: 1})
	m.SetObjective([]mip.Term{{Var: a, Coef: -5}, {Var: b, Coef: -4}, {Var: c, Coef: -3}})

	return m
}

func TestSolvers(t *testing.T) {
	t.Parallel()

	solvers := []mip.Solver{gomilp.New()}

	for _, solver := range solvers {
		t.Run(solver.Name(), func(t *testing.T) {
			t.Parallel()

			m := knapsack()
			start := []float64{0, 1, 1, 0} // feasible, value 7

			sol, err := solver.Solve(m, mip.Params{AbsGap: 1e-6, Start: start})
			require.NoError(t, err)
			require.True(t, sol.Proven)
			require.InDelta(t, -8, mip.Eval([]mip.Term{{Var: 0, Coef: -5}, {Var: 1, Coef: -4}, {Var: 2, Coef: -3}}, sol.Values), 1e-6)
		})
	}
}
