#!/bin/sh
# Builds the ready-to-use zips of a release, one per system, into dist/:
# the program, the problem file of the beginner's tutorial (problem.yaml),
# the full commented example (example-full.yaml), the examples of a panel
# and of other uses (example-panel.yaml, example-show.yaml…), and all of
# them in French (exemples-fr/).
#
#   scripts/package.sh v0.1.0
set -eu

version=${1:?usage: scripts/package.sh <version>}
root=$(cd "$(dirname "$0")/.." && pwd)
dist=$root/dist

rm -rf "$dist"
mkdir -p "$dist"

# name of the zip, GOOS, GOARCH
for target in windows:windows:amd64 mac-apple:darwin:arm64 mac-intel:darwin:amd64 linux:linux:amd64; do
	name=${target%%:*}
	rest=${target#*:}
	goos=${rest%%:*}
	goarch=${rest#*:}

	program=igiveyouarendezvous
	if [ "$goos" = windows ]; then
		program=igiveyouarendezvous.exe
	fi

	dir=$dist/igiveyouarendezvous-$version-$name
	mkdir -p "$dir"

	(cd "$root" && CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch \
		go build -trimpath -ldflags "-s -w" -o "$dir/$program" ./cmd/igiveyouarendezvous)

	cp "$root/example/simple.yaml" "$dir/problem.yaml"
	cp "$root/example/problem.yaml" "$dir/example-full.yaml"
	for other in panel show colloquium roleplay; do # a panel, and the other uses
		cp "$root/example/$other.yaml" "$dir/example-$other.yaml"
	done
	mkdir "$dir/exemples-fr"
	cp "$root"/example/fr/*.yaml "$dir/exemples-fr/"
	cp "$root/LICENSE" "$root/WTFPL.txt" "$dir/"

	# The schema comment of the examples points into the repository.
	sed -i.bak 's#\$schema=\(\.\./\)*schema/#$schema=https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/#' \
		"$dir"/*.yaml "$dir"/exemples-fr/*.yaml
	rm "$dir"/*.bak "$dir"/exemples-fr/*.bak

	(cd "$dist" && zip -qr "igiveyouarendezvous-$name.zip" "igiveyouarendezvous-$version-$name")
	rm -rf "$dir"
done

ls -l "$dist"
