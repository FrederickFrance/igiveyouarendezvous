package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Format is the syntax of a file.
type Format int

// The formats. Both are read the same way; JSON is read by encoding/json,
// whose strings are not all valid YAML.
const (
	YAML Format = iota
	JSON
)

// FormatOf returns the format of a file from its name: JSON for ".json",
// YAML otherwise.
func FormatOf(path string) Format {
	if strings.EqualFold(filepath.Ext(path), ".json") {
		return JSON
	}

	return YAML
}

// parse reads a YAML or JSON document (JSON when it starts with "{") into a
// YAML document node, with the line numbers of the text.
func parse(data []byte) (*yaml.Node, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // a byte order mark

	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '{' {
		root, err := parseJSON(data)
		if err != nil {
			return nil, err
		}

		return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}, Line: 1}, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	if doc.Kind == 0 { // empty
		return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{mapping()}, Line: 1}, nil
	}

	return &doc, nil
}

// parseJSON converts JSON into a YAML node, keeping the order of the keys.
func parseJSON(data []byte) (*yaml.Node, error) {
	var lines []int // offsets of the line breaks

	for i, b := range data {
		if b == '\n' {
			lines = append(lines, i)
		}
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	// line returns the line of the next token: after the offset of the
	// decoder, which stops right after the previous token.
	line := func() int {
		at := int(dec.InputOffset())
		for at < len(data) && strings.IndexByte(" \t\r\n,:", data[at]) >= 0 {
			at++
		}

		return 1 + sort.SearchInts(lines, at)
	}

	var value func() (*yaml.Node, error)

	value = func() (*yaml.Node, error) {
		at := line()

		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", at, err)
		}

		switch t := tok.(type) {
		case json.Delim:
			n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Line: line()}
			if t == '{' {
				n.Kind, n.Tag = yaml.MappingNode, "!!map"
			}

			seen := map[string]bool{}

			for dec.More() {
				if n.Kind == yaml.MappingNode {
					keyLine := line()

					k, err := dec.Token()
					if err != nil {
						return nil, fmt.Errorf("line %d: %w", keyLine, err)
					}

					key, _ := k.(string)
					if seen[key] {
						return nil, fmt.Errorf("line %d: %q is given twice", keyLine, key)
					}

					seen[key] = true
					n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key, Line: keyLine})
				}

				item, err := value()
				if err != nil {
					return nil, err
				}

				n.Content = append(n.Content, item)
			}

			if _, err := dec.Token(); err != nil { // the closing delimiter
				return nil, fmt.Errorf("line %d: %w", line(), err)
			}

			if n.Kind == yaml.SequenceNode && !slices.ContainsFunc(n.Content, func(item *yaml.Node) bool { return item.Kind != yaml.ScalarNode }) {
				n.Style = yaml.FlowStyle // [a, b] once written in YAML
			}

			return n, nil
		case string:
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: t, Line: at}, nil
		case json.Number:
			tag := "!!int"
			if strings.ContainsAny(t.String(), ".eE") {
				tag = "!!float"
			}

			return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: t.String(), Line: at}, nil
		case bool:
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(t), Line: at}, nil
		default: // null
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null", Line: at}, nil
		}
	}

	n, err := value()
	if err != nil {
		return nil, err
	}

	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("line %d: unexpected text after the end of the document", line())
	}

	return n, nil
}

// shape lists the keys a mapping may hold, with the shape of their values;
// a nil shape is not checked, and the key "*" stands for any name (of a
// person).
type shape map[string]shape

// checkKeys reports the first unknown key of n, a typo most of the time.
// The elements of a sequence all follow the shape.
func checkKeys(n *yaml.Node, sh shape, where string) error {
	if sh == nil {
		return nil
	}

	in := where
	if in == "" {
		in = "the top level"
	}

	switch n.Kind {
	case yaml.SequenceNode:
		for _, item := range n.Content {
			if err := checkKeys(item, sh, where); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, value := n.Content[i], n.Content[i+1]

			sub, known := sh[key.Value]
			if wild, ok := sh["*"]; ok {
				sub, known = wild, true
			}

			if !known {
				keys := make([]string, 0, len(sh))
				for k := range sh {
					keys = append(keys, k)
				}

				slices.Sort(keys)

				hint := ""
				if _, err := ParseInterval(key.Value); err == nil {
					hint = `; between braces, write the time spans between quotes: "09:00-12:00, 14:00-16:00"`
				}

				return fmt.Errorf("line %d: unknown key %q in %s (known: %s)%s", key.Line, key.Value, in, strings.Join(keys, ", "), hint)
			}

			inner := key.Value
			if where != "" {
				inner = where + " > " + key.Value
			}

			if err := checkKeys(value, sub, inner); err != nil {
				return err
			}
		}
	default:
	}

	return nil
}

// write writes n in format f.
func write(w io.Writer, n *yaml.Node, f Format) error {
	if f == YAML {
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)

		if err := enc.Encode(n); err != nil {
			return err
		}

		return enc.Close()
	}

	if n.Kind == yaml.DocumentNode {
		n = n.Content[0]
	}

	var b bytes.Buffer
	if err := writeJSON(&b, n, 0); err != nil {
		return err
	}

	b.WriteByte('\n')

	_, err := w.Write(b.Bytes())

	return err
}

// writeJSON writes n as indented JSON, keeping the order of the keys; lists
// of scalars, and mappings in flow style, stay on one line.
func writeJSON(b *bytes.Buffer, n *yaml.Node, depth int) error {
	outdent := strings.Repeat("  ", depth) // the closing brace or bracket
	indent := outdent + "  "               // the contents

	switch n.Kind {
	case yaml.MappingNode:
		if len(n.Content) == 0 {
			b.WriteString("{}")

			return nil
		}

		if n.Style&yaml.FlowStyle != 0 { // on one line
			b.WriteString("{")

			for i := 0; i+1 < len(n.Content); i += 2 {
				if i > 0 {
					b.WriteString(", ")
				}

				writeString(b, n.Content[i].Value)
				b.WriteString(": ")

				if err := writeJSON(b, n.Content[i+1], depth+1); err != nil {
					return err
				}
			}

			b.WriteString("}")

			return nil
		}

		b.WriteString("{\n")

		for i := 0; i+1 < len(n.Content); i += 2 {
			b.WriteString(indent)
			writeString(b, n.Content[i].Value)
			b.WriteString(": ")

			if err := writeJSON(b, n.Content[i+1], depth+1); err != nil {
				return err
			}

			if i+2 < len(n.Content) {
				b.WriteByte(',')
			}

			b.WriteByte('\n')
		}

		b.WriteString(outdent)
		b.WriteByte('}')
	case yaml.SequenceNode:
		inline := !slices.ContainsFunc(n.Content, func(item *yaml.Node) bool { return item.Kind != yaml.ScalarNode })

		b.WriteByte('[')

		for i, item := range n.Content {
			if !inline {
				b.WriteByte('\n')
				b.WriteString(indent)
			}

			if err := writeJSON(b, item, depth+1); err != nil {
				return err
			}

			if i+1 < len(n.Content) {
				b.WriteByte(',')

				if inline {
					b.WriteByte(' ')
				}
			}
		}

		if !inline && len(n.Content) > 0 {
			b.WriteByte('\n')
			b.WriteString(outdent)
		}

		b.WriteByte(']')
	case yaml.ScalarNode:
		switch tag := n.ShortTag(); {
		case tag == "!!null":
			b.WriteString("null")
		case tag == "!!bool":
			b.WriteString(strconv.FormatBool(n.Value == "true" || n.Value == "True" || n.Value == "TRUE"))
		case (tag == "!!int" || tag == "!!float") && json.Valid([]byte(n.Value)):
			b.WriteString(n.Value)
		default: // strings, and numbers JSON does not write such as 0x1F
			writeString(b, n.Value)
		}
	case yaml.AliasNode:
		return writeJSON(b, n.Alias, depth)
	default:
		return fmt.Errorf("config: cannot write a node of kind %d", n.Kind)
	}

	return nil
}

// writeString writes s as a JSON string, without escaping <, > and &.
func writeString(b *bytes.Buffer, s string) {
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s) // a string always encodes
	b.Truncate(b.Len() - 1)
}

func scalar(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
}

// plain is a scalar written without quotes in YAML: a date, not a name
// that could read as a number or a boolean.
func plain(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: v}
}

func number(v int) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(v)}
}

func boolean(v bool) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(v)}
}

func mapping(content ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: content}
}

// list is a sequence, on one line in YAML when it only holds scalars.
func list(items ...*yaml.Node) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: items}
	if !slices.ContainsFunc(items, func(item *yaml.Node) bool { return item.Kind != yaml.ScalarNode }) {
		n.Style = yaml.FlowStyle
	}

	return n
}

func names(values []string) *yaml.Node {
	items := make([]*yaml.Node, len(values))
	for i, v := range values {
		items[i] = scalar(v)
	}

	return list(items...)
}
