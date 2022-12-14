set -e

cd "$(dirname "$0")"

TMP="$(mktemp -d)"
trap "rm -f -r \"$TMP\"" EXIT INT QUIT TERM

set -x

# Ensure that running install.sh with the actual latest version installs
# that very version.
VERSION="$(git describe --tags HEAD | cut -d"-" -f"1")"
COMMIT="$(git show --format=%h --no-patch "$VERSION")"
sh "install.sh" -d"$TMP/from-latest" -p"src-bin-test1" -v"$VERSION-$COMMIT" | tee "$TMP/latest"

# Ensure prior releases eventually find their way to the same latest version.
sh "install.sh" -d"$TMP/from-2022.12" -p"src-bin-test1" -v"2022.12-eb42cd3" | diff "$TMP/latest" "-"
sh "install.sh" -d"$TMP/from-2022.11" -p"src-bin-test1" -v"2022.11-d7cce75" | diff "$TMP/latest" "-"
sh "install.sh" -d"$TMP/from-2022.10" -p"src-bin-test1" -v"2022.10-a9f1d56" | diff "$TMP/latest" "-"

set +x
echo "ok" >&2
