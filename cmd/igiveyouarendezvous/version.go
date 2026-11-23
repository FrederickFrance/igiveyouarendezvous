package main

import "fmt"

// The version of the program, its code name and its mark. They are written
// here rather than taken from the build so that the WebAssembly demo, which
// is built from a working tree and not from a tag, shows the same thing as
// the command: somebody who only ever uses the demo has no other way of
// seeing that a new version is out.
//
// TestVersionMatchesTheChangelog keeps them equal to the first section of
// CHANGELOG.md: raising one and forgetting the other fails the tests.
const (
	version     = "0.17.0"
	releaseName = "X-Or, le shérif de l'espace"
	releaseMark = "⚡"
)

// release is the version as the release page writes it: v0.15.0 "Bullseye" 🎯.
func release() string {
	return fmt.Sprintf("v%s %q %s", version, releaseName, releaseMark)
}
