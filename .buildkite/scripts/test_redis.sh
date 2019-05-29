#! /bin/bash

set -euxo pipefail

echo "---- Run tests"
EXIT_CODE=0

# if there is already a redis container, remove it so that
# a new one can be created
REDIS_CONTAINER=$(docker ps -a | grep redis | awk '{print $1}') || true
if [ ! -z "$REDIS_CONTAINER" ]; then
    docker rm -f "$REDIS_CONTAINER"
fi

# create a new redis container
REDIS_CONTAINER=$(docker run --rm -p 6379:6379 -v "$(pwd)":/app -d redis:5.0.5-alpine)
sleep 5

# run tests
docker exec -ti "$REDIS_CONTAINER" redis-cli --eval /app/mqueue/redis/redis.lua , test
docker rm -f "$REDIS_CONTAINER"

# report coverage
if [ $EXIT_CODE -ne 0 ]; then
	  exit 1
fi

exit 0
