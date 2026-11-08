#!/bin/sh
# Prints the title of the release of a version: the version, then the name
# that its section of CHANGELOG.md gives after the date, without quotes.
#
#   scripts/title.sh v0.4.0   # ## [0.4.0] - 2026-11-06 "Cinderella" 👠  ->  v0.4.0 Cinderella 👠
set -eu

tag=${1:?usage: scripts/title.sh <version>}
version=${tag#v}
root=$(cd "$(dirname "$0")/.." && pwd)

name=$(awk -v v="$version" '
	index($0, "## [" v "]") == 1 {
		sub(/^## \[[^]]*\]/, "")
		sub(/^ - [0-9][0-9-]*/, "") # the date
		gsub(/"/, "")
		sub(/^ +/, "")
		print
		exit
	}
' "$root/CHANGELOG.md")

if [ -n "$name" ]; then
	echo "$tag $name"
else
	echo "$tag"
fi
