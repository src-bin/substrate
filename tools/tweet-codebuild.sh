set -e

# Only tweet if we're succeeding and have made it all the way to the end.
if [ "$CODEBUILD_BUILD_SUCCEEDING" != "1" ]
then exit 0
fi

# Also only tweet if this is a tagged build.
if ! git describe --exact-match --tags "HEAD"
then exit 0
fi

VERSION="$(cat "substrate.version")"

TMP="$(mktemp)"
trap "rm -f \"$TMP\"" EXIT INT QUIT TERM
curl \
    -X"POST" \
    -H"Authorization: Basic $(
        aws secretsmanager get-secret-value --secret-id "TwitterBase64ClientIDSecret" |
        jq -r ".SecretString"
    )" \
    -H"Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "grant_type=refresh_token" \
    --data-urlencode "refresh_token=$(
        aws secretsmanager get-secret-value --secret-id "TwitterRefreshToken" |
        jq -r ".SecretString"
    )" \
    -s \
    "https://api.twitter.com/2/oauth2/token" >"$TMP"

aws secretsmanager put-secret-value --secret-id "TwitterRefreshToken" --secret-string "$(
    jq -r ".refresh_token" <"$TMP"
)"

curl \
    -H"Authorization: Bearer $(jq -r ".access_token" <"$TMP")" \
    -H"Content-Type: application/json; charset=utf-8" \
    -X"POST" \
    -d '{"text": "Substrate '"$VERSION"' is out! Release notes: https://docs.src-bin.com/substrate/releases#'"$VERSION"'"}' \
    -s \
    "https://api.twitter.com/2/tweets"
