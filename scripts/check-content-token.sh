#!/bin/sh
# Compare PLUGPIN_CONTENT_TOKEN with policy/content-token.sha256 and search the tree.
# The token is not stored in the repository.
set -eu

token=${PLUGPIN_CONTENT_TOKEN:?set PLUGPIN_CONTENT_TOKEN locally}
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
want=$(awk '/^[0-9a-f]{64}$/ {print; exit}' "$root/policy/content-token.sha256")
if [ -z "$want" ]; then
	echo "policy hash missing" >&2
	exit 1
fi

hash_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		printf '%s' "$1" | sha256sum | awk '{print $1}'
	else
		printf '%s' "$1" | shasum -a 256 | awk '{print $1}'
	fi
}

got=$(hash_of "$token")
if [ "$got" != "$want" ]; then
	echo "token does not match policy/content-token.sha256" >&2
	exit 1
fi

cd "$root"
found=0
set +e
git grep -I -i -n -F -e "$token" HEAD -- . >/tmp/plugpin-content-hits 2>/dev/null
status=$?
set -e
if [ "$status" -eq 0 ]; then
	echo "banned substring found in tracked content" >&2
	found=1
elif [ "$status" -gt 1 ]; then
	echo "git grep failed: $status" >&2
	exit "$status"
fi

set +e
git ls-files | grep -i -F -e "$token" >/dev/null
name_status=$?
set -e
if [ "$name_status" -eq 0 ]; then
	echo "banned substring found in a file name" >&2
	found=1
fi

if [ "$found" -ne 0 ]; then
	exit 1
fi
echo "content token: tree is clear"
