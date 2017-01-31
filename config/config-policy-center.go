package config

import (
	log "github.com/Sirupsen/logrus"
	"github.com/obieq/gas"
)

// ConstConfigPCConnectionName => constant key for locating policy center config info
const ConstConfigPCConnectionName = "policy_center"

// PolicyCenterConfig => contains policy center connection info
type PolicyCenterConfig struct {
	ConnectionName string
	UserName       string
	Password       string
	URI            string
}

func (c *config) loadPolicyCenter() {
	path := c.Environment + "." + ConstConfigPCConnectionName
	connName := c.Environment + "_" + ConstConfigPCConnectionName
	pc := &PolicyCenterConfig{}

	pc.ConnectionName = connName
	pc.URI = gas.GetString(path + ".edge_services_address")
	pc.UserName = gas.GetString(path + ".edge_services_user")
	pc.Password = gas.GetString(path + ".edge_services_password")

	if pc.URI == "" {
		log.Panic("PC Connection URI cannot be blank")
	} else if pc.UserName == "" {
		log.Panic("PC Connection UserName cannot be blank")
	} else if pc.Password == "" {
		log.Panic("PC Connection Password cannot be blank")
	} else {
		log.Info("(config) PC Connection URI: " + pc.URI)
	}

	c.PCConnections[connName] = pc
}
