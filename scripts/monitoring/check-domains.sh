#!/usr/bin/env bash
set -euo pipefail

# External uptime + certificate check for one or more public hostnames.
#
# Usage: scripts/monitoring/check-domains.sh <host> [host...]
#    or: VICTORY_CHECK_DOMAINS="a.example b.example" scripts/monitoring/check-domains.sh
#
# Exists because of a real incident (2026-10-04): a subdomain's A/AAAA
# records pointed at a Hetzner IP that had been released and reassigned to
# an unrelated customer. The app itself was healthy on its own box the
# whole time and the reverse proxy was configured correctly -- there was
# simply no DNS aimed at it, so every visitor got a TLS name mismatch.
# Nothing surfaced it for however long it had been that way; it was found
# by a stray curl during an unrelated deploy. This script is the thing
# that should have caught it.
#
# Deliberately checks the two failure modes that are invisible from the
# server's own point of view:
#
#   1. Does the name, resolved publicly, actually serve this site?
#      A dangling record, a hijacked IP, or a proxy misconfiguration all
#      fail here while every on-box health check stays green.
#   2. Does the certificate still have life left in it?
#      Automatic renewal failing silently looks completely fine until the
#      day it expires. CERT_WARN_DAYS is the early warning.
#
# RUN THIS FROM OUTSIDE THE SERVER IT MONITORS. Run on-box it cannot
# detect the box being down, and its DNS/routing may not reflect what a
# real visitor sees -- during the incident above, the affected host timed
# out from the server itself but connected instantly from anywhere else,
# which sent the diagnosis down a false path for a while.
#
# Hostnames are arguments, never hardcoded: this script ships in the repo
# and must carry no particular deployment's domains.
#
# Exit status: 0 all checks passed, 1 at least one failed. Prints one line
# per host. Intended to be quiet enough to run unattended and loud enough
# to notice when a timer mails the output.

CERT_WARN_DAYS="${CERT_WARN_DAYS:-14}"
TIMEOUT="${TIMEOUT:-15}"
EXPECT_STATUS="${EXPECT_STATUS:-200}"

domains=("$@")
if [[ ${#domains[@]} -eq 0 ]]; then
  # shellcheck disable=SC2206
  domains=(${VICTORY_CHECK_DOMAINS:-})
fi
if [[ ${#domains[@]} -eq 0 ]]; then
  echo "usage: $0 <host> [host...]   (or set VICTORY_CHECK_DOMAINS)" >&2
  exit 2
fi

failed=0
stamp="$(date -Is)"

for host in "${domains[@]}"; do
  [[ -z "$host" ]] && continue

  status="$(curl -s -m "$TIMEOUT" -o /dev/null -w '%{http_code}' "https://${host}/" 2>/dev/null || true)"
  # curl writes 000 on any pre-response failure: DNS, connect, TLS. Those
  # are exactly the interesting cases, so report the reason rather than
  # just the code.
  if [[ "$status" != "$EXPECT_STATUS" ]]; then
    reason="$(curl -sS -m "$TIMEOUT" -o /dev/null "https://${host}/" 2>&1 | head -1 || true)"
    echo "${stamp} FAIL ${host} http=${status:-none} ${reason}"
    failed=1
    continue
  fi

  # Certificate expiry. Checked only once the host is known to answer, so
  # a hard-down host produces one clear failure rather than two.
  not_after="$(echo \
    | timeout "$TIMEOUT" openssl s_client -connect "${host}:443" -servername "$host" 2>/dev/null \
    | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2 || true)"

  if [[ -z "$not_after" ]]; then
    echo "${stamp} WARN ${host} http=${status} cert=unreadable"
    failed=1
    continue
  fi

  expiry_epoch="$(date -d "$not_after" +%s 2>/dev/null || echo 0)"
  if [[ "$expiry_epoch" -eq 0 ]]; then
    echo "${stamp} WARN ${host} http=${status} cert_expiry_unparseable='${not_after}'"
    failed=1
    continue
  fi

  days_left=$(( (expiry_epoch - $(date +%s)) / 86400 ))
  if [[ "$days_left" -lt "$CERT_WARN_DAYS" ]]; then
    echo "${stamp} FAIL ${host} http=${status} cert_expires_in=${days_left}d (threshold ${CERT_WARN_DAYS}d)"
    failed=1
    continue
  fi

  echo "${stamp} OK   ${host} http=${status} cert_expires_in=${days_left}d"
done

exit "$failed"
