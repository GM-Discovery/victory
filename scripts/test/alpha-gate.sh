#!/usr/bin/env bash
set -uo pipefail

# Kernel 70A alpha-gate: orchestrates existing tools only (no new checks
# invented here) and prints an explicit PASS/FAIL line per step, so a
# skimmed run still shows what actually happened. This script intentionally
# does NOT run browser/manual visual-acceptance proof -- see the final
# summary line and Kernel 70A's manual checklist (kernel doc SS11) for that.
#
# Usage:
#   TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
#     scripts/test/alpha-gate.sh

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKEND_DIR="$ROOT/backend"

# Test titles from the nine pre-existing dice.test.js failures
# (Construction/Canon/roadmap.md's tracked, non-blocking exception). Kernel 72
# merged the mirrored first-theater/catharsis suites into tests/stage-runtime,
# so the nine titles appear once now, not twice. Every one of these titles
# starts with "dice tray"; nothing else in the suite does, so that prefix is
# used below as the allowlist check.
KNOWN_DICE_FAILURE_PREFIX="dice tray"
KNOWN_DICE_FAILURE_COUNT=9

STEP_RESULTS=()
OVERALL_PASS=1

record_step() {
  local name="$1"
  local status="$2"
  STEP_RESULTS+=("$status  $name")
  # PASS (with tracked exception) is a pass for gating purposes -- only the
  # dice.test.js exception failed, and that's tracked/expected, not a
  # regression. The manual/browser step is deliberately never PASS/FAIL --
  # it's a reminder, not an automated result -- so it never affects gating.
  case "$status" in
    PASS|"PASS (with tracked exception)"|"PENDING (not run by this script)") ;;
    *) OVERALL_PASS=0 ;;
  esac
}

echo "=== Kernel 70A alpha-gate ==="
echo

# --- Step 0: prerequisite check -------------------------------------------
if [[ -z "${TEST_DATABASE_URL:-}" ]]; then
  echo "FAIL: TEST_DATABASE_URL is required (see usage above)." >&2
  record_step "TEST_DATABASE_URL set" "FAIL"
  echo
  echo "=== Summary ==="
  printf '%s\n' "${STEP_RESULTS[@]}"
  exit 1
fi
record_step "TEST_DATABASE_URL set" "PASS"

# --- Step 1: test database setup -------------------------------------------
echo "--- Step 1/7: scripts/test/setup-test-database.sh ---"
if bash "$ROOT/scripts/test/setup-test-database.sh"; then
  record_step "Test database setup" "PASS"
else
  record_step "Test database setup" "FAIL"
fi
echo

# --- Step 2: go build / go vet ----------------------------------------------
echo "--- Step 2/7: go build ./... && go vet ./... ---"
if (cd "$BACKEND_DIR" && go build ./... && go vet ./...); then
  record_step "go build / go vet" "PASS"
else
  record_step "go build / go vet" "FAIL"
fi
echo

# --- Step 3: go test ----------------------------------------------------
# Kernel 101 (101-16): -p 1 forces Go to test one package at a time rather
# than its default (one test binary per package, up to NumCPU running
# concurrently). Several packages (shows, showtime, and others touching
# real Show/Session state) all start real Sessions on the one, real,
# shared "catharsis" venue this test database seeds -- there is no
# disposable per-test venue for that, since the venue slug is hardcoded in
# production code (scenes/capture.go's liveBridgeVenueSlug and friends).
# Confirmed empirically: every affected package passes cleanly alone or
# with -p 1 on a fresh database; the same combination run with Go's
# default concurrency reproduced a real, reliable (3/3 attempts) race on
# catharsis's one-live-session-at-a-time constraint, with two different
# packages' test binaries both starting a live session on it at genuinely
# the same moment. Not a bug in any individual test -- every one already
# cleans up correctly -- purely a consequence of shared, unavoidably
# global fixture state under true concurrent execution. Slower, but this
# gate should never be flaky.
echo "--- Step 3/7: go test -p 1 -count=1 ./... ---"
if (cd "$BACKEND_DIR" && TEST_DATABASE_URL="$TEST_DATABASE_URL" go test -p 1 -count=1 ./...); then
  record_step "go test ./..." "PASS"
else
  record_step "go test ./..." "FAIL"
fi
echo

# --- Step 4: Node test suites (shared stage engine + contract + ewrite + storyboards) ----
echo "--- Step 4/7: node --test tests/stage-runtime tests/contract tests/ewrite tests/storyboards ---"
NODE_TEST_LOG="$(mktemp)"
trap 'rm -f "$NODE_TEST_LOG"' EXIT
node --test "$ROOT"/tests/stage-runtime/*.test.js "$ROOT"/tests/contract/*.test.js "$ROOT"/tests/ewrite/*.test.js "$ROOT"/tests/storyboards/*.test.js >"$NODE_TEST_LOG" 2>&1
node_test_exit=$?

# Every failing test's title, one per line, from the TAP "not ok N - <title>" lines.
failing_titles="$(grep -E '^not ok [0-9]+ - ' "$NODE_TEST_LOG" | sed -E 's/^not ok [0-9]+ - //')"
failing_count=0
unexpected_count=0
if [[ -n "$failing_titles" ]]; then
  failing_count="$(printf '%s\n' "$failing_titles" | wc -l | tr -d ' ')"
  unexpected_count="$(printf '%s\n' "$failing_titles" | grep -vc "^${KNOWN_DICE_FAILURE_PREFIX}" || true)"
fi

echo "Node test run exit code: $node_test_exit"
echo "Total failing tests: $failing_count (expected exactly $KNOWN_DICE_FAILURE_COUNT known '$KNOWN_DICE_FAILURE_PREFIX' failures, tracked in Construction/Canon/roadmap.md)"
if [[ "$unexpected_count" -gt 0 ]]; then
  echo "UNEXPECTED failing tests (not the tracked dice exception):"
  printf '%s\n' "$failing_titles" | grep -v "^${KNOWN_DICE_FAILURE_PREFIX}" || true
fi

if [[ "$unexpected_count" -eq 0 && "$failing_count" -eq "$KNOWN_DICE_FAILURE_COUNT" ]]; then
  echo "Node suite result: only the tracked, non-blocking dice.test.js exception failed ($failing_count/$failing_count)."
  record_step "Node test suites (stage-runtime/contract)" "PASS (with tracked exception)"
elif [[ "$unexpected_count" -eq 0 && "$failing_count" -ne "$KNOWN_DICE_FAILURE_COUNT" ]]; then
  echo "Node suite result: dice.test.js failure count drifted from the tracked $KNOWN_DICE_FAILURE_COUNT (now $failing_count) -- update the tracked count in this script and Construction/Canon/roadmap.md if this is an intentional partial fix or new break, and confirm which before treating this as a pass."
  record_step "Node test suites (stage-runtime/contract)" "FAIL"
else
  echo "Node suite result: unexpected failures outside the tracked dice.test.js exception."
  record_step "Node test suites (stage-runtime/contract)" "FAIL"
fi
echo

# --- Step 5: fresh install smoke test ---------------------------------------
echo "--- Step 5/7: scripts/smoke/fresh-install.sh --local ---"
if bash "$ROOT/scripts/smoke/fresh-install.sh" --local; then
  record_step "Fresh install smoke test" "PASS"
else
  record_step "Fresh install smoke test" "FAIL"
fi
echo

# --- Step 6: git diff --check (whitespace/conflict-marker hygiene) --------
echo "--- Step 6/7: git diff --check ---"
if (cd "$ROOT" && git diff --check); then
  record_step "git diff --check" "PASS"
else
  record_step "git diff --check" "FAIL"
fi
echo

# --- Step 7: manual/browser proof reminder ----------------------------------
echo "--- Step 7/7: manual/browser visual-acceptance proof ---"
echo "NOT RUN by this script. Kernel 70A's manual checklist (Director start/GO/"
echo "end/restart persistence, Player auto-resolved theater context and empty-"
echo "state messaging, Audience no-leakage/forged-Cue-rejection) must still be"
echo "walked by hand, with screenshots where browser automation is available."
record_step "Manual/browser visual-acceptance proof" "PENDING (not run by this script)"
echo

echo "=== Summary ==="
printf '%s\n' "${STEP_RESULTS[@]}"
echo

if [[ "$OVERALL_PASS" -eq 1 ]]; then
  echo "Overall: automated steps PASS. Manual/browser checklist still pending -- see Step 7."
  exit 0
else
  echo "Overall: FAIL -- see the steps marked FAIL above."
  exit 1
fi
