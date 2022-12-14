#!/bin/sh

set -e

TMP="$(mktemp -d)"
trap "rm -f -r \"$TMP\"" EXIT INT QUIT TERM

usage() {
    echo "Usage: $0 [-d <dirname>] -p <prefix> -v <known-version>" >&2
    echo "  -d <dirname>       directory where the latest version of Substrate should be installed (defaults to the first writable directory on your PATH)" >&2
    echo "  -p <prefix>        contents of substrate.prefix for a Substrate customer" >&2
    echo "  -v <known-version> version string from any past release of Substrate, even a distant past release, in full YYYY.MM-01234abc form" >&2
    exit "$1"
}

DIRNAME="" PREFIX="" VERSION=""
while [ "$#" -gt 0 ]
do
    case "$1" in
        "-d"|"--dirname") DIRNAME="$2" shift 2;;
        "-d"*) DIRNAME="$(echo "$1" | cut -c"3-")" shift;;
        "--dirname="*) DIRNAME="$(echo "$1" | cut -d"=" -f"2-")" shift;;
        "-p"|"--prefix") PREFIX="$2" shift 2;;
        "-p"*) PREFIX="$(echo "$1" | cut -c"3-")" shift;;
        "--prefix="*) PREFIX="$(echo "$1" | cut -d"=" -f"2-")" shift;;
        "-v"|"--version") VERSION="$2" shift 2;;
        "-v"*) VERSION="$(echo "$1" | cut -c"3-")" shift;;
        "--version="*) VERSION="$(echo "$1" | cut -d"=" -f"2-")" shift;;
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
if [ -z "$DIRNAME" -o -z "$PREFIX" -o -z "$VERSION" ]
then usage 1
fi

# Search for the latest Substrate release by iterating over the upgrade
# pointers until one responds 403 Forbidden (which is
# anti-information-disclosure for 404 Not Found). The one without an upgrade
# available is the latest release.
echo "searching for the latest version of Substrate, starting from $VERSION" >&2
while true
do
    set +e
    curl \
        -f \
        -o"$TMP/upgrade" \
        -s \
        "https://src-bin.com/substrate/upgrade/$VERSION/$PREFIX"
    STATUS="$?"
    set -e
    case "$STATUS" in
        "0") # HTTP 200 OK
            VERSION="$(cat "$TMP/upgrade")";; # keep upgrading
        "22") # HTTP 400+, particularly HTTP 403 Forbidden
            break;; # we've found the latest version
        *) exit "$STATUS";;
    esac
done
echo "found the latest version of Substrate, $VERSION" >&2

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
"$DIRNAME/substrate" -version 2>&1
