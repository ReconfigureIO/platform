#!/bin/bash
set -x

CONFIG=$(curl http://169.254.169.254/latest/user-data/)
IMAGE=$(echo "$CONFIG" | jq -r ".container.image")
COMMAND=$(echo "$CONFIG" | jq -r ".container.command")
LOG_GROUP=$(echo "$CONFIG" | jq -r ".logs.group")
LOG_PREFIX=$(echo "$CONFIG" | jq -r ".logs.prefix")
DIST_URL=$(echo "$CONFIG" | jq -r ".build.dist_url")

CALLBACK_URL=$(echo "$CONFIG" | jq -r ".callback_url")

curl -XPOST -H "Content-Type: application/json"  -d '{"status": "STARTED"}' "$CALLBACK_URL" &> /dev/null

aws s3 cp "$DIST_URL" /tmp/bundle.zip
unzip /tmp/bundle.zip -d "$PWD"

timeout --kill-after 1m 45m docker run -v "$PWD/.reco-work/sdaccel/dist/:/mnt/dist" \
        --privileged \
        -e XCL_EMULATION_MODE=hw -e XCL_BINDIR="/mnt/xclbin" \
        -e XILINX_SDX=/opt/Xilinx/SDx/2017.1.op \
        -v /opt/Xilinx:/opt/Xilinx \
        --log-driver=awslogs --log-opt awslogs-region=us-east-1 --log-opt awslogs-group="$LOG_GROUP" --log-opt awslogs-stream="$LOG_PREFIX" \
        "$IMAGE" bash -c "cd /mnt/dist && export PATH=/mnt/dist:\$PATH && source \$XILINX_SDX/settings64.sh && $COMMAND"

exit="$?"

if [ $exit -ne 0 ]; then
    curl -XPOST -H "Content-Type: application/json"  -d "{\"status\": \"ERRORED\", \"code\": $exit}" "$CALLBACK_URL" &> /dev/null
fi

curl -XPOST -H "Content-Type: application/json"  -d '{"status": "COMPLETED"}' "$CALLBACK_URL" &> /dev/null
shutdown -H now
