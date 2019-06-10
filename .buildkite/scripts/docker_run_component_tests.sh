#! /bin/bash
# shellcheck disable=SC2046

echo "---- Run Component tests"
EXIT_CODE=0

if [ -z "$BUILDKITE_PULL_REQUEST" ]; then
    BUILDKITE_PULL_REQUEST=0
fi

function run_dev_tests() {
    docker run \
           --rm \
	         --volume="$(pwd)":/app \
	         oasislabs/developer-gateway:build \
	         /bin/sh -c 'cd /app && make test-component-dev' || EXIT_CODE=$? ;
}

function run_redis_single_tests() {
    # start redis container
    NETWORK=$1

    GATEWAY_IP=$(docker network inspect "$NETWORK" -f '{{(index .IPAM.Config 0).Gateway}}')
    REDIS_CONTAINER=$(docker run -p 40001:40001 --net="$NETWORK" -v "$(pwd)":/app \
           -d redis:5.0.5-alpine \
           /bin/sh -c '/app/.buildkite/scripts/redis_single.sh start')
    sleep 1
    docker exec -ti "$REDIS_CONTAINER" /app/.buildkite/scripts/redis_single.sh create

    # modify the configuration file to point to the deployed
    # redis instance
    cp tests/config/redis_single.toml tests/config/redis_single.toml.back
    sed "s/127.0.0.1:6379/$GATEWAY_IP:40001/g" tests/config/redis_single.toml.back > tests/config/redis_single.toml
    echo "REDIS CONFIG"
    cat tests/config/redis_single.toml

    docker run \
           --rm \
           --net="$NETWORK"\
	         --volume="$(pwd)":/app \
	         oasislabs/developer-gateway:build \
	         make test-component-redis-single || EXIT_CODE=$? ;

    # revert back the configuration change
    cp tests/config/redis_single.toml.back tests/config/redis_single.toml
    rm tests/config/redis_single.toml.back

    # stop and remove the redis container
    docker exec -ti "$REDIS_CONTAINER"  /bin/sh -c '/app/.buildkite/scripts/redis_single.sh stop'
    docker rm -f "$REDIS_CONTAINER"
}

# create the network used for the tests
NETWORK_NAME="developer-gateway-network-$BUILDKITE_PULL_REQUEST"
NETWORK=$(docker network ls 2> /dev/null | (grep "$NETWORK_NAME" || true) | cut -d ' ' -f 1)

if [ ! -z "$NETWORK" ]; then
    docker rm -f $(docker ps -a -q)
    docker network rm "$NETWORK"
fi

NETWORK=$(docker network create "$NETWORK_NAME")

run_dev_tests
run_redis_single_tests "$NETWORK"

# remove network for the tests
docker network rm "$NETWORK"

if [ $EXIT_CODE -ne 0 ]; then
	  exit 1
fi

exit 0

