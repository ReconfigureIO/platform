#!/bin/bash 
# This script mimics the equivalent script (reco-sdaccel/aws/build.sh)
# which is used in our production compiler. It produces some basic logs and
# posts events to platform's events API but it has fewer system requirements
# than the production script making it easier to test against during end to end
# testing, for instance it does not require Vivado or run for 4+ hours. 
set -e

function post_event {
    curl -XPOST -H "Content-Type: application/json"  -d '{"status": "'"$1"'", "message": "'"$2"'", "code": '${3-0}'}' "$CALLBACK_URL" &> /dev/null
}

post_event STARTED

echo "downloading source code... done"

echo "compiling host cmds... done"

echo "compiling fpga kernel..."

for i in $(seq 1 100)
    do echo $i%
    sleep $SLEEP_PERIOD
done

echo "uploading artifacts... done"

post_event COMPLETED
