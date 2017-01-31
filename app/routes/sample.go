package routes

import (
	"github.com/go-martini/martini"
	c "github.com/mccraymt/ms-crypto/app/controllers"
	gapi "github.com/obieq/gson-api"
)

var sampleServerInfo gapi.JSONApiServerInfo

func getSampleServerInfo(c martini.Context) {
	c.Map(sampleServerInfo)
	c.Next()
}

func LoadSampleRoutes(r martini.Router, serverInfo gapi.JSONApiServerInfo) {
	sampleServerInfo = serverInfo
	r.Get("/sample", getSampleServerInfo, c.HandleGetSample)
}
