#! /bin/bash
# shellcheck disable=SC1117,SC2103

# some binaries are installed in /tmp/bin and
# we make them globally available
export PATH=$PATH:/tmp/bin

set -euxo pipefail

# golang service backend linting / testing
echo "--- Installing Dependencies"
go get -v
set -x
make build
echo "--- Linting Backend"
make lint
echo "+++ Testing Backend"
make test-coverage


# Integration testing
exit 0
