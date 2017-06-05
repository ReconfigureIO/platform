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

        stage('install') {
            steps {
                sh 'docker build -t "reco-api-builder:latest" build'
                sh 'docker run -v $PWD:/go/src/github.com/ReconfigureIO/platform -w /go/src/github.com/ReconfigureIO/platform "reco-api-builder:latest" make install'
            }
        }

        stage('test') {
            steps {
                sh 'docker run -v $PWD:/go/src/github.com/ReconfigureIO/platform -w /go/src/github.com/ReconfigureIO/platform "reco-api-builder:latest" make test'
            }
        }

        stage('clean') {
            steps {
                sh 'docker run -v $PWD:/go/src/github.com/ReconfigureIO/platform -w /go/src/github.com/ReconfigureIO/platform "reco-api-builder:latest" make clean'
            }
        }

        stage('build') {
            steps {
                sh 'docker run -v $PWD:/go/src/github.com/ReconfigureIO/platform -w /go/src/github.com/ReconfigureIO/platform "reco-api-builder:latest" make all'
                sh 'make image'
                sh 'docker tag reco-api:latest 398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:latest'
                sh 'docker tag reco-api:latest-worker 398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:latest-worker'
            }
        }

        stage ('deploy') {
            when {
                expression { env.BRANCH_NAME in ["master"] }
            }
            steps {
                sh '$(aws ecr get-login --region us-east-1)'
                script {
                    docker.withRegistry("https://398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:latest") {
                        docker.image("398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:latest").push()
                        docker.image("398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:latest-worker").push()
                    }
                }
                sh 'make deploy-staging'
                sh 'make deploy-production'
            }
        }
    }
}