pipeline {
    agent { label "master" }
    environment {
        AWS_DEFAULT_REGION = "us-east-1"
    }
    options {
        buildDiscarder(logRotator(daysToKeepStr: '', numToKeepStr: '20'))
        disableConcurrentBuilds()
    }
    post {
        failure {
            slackSend (color: '#FF0000', message: "FAILED: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]' (${env.BUILD_URL})")
        }
        success {
            slackSend (color: '#00FF00', message: "SUCCESSFUL: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]' (${env.BUILD_URL})")
        }
    }
    stages {
        stage("notify") {
            steps {
                slackSend (color: '#FFFF00', message: "STARTED: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]' (${env.BUILD_URL})")
            }
        }

        stage('checkout') {
            steps {
                checkout scm
            }
        }

        stage('zip api dir') {
            steps {
                dir('api'){
                    sh 'zip ../api.zip -r * .[^.]*'
                }
            }
        }

        stage('upload zip to S3') {
            steps {
                step([$class: 'S3BucketPublisher', dontWaitForConcurrentBuildCompletion: false, entries: [[bucket: 'nerabus/platform', excludedFile: '', flatten: false, gzipFiles: false, keepForever: false, managedArtifacts: false, noUploadOnFailure: true, selectedRegion: 'us-east-1', showDirectlyInBrowser: false, sourceFile: "*.zip", storageClass: 'STANDARD', uploadFromSlave: false, useServerSideEncryption: false]], profileName: 's3', userMetadata: []])
            }
        }

        stage('trigger ElasticBeanstalk') {
            steps {
                sh 'aws elasticbeanstalk create-application-version --application-name platform --version-label v1 --description platformv1 --source-bundle S3Bucket="nerabus",S3Key="platform/api.zip" --auto-create-application'
                sh 'aws elasticbeanstalk create-environment --application-name platform --environment-name platform --version-label v1 --solution-stack-name \\"64bit Amazon Linux 2016.09 v2.5.0 running Docker 1.12.6\\"'
            }
        }
    }
}
