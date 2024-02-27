set -e -x

VERSION="$(cat "substrate.version" || :)"
if [ -z "$VERSION" ]
then exit 0 # nothing to do for untagged versions
fi

cat >"substrate.download.html" <<EOF
<table border="0" cellpadding="0" cellspacing="16">
    <tr>
        <th>macOS (Intel)</th>
        <td><a href="https://src-bin.com/substrate-$VERSION-darwin-amd64.tar.gz">substrate-$VERSION-darwin-amd64.tar.gz</a></td>
    </tr>
    <tr>
        <th>macOS (Apple Silicon)</th>
        <td><a href="https://src-bin.com/substrate-$VERSION-darwin-arm64.tar.gz">substrate-$VERSION-darwin-arm64.tar.gz</a></td>
    </tr>
    <tr>
        <th>Linux (Intel/AMD)</th>
        <td><a href="https://src-bin.com/substrate-$VERSION-linux-amd64.tar.gz">substrate-$VERSION-linux-amd64.tar.gz</a></td>
    </tr>
    <tr>
        <th>Linux (ARM)</th>
        <td><a href="https://src-bin.com/substrate-$VERSION-linux-arm64.tar.gz">substrate-$VERSION-linux-arm64.tar.gz</a></td>
    </tr>
</table>
EOF
