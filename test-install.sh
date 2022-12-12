set -e -x

cd "$(dirname "$0")"

TMP="$(mktemp)"
trap "rm -f \"$TMP\"" EXIT INT QUIT TERM

# When this test was written, this was the latest release, so it went directly
# to the loop's termination condition.
sh "install.sh" "src-bin-test1" "2022.12-eb42cd3" | tee "$TMP"

# Prior releases do eventually find their way to the same result, though.
sh "install.sh" "src-bin-test1" "2022.11-d7cce75" | diff -u "-" "$TMP"
sh "install.sh" "src-bin-test1" "2022.10-a9f1d56" | diff -u "-" "$TMP"

echo "ok" >&2
