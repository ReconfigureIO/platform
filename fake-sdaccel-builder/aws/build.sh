#!/bin/bash
set -e

function post_event {
    curl -XPOST -H "Content-Type: application/json"  -d '{"status": "'"$1"'", "message": "'"$2"'", "code": '${3-0}'}' "$CALLBACK_URL" &> /dev/null
}

post_event STARTED

echo "downloading source code... done"

echo "compiling host cmds... done"

echo "compiling fpga kernel... done"

echo "uploading artifacts... done"

post_event COMPLETED
