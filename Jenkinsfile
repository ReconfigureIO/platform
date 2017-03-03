properties([buildDiscarder(logRotator(daysToKeepStr: '', numToKeepStr: '20'))])

notifyStarted()

node ("docker") {
    try {
        stage "checkout"
        checkout scm

        stage 'zip api dir'
        dir('api'){
            sh 'zip ../api.zip -r * .[^.]*'
        }

        stage 'upload to S3'
        step([$class: 'S3BucketPublisher', dontWaitForConcurrentBuildCompletion: false, entries: [[bucket: 'nerabus/platform', excludedFile: '', flatten: false, gzipFiles: false, keepForever: false, managedArtifacts: false, noUploadOnFailure: true, selectedRegion: 'us-east-1', showDirectlyInBrowser: false, sourceFile: "*.zip", storageClass: 'STANDARD', uploadFromSlave: false, useServerSideEncryption: false]], profileName: 's3', userMetadata: []])

        stage 'trigger ElasticBeanstalk'
        sh 'aws elasticbeanstalk create-application-version --application-name platform --version-label v1 --description platformv1 --source-bundle S3Bucket="nerabus",S3Key="platform/api.zip" --auto-create-application -Region us-east-1'
        sh 'aws elasticbeanstalk create-environment --application-name platform --environment-name platform --version-label v1 --solution-stack-name \\"64bit Amazon Linux 2016.09 v2.5.0 running Docker 1.12.6\\" -Region us-east-1'

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
