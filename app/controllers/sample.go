package controllers

import (
	"log"

	"github.com/fatih/stopwatch"
	"github.com/martini-contrib/render"
	"github.com/mccraymt/ms-crypto/app/utils"
	gapi "github.com/obieq/gson-api"
)

// HandleGetSample  => Creates some sample crap and spits it out
func HandleGetSample(rendr render.Render, serverInfo gapi.JSONApiServerInfo) {
	// TODO: enable stop watch timings via debug flag
	s := stopwatch.Start(0)

	var jsonAPIError *gapi.JsonApiError
	var sampleCrap = []string{
		"Ima ignore whatever that was you posted.",
		"This right here is a sample.",
		"Enjoy responsibly.",
	}

	HandleIndexResponse(serverInfo, jsonAPIError, sampleCrap, rendr)

	duration := s.ElapsedTime()
	utils.PM("HandlesGetSample Duration")
	log.Println(duration)
}
