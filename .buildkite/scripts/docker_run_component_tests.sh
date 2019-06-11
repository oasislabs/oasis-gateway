#! /bin/bash
# shellcheck disable=SC2046

set -euxo pipefail

echo "---- Run Component tests"
EXIT_CODE=0

function run_dev_tests() {
    docker run \
           --rm \
	         --volume="$(pwd)":/app \
	         oasislabs/developer-gateway:build \
	         /bin/sh -c 'cd /app && make test-component-dev' || EXIT_CODE=$? ;
}

function run_redis_cluster_tests() {
    # start redis container
    NETWORK=$1

    GATEWAY_IP=$(docker network inspect "$NETWORK" -f '{{(index .IPAM.Config 0).Gateway}}')

    REDIS_CONTAINER=$(docker run --rm -p 30001:30001 -p 30002:30002 -p 30003:30003 -p 30004:30004 \
                             -p 30005:30005 -p 30006:30006 \
                             --net="$NETWORK" -v "$(pwd)":/app \
                             -d redis:5.0.5-alpine \
                             /bin/sh -c '/app/.buildkite/scripts/redis_cluster.sh start')
    sleep 1
    docker exec -ti "$REDIS_CONTAINER" /bin/sh -c 'echo yes | /app/.buildkite/scripts/redis_cluster.sh create'

    # modify the configuration file to point to the deployed
    # redis instance
    sed "s/127.0.0.1/$GATEWAY_IP/g" tests/config/redis_cluster.toml > redis_cluster.toml
    echo "REDIS CONFIG"
    cat redis_cluster.toml

    docker run \
           --rm \
           --net="$NETWORK"\
           --env OASIS_DG_CONFIG_PATH=/app/redis_cluster.toml \
	         --volume="$(pwd)":/app \
	         oasislabs/developer-gateway:build \
           make test-component || EXIT_CODE=$?;

    # stop and remove the redis container
    docker rm -f "$REDIS_CONTAINER"
}

function run_redis_single_tests() {
    # start redis container
    NETWORK=$1

    GATEWAY_IP=$(docker network inspect "$NETWORK" -f '{{(index .IPAM.Config 0).Gateway}}')
    REDIS_CONTAINER=$(docker run -p 6379:6379 --net="$NETWORK" -v "$(pwd)":/app \
           -d redis:5.0.5-alpine \
           /bin/sh -c '/app/.buildkite/scripts/redis_single.sh start')
    sleep 1
    docker exec -ti "$REDIS_CONTAINER" /app/.buildkite/scripts/redis_single.sh create

    # modify the configuration file to point to the deployed
    # redis instance
    sed "s/127.0.0.1/$GATEWAY_IP/g" tests/config/redis_single.toml > redis_single.toml
    echo "REDIS CONFIG"
    cat redis_single.toml

    docker run \
           --rm \
           --net="$NETWORK"\
           --env OASIS_DG_CONFIG_PATH=/app/redis_single.toml \
	         --volume="$(pwd)":/app \
	         oasislabs/developer-gateway:build \
	         make test-component || EXIT_CODE=$? ;

    # stop and remove the redis container
    docker rm -f "$REDIS_CONTAINER"
}

mkdir -p output

# create the network used for the tests
NETWORK_NAME="developer-gateway-network-$BUILDKITE_PULL_REQUEST"
NETWORK=$(docker network ls 2> /dev/null | (grep "$NETWORK_NAME" || true) | cut -d ' ' -f 1)

if [ ! -z "$NETWORK" ]; then
    if [ $(docker ps -a -q | wc -l) != 0 ]; then
        docker rm -f $(docker ps -a -q)
    fi

    docker network rm "$NETWORK"
fi

NETWORK=$(docker network create "$NETWORK_NAME")

run_dev_tests
run_redis_single_tests "$NETWORK"
run_redis_cluster_tests "$NETWORK"

# remove network for the tests
docker network rm "$NETWORK"

if [ $EXIT_CODE -ne 0 ]; then
	  exit 1
fi

exit 0

