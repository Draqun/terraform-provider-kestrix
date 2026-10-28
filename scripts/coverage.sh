#!/usr/bin/env bash
set -Eeuo pipefail
# Enforce the 100% statement gate on the provider package. main.go is the plugin
# entry point: it is exercised as a real process by scripts/protocol.sh, which
# Go's coverage runtime cannot attribute to this profile.
export LC_ALL=C
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"
go test ./internal/... -count=1 -covermode=atomic -coverprofile=coverage.out
grep -vF '/terraform-provider-kestrix/main.go' coverage.out > coverage.testable.out
mv coverage.testable.out coverage.out
read -r covered total percent < <(awk '
  /^mode:/ { next }
  { n = $2 + 0; if (n == 0) next; total += n; if ($3 + 0 > 0) cov += n }
  END { if (total == 0) print "0 0 0"; else printf "%d %d %.1f\n", cov, total, (cov / total) * 100 }
' coverage.out)
echo "Coverage: ${percent}% (${covered}/${total} statements)"
if [[ "${covered}" != "${total}" ]]; then
  echo "FAIL: uncovered statements remain:" >&2
  awk 'NR > 1 && $3 == 0 { print "  " $1, "statements=" $2 }' coverage.out >&2
  exit 1
fi
echo "✓ Coverage gate met"
