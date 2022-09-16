set -e

# Get the previous tagged version and commit, from which customers will be
# upgrading when they come looking for the release being built now.
V="$(git describe --tags "HEAD^" | sed -E 's/-[0-9]+-g[0-9a-f]+$//')"
C="$(git show --format="%h" --no-patch "$V")"
FROM_VERSION="$V-$C"

# Create the base directory which CodeBuild will upload to S3.
mkdir -p "upgrade/$FROM_VERSION"

# Get this tagged version and commit, to which customers will be upgraded when
# they invoke `substrate upgrade`.
V="$(cat "substrate.version")"
C="$(git show --format="%h" --no-patch)"
TO_VERSION="$V-$C"

# Write a breadcrumb for each paying customer, their unique prefixes being
# magically found in the UPGRADES environment variable.
echo "$UPGRADES" |
tr "," "\n" |
if git describe --exact-match --tags "HEAD" >"/dev/null"
then cat # leave an upgrade breadcrumb for everyone on tagged releases
else grep "^src-bin-test" # only leave upgrade breadcrumbs for test orgs on untagged releases
fi |
while read PREFIX
do echo "$TO_VERSION" >"upgrade/$FROM_VERSION/$PREFIX"
done
