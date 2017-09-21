pipeline {
    agent { label "master" }
    environment {
        AWS_DEFAULT_REGION = "us-east-1"
        KOPS_STATE_STORE=S3://k8s-reconfigure-infra
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

        stage('pre-clean') {
            steps {
                sh 'docker-compose run --rm web-base make clean'
            }
        }

        stage('install') {
            steps {
                sh 'docker-compose run --rm web-base make install'
            }
        }

        stage('test') {
            steps {
                sh 'docker-compose run --rm test make vet integration-tests'
            }
        }

        stage('clean') {
            steps {
                sh 'docker-compose run --rm web-base make clean'
                sh 'docker-compose down'
            }
        }

        stage('build') {
            steps {
                sh 'docker-compose run --rm web-base make all'
                sh 'make image'
            }
        }

        stage ('deploy') {
            when {
                expression { env.BRANCH_NAME in ["master"] }
            }
            steps {
                sh 'make push-image migrate-staging deploy-staging'
                sh 'make DOCKER_TAG=latest image push-image deploy-production'
            }
        }
    }
}
