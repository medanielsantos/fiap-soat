#!/usr/bin/env bash
set -euo pipefail

COVERAGE_MIN="${COVERAGE_MIN:-85}"
PROFILE="${COVERAGE_PROFILE:-coverage.out}"
MODULE="$(go list -m)"
SCOPE="${COVERAGE_SCOPE:-./cmd/... ./internal/...}"
TESTS="${COVERAGE_TESTS:-./...}"

pkgs="$(go list $SCOPE | grep -v '/internal/platform/testdb$' | paste -sd, -)"

go test -p 1 -count=1 -covermode=atomic -coverpkg="$pkgs" -coverprofile="$PROFILE.raw" $TESTS

awk 'NR == 1 { print; next } { n[$1] = $2; if (!($1 in c) || $3 > c[$1]) c[$1] = $3 }
     END { for (k in n) print k, n[k], c[k] }' "$PROFILE.raw" > "$PROFILE"
rm -f "$PROFILE.raw"

awk -v min="$COVERAGE_MIN" -v module="$MODULE/" '
  NR == 1 { next }
  {
    split($1, loc, ":"); pkg = loc[1]; sub(/\/[^\/]+$/, "", pkg); sub(module, "", pkg)
    total[pkg] += $2; all += $2
    if ($3 > 0) { covered[pkg] += $2; hit += $2 }
  }
  END {
    failed = 0
    for (pkg in total) {
      pct = 100 * covered[pkg] / total[pkg]
      flag = pct < min ? "FAIL" : "ok"
      if (pct < min) failed = 1
      printf "%-4s %6.1f%%  %s\n", flag, pct, pkg | "sort -k3"
    }
    close("sort -k3")
    pct = 100 * hit / all
    printf "\ntotal %.1f%% (minimum %s%% per package and overall)\n", pct, min
    if (pct < min) failed = 1
    exit failed
  }' "$PROFILE"
