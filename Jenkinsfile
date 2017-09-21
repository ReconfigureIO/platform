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
                sh 'docker tag reco-api:latest 398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:latest'
                sh 'docker tag reco-api:latest-worker 398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:latest-worker'
                sh 'docker tag reco-api:latest ${env.GIT_COMMIT}'
                sh 'docker tag reco-api:latest-worker ${env.GIT_COMMIT}'
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
                sh 'kops export kubecfg k8s.reconfigure.io'
                sh 'make deploy-staging 398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/api:${env.GIT_COMMIT}'
                sh 'make deploy-production'
            }
        }
    }
}
