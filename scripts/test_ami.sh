#!/bin/bash
aws ec2 run-instances --image-id "$1" --count 1 --key-name josh \
    --instance-type f1.2xlarge --security-groups default --region us-east-1 \
    --user-data file://scripts/config.json --iam-instance-profile Name=jenkins \
    --instance-initiated-shutdown-behavior terminate
