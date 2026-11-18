//go:build !(js && wasm)

package main

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

// The ceiling the command gives the garbage collector is its own business,
// but what the person asked for is theirs: a GOGC or a GOMEMLIMIT of their
// own must come out the other side untouched, since they know what else is
// running on the machine and this program does not.
//
// Not parallel: t.Setenv forbids it, and what it changes belongs to the
// whole process.
func TestTuneMemoryLeavesWhatThePersonAskedFor(t *testing.T) {
	for _, env := range []string{"GOGC", "GOMEMLIMIT"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv(env, "123")

			before, beforePercent := debug.SetMemoryLimit(-1), debug.SetGCPercent(-2)
			debug.SetGCPercent(beforePercent)

			tuneMemory()

			after, afterPercent := debug.SetMemoryLimit(-1), debug.SetGCPercent(-2)
			debug.SetGCPercent(afterPercent)

			require.Equal(t, before, after, "the memory limit was changed")
			require.Equal(t, beforePercent, afterPercent, "the collection target was changed")
		})
	}

	// With nothing asked for, the ceiling is the one this command chose.
	t.Run("nothing asked for", func(t *testing.T) {
		t.Setenv("GOGC", "")
		t.Setenv("GOMEMLIMIT", "")

		was, wasPercent := debug.SetMemoryLimit(-1), debug.SetGCPercent(-2)
		debug.SetGCPercent(wasPercent)

		t.Cleanup(func() {
			debug.SetMemoryLimit(was)
			debug.SetGCPercent(wasPercent)
		})

		tuneMemory()

		limit := debug.SetMemoryLimit(-1)
		percent := debug.SetGCPercent(-2)
		debug.SetGCPercent(percent)

		require.Equal(t, int64(softLimit), limit)
		require.Equal(t, gcPercent, percent)
	})
}
