// Package planning schedules interviews between candidates and recruiters.
//
// Each candidate must meet some recruiters: the requests of a [Problem]. A
// recruiter meets one candidate at a time; a candidate may meet several
// recruiters of a same joint group ([Problem.JointGroups]) in one joint
// interview. Interviews have a duration per recruiter, and everybody has
// availabilities for the day to plan (package config picks them by date).
//
// A candidate meets all their recruiters the same day, or none of them that
// day, in interviews that follow each other ([Options.MaxWait]) and never lie on both sides of a break
// ([Options.Breaks]). Since everything rarely fits in one day, [Search] and
// [Solve] plan one day and [Schedule.Remaining] returns what is left for the
// next one.
//
// # Objectives
//
// The objectives are optimised lexicographically, in the order of
// [Options.Objectives]: a schedule is better than another when it does better
// on the first objective, or as well on the first and better on the second,
// and so on. [DefaultObjectives] gives the default order: the candidates who
// meet all their recruiters, priority level by priority level
// ([Candidate.Priority], [MaxPriorities]), then the meetings, then the
// recruiters mobilised in the day and their half-days, then the interviews
// and the presence of the recruiters.
//
// # Engines
//
// [Search], a local search in pure Go, improves the schedule as long as it
// runs (until a deadline, a number of moves or a patience) without proving
// optimality, except for obvious bounds. It is the engine to use. It starts
// from a greedy schedule: the highest priorities first, then the candidates
// with the fewest requests, each at the first place found.
//
// [Solve] writes the problem as a mixed-integer program and hands it to a
// [mip.Solver], such as mip/gomilp, which proves optimality on small
// problems only.
//
// [Check] verifies any schedule independently of the engines, and [Evaluate]
// scores it.
//
// # Following, stopping, resuming
//
// [SearchOptions.Progress] is called with every better schedule found.
// Canceling the context of [Search] stops it, and it then returns the best
// schedule found so far. [SearchOptions.Start] resumes from an earlier
// schedule, even after the problem changed: the plans of its fixed
// candidates ([Interview.Fixed]), already announced, are added one by one
// and kept as they are, unless they no longer hold; the other interviews are
// kept when they still fit. [Clean] tells beforehand what is kept, and why
// the rest is not.
//
// # Example
//
//	p := &planning.Problem{
//		Recruiters: []planning.Recruiter{{Name: "R1"}, {Name: "R2", Duration: 30 * time.Minute}},
//		Candidates: []planning.Candidate{{Name: "A"}, {Name: "B"}},
//		Requests: []planning.Request{
//			{Candidate: 0, Recruiter: 0},
//			{Candidate: 0, Recruiter: 1},
//			{Candidate: 1, Recruiter: 0},
//		},
//		JointGroups: [][]int{{0, 1}}, // R1 and R2 may interview together
//	}
//
//	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
//	defer cancel()
//
//	s, err := planning.Search(ctx, p, planning.Options{}, planning.SearchOptions{Patience: 10 * time.Second})
//	if err != nil {
//		return err
//	}
//
//	for _, iv := range s.Interviews {
//		fmt.Println(planning.Clock(iv.Start), p.Candidates[iv.Candidate].Name, iv.Recruiters)
//	}
//
//	tomorrow := s.Remaining(p)
package planning
