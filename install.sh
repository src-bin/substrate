#!/bin/sh

set -e

usage() {
    echo "Usage: $0 <prefix> <known-version>" >&2
    echo "  <prefix>        contents of substrate.prefix for a once-paying Substrate customer" >&2
    echo "  <known-version> version string from any past release of Substrate, even a distant past release, in full YYYY.MM-01234abc form" >&2
    exit "$1"
}

while [ "$#" -gt 0 ]
do
    case "$1" in
        "-h"|"--help") usage 0;;
        *) break;;
    esac
done
if [ -z "${PREFIX:="$1"}" ]
then usage 1
fi
if [ -z "${VERSION:="$2"}" ]
then usage 1
fi

TMP="$(mktemp)"
trap "rm -f \"$TMP\"" EXIT INT QUIT TERM

while true
do
    set +e
    curl -f -o"$TMP" -s "https://src-bin.com/substrate/upgrade/$VERSION/$PREFIX"
    STATUS="$?"
    set -e
    case "$STATUS" in
        "0") # HTTP 200 OK
            VERSION="$(cat "$TMP")";;
        "22") # HTTP 400+, particularly HTTP 403 Forbidden
            break;; # we've found the latest version
        *) exit "$STATUS";;
    esac
done

ARCH="$(uname -m)"
case "$ARCH" in
    "aarch64") ARCH="arm64";;
    "x86_64") ARCH="amd64";;
esac
OS="$(uname -s | tr "[:upper:]" "[:lower:]")"
echo "https://src-bin.com/substrate-$VERSION-$OS-$ARCH.tar.gz" # TODO install this version
