#!/bin/sh
# Install a local pre-push hook. The hook is written under .git and is not committed.
# Usage: PLUGPIN_CONTENT_TOKEN=... sh scripts/install-local-hook.sh
set -eu

token=${PLUGPIN_CONTENT_TOKEN:?set PLUGPIN_CONTENT_TOKEN locally}
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
# shellcheck disable=SC1091
# Reuse the hash check by invoking the checker after the hook exists. First compare here.
want=$(awk '/^[0-9a-f]{64}$/ {print; exit}' "$root/policy/content-token.sha256")
if command -v sha256sum >/dev/null 2>&1; then
	got=$(printf '%s' "$token" | sha256sum | awk '{print $1}')
else
	got=$(printf '%s' "$token" | shasum -a 256 | awk '{print $1}')
fi
if [ "$got" != "$want" ]; then
	echo "token does not match policy/content-token.sha256" >&2
	exit 1
fi

hook="$root/.git/hooks/pre-push"
cat >"$hook" <<EOF
#!/bin/sh
set -eu
token='$token'
cd "$root"
set +e
git grep -I -i -n -F -e "\$token" HEAD -- . >/dev/null 2>&1
status=\$?
set -e
if [ "\$status" -eq 0 ]; then
	echo "pre-push: banned substring is in the tree" >&2
	exit 1
fi
while read -r local_ref local_sha remote_ref remote_sha; do
	if [ "\$local_sha" = "0000000000000000000000000000000000000000" ]; then
		continue
	fi
	if [ "\$remote_sha" = "0000000000000000000000000000000000000000" ]; then
		range="\$local_sha"
	else
		range="\$remote_sha..\$local_sha"
	fi
	RANGE_BEFORE="\$remote_sha" RANGE_SHA="\$local_sha" EVENT_NAME=push \\
		sh "$root/scripts/check-commit-identity.sh" "\$range"
	set +e
	git log --format='%B%n%an%n%ae%n%cn%n%ce' "\$range" | grep -i -F -e "\$token" >/dev/null
	hit=\$?
	set -e
	if [ "\$hit" -eq 0 ]; then
		echo "pre-push: banned substring is in commit metadata" >&2
		exit 1
	fi
done
exit 0
EOF
chmod 0755 "$hook"
echo "installed $hook"
