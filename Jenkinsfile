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

        stage('create builder') {
            steps {
                sh 'docker build -t "reco-api-builder:latest" build'
            }
        }

        stage('build binary') {
            steps {
                sh 'docker run -v $PWD:/go/src/github.com/ReconfigureIO/platform -w /go/src/github.com/ReconfigureIO/platform "reco-api-builder:latest" go build -o ./api/main main.go'
            }
        }

        stage('build container for EB') {
            steps {
                sh 'docker build -t "reco-api:latest" api'
            }
        }

        stage('tag EB container for ECR repo') {
            steps {
                sh 'docker tag reco-api:latest 398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:latest'
            }
        }

        stage('get ECR token') {
            steps {
                sh '$(aws ecr get-login --region us-east-1)'
            }
        }

        stage('upload container to ECR') {
            steps {
                script {
                    docker.withRegistry("https://398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:latest") {
                        docker.image("398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:latest").push()
                    }
                }
            }
        }

        stage('zip api dir') {
            steps {
                dir('EB'){
                    sh 'zip ../EB.zip -r * .[^.]*'
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
                sh 'aws elasticbeanstalk create-application-version --application-name platform --version-label env.GIT_COMMIT --description platform --source-bundle S3Bucket="nerabus",S3Key="platform/EB.zip" --auto-create-application'
                sh 'aws elasticbeanstalk create-environment --application-name platform --environment-name platform --version-label env.GIT_COMMIT --solution-stack-name "64bit Amazon Linux 2016.09 v2.5.0 running Docker 1.12.6"'
            }
        }
    }
}
