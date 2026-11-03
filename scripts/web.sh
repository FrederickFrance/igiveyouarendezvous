#!/bin/sh
# Builds the online demo into a folder (dist/web by default), ready to be
# served as static files, by GitHub Pages for instance: the page (web/), the
# program compiled to WebAssembly, the JavaScript glue of the same Go
# version, and the examples.
#
#   scripts/web.sh [folder]
#   python3 -m http.server -d dist/web   # then open http://localhost:8000
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
out=${1:-$root/dist/web}

rm -rf "$out"
mkdir -p "$out/examples/fr"

cp "$root"/web/* "$out/"
cp "$(cd "$root" && go env GOROOT)/lib/wasm/wasm_exec.js" "$out/"

(cd "$root" && GOOS=js GOARCH=wasm go build -trimpath -ldflags "-s -w" \
	-o "$out/igiveyouarendezvous.wasm" ./cmd/igiveyouarendezvous)

# The examples, in English and in French; their schema comment points into
# the repository.
for example in problem simple panel show colloquium roleplay; do
	for lang in . fr; do
		sed 's#\$schema=\(\.\./\)*schema/#$schema=https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/#' \
			"$root/example/$lang/$example.yaml" > "$out/examples/$lang/$example.yaml"
	done
done

ls -l "$out"
