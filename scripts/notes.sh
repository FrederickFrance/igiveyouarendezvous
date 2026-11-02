#!/bin/sh
# Prints the section of a version in CHANGELOG.md, the notes of its release;
# nothing when the changelog has no such section.
#
#   scripts/notes.sh v0.2.0
set -eu

version=${1:?usage: scripts/notes.sh <version>}
version=${version#v}
root=$(cd "$(dirname "$0")/.." && pwd)

awk -v v="$version" '
	/^## \[/ { in_section = index($0, "## [" v "]") == 1; next }
	/^\[[^]]*\]: / { next } # the links at the end
	in_section
' "$root/CHANGELOG.md" | sed -e '/./,$!d'
