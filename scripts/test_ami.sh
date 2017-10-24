#!/bin/bash
aws ec2 run-instances --image-id "$1" --count 1 --key-name josh \
    --instance-type f1.2xlarge --security-group-ids sg-7fbfbe0c --region us-east-1 \
    --user-data file://scripts/config.json --iam-instance-profile Name=deployment-worker \
    --instance-initiated-shutdown-behavior terminate \
    --subnet-id subnet-ef8096b5 \
    --associate-public-ip-address
