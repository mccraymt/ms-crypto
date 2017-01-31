package gsonapi

import (
	"log"

	"github.com/obieq/gas"
)

// Config => global variable that stores the config values
var Config *config

func init() {
	Config = newConfig()
}

// config => stores a map[string]string of all values parse from an .env file
type config struct {
	gas.Config
	URL string
}

func newConfig() *config {
	c := &config{}
	c.ParseConfigFile("config")
	c.Validate()

	return c
}

// ParseConfigFile => reads the config values from the config.json file
func (c *config) ParseConfigFile(configFileName string) (err error) {
	err = c.Load("gson-api", "config.json", true)

	// get api base url
	c.URL = gas.GetString("gson_api_url")

	return err
}

func (c *config) Validate() {
	if c.URL == "" {
		log.Panicln("gson api config error: URL cannot be blank")
	}
}
