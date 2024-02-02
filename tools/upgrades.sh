set -e

# Get the previous tagged version and commit, from which customers will be
# upgrading when they come looking for the release being built now.
FROM_VERSION="$(git describe --tags "HEAD^" | sed -E 's/-[0-9]+-g[0-9a-f]+$//')"

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

# Write a breadcrumb to help folks get from one version to the next.
mkdir "upgrade"
echo "$TO_VERSION" >"upgrade/$FROM_VERSION"
