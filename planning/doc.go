// Package planning schedules interviews between candidates and recruiters.
//
// Each candidate must meet some recruiters: the requests of a [Problem]. A
// recruiter meets one candidate at a time; a candidate may meet several
// recruiters of a same joint group ([Problem.JointGroups]) in one joint
// interview, and always meets the members of a panel ([Problem.Panels]),
// such as a jury, at once. Interviews have a duration per recruiter, and
// everybody has availabilities over the period to plan (package config reads
// them by date).
//
// # Period
//
// The period is [Options.Days] days, one by default, with [Options.Closed]
// days without interviews. Times count from the midnight of its first day:
// day d runs from d*[DayLength], and every day has the hours of [Options.Day]
// and the breaks of [Options.Breaks]. An interview never crosses midnight.
//
// A candidate meets all their recruiters in the period, or none of them:
// [Schedule.Remaining] returns what is left for the next period. Between two
// interviews of a candidate, the gap is at least [Options.MinGap] and at most
// [Options.MaxGap], a duration or a number of calendar days ([Gap]); by
// default, interviews follow each other. Two interviews of the same day never
// lie on both sides of a break. A request may have a rank
// ([Request.Rank]): the meetings of rank 1 come first, then those of rank 2,
// and so on; rank 0 meets at any time. A joint interview never mixes two
// ranks.
//
// A candidate with [Candidate.Together] meets all their recruiters in one
// single interview, or not at all: a meeting rather than interviews, such
// as the recording of a show with its host and guests. [Candidate.Duration]
// then gives its length.
//
// # Objectives
//
// The objectives are optimised lexicographically, in the order of
// [Options.Objectives]: a schedule is better than another when it does better
// on the first objective, or as well on the first and better on the second,
// and so on. [DefaultObjectives] gives the default order: the candidates who
// meet all their recruiters, priority level by priority level
// ([Candidate.Priority], [MaxPriorities]), then the candidates who have the
// fewest days left ([MaxUrgency]), then the meetings, then the recruiters
// mobilised and their half-days, day by day, then the interviews and the
// presence of the recruiters.
//
// # Search and check
//
// [Search], a local search in pure Go, improves the schedule as long as it
// runs (until a deadline, a number of moves or a patience) without proving
// optimality, except for obvious bounds. It starts from a greedy schedule:
// the highest priorities first, then the candidates with the fewest
// requests, each at the first place found.
//
// [Check] verifies any schedule independently of the search, and [Evaluate]
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
