set -e -x

# Get the previous tagged version and commit, from which customers will be
# upgrading when they come looking for the release being built now.
V="$(git describe --tags "HEAD^" | sed -E 's/-[0-9]+-g[0-9a-f]+$//')"
C="$(git show --format="%h" --no-patch "$V")"
FROM_VERSION="$V-$C"

# Create the base directory which CodeBuild will upload to S3.
mkdir -p "upgrade/$FROM_VERSION"

# Get this tagged version and commit, to which customers will be upgraded when
# they invoke `substrate upgrade`.
V="$(make release-version)"
C="$(git show --format="%h" --no-patch "$V")"
TO_VERSION="$V-$C"

# Write a breadcrumb for each paying customer, their unique prefixes being
# magically found in the UPGRADES environment variable.
echo "$UPGRADES" | tr "," "\n" | while read PREFIX
do echo "$TO_VERSION" >"upgrade/$FROM_VERSION/$PREFIX"
done

# TODO remove this after confirming everything's in order.
find "upgrade" | while read PATHNAME
do cat "$PATHNAME"
done
