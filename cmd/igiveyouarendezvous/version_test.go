package main

import (
	"bytes"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// heading matches the heading of a version in CHANGELOG.md, such as
//
//	## [0.11.0] - 2026-11-14 "Jack Burton" 🚚
var heading = regexp.MustCompile(`(?m)^## \[(\d+\.\d+\.\d+)\] - \S+ "([^"]+)" (\S+)$`)

// TestVersionMatchesTheChangelog is the whole point of writing the version
// in the code: without it, the demo would keep showing the version of the
// day somebody last thought about it.
func TestVersionMatchesTheChangelog(t *testing.T) {
	t.Parallel()

	text, err := os.ReadFile("../../CHANGELOG.md")
	require.NoError(t, err)

	found := heading.FindSubmatch(text)
	require.NotNil(t, found, "no version heading in CHANGELOG.md: the format is `## [1.2.3] - date \"Name\" mark`")

	require.Equal(t, version, string(found[1]), "version: the code and the first section of CHANGELOG.md disagree")
	require.Equal(t, releaseName, string(found[2]), "code name: the code and the first section of CHANGELOG.md disagree")
	require.Equal(t, releaseMark, string(found[3]), "mark: the code and the first section of CHANGELOG.md disagree")
}

// TestVersionFlagPrintsTheRelease checks what the demo shows in its footer
// and what -version prints: the same line, from the same constants.
func TestVersionFlagPrintsTheRelease(t *testing.T) {
	t.Parallel()

	require.Equal(t, `v`+version+` "`+releaseName+`" `+releaseMark, release())

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-version"}, nil, &out, &out))
	require.Equal(t, release()+"\n", out.String())
}
