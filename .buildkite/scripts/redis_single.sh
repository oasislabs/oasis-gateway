#!/bin/sh
# shellcheck disable=SC1117,SC2103,SC2086,SC2039

# Settings
PORT=40000

if [ "$1" == "start" ]
then
    PORT=$((PORT+1))
    echo "Starting $PORT"
    redis-server --port $PORT --appendonly yes --appendfilename appendonly-${PORT}.aof --dbfilename dump-${PORT}.rdb --logfile ${PORT}.log --daemonize no
    exit 0
fi

if [ "$1" == "create" ]
then
    PORT=$((PORT+1))
    echo "Loading script $PORT"
    redis-cli -p $PORT --eval /app/mqueue/redis/redis.lua , test
    exit 0
fi

if [ "$1" == "stop" ]
then
    PORT=$((PORT+1))
    echo "Stopping $PORT"
    redis-cli -p $PORT shutdown nosave
    exit 0
fi

if [ "$1" == "watch" ]
then
    PORT=$((PORT+1))
    while true; do
        clear
        date
        redis-cli -p $PORT cluster nodes | head -30
        sleep 1
    done
    exit 0
fi

if [ "$1" == "tail" ]
then
    PORT=$((PORT+1))
    tail -f ${PORT}.log
    exit 0
fi

if [ "$1" == "call" ]
then
    PORT=$((PORT+1))
    redis-cli -p $PORT $2 $3 $4 $5 $6 $7 $8 $9
    exit 0
fi

if [ "$1" == "clean" ]
then
    rm -rf ./*.log
    rm -rf ./appendonly*.aof
    rm -rf ./dump*.rdb
    rm -rf ./nodes*.conf
    exit 0
fi

if [ "$1" == "clean-logs" ]
then
    rm -rf ./*.log
    exit 0
fi

echo "Usage: $0 [start|create|stop|watch|tail|clean]"
echo "start       -- Launch Redis instance."
echo "create      -- Create instance and default resources."
echo "stop        -- Stop Redis instances."
echo "watch       -- Show CLUSTER NODES output (first 30 lines) of first node."
echo "tail   -- Run tail -f of instance at base port + ID."
echo "clean       -- Remove data, logs, configs."
echo "clean-logs  -- Remove just logs."
