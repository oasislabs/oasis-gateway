#! /bin/bash

set -euxo pipefail

echo "---- Build and run unit tests"
EXIT_CODE=0

docker run \
  --rm \
  --env BUILDKITE_BUILD_NUMBER="$BUILDKITE_BUILD_NUMBER" \
  --env BUILDKITE_PULL_REQUEST="$BUILDKITE_PULL_REQUEST" \
  --env BUILDKITE_BRANCH="$BUILDKITE_BRANCH" \
	--volume="$(pwd)":/app \
	oasislabs/developer-gateway:build \
	/app/.buildkite/scripts/build.sh || EXIT_CODE=$? ;

# report coverage
echo "--- Uploading Coverage"
set +e
bash <(curl -s https://codecov.io/bash) -Z

if [ $EXIT_CODE -ne 0 ]; then
	exit 1
fi

exit 0
