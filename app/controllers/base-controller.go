package controllers

import (
	"github.com/martini-contrib/render"
	"github.com/mccraymt/ms-crypto/app/utils"
	gapi "github.com/obieq/gson-api"
)

// HandleIndexResponse => wraps gson-api handler
func HandleIndexResponse(jasi gapi.JSONApiServerInfo, err *gapi.JsonApiError, result interface{}, r render.Render) {
	logError(err)
	gapi.HandleIndexResponse(jasi, err, result, r)
}

// HandleGetResponse => wraps gson-api handler
func HandleGetResponse(jasi gapi.JSONApiServerInfo, err *gapi.JsonApiError, result interface{}, r render.Render) {
	logError(err)
	gapi.HandleGetResponse(jasi, err, result, r)
}

// HandlePostResponse => wraps gson-api handler
func HandlePostResponse(jasi gapi.JSONApiServerInfo, success bool, err *gapi.JsonApiError, resource gapi.JsonApiResourcer, r render.Render) {
	logError(err)
	gapi.HandlePostResponse(jasi, success, err, resource, r)
}

// HandlePatchResponse => wraps gson-api handler
func HandlePatchResponse(jasi gapi.JSONApiServerInfo, success bool, err *gapi.JsonApiError, resource gapi.JsonApiResourcer, r render.Render) {
	logError(err)
	gapi.HandlePatchResponse(jasi, success, err, resource, r)
}

// HandleDeleteResponse => wraps gson-api handler
func HandleDeleteResponse(err *gapi.JsonApiError, r render.Render) {
	logError(err)
	gapi.HandleDeleteResponse(err, r)
}

func logError(err *gapi.JsonApiError) {
	if err != nil {
		// fields := map[string]interface{}{"err": err.Detail}
		// utils.LogError(err.Detail, &fields)
		utils.LogError(err.Detail, nil)
	}
}
