set -e

TMP="$(mktemp)"
trap "rm -f \"$TMP\"" EXIT INT QUIT TERM

# Find out if this is a tagged release. If so, note the version string. If not,
# exit zero because we have no more work to do.
git describe --exact-match --tags "HEAD" >"$TMP" || exit 0

# Ensure we've at least pushed release notes covering this tagged release.
curl -s "https://docs.src-bin.com/substrate/releases" |
grep "<h2 id=\"$(cat "$TMP")\">$(cat "$TMP")</h2>"
