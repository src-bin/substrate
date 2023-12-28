#!/bin/sh

set -e

TMP="$(mktemp -d)"
trap "rm -f -r \"$TMP\"" EXIT INT QUIT TERM

usage() {
    echo "Usage: $0 [-d <dirname>]" >&2
    echo "  -d <dirname> directory where the latest version of Substrate should be installed (defaults to the first writable directory on your PATH)" >&2
    exit "$1"
}

DIRNAME="" PREFIX="" VERSION=""
while [ "$#" -gt 0 ]
do
    case "$1" in
        "-d"|"--dirname") DIRNAME="$2" shift 2;;
        "-d"*) DIRNAME="$(echo "$1" | cut -c"3-")" shift;;
        "--dirname="*) DIRNAME="$(echo "$1" | cut -d"=" -f"2-")" shift;;
        "-h"|"--help") usage 0;;
        *) usage 1;;
    esac
done
if [ -z "$DIRNAME" ]
then
    echo "$PATH" | tr ":" "\n" >"$TMP/path"
    while read D
    do
        if [ -d "$D" -a -w "$D" ]
        then DIRNAME="$D" break
        fi
    done <"$TMP/path"
fi
if [ -z "$DIRNAME" ]
then usage 1
fi

# Find the latest version of Substrate.
curl -f -o"$TMP/version" -s "https://src-bin.com/substrate.version"
VERSION="$(cat "$TMP/version")"
echo "the latest version of Substrate is $VERSION" >&2

# Download the latest Substrate release for this OS and architecture.
ARCH="$(uname -m)"
case "$ARCH" in
    "aarch64") ARCH="arm64";;
    "x86_64") ARCH="amd64";;
esac
OS="$(uname -s | tr "[:upper:]" "[:lower:]")"
echo "downloading <https://src-bin.com/substrate-$VERSION-$OS-$ARCH.tar.gz>" >&2
curl \
    -f \
    -o"$TMP/substrate-$VERSION-$OS-$ARCH.tar.gz" \
    -s \
    "https://src-bin.com/substrate-$VERSION-$OS-$ARCH.tar.gz"

# Untar just the binary from the latest Substrate release tarball.
echo "untarring $TMP/substrate-$VERSION-$OS-$ARCH.tar.gz" >&2
mkdir -p "$DIRNAME"
tar \
    -C"$DIRNAME" \
    -f"$TMP/substrate-$VERSION-$OS-$ARCH.tar.gz" \
    --strip-components 2 \
    -x \
    "substrate-$VERSION-$OS-$ARCH/bin/substrate"
echo "installed $DIRNAME/substrate" >&2

# Prove Substrate's installed and let it declare its version.
"$DIRNAME/substrate" --version 2>&1
