#! /bin/bash

set -euxo pipefail

echo "---- Build and run unit tests"

docker build -t oasislabs/developer-gateway:build -f .buildkite/Dockerfile.ci .
docker push oasislabs/developer-gateway:build

exit 0
