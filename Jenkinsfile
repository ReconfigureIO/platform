properties([buildDiscarder(logRotator(daysToKeepStr: '', numToKeepStr: '20'))])

notifyStarted()

node ("docker") {
    try {
        stage "checkout"
        checkout scm

        stage 'zip api dir'
        dir('api'){
                    sh 'zip ../myapp.zip -r * .[^.]*'
                }

        stage 'upload to S3'

        stage 'trigger ElasticBeanstalk'

        notifySuccessful()
    } catch (e) {
      currentBuild.result = "FAILED"
      notifyFailed()
      throw e
    }
}

def notifyStarted() {
    slackSend (color: '#FFFF00', message: "STARTED: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]' (${env.BUILD_URL})")
}

def notifySuccessful() {
    slackSend (color: '#00FF00', message: "SUCCESSFUL: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]' (${env.BUILD_URL})")
}

def notifyFailed() {
  slackSend (color: '#FF0000', message: "FAILED: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]' (${env.BUILD_URL})")
}
