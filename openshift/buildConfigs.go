package openshift

import (
	"fmt"
	"log"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/fabric8io/golang-jenkins"

	kapi "k8s.io/kubernetes/pkg/api/v1"
	oapi "github.com/openshift/origin/pkg/build/api/v1"
	tapi "github.com/openshift/origin/pkg/template/api/v1"
)

type BuildConfigsResource struct {
	JenkinsURL	string
	Jenkins 	*gojenkins.Jenkins
}

func (r BuildConfigsResource) Register(container *restful.Container) {
	ws := new(restful.WebService)
	ws.
	Path("/namespaces/{namespace}").
	Consumes(restful.MIME_XML, restful.MIME_JSON).
	Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/buildconfigs/").To(r.getBuildConfigs))
	ws.Route(ws.GET("/buildconfigs/{name}").To(r.getBuildConfig))
	/*
	ws.Route(ws.POST("/buildconfigs").To(r.updateBuildConfig))
	ws.Route(ws.PUT("/buildconfigs/{name}").To(r.createBuildConfig))
	ws.Route(ws.DELETE("/buildconfigs/{name}").To(r.removeBuildConfig))
	*/


	// lets add a dummy templates REST service to avoid errors in the current fabric8 console ;)
	ws.Route(ws.GET("/templates/").To(r.getTemplates))

	container.Add(ws)
}

func respondError(request *restful.Request, response *restful.Response, err error) {
	message := fmt.Sprintf("%s", err)
	respondErrorMessage(request, response, message)
}

func respondErrorMessage(request *restful.Request, response *restful.Response, message string) {
	response.AddHeader("Content-Type", "text/plain")
	response.WriteErrorString(http.StatusNotFound, message)
}

// GET http://localhost:8080/namespaces/{namespaces}/buildconfigs
//
func (r BuildConfigsResource) getBuildConfigs(request *restful.Request, response *restful.Response) {
	ns := request.PathParameter("namespace")

	jenkins := r.Jenkins
	jobs, err := jenkins.GetJobs()
	if err != nil {
		respondError(request, response, err)
		return
	}

	buildConfigs := []oapi.BuildConfig{}

	for _, job := range jobs {
		buildConfig, err := r.loadBuildConfig(ns, job.Name, job.Url)
		if err != nil {
			log.Printf("Failed to find job %s due to %s", job.Name, err)
		} else if buildConfig != nil {
			buildConfigs = append(buildConfigs, *buildConfig)
		}
	}
	buildConfigList := oapi.BuildConfigList{
		Items: buildConfigs,
	}
	response.WriteEntity(buildConfigList)
}

// GET http://localhost:8080/namespaces/{namespaces}/buildconfigs/{name}
//
func (r BuildConfigsResource) getBuildConfig(request *restful.Request, response *restful.Response) {
	ns := request.PathParameter("namespace")
	jobName := request.PathParameter("name")
	if len(jobName) == 0 {
		respondErrorMessage(request, response, "No BuildConfig name specified in URL")
		return
	}
	jobUrl := r.Jenkins.GetJobUrl(jobName)

	buildConfig, err := r.loadBuildConfig(ns, jobName, jobUrl)
	if err != nil {
		respondError(request, response, err)
		return
	}
	if buildConfig == nil {
		respondErrorMessage(request, response, fmt.Sprintf("No BuildConfig could be found for job %s", jobName))
		return
	}
	response.WriteEntity(buildConfig)
}

// loadBuildConfig loads a BuildConfig for a given jobName
func (r BuildConfigsResource) loadBuildConfig(ns string, jobName string, jobUrl string) (*oapi.BuildConfig, error) {
	jenkins := r.Jenkins
	item, err := jenkins.GetJobConfig(jobName)
	gitUrl := ""
	if err != nil {
		return nil, err
	}
	mavenJob := item.MavenJobItem
	pipelineJob := item.PipelineJobItem
	if mavenJob != nil {
		//log.Printf("Found maven job: (%+v)", mavenJob)
		gitUrl = getGitUrlFromScm(mavenJob.Scm)
	} else if pipelineJob != nil {
		//log.Printf("Found pipeline job: (%+v)", pipelineJob)
		gitUrl = getGitUrlFromScm(pipelineJob.Definition.Scm)
	} else {
		//log.Printf("Unknown job type (%+v)", item);
		return nil, nil
	}
	return &oapi.BuildConfig{
		ObjectMeta: kapi.ObjectMeta{
			Name: jobName,
			Namespace: ns,
			Annotations: map[string]string{
				"fabric8.io/jenkins-url": jobUrl,
			},
		},
		Spec: oapi.BuildConfigSpec{
			BuildSpec: oapi.BuildSpec{
				Source: oapi.BuildSource{
					Type: oapi.BuildSourceGit,
					Git: &oapi.GitBuildSource{
						URI: gitUrl,
					},

				},
			},
		},
	}, nil
}

// GET http://localhost:8080/namespaces/{namespaces}/templates
//
func (r BuildConfigsResource) getTemplates(request *restful.Request, response *restful.Response) {
	templateList := tapi.TemplateList{}
	response.WriteEntity(templateList)
}


func getGitUrlFromScm(scm gojenkins.Scm) string {
	answer := ""
	scmContent := scm.ScmContent
	switch t := scmContent.(type) {
	case *gojenkins.ScmGit:
		urls := t.UserRemoteConfigs.UserRemoteConfig.Urls
		if len(urls) > 0 {
			answer = urls[0]
		}
		if len(answer) == 0 {
			answer = t.GitBrowser.Url
		}
	}
	return answer
}

