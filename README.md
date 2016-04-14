# Jenkinshift: a simple Jenkins OpenShift Facade

A simple microservice which exposes an OpenShift REST API for BuildConfig and Build which proxies to Jenkins Jobs and Build resources for cases where you wish to use Jenkins on vanilla Kubernetes.

## Configuring Jenkinshift

To configure the `jenkinshift` process just set the `$JENKINS_URL` variable to point to the URL of the Jenkins server you wish to connect to.

The default is `http://jenkins/` to discover jenkins via DNS inside Kubernetes
