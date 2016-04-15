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
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "9191"
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

	log.Printf("jenkinshift start listening on localhost:%s", port)
	server := &http.Server{Addr: ":" + port, Handler: wsContainer}
	log.Fatal(server.ListenAndServe())
}
