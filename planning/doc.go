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
// then gives its length. The requests with the same [Request.Meeting] are
// such meetings too, as given, and a candidate may meet a recruiter in
// several of them: the shoot of an episode, with the host and the camera
// operator, then with them and a guest.
//
// A recruiter may need time between two of their interviews, nights
// included: [Recruiter.MinGap], to write notes for instance. When the
// candidates have places ([Candidate.Place]), such as the cities of shoots,
// the journey between them ([Problem.Travel]) comes on top.
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
// Most moves re-time a whole day, keeping who meets whom and packing the
// interviews as tight as they go; one in fifty, and whenever re-timing
// changes nothing, removes some candidates and puts them back
// ([SearchOptions.Retime]). With [SearchOptions.Exact], some moves take the
// day of one recruiter that has a break and put its interviews back where
// they leave the fewest breaks there can be, for the recruiter and for
// everybody they meet: worked out rather than tried, on days of one-slot
// meetings.
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
// # References
//
// The search puts together methods of the literature, cited where each is
// used and listed here, in the style of APA 7: accepting a move not worse
// than the schedule of a thousand moves ago (Burke & Bykov, 2017); removing
// some candidates and putting them back (Schrimpf et al., 2000), chosen
// among related ones, such as those sharing a recruiter or a time window
// (Shaw, 1998); objectives in a strict order (Ehrgott, 2005); the exact
// re-timing of one recruiter's day, a dynamic programme over the subsets of
// its interviews (Held & Karp, 1962); and, to know when no schedule can
// serve more candidates, a knapsack of each recruiter's hours (Kellerer et
// al., 2004).
//
//   - Burke, E. K., & Bykov, Y. (2017). The late acceptance hill-climbing
//     heuristic. European Journal of Operational Research, 258(1), 70–78.
//     https://doi.org/10.1016/j.ejor.2016.07.012
//   - Ehrgott, M. (2005). Multicriteria optimization (2nd ed.). Springer.
//     https://doi.org/10.1007/3-540-27659-9
//   - Held, M., & Karp, R. M. (1962). A dynamic programming approach to
//     sequencing problems. Journal of the Society for Industrial and Applied
//     Mathematics, 10(1), 196–210. https://doi.org/10.1137/0110015
//   - Kellerer, H., Pferschy, U., & Pisinger, D. (2004). Knapsack problems.
//     Springer. https://doi.org/10.1007/978-3-540-24777-7
//   - Schrimpf, G., Schneider, J., Stamm-Wilbrandt, H., & Dueck, G. (2000).
//     Record breaking optimization results using the ruin and recreate
//     principle. Journal of Computational Physics, 159(2), 139–171.
//     https://doi.org/10.1006/jcph.1999.6413
//   - Shaw, P. (1998). Using constraint programming and local search methods
//     to solve vehicle routing problems. In M. Maher & J.-F. Puget (Eds.),
//     Principles and practice of constraint programming — CP98 (Lecture Notes
//     in Computer Science, Vol. 1520, pp. 417–431). Springer.
//     https://doi.org/10.1007/3-540-49481-2_30
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
