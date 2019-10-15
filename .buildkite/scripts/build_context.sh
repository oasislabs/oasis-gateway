#!/bin/bash

# Build a Docker context tarball.

set -euxo pipefail

dst="/tmp/context.tar.gz"

tar -czf "$dst" .
