#! /bin/bash

set -euxo pipefail

echo "---- Run tests"
EXIT_CODE=0
docker build -t oasislabs/developer-gateway-ci -f .buildkite/Dockerfile.ci .
docker run \
  --rm \
	--volume="$(pwd)":/app \
	oasislabs/developer-gateway-ci:latest \
	/app/.buildkite/scripts/run_tests.sh || EXIT_CODE=$? ;

if [ $EXIT_CODE -ne 0 ]; then
	exit 1
fi

# report coverage
echo "--- Uploading Coverage"
set +e
COVER=${COVERALLS_TOKEN:-}
if [ -n "$COVER" ]; then
  # map env vars for goveralls
  BUILD_NUMBER=$BUILDKITE_BUILD_NUMBER CI_PULL_REQUEST=$BUILDKITE_PULL_REQUEST CI_BRANCH=$BUILDKITE_BRANCH goveralls -coverprofile=coverage.out -service=travis-ci -repotoken "$COVER"
fi

exit 0
