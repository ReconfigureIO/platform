#!/bin/bash
set -x

CONFIG=$(curl http://169.254.169.254/latest/user-data/)
IMAGE=$(echo "$CONFIG" | jq -r ".container.image")
COMMAND=$(echo "$CONFIG" | jq -r ".container.command")
LOG_GROUP=$(echo "$CONFIG" | jq -r ".logs.group")
LOG_PREFIX=$(echo "$CONFIG" | jq -r ".logs.prefix")

CALLBACK_URL=$(echo "$CONFIG" | jq -r ".callback_url")

curl -XPOST -H "Content-Type: application/json"  -d '{"status": "STARTED"}' "$CALLBACK_URL" &> /dev/null

timeout --kill-after 1m 45m docker run --privileged --log-driver=awslogs --log-opt awslogs-region=us-east-1 --log-opt awslogs-group="$LOG_GROUP" --log-opt awslogs-stream="$LOG_PREFIX" "$IMAGE" bash -c "$COMMAND"

exit="$?"

if [ $exit -ne 0 ]; then
    curl -XPOST -H "Content-Type: application/json"  -d "{\"status\": \"ERRORED\", \"code\": $exit}" "$CALLBACK_URL" &> /dev/null
fi

curl -XPOST -H "Content-Type: application/json"  -d '{"status": "COMPLETED"}' "$CALLBACK_URL" &> /dev/null
shutdown -H now
