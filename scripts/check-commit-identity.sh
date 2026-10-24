#!/bin/sh
# Reject commits whose author or committer is not the repository owner,
# and reject attribution trailers such as Co-authored-by.
# The preexisting parentless root commit is skipped: it predates this policy
# and is not rewritten.
set -eu

name="Ali Kesik"
email="alikesk222@users.noreply.github.com"

event="${EVENT_NAME:-local}"
if [ "$event" = "pull_request" ]; then
	range="${BASE_SHA:?}..${HEAD_SHA:?}"
elif [ -n "${RANGE_BEFORE:-}" ] && [ "$RANGE_BEFORE" != "0000000000000000000000000000000000000000" ]; then
	range="${RANGE_BEFORE}..${RANGE_SHA:?}"
elif [ -n "${RANGE_SHA:-}" ]; then
	range="$RANGE_SHA"
else
	range="${1:-HEAD}"
fi

commits=$(git rev-list "$range")
if [ -z "$commits" ]; then
	echo "commit-policy: no commits in $range"
	exit 0
fi

fail=0
for c in $commits; do
	if ! git rev-parse --verify --quiet "$c^" >/dev/null; then
		echo "commit-policy: skip parentless root $c"
		continue
	fi
	an=$(git log -1 --format='%an' "$c")
	ae=$(git log -1 --format='%ae' "$c")
	cn=$(git log -1 --format='%cn' "$c")
	ce=$(git log -1 --format='%ce' "$c")
	if [ "$an" != "$name" ] || [ "$ae" != "$email" ] || [ "$cn" != "$name" ] || [ "$ce" != "$email" ]; then
		echo "commit-policy: $c author '$an <$ae>' committer '$cn <$ce>'" >&2
		fail=1
	fi
	body=$(git log -1 --format='%B' "$c")
	if printf '%s\n' "$body" | grep -E -i -q '^[[:space:]]*[A-Za-z0-9-]+-by:'; then
		echo "commit-policy: $c contains an attribution trailer" >&2
		fail=1
	fi
done

if [ "$fail" -ne 0 ]; then
	exit 1
fi
echo "commit-policy: $range ok"
