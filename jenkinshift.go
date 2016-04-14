package main

import (
	"log"
	"net/http"
	"os"

	"github.com/emicklei/go-restful"
	"github.com/fabric8io/golang-jenkins"
	"github.com/fabric8io/jenkinshift/openshift"
)

func main() {
	wsContainer := restful.NewContainer()
	wsContainer.Router(restful.CurlyRouter{})

	jenkinsUrl := os.Getenv("JENKINS_URL")
	if len(jenkinsUrl) == 0 {
		jenkinsUrl = "http://jenkins/"
	}

	log.Printf("Invoking Jenkins on URL %s", jenkinsUrl)

	auth := &gojenkins.Auth{
	   /*
	   Username: "[jenkins user name]",
	   ApiToken: "[jenkins API token]",
	   */
	}
	jenkins := gojenkins.NewJenkins(auth, jenkinsUrl)


	bcr := openshift.BuildConfigsResource{
		JenkinsURL: jenkinsUrl,
		Jenkins: jenkins,
	}
	bcr.Register(wsContainer)

	log.Printf("jenkinshift start listening on localhost:9090")
	server := &http.Server{Addr: ":9090", Handler: wsContainer}
	log.Fatal(server.ListenAndServe())
}
