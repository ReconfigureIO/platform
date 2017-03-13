#!/bin/bash
set -ex

aws elasticbeanstalk create-application \
    --application-name platform \
    --description=api-platform \
    --resource-lifecycle-config "ServiceRole=arn:aws:iam::398048034572:role/aws-elasticbeanstalk-service-role,VersionLifecycleConfig={MaxCountRule={Enabled=true,MaxCount=100,DeleteSourceFromS3=true}}"

aws elasticbeanstalk create-environment \
    --application-name platform \
    --environment-name production \
    --solution-stack-name '64bit Amazon Linux 2016.09 v2.5.0 running Docker 1.12.6'
