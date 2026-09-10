#!/usr/bin/env bash
# Compile and run every Go snippet under docs/upstream.
#
# Contract for markdown authors:
#   - A ```go block that is a full self-contained program (starts with
#     "package ") is compiled and executed verbatim; a non-zero exit fails
#     the check.
#   - A ```go block quoting upstream source (not standalone-compilable) must
#     say so in the fence info string: ```go snippet-skip
#   - Anything else fails: an unclassified snippet is exactly how a broken
#     repro gets filed.
#
# Network note: each snippet becomes a temp module requiring samber/do;
# `go mod tidy` reads the local module cache first and only hits the proxy
# on a miss. Hermetic nix checks are therefore intentionally not wired to
# this script; GitHub CI is.
set -euo pipefail

cd "$(dirname "$0")/.."

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

checked=0
failed=0

for md in docs/upstream/*.md; do
	[ -e "$md" ] || continue

	base="$(basename "$md" .md)"
	awk -v dir="$work" -v base="$base" '
		function flush(idx, startn,    f, i) {
			f = sprintf("%s/%s.%03d.go", dir, base, idx)
			for (i = 1; i <= n; i++) print lines[i] > f
			printf "%s\n", info > (f ".info")
			printf "%s:%d\n", srcname, startn > (f ".loc")
			close(f); close(f ".info"); close(f ".loc")
		}
		BEGIN { inb = 0; idx = 0; n = 0 }
		/^```/ {
			if (inb) { flush(idx, start); idx++; n = 0; inb = 0 }
			else {
				info = substr($0, 4)
				if (info ~ /^go( |$)/) { inb = 1; start = NR; n = 0 }
			}
			next
		}
		inb { lines[++n] = $0 }
	' srcname="$md" "$md"
done

for gofile in "$work"/*.go; do
	[ -e "$gofile" ] || continue

	loc="$(cat "$gofile.loc")"
	info="$(cat "$gofile.info")"

	if [[ "$info" == *"snippet-skip"* ]]; then
		echo "SKIP  $loc (snippet-skip)"
		continue
	fi

	if ! grep -q '^package ' "$gofile"; then
		echo "FAIL  $loc: go block neither starts with 'package ' nor is marked snippet-skip"
		failed=$((failed + 1))
		continue
	fi

	mod="$work/mod$(basename "$gofile" .go)"
	mkdir -p "$mod"
	cp "$gofile" "$mod/main.go"
	cat > "$mod/go.mod" <<EOF
module snippet/local

go 1.26

require github.com/samber/do/v2 v2.1.0
EOF

	if (
		cd "$mod" &&
			GOFLAGS=-mod=mod go mod tidy >/dev/null 2>&1 &&
			GOEXPERIMENT=jsonv2 CGO_ENABLED=0 go run .
	); then
		echo "PASS  $loc"
		checked=$((checked + 1))
	else
		echo "FAIL  $loc: snippet did not compile/run cleanly"
		failed=$((failed + 1))
	fi
done

echo "----"
if [ "$failed" -gt 0 ]; then
	echo "snippet check: $failed failed (see above)"
	exit 1
fi
echo "snippet check: $checked passed"
