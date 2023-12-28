set -e

cd "$(dirname "$0")"

TMP="$(mktemp -d)"
trap "rm -f -r \"$TMP\"" EXIT INT QUIT TERM

set -x

sh "install.sh" -d"$TMP/bin"

curl -f -o"$TMP/version" -s "https://src-bin.com/substrate.version"
"$TMP/bin/substrate" --version | grep -q "$(cat "$TMP/version")"

set +x
echo "ok" >&2
