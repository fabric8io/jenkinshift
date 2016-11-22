package main

import (
	"log"
	"net/http"
	"os"

	"github.com/emicklei/go-restful"
	"github.com/fabric8io/golang-jenkins"
	"github.com/fabric8io/jenkinshift/openshift"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	client "k8s.io/kubernetes/pkg/client/unversioned"
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


	log.Println("About to query Kubernetes")

	f := cmdutil.NewFactory(nil)
	cfg, err := f.ClientConfig()
	if err != nil {
		log.Fatal("Could not initialise a client - is your server setting correct?\n\n")
		log.Fatalf("%v", err)
	}
	c, err := client.New(cfg)
	if err != nil {
		log.Fatalf("Could not initialise a client: %v", err)
	}
	//ns, _, _ := f.DefaultNamespace()

	ns := "default"
	cm, err := c.ConfigMaps(ns).Get("fabric8")
	if err != nil {
		log.Fatalf("Could not load ConfigMap: %v", err)
	}
	log.Printf("Loaded ConfigMap %s", cm.Name)

	jenkins := gojenkins.NewJenkins(auth, jenkinsUrl)


	bcr := openshift.BuildConfigsResource{
		JenkinsURL: jenkinsUrl,
		Jenkins: jenkins,
		KubeClient: c,
		Namespace: ns,
	}
	bcr.Register(wsContainer)

	log.Printf("jenkinshift start listening on localhost:%s", port)
	server := &http.Server{Addr: ":" + port, Handler: wsContainer}
	log.Fatal(server.ListenAndServe())
}
