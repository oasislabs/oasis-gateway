#! /bin/bash
# shellcheck disable=SC1117,SC2103

function fmt()
{
  while read -r data; do
    echo "$data" | sed ''/PASS/s//"$(printf "\033[32mPASS\033[0m")"/'' | sed ''/FAIL/s//"$(printf "\033[31mFAIL\033[0m")"/'' | sed ''/---/s//--/''
  done
}

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
make lint | fmt
echo "+++ Testing Backend"
make test-coverage | fmt

# Integration testing
exit 0
