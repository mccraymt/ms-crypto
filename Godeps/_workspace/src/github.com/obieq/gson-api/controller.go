package gsonapi

import (
	"encoding/json"

	"github.com/manyminds/api2go/jsonapi"
	"github.com/martini-contrib/render"
)

const GSON_API_RESPONSE_HEADER = "application/vnd.api+json; charset=UTF-8"

// JSONApiServerInfo => contains necessary info for building an api's route
type JSONApiServerInfo struct {
	BaseURL string
	Prefix  string
}

// GetBaseURL => api routes base url
// EX: https://test.myapi.com/.....
func (jasi JSONApiServerInfo) GetBaseURL() string {
	return jasi.BaseURL
}

// GetPrefix => api route's prefix
// EX: https://xxxxx.com/v1/xxxxx (v1 is the prefix)
func (jasi JSONApiServerInfo) GetPrefix() string {
	return jasi.Prefix
}

// JSON => wraps the martini contrib Render method in order to set the
//         appropriate header values
func JSON(r render.Render, status int, v interface{}) {
	r.JSON(status, v)
	r.Header().Set("Content-Type", GSON_API_RESPONSE_HEADER)
}

func HandleIndexResponse(jasi JSONApiServerInfo, err *JsonApiError, result interface{}, r render.Render) {
	var j []byte
	var jsonError error
	var response interface{}

	if err == nil {
		// JSON(r,200, map[string]interface{}{"links": link, "data": result}) // TODO: return links before data
		if j, jsonError = jsonapi.MarshalToJSONWithURLs(result, jasi); jsonError != nil {
			JSON(r, 400, map[string]interface{}{"errors": jsonError})
		}
		if jsonError = json.Unmarshal(j, &response); jsonError != nil {
			JSON(r, 400, map[string]interface{}{"errors": jsonError})
		}
		JSON(r, 200, response)
	} else {
		JSON(r, 404, map[string]interface{}{"errors": err})
	}
}

func HandleGetResponse(jasi JSONApiServerInfo, err *JsonApiError, result interface{}, r render.Render) {
	var j []byte
	var jsonError error
	var response interface{}

	if err == nil {
		if j, jsonError = jsonapi.MarshalToJSONWithURLs(result, jasi); jsonError != nil {
			JSON(r, 400, map[string]interface{}{"errors": jsonError})
		}
		if jsonError = json.Unmarshal(j, &response); jsonError != nil {
			JSON(r, 400, map[string]interface{}{"errors": jsonError})
		}
		JSON(r, 200, response)
	} else {
		JSON(r, 404, map[string]interface{}{"errors": err})
	}
}

// HandlePostResponse => formats appropriate JSON response based on success vs. error
func HandlePostResponse(jasi JSONApiServerInfo, success bool, err *JsonApiError, resource JsonApiResourcer, r render.Render) {
	// TODO: return 404 if resource not found
	var j []byte
	var jsonError error
	var response interface{}

	if success {
		// TODO: retrieve from the database instead of re-using instance
		// TODO: implement via Api2Go => r.Header().Set("Location", LinkSelfInstance(resource))
		if j, jsonError = jsonapi.MarshalToJSONWithURLs(resource, jasi); jsonError != nil {
			JSON(r, 400, map[string]interface{}{"errors": jsonError})
		}
		if jsonError = json.Unmarshal(j, &response); jsonError != nil {
			JSON(r, 400, map[string]interface{}{"errors": jsonError})
		}
		JSON(r, 201, response)
	} else if err != nil {
		// TODO: how do I parse the status code?
		JSON(r, 400, map[string]interface{}{"errors": err})
	} else {
		JSON(r, 422, map[string]interface{}{"errors": resource.Errors()})
	}
}

// HandlePatchResponse => formats appropriate JSON response based on success vs. error
func HandlePatchResponse(jasi JSONApiServerInfo, success bool, err *JsonApiError, resource JsonApiResourcer, r render.Render) {
	var j []byte
	var jsonError error
	var response interface{}

	if success {
		// TODO: retrieve from the database instead of re-using instance
		// TODO: implement via Api2Go => r.Header().Set("Location", LinkSelfInstance(resource))
		if j, jsonError = jsonapi.MarshalToJSONWithURLs(resource, jasi); jsonError != nil {
			JSON(r, 400, map[string]interface{}{"errors": jsonError})
		}
		if jsonError = json.Unmarshal(j, &response); jsonError != nil {
			JSON(r, 400, map[string]interface{}{"errors": jsonError})
		}
		JSON(r, 200, response) // given that updated-at is set, a 200 w/ content must be returned
	} else if err != nil {
		// TODO: how do I parse the status code?
		JSON(r, 400, map[string]interface{}{"errors": err})
		//JSON(r,412, map[string]interface{}{"errors": err})
	} else {
		JSON(r, 422, map[string]interface{}{"errors": resource.Errors()})
	}
}

func HandleDeleteResponse(err *JsonApiError, r render.Render) {
	if err == nil {
		JSON(r, 204, map[string]interface{}{})
	} else {
		JSON(r, 400, map[string]interface{}{"errors": err})
	}
}
