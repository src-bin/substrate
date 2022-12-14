set -e

cd "$(dirname "$0")"

TMP="$(mktemp -d)"
trap "rm -f -r \"$TMP\"" EXIT INT QUIT TERM

set -x

# When this test was written, this was the latest release, so it went directly
# to the loop's termination condition. This test won't always start at loop
# termination condition, which isn't great, but there's no way to predict the
# necessary argument to the -v option.
sh "install.sh" -d"$TMP/from-2022.12" -p"src-bin-test1" -v"2022.12-eb42cd3" | tee "$TMP/version"

# Ensure prior releases do eventually find their way to the same result, though.
sh "install.sh" -d"$TMP/from-2022.11" -p"src-bin-test1" -v"2022.11-d7cce75" | diff "$TMP/version" "-"
sh "install.sh" -d"$TMP/from-2022.10" -p"src-bin-test1" -v"2022.10-a9f1d56" | diff "$TMP/version" "-"

set +x
echo "ok" >&2
