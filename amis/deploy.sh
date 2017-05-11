#!/bin/bash
set -x

CONFIG=$(curl http://169.254.169.254/latest/user-data/)
IMAGE=$(echo "$CONFIG" | jq -r ".container.image")
COMMAND=$(echo "$CONFIG" | jq -r ".container.command")
LOG_GROUP=$(echo "$CONFIG" | jq -r ".logs.group")
timeout --kill-after 1m 45m docker run --privileged --log-driver=awslogs --log-opt awslogs-region=us-east-1 --log-opt awslogs-group="$LOG_GROUP" "$IMAGE" bash -c "$COMMAND"
shutdown -H now