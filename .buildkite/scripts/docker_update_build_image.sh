#! /bin/bash

set -euxo pipefail

echo "---- Build and run unit tests"

docker push oasislabs/oasis-gateway:build

exit 0
