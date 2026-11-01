package planning_test

import (
	"context"
	"fmt"
	"time"

	"github.com/FrederickFrance/igiveyouarendezvous/mip/gomilp"
	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

func ExampleSolve() {
	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "R1"},
			{Name: "R2", Duration: 30 * time.Minute},
		},
		Candidates: []planning.Candidate{
			{Name: "A"},
			{Name: "B", Availability: []planning.Interval{{Start: 9 * time.Hour, End: 12 * time.Hour}}},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1},
			{Candidate: 1, Recruiter: 0},
		},
		JointGroups: [][]int{{0, 1}}, // A may meet R1 and R2 together
	}

	o := planning.Options{
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinInterviews, planning.MinEndTimes},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s, err := planning.Solve(ctx, gomilp.New(), p, o)
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, iv := range s.Interviews {
		var names []string
		for _, h := range iv.Recruiters {
			names = append(names, p.Recruiters[h].Name)
		}

		fmt.Println(planning.Interval{Start: iv.Start, End: iv.End}, p.Candidates[iv.Candidate].Name, names)
	}

	for _, sc := range s.Scores {
		fmt.Println(sc.Objective, sc.Value, sc.Proven)
	}

	fmt.Println("left for tomorrow:", len(s.Remaining(p).Requests))

	// Output:
	// 08:00-09:00 A [R1 R2]
	// 09:00-10:00 B [R1]
	// complete-candidates 2 true
	// interviews 2 true
	// earliest 1140 true
	// left for tomorrow: 0
}

func ExampleSearch() {
	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "R1"}, {Name: "R2"}},
		Candidates: []planning.Candidate{{Name: "A"}, {Name: "B"}, {Name: "C"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1},
			{Candidate: 1, Recruiter: 1},
			{Candidate: 2, Recruiter: 0},
		},
		JointGroups: [][]int{{0, 1}},
	}

	// A time limit is the usual stop; a number of moves keeps the example
	// reproducible.
	s, err := planning.Search(context.Background(), p, planning.Options{}, planning.SearchOptions{Seed: 1, Iterations: 1000})
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, sc := range s.Scores {
		fmt.Println(sc.Objective, sc.Value, sc.Proven)
	}

	// Output:
	// priorities 3 true
	// meetings 4 true
	// recruiters 2 false
	// recruiter-half-days 2 false
	// interviews 3 false
	// recruiter-span 240 false
}

// busyDay returns a day too short for everybody: R1 has five one-hour
// interviews to hold in four hours.
func busyDay() *planning.Problem {
	p := &planning.Problem{Recruiters: []planning.Recruiter{{Name: "R1"}, {Name: "R2"}}}

	for i, name := range []string{"A", "B", "C", "D", "E"} {
		p.Candidates = append(p.Candidates, planning.Candidate{Name: name})
		p.Requests = append(p.Requests, planning.Request{Candidate: i, Recruiter: 0})
	}

	return p
}

// The search is stopped on demand by canceling its context, here as soon as
// four candidates are served; it returns the best schedule found so far.
func ExampleSearch_stop() {
	p := busyDay()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := planning.Search(ctx, p, planning.Options{}, planning.SearchOptions{
		Progress: func(pr planning.Progress) {
			if pr.Scores[0].Value >= 4 { // pr.Schedule is the schedule itself
				cancel()
			}
		},
	})
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println("complete candidates:", s.Scores[0].Value)
	fmt.Println("requests left for tomorrow:", len(s.Remaining(p).Requests))

	// Output:
	// complete candidates: 4
	// requests left for tomorrow: 1
}

// A search resumes from the schedule of an earlier one, which it never makes
// worse; here the problem changed in between: E is no longer to be seen.
func ExampleSearchOptions_start() {
	p := busyDay()

	first, err := planning.Search(context.Background(), p, planning.Options{}, planning.SearchOptions{Iterations: 100})
	if err != nil {
		fmt.Println(err)

		return
	}

	p.Requests = p.Requests[:4]

	again, err := planning.Search(context.Background(), p, planning.Options{}, planning.SearchOptions{
		Iterations: 100,
		Start:      first,
	})
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println("complete candidates:", again.Scores[0].Value, "of", len(p.Requests))

	// Output:
	// complete candidates: 4 of 4
}

// Remaining returns the problem of the next day: the same people, with the
// requests left.
func ExampleSchedule_Remaining() {
	p := busyDay()

	s, err := planning.Search(context.Background(), p, planning.Options{}, planning.SearchOptions{Iterations: 100})
	if err != nil {
		fmt.Println(err)

		return
	}

	tomorrow := s.Remaining(p)
	r := tomorrow.Requests[0] // which candidate waits depends on the search
	fmt.Println(len(tomorrow.Requests), "request left, with", tomorrow.Recruiters[r.Recruiter].Name)

	// Output:
	// 1 request left, with R1
}
