package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// asOnWindows is a file as a Windows editor such as Notepad may save it: a
// byte order mark in front, and every line ending in a carriage return and a
// line feed.
func asOnWindows(data []byte) []byte {
	return append([]byte("\xef\xbb\xbf"), bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))...)
}

// Whoever writes a problem by hand on Windows saves it with the line endings
// and, often, the byte order mark of Windows: the file must say the same as
// the one the examples were written as, problem and options alike.
func TestAProblemSavedOnWindowsReadsTheSame(t *testing.T) {
	t.Parallel()

	names, err := filepath.Glob(filepath.Join("..", "example", "*.*"))
	require.NoError(t, err)

	french, err := filepath.Glob(filepath.Join("..", "example", "fr", "*.yaml"))
	require.NoError(t, err)

	names = append(names, french...)
	require.GreaterOrEqual(t, len(names), 10, "the examples are where the test looks")

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(name)
			require.NoError(t, err)

			// Git on Windows may already have given the example the line
			// endings of Windows: start from those of Unix either way.
			data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))

			want, err := Parse(data)
			require.NoError(t, err)

			got, err := Parse(asOnWindows(data))
			require.NoError(t, err)
			require.Equal(t, want, got)

			wantP, wantO, err := want.Problem(string(want.From), "")
			require.NoError(t, err)

			gotP, gotO, err := got.Problem(string(got.From), "")
			require.NoError(t, err)
			require.Equal(t, wantP, gotP)
			require.Equal(t, wantO, gotO)
		})
	}
}

// A plan file opened and saved again on Windows must give back the same
// interviews.
func TestAPlanSavedOnWindowsReadsTheSame(t *testing.T) {
	t.Parallel()

	_, p, _ := smallProblem(t, "2026-11-03")

	for _, fm := range []Format{YAML, JSON} {
		var b bytes.Buffer
		require.NoError(t, WritePlan(&b, p, smallSchedule(), "2026-11-03", "2026-11-03", fm))

		want, err := ReadPlan(b.Bytes(), p, false)
		require.NoError(t, err)

		got, err := ReadPlan(asOnWindows(b.Bytes()), p, false)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
}
