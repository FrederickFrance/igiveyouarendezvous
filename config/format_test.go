package config

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestFormatOf(t *testing.T) {
	t.Parallel()

	require.Equal(t, JSON, FormatOf("dir/plan.JSON"))
	require.Equal(t, YAML, FormatOf("problem.yaml"))
	require.Equal(t, YAML, FormatOf("problem.yml"))
	require.Equal(t, YAML, FormatOf("problem"))
}

func TestParseJSONKeepsOrderAndLines(t *testing.T) {
	t.Parallel()

	doc, err := parse([]byte("\xef\xbb\xbf{\n  \"b\": 1,\n  \"a\": [true, null, 2.5, \"x\\/y\"],\n\t\"c\": {}\n}\n"))
	require.NoError(t, err)

	root := doc.Content[0]
	require.Equal(t, yaml.MappingNode, root.Kind)
	require.Len(t, root.Content, 6)

	b, a, c := root.Content[0], root.Content[2], root.Content[4]
	require.Equal(t, []string{"b", "a", "c"}, []string{b.Value, a.Value, c.Value})
	require.Equal(t, []int{2, 3, 4}, []int{b.Line, a.Line, c.Line})

	list := root.Content[3]
	require.Equal(t, yaml.FlowStyle, list.Style, "a list of scalars stays on one line in YAML")
	require.Equal(t, []string{"!!bool", "!!null", "!!float", "!!str"},
		[]string{list.Content[0].Tag, list.Content[1].Tag, list.Content[2].Tag, list.Content[3].Tag})
	require.Equal(t, "x/y", list.Content[3].Value, "an escape YAML does not know")
	require.Equal(t, "!!int", root.Content[1].Tag)
}

func TestParseJSONErrors(t *testing.T) {
	t.Parallel()

	for text, msg := range map[string]string{
		"{\n\"a\": 1,\n\"a\": 2}": `line 3: "a" is given twice`,
		`{"a": 1} {"b": 2}`:       "unexpected text after the end",
		"{\n\"a\": [1,\n}":        "line 3",
		`{"a": `:                  "line 1",
		`{"a" 1}`:                 "line 1",
	} {
		_, err := parse([]byte(text))
		require.ErrorContains(t, err, msg, text)
	}
}

func TestParseYAML(t *testing.T) {
	t.Parallel()

	doc, err := parse([]byte("# nothing\n"))
	require.NoError(t, err)
	require.Equal(t, yaml.MappingNode, doc.Content[0].Kind, "an empty document is an empty mapping")

	doc, err = parse(nil)
	require.NoError(t, err)
	require.Equal(t, yaml.MappingNode, doc.Content[0].Kind)

	_, err = parse([]byte("a: [1\n"))
	require.Error(t, err)

	doc, err = parse([]byte("[1, 2]"))
	require.NoError(t, err, "a list starts like YAML")
	require.Equal(t, yaml.SequenceNode, doc.Content[0].Kind)
}

func TestCheckKeys(t *testing.T) {
	t.Parallel()

	sh := shape{"a": nil, "people": {"*": {"age": nil, "pets": {"name": nil}}}}

	for text, msg := range map[string]string{
		"a: 1\nb: 2":                                   `line 2: unknown key "b" in the top level (known: a, people)`,
		"people:\n  Ann:\n    agee: 3":                 `line 3: unknown key "agee" in people > Ann (known: age, pets)`,
		"people:\n  Ann:\n    pets:\n      - nom: Rex": `line 4: unknown key "nom" in people > Ann > pets (known: name)`,
	} {
		doc, err := parse([]byte(text))
		require.NoError(t, err)
		require.EqualError(t, checkKeys(doc.Content[0], sh, ""), msg, text)
	}

	doc, err := parse([]byte("a: {free: 1}\npeople:\n  Ann:\n  Bob: {age: 3, pets: [{name: Rex}]}"))
	require.NoError(t, err)
	require.NoError(t, checkKeys(doc.Content[0], sh, ""))
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	doc, err := parse([]byte(`# a comment
b: 0x1F
a: [x, "1", 2, true, ~, <&>]
c: {}
d: []
e:
  - {f: 1, g: [h]}
  - k: v
alias: &x {y: 1}
ref: *x
`))
	require.NoError(t, err)

	inline := doc.Content[0].Content[9].Content[0]
	require.Equal(t, yaml.FlowStyle, inline.Style&yaml.FlowStyle)

	var b bytes.Buffer
	require.NoError(t, write(&b, doc, JSON))
	require.JSONEq(t, `{"b": "0x1F", "a": ["x", "1", 2, true, null, "<&>"], "c": {}, "d": [],
		"e": [{"f": 1, "g": ["h"]}, {"k": "v"}], "alias": {"y": 1}, "ref": {"y": 1}}`, b.String())
	require.Contains(t, b.String(), `"a": ["x", "1", 2, true, null, "<&>"]`, "a list of scalars on one line, no HTML escapes")
	require.Contains(t, b.String(), `{"f": 1, "g": ["h"]}`, "a flow mapping on one line")
	require.True(t, json.Valid(b.Bytes()))

	// Read back, the order is kept.
	back, err := parse(b.Bytes())
	require.NoError(t, err)
	require.Equal(t, "b", back.Content[0].Content[0].Value)

	var y bytes.Buffer
	require.NoError(t, write(&y, back, YAML))
	require.Contains(t, y.String(), `a: [x, "1", 2, true, null, <&>]`)
}

func TestWriteKeepsYAMLComments(t *testing.T) {
	t.Parallel()

	text := "# head\n\na: 1 # one\n# before b\nb: [x, y]\n"

	doc, err := parse([]byte(text))
	require.NoError(t, err)

	var b bytes.Buffer
	require.NoError(t, write(&b, doc, YAML))
	require.Equal(t, text, b.String())
}

func TestWriteRejectsUnknownNodes(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	require.Error(t, write(&b, mapping(scalar("a"), &yaml.Node{}), JSON))
}

func TestNodeBuilders(t *testing.T) {
	t.Parallel()

	n := mapping(scalar("name"), scalar("true"), scalar("n"), number(3), scalar("ok"), boolean(true),
		scalar("date"), plain("2026-11-03"), scalar("who"), names([]string{"a", "1"}), scalar("deep"), list(mapping()))

	var b bytes.Buffer
	require.NoError(t, write(&b, n, YAML))
	require.Equal(t, "name: \"true\"\nn: 3\nok: true\ndate: 2026-11-03\nwho: [a, \"1\"]\ndeep:\n  - {}\n", b.String())
}
