package main

import (
	"encoding/json"
	"fmt"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// progressLine is one improvement of the search, as read by a program: the
// demo draws the curve of each objective from it, and a script can follow a
// long run. The format is a JSON object per line, on the standard error
// output, next to the notes.
type progressLine struct {
	Seconds    float64         `json:"seconds"`
	Iterations int             `json:"iterations"`
	Scores     []progressScore `json:"scores"`
}

// progressScore is the value of one criterion. Label is what the text
// output shows, so a reader needs no table of its own; Unit tells how to
// read Value, and Direction which way is better.
type progressScore struct {
	Objective string `json:"objective"`
	Label     string `json:"label"`
	Value     int    `json:"value"`
	Unit      string `json:"unit"`      // "count" or "minutes"
	Direction string `json:"direction"` // "max" or "min"
	Proven    bool   `json:"proven"`
	Priority  int    `json:"priority,omitempty"`
	DaysLeft  int    `json:"daysLeft,omitempty"`
}

// progressJSON returns the JSON line of a progress report.
func progressJSON(pr planning.Progress) (string, error) {
	line := progressLine{
		Seconds:    pr.Elapsed.Seconds(),
		Iterations: pr.Iterations,
		Scores:     make([]progressScore, len(pr.Scores)),
	}

	for i, sc := range pr.Scores {
		line.Scores[i] = progressScore{
			Objective: sc.Objective.String(),
			Label:     label(sc),
			Value:     sc.Value,
			Unit:      scoreUnit(sc.Objective),
			Direction: scoreDirection(sc.Objective),
			Proven:    sc.Proven,
			Priority:  sc.Priority,
			DaysLeft:  sc.DaysLeft,
		}
	}

	text, err := json.Marshal(line)
	if err != nil {
		return "", fmt.Errorf("the progress could not be written as JSON: %w", err)
	}

	return string(text), nil
}

// scoreUnit tells what the value of an objective counts.
func scoreUnit(obj planning.Objective) string {
	switch obj {
	case planning.MinRecruiterSpan, planning.MinCandidateWait, planning.MinEndTimes,
		planning.MinDayEnds, planning.MinSpan:
		return "minutes"
	default:
		return "count"
	}
}

// scoreDirection tells which way is better for an objective.
func scoreDirection(obj planning.Objective) string {
	switch obj {
	case planning.MaxCompleteCandidates, planning.MaxMeetings, planning.MaxPriorities, planning.MaxUrgency:
		return "max"
	default:
		return "min"
	}
}
