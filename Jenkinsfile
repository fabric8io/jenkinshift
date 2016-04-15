#!/usr/bin/groovy
node{

  git 'https://github.com/fabric8io/jenkinshift.git'

  kubernetes.pod('buildpod').withImage('fabric8/go-builder')
  .withEnvVar('GOPATH','/home/jenkins/workspace/workspace/go')
  .withPrivileged(true).inside {

    stage 'build binary'

    sh "mkdir -p ../go/src/github.com/fabric8io; cp -R ../jenkinshift ../go/src/github.com/fabric8io/; cd ../go/src/github.com/fabric8io/jenkinshift; make build test lint"

    sh "mv ../go/src/github.com/fabric8io/jenkinshift/bin ."

    def imageName = 'jenkinshift'
    def tag = 'latest'

    stage 'build image'
    kubernetes.image().withName(imageName).build().fromPath(".")

    stage 'tag'
    kubernetes.image().withName(imageName).tag().inRepository('docker.io/fabric8/'+imageName).force().withTag(tag)

    stage 'push'
    kubernetes.image().withName('docker.io/fabric8/'+imageName).push().withTag(tag).toRegistry()

  }
}
