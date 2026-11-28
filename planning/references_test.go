package planning

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	listItem     = regexp.MustCompile(`^\s*// {3}- (.*)$`)
	listMore     = regexp.MustCompile(`^\s*// {5}(.*)$`)
	markdownItem = regexp.MustCompile(`^- (.*)$`)
	markdownMore = regexp.MustCompile(`^ {2}(.*)$`)
)

// references gathers the entries of the reference lists of a text: the
// items holding a DOI, each joined into one line, emphasis dropped.
func references(text string, item, more *regexp.Regexp) []string {
	var (
		entries []string
		entry   []string
	)

	flush := func() {
		if joined := strings.Join(entry, " "); strings.Contains(joined, "https://doi.org/") {
			entries = append(entries, strings.ReplaceAll(joined, "*", ""))
		}

		entry = nil
	}

	for line := range strings.SplitSeq(text, "\n") {
		switch {
		case item.MatchString(line):
			flush()
			entry = append(entry, item.FindStringSubmatch(line)[1])
		case entry != nil && more.MatchString(line):
			entry = append(entry, more.FindStringSubmatch(line)[1])
		default:
			flush()
		}
	}

	flush()

	return entries
}

// A reference is written out in full wherever a method is used, so that
// hovering over the name in an editor shows it: each copy must be the very
// entry of the package documentation, and both readmes must list them all.
func TestEveryReferenceIsTheOneOfThePackageDocumentation(t *testing.T) {
	t.Parallel()

	doc, err := os.ReadFile("doc.go")
	require.NoError(t, err)

	canonical := references(string(doc), listItem, listMore)
	require.Len(t, canonical, 7, "the references of the package documentation")

	files, err := filepath.Glob("*.go")
	require.NoError(t, err)

	cited := map[string]bool{}

	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") || name == "doc.go" {
			continue
		}

		data, err := os.ReadFile(name)
		require.NoError(t, err)

		for _, entry := range references(string(data), listItem, listMore) {
			require.Contains(t, canonical, entry, "%s: a reference that is not the one of doc.go", name)

			cited[entry] = true
		}
	}

	for _, entry := range canonical {
		require.True(t, cited[entry], "listed in doc.go but cited nowhere in the code: %s", entry)
	}

	for _, readme := range []string{"README.md", "README.fr.md"} {
		data, err := os.ReadFile(filepath.Join("..", readme))
		require.NoError(t, err)

		require.ElementsMatch(t, canonical, references(string(data), markdownItem, markdownMore), readme)
	}
}
