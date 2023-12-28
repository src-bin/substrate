set -e

# Get the previous tagged version and commit, from which customers will be
# upgrading when they come looking for the release being built now.
# TODO for 2024.02: FROM_VERSION doesn't have a commit SHA in it anymore
V="$(git describe --tags "HEAD^" | sed -E 's/-[0-9]+-g[0-9a-f]+$//')"
C="$(git show --format="%h" --no-patch "$V")"
FROM_VERSION="$V-$C"
FROM_VERSION_TRIAL="$V-trial" # TODO remove in 2024.02

# Create the base directory which CodeBuild will upload to S3.
mkdir -p "upgrade/$FROM_VERSION" "upgrade/$FROM_VERSION_TRIAL" # TODO remove trial in 2024.02

# Figure out which version customers on the previous version will be upgraded
# when they invoke `substrate upgrade`. If it's a tagged build, additionally
# write the latest version pointer which CodeBuild will upload to S3. If not,
# use a short commit SHA as the version. In either case, annotate builds from
# dirty work trees.
TO_VERSION="$(git describe --exact-match --tags "HEAD" 2>"/dev/null" || :)"
if [ "$TO_VERSION" ]
then echo "$TO_VERSION" >"substrate.version"
else TO_VERSION="$(git show --format="%h" --no-patch)"
fi
TO_VERSION="$TO_VERSION$(git diff --quiet || echo "-dirty")"

# Write a breadcrumb for each paying customer, their unique prefixes being
# magically found in the UPGRADES environment variable.
echo "$UPGRADES" |
tr "," "\n" |
if git describe --exact-match --tags "HEAD" >"/dev/null" 2>"/dev/null"
then cat # leave an upgrade breadcrumb for everyone on tagged releases
else grep "^src-bin-test" # only leave upgrade breadcrumbs for test orgs on untagged releases
fi |
while read PREFIX
do
    echo "$TO_VERSION" >"upgrade/$FROM_VERSION/$PREFIX"
    echo "$TO_VERSION" >"upgrade/$FROM_VERSION_TRIAL/trial" # TODO remove in 2024.02
done
# echo "$TO_VERSION" >"upgrade/$FROM_VERSION" # TODO starting in 2024.02
