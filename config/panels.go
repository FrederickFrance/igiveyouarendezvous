package config

import (
	"fmt"
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// Panels is the key panels: a list of groups of recruiters who always
// interview together (Unnamed), or a mapping from names to panels met by
// their name (Named), each with its members, the times it sits and where.
// A named panel is met as one, by naming it in a candidate's meet, and may
// be one of a choice: a defence before one of the juries of the week.
type Panels struct {
	Unnamed [][]string
	Named   Named[Panel]
}

// Panel is a named panel of the problem file.
type Panel struct {
	Members []string `yaml:"members"`
	// Availability is when the panel sits, on top of the availabilities of
	// its members, who may sit on other panels at other times.
	Availability Availability `yaml:"availability"`
	// Place is where it sits, for the journeys (see Candidate.Place).
	Place string `yaml:"place"`
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (ps *Panels) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.SequenceNode:
		return n.Decode(&ps.Unnamed)
	case yaml.MappingNode:
		for i := 1; i < len(n.Content); i += 2 {
			if v := n.Content[i]; v.Kind == yaml.MappingNode {
				for j := 0; j < len(v.Content); j += 2 {
					if k := v.Content[j].Value; !slices.Contains([]string{"members", "availability", "place"}, k) {
						return fmt.Errorf("line %d: panels: %q is not a key of a panel (members, availability, place)", v.Content[j].Line, k)
					}
				}
			}
		}

		return n.Decode(&ps.Named)
	default:
		return fmt.Errorf("line %d: panels: a list of groups, or panels by name, is expected", n.Line)
	}
}

// jury is a named panel, ready to be met.
type jury struct {
	name         string
	members      []int
	availability []planning.Interval // nil: whenever its members are
	place        string
}

// juries resolves the named panels against the recruiters of the file.
func (ps Panels) juries(recruiters map[string]int, dates []string) (map[string]jury, error) {
	out := map[string]jury{}

	for _, e := range ps.Named {
		if _, ok := recruiters[e.Name]; ok {
			return nil, fmt.Errorf("line %d: panels: %s is also a recruiter: a panel needs a name of its own", e.Line, e.Name)
		}

		j := jury{name: e.Name, place: NormalizeName(e.Value.Place)}

		if len(e.Value.Members) == 0 {
			return nil, fmt.Errorf("line %d: panels: %s has no members", e.Line, e.Name)
		}

		for _, name := range e.Value.Members {
			h, ok := recruiters[NormalizeName(name)]

			switch {
			case !ok:
				return nil, fmt.Errorf("line %d: panels: %s: %q is not a recruiter of the file", e.Line, e.Name, name)
			case slices.Contains(j.members, h):
				return nil, fmt.Errorf("line %d: panels: %s: %s is given twice", e.Line, e.Name, name)
			}

			j.members = append(j.members, h)
		}

		spans, set, err := on(e.Value.Availability, dates)
		if err != nil {
			return nil, fmt.Errorf("line %d: panels: %s: %w", e.Line, e.Name, err)
		}

		if set {
			j.availability = spans
		}

		out[e.Name] = j
	}

	return out, nil
}

// request is the request of candidate c to meet the panel: its first
// member, with the others coming along (see planning.Request.With).
func (j jury) request(c int, m Meet, choices map[string]int) (planning.Request, error) {
	req := planning.Request{
		Candidate: c, Recruiter: j.members[0], With: j.members[1:], Rank: m.Rank, Duration: m.Duration,
		Optional: m.Optional, Availability: j.availability, Choice: choiceNumber(choices, m.Choice),
	}

	if len(req.With) == 0 {
		req.With = nil
	}

	switch {
	case m.Place != "":
		return req, fmt.Errorf("%s: a panel sits at its own place, given in panels", j.name)
	case j.place != "" && req.Choice == 0:
		return req, fmt.Errorf("%s sits at %s: met alone, give the candidate that place instead", j.name, j.place)
	}

	req.Place = j.place

	return req, nil
}

// choiceNumber returns the number of the choice named, from 1, numbering a
// new one; 0 for no choice.
func choiceNumber(choices map[string]int, name string) int {
	if name == "" {
		return 0
	}

	if _, ok := choices[name]; !ok {
		choices[name] = len(choices) + 1
	}

	return choices[name]
}
