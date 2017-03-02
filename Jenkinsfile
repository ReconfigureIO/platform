properties([buildDiscarder(logRotator(daysToKeepStr: '', numToKeepStr: '20'))])

notifyStarted()

node ("docker") {
    try {
        stage "checkout"
        checkout scm

        stage 'install'
        sh 'docker build -t "reco-builder:latest" docker'
        sh "docker run -v ${env.WORKSPACE}:/go/src/github.com/ReconfigureIO/reco -w /go/src/github.com/ReconfigureIO/reco reco-builder make clean dependencies"

        stage 'test'
        sh "docker run -v ${env.WORKSPACE}:/go/src/github.com/ReconfigureIO/reco -w /go/src/github.com/ReconfigureIO/reco reco-builder make test benchmark"

        stage 'build'
        sh "docker run -v ${env.WORKSPACE}:/go/src/github.com/ReconfigureIO/reco -w /go/src/github.com/ReconfigureIO/reco reco-builder make VERSION=${env.BRANCH_NAME} packages"

        stage 'post build'
        if (env.BRANCH_NAME == "master") {
            step([$class: 'S3BucketPublisher', dontWaitForConcurrentBuildCompletion: false, entries: [[bucket: 'nerabus/reco', excludedFile: '', flatten: false, gzipFiles: false, keepForever: false, managedArtifacts: false, noUploadOnFailure: true, selectedRegion: 'us-east-1', showDirectlyInBrowser: false, sourceFile: "dist/*.tar.gz", storageClass: 'STANDARD', uploadFromSlave: false, useServerSideEncryption: false]], profileName: 's3', userMetadata: []])
        }
        sh "docker run -v ${env.WORKSPACE}:/go/src/github.com/ReconfigureIO/reco -w /go/src/github.com/ReconfigureIO/reco reco-builder make clean"

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
