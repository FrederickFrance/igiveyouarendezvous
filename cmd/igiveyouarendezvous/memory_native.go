//go:build !(js && wasm)

package main

import (
	"os"
	"runtime/debug"
)

// softLimit is how much memory the command lets itself use before the
// garbage collector is made to work for it, and gcPercent how much the heap
// may grow under that ceiling.
//
// The search allocates heavily, so by default a collection runs every few
// milliseconds and costs about a quarter of its speed. A ceiling buys that
// back. Measured on a forum of 604 meetings in one day, the whole of it
// re-timed on every move: a third more moves a second, for a peak of 58 MiB
// instead of 13.
//
// A ceiling and not a bigger ratio, because the ratio is a multiple of what
// is live and what is live here is small — a few megabytes of plans beside
// an enormous churn. Asking for eight times that reached the same speed and
// 168 MiB, which is a lot of memory to spend on a number that says nothing
// about this problem. A ceiling says what it means.
const (
	softLimit = 64 << 20
	gcPercent = 400
)

// tuneMemory gives the garbage collector a ceiling to work to, unless the
// person has already said what they want: their GOGC or GOMEMLIMIT wins,
// always, since they know what else is running on their machine and this
// program does not.
//
// Not built for the demo, where the browser tab decides what it may use and
// the page is not ours to spend.
func tuneMemory() {
	if os.Getenv("GOGC") != "" || os.Getenv("GOMEMLIMIT") != "" {
		return
	}

	debug.SetMemoryLimit(softLimit)
	debug.SetGCPercent(gcPercent)
}
