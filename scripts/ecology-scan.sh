#!/usr/bin/env bash
# Ecology scan: run samber-linter over every local samber/do v2 consumer and
# print a pseudonymous triage table.
#
# Usage: scripts/ecology-scan.sh [scan-root]   (default: ~/projects)
#
# Output contract:
#   - Project names never appear in the output. Each project becomes
#     "p-" + first 4 hex of sha256 of its absolute path, matching the keyfile
#     written by earlier scans (~/backups/ecology/keyfile.json by default), so
#     scan-to-scan pseudonyms stay stable and comparable.
#   - Rows are ranked by unprotected services (registered - checked), the
#     health-washing exposure the HW-6 ratchet exists to shrink.
#   - Exit 0 even when findings exist: this is a survey, not a gate. Load
#     errors are listed with stderr excerpts, never silently dropped.
#
# Notes:
#   - Building the scanner requires GOEXPERIMENT=jsonv2 (go-finding imports
#     encoding/json/v2); the built binary does not.
#   - A project with a go.work is scanned per workspace module with `./...`:
#     the workspace pattern `all` would expand to the full dependency
#     closure, dragging dependency packages (and their own samber/do
#     registrations) into a survey row the consumer cannot own. `./...`
#     per module covers exactly the consumer's own packages.
#   - Scans run under --check (advisory): baselines are read, never written.
set -euo pipefail

SCAN_ROOT="${1:-$HOME/projects}"
REPO="$(cd "$(dirname "$0")/.." && pwd)"
KEYFILE="${ECOLOGY_KEYFILE:-$HOME/backups/ecology/keyfile.json}"
TIMEOUT_SECS=180

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

echo "building scanner..." >&2
BIN="$work/samber-linter"
(cd "$REPO" && GOEXPERIMENT=jsonv2 CGO_ENABLED=0 go build -o "$BIN" ./cmd/samber-linter)

# Discover candidate modules: a go.mod requiring samber/do v2. Nested modules
# inside an already-accepted project are skipped (outermost module wins).
candidates=()
while IFS= read -r gomod; do
	dir="$(dirname "$gomod")"
	[ "$dir" = "$REPO" ] && continue # never scan the linter with itself here
	grep -q "github.com/samber/do" "$gomod" || continue
	nested=0
	for accepted in ${candidates[@]:-}; do
		case "$dir/" in "$accepted"/*) nested=1; break ;; esac
	done
	[ "$nested" = 0 ] && candidates+=("$dir")
done < <(find "$SCAN_ROOT" -maxdepth 3 -name go.mod -type f 2>/dev/null | sort)

if [ "${#candidates[@]}" -eq 0 ]; then
	echo "no samber/do v2 consumers found under $SCAN_ROOT" >&2
	exit 1
fi

echo "scanning ${#candidates[@]} project(s) under $SCAN_ROOT..." >&2

per_project="$work/per-project.tsv"
scanned=0
load_errors=0

for dir in "${candidates[@]}"; do
	pseudo="p-$(printf %s "$dir" | sha256sum | cut -c1-4)"
	: >"$work/err-$pseudo.txt"

	out=""
	rc=0
	if [ -f "$dir/go.work" ]; then
		mods="$(cd "$dir" && go work edit -json 2>>"$work/err-$pseudo.txt" \
			| jq -r --arg dir "$dir" '.Use[].DiskPath | if startswith("/") then . else $dir + "/" + . end' || true)"
		if [ -z "$mods" ]; then
			rc=2
		else
			while IFS= read -r mod; do
				[ -n "$mod" ] || continue
				part=""
				part_rc=0
				part="$(cd "$mod" && timeout "$TIMEOUT_SECS" "$BIN" --check ./... 2>>"$work/err-$pseudo.txt")" || part_rc=$?
				out+="$part"$'\n'
				[ "$part_rc" -ne 0 ] && rc=$part_rc
				done <<<"$mods"
		fi
	else
		out="$(cd "$dir" && timeout "$TIMEOUT_SECS" "$BIN" --check ./... 2>>"$work/err-$pseudo.txt")" || rc=$?
	fi

	# A module whose packages fail to load does not exit 2: the driver
	# reports per-package errors on stderr and still exits 0 under --check.
	# Counting that as "clean" would make the survey lie, so treat it as a
	# load error and surface the stderr excerpt.
	if grep -q "package error(s) during load" "$work/err-$pseudo.txt" 2>/dev/null; then
		rc=2
	fi

	if [ "$rc" -eq 2 ]; then
		load_errors=$((load_errors + 1))
		status="LOAD_ERROR"
		registered=-1
		checked=-1
		findings=0
		rule_summary=""
	elif [ "$rc" -eq 124 ]; then
		load_errors=$((load_errors + 1))
		status="TIMEOUT"
		registered=-1
		checked=-1
		findings=0
		rule_summary=""
	else
		scanned=$((scanned + 1))
		status="FINDINGS"
		registered=0
		checked=0

		while IFS= read -r counts; do
			[ -n "$counts" ] || continue
			checked=$((checked + ${counts%/*}))
			registered=$((registered + ${counts#*/}))
			done < <(grep -oE 'health-coverage: [0-9]+/[0-9]+' <<<"$out" | sed 's/health-coverage: //')

		findings="$(grep -cE ': (HW-[0-9]|HW-unresolved):' <<<"$out" || true)"
		if [ "$findings" -eq 0 ]; then
			status="CLEAN"
			rule_summary=""
		else
			rule_summary="$(grep -oE ': HW-[0-9a-z-]+:' <<<"$out" | tr -d ' :' | sort | uniq -c \
				| awk '{printf "%s=%s ", $2, $1}')"
		fi
	fi

	printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
		"$pseudo" "$registered" "$checked" "$findings" "$status" "$rule_summary" "$dir" \
		>>"$per_project"
done

# Keyfile: real names live ONLY outside any repository. Merge, never clobber:
# earlier scan statuses are historical evidence.
if [ ! -f "$KEYFILE" ]; then
	mkdir -p "$(dirname "$KEYFILE")"
	echo '{}' >"$KEYFILE"
fi
scan_map="$(jq -Rs '
	split("\n")
	| map(select(length > 0) | split("\t") | {(.[0]): {path: .[6], status: .[4]}})
	| add // {}
' - <"$per_project")"
jq -s '.[0] * .[1]' "$KEYFILE" <(echo "$scan_map") >"$work/keyfile.json"
mv "$work/keyfile.json" "$KEYFILE"

# Report: ranked by unprotected (registered - checked), then findings.
{
	echo
	echo "Ecology scan ($(date +%F)) — root: $SCAN_ROOT"
	echo
	printf '%-4s %-8s %11s %8s %12s %9s  %s\n' \
		"rank" "project" "registered" "checked" "unprotected" "findings" "rules"
	awk -F'\t' '{
		unprot = ($2 < 0) ? -1 : ($2 - $3)
		print unprot "\t" $0
	}' "$per_project" | sort -t$'\t' -k1,1rn -k5,5rn | awk -F'\t' '
		{
			rank++
			if ($3 < 0) {
				printf "%-4d %-8s %11s %8s %12s %9s  %s\n", rank, $2, "-", "-", "-", "-", $6
			} else {
				printf "%-4d %-8s %11d %8d %12d %9d  %s\n", rank, $2, $3, $4, $1, $5, $7
			}
		}
	'

	total_findings="$(awk -F'\t' '{s+=$4} END {print s+0}' "$per_project")"
	clean="$(awk -F'\t' '$5 == "CLEAN"' "$per_project" | wc -l)"
	with_findings="$(awk -F'\t' '$5 == "FINDINGS"' "$per_project" | wc -l)"
	echo
	echo "Aggregate: $scanned analyzed · $clean clean · $with_findings with findings · $total_findings findings total · $load_errors load/timeout error(s)"
	echo "Keyfile updated: $KEYFILE"
} | tee "$work/report.txt"

failed="$(awk -F'\t' '$5 == "LOAD_ERROR" || $5 == "TIMEOUT" {print $1 "\t" $7}' "$per_project")"
if [ -n "$failed" ]; then
	echo >&2
	echo "load/timeout details:" >&2
	while IFS=$'\t' read -r pseudo dir; do
		echo "--- $pseudo ($dir)" >&2
		head -3 "$work/err-$pseudo.txt" | sed 's/^/    /' >&2
	done <<<"$failed"
fi
