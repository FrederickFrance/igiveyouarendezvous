package config_test

import (
	"fmt"
	"os"
	"time"

	"github.com/FrederickFrance/igiveyouarendezvous/config"
	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

const problemFile = `
day: 08:00-17:00
breaks: 12:00-13:30
joint-groups: [[recruiter 1, recruiter 2]]
recruiters:
  recruiter 1:
  recruiter 2:
    duration: 30m
candidates:
  candidate A:
    meet: [recruiter 1, recruiter 2]
  candidate B:
    meet: [recruiter 2]   # a comment
    availability:
      2026-11-03: 14:00-17:00
      2026-11-04: 09:00-11:00
`

// The problem file gives the problem and the options of a period, here two
// days: the times count from the midnight of its first day.
func ExampleFile_Problem() {
	f, err := config.Parse([]byte(problemFile))
	if err != nil {
		fmt.Println(err)

		return
	}

	p, o, err := f.Problem("2026-11-03", "2026-11-04")
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println("dates:", f.Dates())
	fmt.Println("days:", o.Days, "day:", o.Day, "breaks:", o.Breaks)

	for _, iv := range p.Candidates[1].Availability {
		fmt.Println("candidate B: day", iv.Day(), iv)
	}

	fmt.Println("requests:", len(p.Requests), "joint groups:", p.JointGroups)

	// Output:
	// dates: [2026-11-03 2026-11-04]
	// days: 2 day: [08:00-17:00] breaks: [12:00-13:30]
	// candidate B: day 0 14:00-17:00
	// candidate B: day 1 09:00-11:00
	// requests: 3 joint groups: [[0 1]]
}

// After a period, the problem file of the next one keeps the candidates who
// were not planned.
func ExampleFile_Next() {
	f, err := config.Parse([]byte(problemFile))
	if err != nil {
		fmt.Println(err)

		return
	}

	p, _, err := f.Problem("2026-11-03", "")
	if err != nil {
		fmt.Println(err)

		return
	}

	// Candidate A met their recruiters, B did not.
	s := &planning.Schedule{Unscheduled: []int{2}}

	next, err := f.Next(p, s)
	if err != nil {
		fmt.Println(err)

		return
	}

	if err := next.Write(os.Stdout, config.YAML); err != nil {
		fmt.Println(err)
	}

	// Output:
	// day: 08:00-17:00
	// breaks: 12:00-13:30
	// joint-groups: [[recruiter 1, recruiter 2]]
	// recruiters:
	//   recruiter 1:
	//   recruiter 2:
	//     duration: 30m
	// candidates:
	//   candidate B:
	//     meet: [recruiter 2] # a comment
	//     availability:
	//       2026-11-03: 14:00-17:00
	//       2026-11-04: 09:00-11:00
}

// A schedule is written with names and dates, to be read back against the
// problem of the period, which may have changed since.
func ExampleWritePlan() {
	f, err := config.Parse([]byte(problemFile))
	if err != nil {
		fmt.Println(err)

		return
	}

	p, _, err := f.Problem("2026-11-03", "2026-11-04")
	if err != nil {
		fmt.Println(err)

		return
	}

	s := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: 9 * time.Hour, End: 10 * time.Hour, Fixed: true},
		{Candidate: 1, Recruiters: []int{1}, Requests: []int{2}, Start: planning.DayLength + 9*time.Hour, End: planning.DayLength + 9*time.Hour + 30*time.Minute},
	}}

	if err := config.WritePlan(os.Stdout, p, s, "2026-11-03", "2026-11-04", config.YAML); err != nil {
		fmt.Println(err)
	}

	// Output:
	// from: 2026-11-03
	// to: 2026-11-04
	// candidates:
	//   candidate A:
	//     fixed: true
	//     interviews:
	//       - date: 2026-11-03
	//         time: 09:00-10:00
	//         recruiters: [recruiter 1, recruiter 2]
	//   candidate B:
	//     fixed: false
	//     interviews:
	//       - date: 2026-11-04
	//         time: 09:00-09:30
	//         recruiters: [recruiter 2]
}
