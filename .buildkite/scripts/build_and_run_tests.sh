#! /bin/bash

set -euxo pipefail

echo "---- Run tests"
EXIT_CODE=0

NETWORK="developer-gateway-network-$BUILDKITE_PULL_REQUEST"
EXISTS=$(docker network ls | (grep "$NETWORK" || true) | wc -l)

if [ "$EXISTS" != "0" ]; then
    docker network rm "$NETWORK"
fi

docker network create "$NETWORK"
GATEWAY_IP=$(docker network inspect "$NETWORK" -f '{{(index .IPAM.Config 0).Gateway}}')

# if there is already a redis container, remove it so that
# a new one can be created
REDIS_CONTAINER=$(docker ps -a | grep redis | awk '{print $1}') || true
if [ ! -z "$REDIS_CONTAINER" ]; then
    docker rm -f "$REDIS_CONTAINER"
fi

# create a new redis container
REDIS_CONTAINER=$(docker run --rm -p 6379:6379 --net="$NETWORK" -v "$(pwd)":/app -d redis:5.0.5-alpine)
sleep 5

cp tests/config/redis_single.toml tests/config/redis_single.toml.back
sed "s/127.0.0.1/$GATEWAY_IP/g" tests/config/redis_single.toml.back > tests/config/redis_single.toml
echo "REDIS CONFIG"
cat tests/config/redis_single.toml

# load redis operations and test lua code
docker exec -ti "$REDIS_CONTAINER" redis-cli --eval /app/mqueue/redis/redis.lua , test

docker build -t oasislabs/developer-gateway-ci -f .buildkite/Dockerfile.ci .
docker run \
  --rm \
  --net="$NETWORK"\
  --env BUILDKITE_BUILD_NUMBER="$BUILDKITE_BUILD_NUMBER" \
  --env BUILDKITE_PULL_REQUEST="$BUILDKITE_PULL_REQUEST" \
  --env BUILDKITE_BRANCH="$BUILDKITE_BRANCH" \
	--volume="$(pwd)":/app \
	oasislabs/developer-gateway-ci:latest \
	/app/.buildkite/scripts/run_tests.sh || EXIT_CODE=$? ;

docker rm -f "$REDIS_CONTAINER"
docker network rm "$NETWORK"

# report coverage
echo "--- Uploading Coverage"
set +e
bash <(curl -s https://codecov.io/bash) -Z

if [ $EXIT_CODE -ne 0 ]; then
	exit 1
fi

exit 0
