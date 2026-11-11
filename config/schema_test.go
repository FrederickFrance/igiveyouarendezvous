package config

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

func compileSchema(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()

	path, err := filepath.Abs(filepath.Join("..", "schema", name)) // absolute: also under WebAssembly
	require.NoError(t, err)

	sch, err := jsonschema.NewCompiler().Compile(path)
	require.NoError(t, err)

	return sch
}

// asJSON returns a YAML or JSON document as the values the schema checks.
func asJSON(t *testing.T, data []byte) any {
	t.Helper()

	doc, err := parse(data)
	require.NoError(t, err)

	var b bytes.Buffer
	require.NoError(t, write(&b, doc, JSON))

	v, err := jsonschema.UnmarshalJSON(&b)
	require.NoError(t, err)

	return v
}

func TestTheExamplesFollowTheSchema(t *testing.T) {
	t.Parallel()

	sch := compileSchema(t, "problem.schema.json")

	names, err := filepath.Glob(filepath.Join("..", "example", "*.*"))
	require.NoError(t, err)

	french, err := filepath.Glob(filepath.Join("..", "example", "fr", "*.yaml"))
	require.NoError(t, err)

	english, err := filepath.Glob(filepath.Join("..", "example", "*.yaml"))
	require.NoError(t, err)

	require.Len(t, french, len(english), "every example has its French version")

	for _, name := range append(names, french...) {
		data, err := os.ReadFile(name)
		require.NoError(t, err)
		require.NoError(t, sch.Validate(asJSON(t, data)), name)
	}

	require.NoError(t, sch.Validate(asJSON(t, []byte(small))))

	// The file of the next day too.
	f, p, _ := smallProblem(t, "2026-11-03")

	next, err := f.Next(p, &planning.Schedule{Unscheduled: []int{0}})
	require.NoError(t, err)

	var b bytes.Buffer
	require.NoError(t, next.Write(&b, JSON))
	require.NoError(t, sch.Validate(asJSON(t, b.Bytes())))
}

func TestWrittenPlansFollowTheSchema(t *testing.T) {
	t.Parallel()

	sch := compileSchema(t, "plan.schema.json")
	_, p, _ := smallProblem(t, "2026-11-03")

	for _, fm := range []Format{YAML, JSON} {
		for _, s := range []*planning.Schedule{smallSchedule(), {}} {
			var b bytes.Buffer
			require.NoError(t, WritePlan(&b, p, s, "2026-11-03", "2026-11-03", fm))
			require.NoError(t, sch.Validate(asJSON(t, b.Bytes())), b.String())
		}
	}
}

// What the code rejects, the schema rejects too, so that editors tell.
func TestTheSchemaRejectsWhatTheCodeRejects(t *testing.T) {
	t.Parallel()

	problem := compileSchema(t, "problem.schema.json")

	for _, text := range []string{
		"recruiters: {R: }",
		"recruiters: {}\ncandidates: {}\nbreak: 12h-13h",
		"recruiters:\n  R: {durations: 1h}\ncandidates: {}",
		"recruiters: {R: }\ncandidates:\n  A: {meets: [R]}",
		"recruiters: {R: }\ncandidates:\n  A: {}",
		"recruiters: {R: }\ncandidates:\n  A: {meet: [R], priority: 0}",
		"recruiters: {R: }\ncandidates:\n  A: {meet: [R, R]}",
		"recruiters: [R]\ncandidates: {}",
		"recruiters:\n  '': {}\ncandidates: {}",
		"recruiters:\n  R: {duration: soon}\ncandidates: {}",
		"recruiters:\n  R: {availability: 9h-12h, 14h}\ncandidates: {}",
		"recruiters:\n  R: {availability: {tomorrow: 9h-12h}}\ncandidates: {}",
		"recruiters:\n  R: {availability: []}\ncandidates: {}",
		"recruiters: {}\ncandidates: {}\nmax-wait: forever",
		"recruiters: {}\ncandidates: {}\nallow-partial: true",
		"recruiters: {}\ncandidates: {}\nobjectives: [fun]",
		"recruiters: {}\ncandidates: {}\nobjectives: [meetings, meetings]",
		"recruiters: {}\ncandidates: {}\nmax-joint-size: -1",
		"recruiters: {R: , S: }\ncandidates: {}\njoint-groups: [[R]]",
		"recruiters: {R: , S: }\ncandidates: {}\njoint-groups: [[R, R]]",
		"recruiters: {}\ncandidates: {}\nmin-gap: soon",
		"recruiters: {R: , S: }\ncandidates: {}\npanels: [[R]]",
		"recruiters: {R: , S: }\ncandidates: {}\npanels: [[R, R]]",
		"recruiters: {}\ncandidates: {}\nmin-gap: unlimited",
		"recruiters: {}\ncandidates: {}\nmax-gap: -1h",
		"recruiters: {}\ncandidates: {}\nweekdays: [lundi]",
		"recruiters: {}\ncandidates: {}\nclosed: [demain]",
		"recruiters: {}\ncandidates: {}\nfrom: 2026-13-01",
		"recruiters: {}\ncandidates: {}\nnot-before: soon",
		"recruiters: {R: }\ncandidates:\n  A: {meet: {R: 0}}",
		"recruiters: {R: }\ncandidates:\n  A: {meet: {R: Y}}",
		"recruiters: {R: }\ncandidates:\n  A: {meet: {R: 1.5}}",
		"recruiters: {R: }\ncandidates:\n  A: {meet: {R: {rank: 0}}}",
		"recruiters: {R: }\ncandidates:\n  A: {meet: {R: {duration: 30}}}",
		"recruiters: {R: }\ncandidates:\n  A: {meet: {R: {length: 1h}}}",
		"recruiters: {R: }\ncandidates:\n  A: {meetings: [[R]]}",
		"recruiters: {R: }\ncandidates:\n  A: {meetings: [{duration: 1h}]}",
		"recruiters: {R: }\ncandidates:\n  A: {meetings: [{with: [R], times: 0}]}",
		"recruiters: {R: }\ncandidates:\n  A: {meetings: [{with: [R], duration: 2}]}",
		"recruiters: {R: }\ncandidates:\n  A: {meetings: [{with: [R], length: 1h}]}",
		"recruiters: {R: }\ncandidates:\n  A: {priority: 2}",
		"recruiters: {}\ncandidates: {}\ntravel: {default: soon}",
		"recruiters: {}\ncandidates: {}\ntravel: {speed: 80}",
		"recruiters: {}\ncandidates: {}\ntravel: {journeys: [[Paris, Lyon]]}",
		"recruiters: {}\ncandidates: {}\ntravel: {journeys: [[Paris, Lyon, 2]]}",
		"recruiters: {}\ncandidates: {}\ntravel: {journeys: [[Paris, Lyon, 2h, 3h]]}",
		"recruiters: {}\ncandidates: {}\ntravel: {journeys: [Paris, Lyon, 2h]}",
	} {
		require.Error(t, problem.Validate(asJSON(t, []byte(text))), text)

		f, err := Parse([]byte(text))
		if err == nil {
			p, o, err2 := f.Problem("", "")
			err = err2

			if err == nil {
				_, err = planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 1})
			}
		}

		require.Error(t, err, text)
	}

	plan := compileSchema(t, "plan.schema.json")

	for _, text := range []string{
		"candidates:\n  A: {interviews: []}",
		"candidates:\n  A: {fixed: true, interview: []}",
		"candidates:\n  A: {fixed: true, interviews: [{time: 9h, recruiters: [R1]}]}",
		"candidates:\n  A: {fixed: true, interviews: [{time: 9h-10h}]}",
		"candidates:\n  A: {fixed: true, interviews: [{time: 9h-10h, recruiters: [R1, R1]}]}",
		"from: 3/11\ncandidates: {}",
		"from: 2026-11-03\ncandidates:\n  A: {fixed: true, interviews: [{date: soon, time: 9h-10h, recruiters: [R1]}]}",
		"candidates: [A]",
		"candidates: {}\nextra: 1",
	} {
		require.Error(t, plan.Validate(asJSON(t, []byte(text))), text)

		_, p, _ := smallProblem(t, "2026-11-03")
		_, err := ReadPlan([]byte(text), p, false)
		require.Error(t, err, text)
	}
}

// The schemas and the code know the same keys and objectives.
func TestTheSchemaKnowsTheSameKeys(t *testing.T) {
	t.Parallel()

	keys := func(m map[string]any) []string { return slices.Sorted(maps.Keys(m)) }
	props := func(v any, path ...string) map[string]any {
		for _, p := range path {
			v = v.(map[string]any)[p]
		}

		return v.(map[string]any)["properties"].(map[string]any)
	}
	shapeKeys := func(sh shape) []string { return slices.Sorted(maps.Keys(sh)) }

	read := func(name string) any {
		data, err := os.ReadFile(filepath.Join("..", "schema", name))
		require.NoError(t, err)

		return asJSON(t, data)
	}

	problem := read("problem.schema.json")
	require.Equal(t, shapeKeys(problemKeys), keys(props(problem)))
	require.Equal(t, shapeKeys(problemKeys["recruiters"]["*"]),
		keys(props(problem, "properties", "recruiters", "additionalProperties")))
	require.Equal(t, shapeKeys(problemKeys["candidates"]["*"]),
		keys(props(problem, "properties", "candidates", "additionalProperties")))
	require.Equal(t, shapeKeys(problemKeys["travel"]), keys(props(problem, "properties", "travel")))

	var objectives []string
	for _, obj := range planning.AllObjectives() {
		objectives = append(objectives, obj.String())
	}

	var enum []string
	for _, v := range props(problem)["objectives"].(map[string]any)["items"].(map[string]any)["enum"].([]any) {
		enum = append(enum, v.(string))
	}

	require.ElementsMatch(t, objectives, enum)

	plan := read("plan.schema.json")
	require.Equal(t, shapeKeys(planKeys), keys(props(plan)))
	require.Equal(t, shapeKeys(planKeys["candidates"]["*"]),
		keys(props(plan, "properties", "candidates", "additionalProperties")))
	require.Equal(t, shapeKeys(planKeys["candidates"]["*"]["interviews"]),
		keys(props(plan, "properties", "candidates", "additionalProperties", "properties", "interviews", "items")))
}
