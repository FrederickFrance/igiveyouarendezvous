package gomilp_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/mip"
	"github.com/FrederickFrance/igiveyouarendezvous/mip/gomilp"
)

func TestSolve(t *testing.T) {
	t.Parallel()

	// minimise -x - 2y with x + y <= 3, x - y == 1, x integer in [0, 5],
	// y continuous in [0, 10]: x = 2, y = 1.
	m := &mip.Model{}
	x := m.AddVar(0, 5, true)
	y := m.AddVar(0, 10, false)
	m.AddRow(math.Inf(-1), 3, mip.Term{Var: x, Coef: 1}, mip.Term{Var: y, Coef: 1})
	m.AddRow(1, 1, mip.Term{Var: x, Coef: 1}, mip.Term{Var: y, Coef: -1})
	m.AddRow(math.Inf(-1), math.Inf(1)) // empty rows are ignored
	m.SetObjective([]mip.Term{{Var: x, Coef: -1}, {Var: y, Coef: -2}})

	sol, err := gomilp.New().Solve(m, mip.Params{AbsGap: 1e-6})
	require.NoError(t, err)
	require.True(t, sol.Proven)
	require.InDelta(t, 2, sol.Values[x], 1e-6)
	require.InDelta(t, 1, sol.Values[y], 1e-6)
	require.Equal(t, "go-milp", gomilp.New().Name())
}

func TestNoObjective(t *testing.T) {
	t.Parallel()

	m := &mip.Model{}
	x := m.AddBinary()
	m.AddRow(1, math.Inf(1), mip.Term{Var: x, Coef: 1})

	sol, err := gomilp.New().Solve(m, mip.Params{})
	require.NoError(t, err)
	require.InDelta(t, 1, sol.Values[x], 1e-6)
}

func TestInfeasible(t *testing.T) {
	t.Parallel()

	m := &mip.Model{}
	x := m.AddBinary()
	m.AddRow(2, math.Inf(1), mip.Term{Var: x, Coef: 1})
	m.SetObjective([]mip.Term{{Var: x, Coef: 1}})

	_, err := gomilp.New().Solve(m, mip.Params{})
	require.ErrorIs(t, err, mip.ErrNoSolution)
}
