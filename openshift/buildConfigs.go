package openshift

import (
	/*
	"log"
	"net/http"
	*/

	"github.com/emicklei/go-restful"
	kapi "k8s.io/kubernetes/pkg/api/v1"
	oapi "github.com/openshift/origin/pkg/build/api/v1"
)

type BuildConfigsResource struct {
}


func (u BuildConfigsResource) Register(container *restful.Container) {
	ws := new(restful.WebService)
	ws.
	Path("/namespaces/{namespace}/buildConfigs").
	Consumes(restful.MIME_XML, restful.MIME_JSON).
	Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("/").To(u.findBuildConfigs))
	/*
	ws.Route(ws.GET("/{user-id}").To(u.findUser))
	ws.Route(ws.POST("").To(u.updateUser))
	ws.Route(ws.PUT("/{user-id}").To(u.createUser))
	ws.Route(ws.DELETE("/{user-id}").To(u.removeUser))
	*/

	container.Add(ws)
}

// GET http://localhost:8080/namespaces/{namespaces}/buildConfigs
//
func (u BuildConfigsResource) findBuildConfigs(request *restful.Request, response *restful.Response) {
	ns := request.PathParameter("namespace")

	buildConfigs := []oapi.BuildConfig{
		oapi.BuildConfig{
			ObjectMeta: kapi.ObjectMeta{
				Name: "foo",
				Namespace: ns,
			},
			Spec: oapi.BuildConfigSpec{
				BuildSpec: oapi.BuildSpec{
					Source: oapi.BuildSource{
						Type: oapi.BuildSourceGit,
						Git: &oapi.GitBuildSource{
							URI: "http://github.com/foo/bar.git",
						},

					},
				},
			},
		},
	}
	response.WriteEntity(buildConfigs)
}


