package openshift

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/fabric8io/golang-jenkins"

	client "k8s.io/kubernetes/pkg/client/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1"
	oapi "github.com/openshift/origin/pkg/build/api/v1"
	tapi "github.com/openshift/origin/pkg/template/api/v1"
	"k8s.io/kubernetes/pkg/api"
)

type BuildConfigsResource struct {
	JenkinsURL	string
	Jenkins 	*gojenkins.Jenkins
	KubeClient      *client.Client
	Namespace       string
}

func (r BuildConfigsResource) Register(container *restful.Container) {
	ws := new(restful.WebService)
	ws.
	Path("/oapi/v1/namespaces/{namespace}").
	Consumes(restful.MIME_XML, restful.MIME_JSON).
	Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/buildconfigs/").To(r.getBuildConfigs))
	ws.Route(ws.GET("/buildconfigs/{name}").To(r.getBuildConfig))
	ws.Route(ws.POST("/buildconfigs").To(r.createBuildConfig))
	ws.Route(ws.POST("/buildconfigs/{name}").To(r.updateBuildConfig))
	ws.Route(ws.PUT("/buildconfigs/{name}").To(r.updateBuildConfig))
	ws.Route(ws.DELETE("/buildconfigs/{name}").To(r.removeBuildConfig))


	ws.Route(ws.GET("/builds/").To(r.getBuilds))

	// lets add a dummy templates REST service to avoid errors in the current fabric8 console ;)
	ws.Route(ws.GET("/templates/").To(r.getTemplates))

	container.Add(ws)
}

// GET http://localhost:8080/namespaces/{namespaces}/builds
//
func (r BuildConfigsResource) getBuilds(request *restful.Request, response *restful.Response) {
	buildList := oapi.BuildList{
		Items: []oapi.Build{},
	}
	response.WriteEntity(buildList)
}

// GET http://localhost:8080/namespaces/{namespaces}/buildconfigs
//
func (r BuildConfigsResource) getBuildConfigs(request *restful.Request, response *restful.Response) {
	ns := request.PathParameter("namespace")

	jenkins := r.Jenkins
	jobs, err := jenkins.GetJobs()
	if err != nil {
		errorText := fmt.Sprintf("%v", err)
		if !strings.Contains(errorText, "no such host") {
			respondError(request, response, err)
			return
		}
	}

	buildConfigs := []oapi.BuildConfig{}

	for _, job := range jobs {
		buildConfig, err := r.loadBuildConfig(ns, job.Name)
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

	buildConfig, err := r.loadBuildConfig(ns, jobName)
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

// POST http://localhost:8080/namespaces/{namespaces}/buildconfigs
//
func (r BuildConfigsResource) createBuildConfig(request *restful.Request, response *restful.Response) {
	buildConfig := oapi.BuildConfig{}
	err := request.ReadEntity(&buildConfig)
	if err != nil {
		respondError(request, response, err)
		return
	}
	ns := request.PathParameter("namespace")
	objectMeta := buildConfig.ObjectMeta
	if len(objectMeta.Namespace) == 0 {
		objectMeta.Namespace = ns
	}
	jobName := objectMeta.Name
	if len(jobName) == 0 {
		respondErrorMessage(request, response, "No BuildConfig name specified in the body")
		return
	}
	jobItem := gojenkins.JobItem{}
	populateJobForBuildConfig(&buildConfig, &jobItem)

	log.Printf("About to create job %s with structure: (%+v)", jobName, jobItem.PipelineJobItem)
	err = r.Jenkins.CreateJob(jobItem, jobName)
	if err != nil  {
		respondError(request, response, err)
		return
	}
	err = r.updateAnnotations(&buildConfig)
	if err != nil  {
		respondError(request, response, err)
		return
	}
	response.WriteHeaderAndEntity(http.StatusCreated, buildConfig)
}


// PUT http://localhost:8080/namespaces/{namespaces}/buildconfigs/{name}
//
func (r BuildConfigsResource) updateBuildConfig(request *restful.Request, response *restful.Response) {
	jobName := request.PathParameter("name")
	if len(jobName) == 0 {
		respondErrorMessage(request, response, "No BuildConfig name specified in URL")
		return
	}
	ns := request.PathParameter("namespace")
	log.Printf("Updating namespace %s buildConfig %s", ns, jobName)
	buildConfig := oapi.BuildConfig{}
	err := request.ReadEntity(&buildConfig)
	if err != nil {
		respondError(request, response, err)
		return
	}
	objectMeta := buildConfig.ObjectMeta
	if len(objectMeta.Namespace) == 0 {
		objectMeta.Namespace = ns
	}
	objectMeta.Name = jobName

	jobItem := gojenkins.JobItem{}
	populateJobForBuildConfig(&buildConfig, &jobItem)

	log.Printf("About to create job %s with structure: (%+v)", jobName, jobItem.PipelineJobItem)
	err = r.Jenkins.UpdateJob(jobItem, jobName)
	if err != nil {
		respondError(request, response, err)
		return
	}
	err = r.updateAnnotations(&buildConfig)
	response.WriteHeaderAndEntity(http.StatusOK, buildConfig)
}

// DELETE http://localhost:8080/namespaces/{namespaces}/buildconfigs/{name}
//
func (r BuildConfigsResource) removeBuildConfig(request *restful.Request, response *restful.Response) {
	jobName := request.PathParameter("name")
	if len(jobName) == 0 {
		respondErrorMessage(request, response, "No BuildConfig name specified in URL")
		return
	}
	err := r.Jenkins.RemoveJob(jobName)
	if err != nil {
		respondError(request, response, err)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, "{}")
}

// loadBuildConfig loads a BuildConfig for a given jobName
func (r BuildConfigsResource) loadBuildConfig(ns string, jobName string) (*oapi.BuildConfig, error) {
	jobUrlPath := r.Jenkins.GetJobURLPath(jobName)
	jenkins := r.Jenkins
	item, err := jenkins.GetJobConfig(jobName)
	gitUrl := ""
	gitRef := ""
	if err != nil {
		return nil, err
	}
	mavenJob := item.MavenJobItem
	pipelineJob := item.PipelineJobItem
	if mavenJob != nil {
		//log.Printf("Found maven job: (%+v)", mavenJob)
		gitUrl, gitRef = getGitUrlFromScm(mavenJob.Scm)
	} else if pipelineJob != nil {
		//log.Printf("Found pipeline job: (%+v)", pipelineJob)
		gitUrl, gitRef  = getGitUrlFromScm(pipelineJob.Definition.Scm)
	} else {
		//log.Printf("Unknown job type (%+v)", item);
		return nil, nil
	}
	buildConfig := &oapi.BuildConfig{
		ObjectMeta: kapi.ObjectMeta{
			Name: jobName,
			Namespace: ns,
			Annotations: map[string]string{
				"fabric8.io/jenkins-url-path": jobUrlPath,
			},
		},
		Spec: oapi.BuildConfigSpec{
			CommonSpec: oapi.CommonSpec{
				Source: oapi.BuildSource{
					Type: oapi.BuildSourceGit,
					Git: &oapi.GitBuildSource{
						URI: gitUrl,
						Ref: gitRef,
					},

				},
			},
		},
	}
	r.loadAnnotations(buildConfig)
	return buildConfig, nil
}

// GET http://localhost:8080/namespaces/{namespaces}/templates
//
func (r BuildConfigsResource) getTemplates(request *restful.Request, response *restful.Response) {
	templateList := tapi.TemplateList{}
	response.WriteEntity(templateList)
}

func (r BuildConfigsResource) updateAnnotations(buildConfig *oapi.BuildConfig) error {
	objectMeta := buildConfig.ObjectMeta
	annotations := objectMeta.Annotations
	if len(annotations) > 0 {
		create := false
		cmResources := r.KubeClient.ConfigMaps(r.Namespace)
		cm, err := cmResources.Get(objectMeta.Name)
		if err != nil || cm == nil {
			cm = &api.ConfigMap{
				ObjectMeta: api.ObjectMeta{
					Name: objectMeta.Name,
					Annotations: make(map[string]string),
					Labels: map[string]string{
						"project": "fabric8",
						"owner": "jenkinshift",
					},
				},
			}
			create = true
		}
		updated := false
		for k, v := range annotations {
			if cm.Annotations[k] != v {
				cm.Annotations[k] = v
				updated = true
			}
		}
		if updated {
			if create {
				_, err = cmResources.Create(cm)
			} else {
				_, err = cmResources.Update(cm)
			}
			return err
		}
	}
	return nil
}

func (r BuildConfigsResource) loadAnnotations(buildConfig *oapi.BuildConfig) {
	cmResources := r.KubeClient.ConfigMaps(r.Namespace)
	cm, err := cmResources.Get(buildConfig.ObjectMeta.Name)
	if err == nil && cm != nil {
		for k, v := range cm.Annotations {
			buildConfig.ObjectMeta.Annotations[k] = v
		}
	}
}

func populateJobForBuildConfig(buildConfig *oapi.BuildConfig, jobItem *gojenkins.JobItem) {
	gitUrls := []string{}
	ref := ""
	gitSource := buildConfig.Spec.CommonSpec.Source.Git
	if gitSource != nil {
		uri := gitSource.URI
		if len(uri) > 0 {
			gitUrls = append(gitUrls, uri)
		}
		ref = gitSource.Ref
	}
	script := `node {
	   stage 'Stage 1'
	   echo 'Hello World 1'
	   stage 'Stage 2'
	   echo 'Hello World 2'
	}`
	branches := gojenkins.Branches{}
	if len(ref) > 0 {
		branches.BranchesSpec = []gojenkins.BranchesSpec{
			{
				Name: ref,
			},
		}
	}
	jobItem.PipelineJobItem = &gojenkins.PipelineJobItem{
	 	Definition: gojenkins.PipelineDefinition{
			Script: script,
			Scm: gojenkins.Scm{
				ScmContent: &gojenkins.ScmGit{
					UserRemoteConfigs: gojenkins.UserRemoteConfigs{
						UserRemoteConfig: gojenkins.UserRemoteConfig{
							Urls: gitUrls,
						},
					},
					Branches: branches,
				},
			},

		},
	}

}

func getGitUrlFromScm(scm gojenkins.Scm) (string, string) {
	url := ""
	ref := ""
	scmContent := scm.ScmContent
	switch t := scmContent.(type) {
	case *gojenkins.ScmGit:
		urls := t.UserRemoteConfigs.UserRemoteConfig.Urls
		if len(urls) > 0 {
			url = urls[0]
		}
		if len(url) == 0 {
			url = t.GitBrowser.Url
		}
		branches := t.Branches.BranchesSpec
		if len(branches) > 0 {
			ref = branches[0].Name
		}
	}
	return url, ref
}

func respondError(request *restful.Request, response *restful.Response, err error) {
	message := fmt.Sprintf("%s", err)
	respondErrorMessage(request, response, message)
}

func respondErrorMessage(request *restful.Request, response *restful.Response, message string) {
	response.AddHeader("Content-Type", "text/plain")
	response.WriteErrorString(http.StatusNotFound, message)
}

func respondOK(request *restful.Request, response *restful.Response) {
	response.AddHeader("Content-Type", "application/json")
	response.WriteEntity("{}")

}


