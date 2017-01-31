package config

import (
	log "github.com/Sirupsen/logrus"
	"github.com/obieq/gas"
)

// ConstConfigInfluxDBConnectionName => constant key for locating policy center config info
const ConstConfigInfluxDBConnectionName = "influx_db"

// InfluxDBConfig => contains policy center connection info
type InfluxDBConfig struct {
	ConnectionName string
	Host           string
	DBName         string
	UserName       string
	Password       string
}

func (c *config) loadInfluxDB() {
	path := c.Environment + "." + ConstConfigInfluxDBConnectionName
	connName := c.Environment + "_" + ConstConfigInfluxDBConnectionName
	ic := &InfluxDBConfig{}

	ic.ConnectionName = connName
	ic.Host = gas.GetString(path + ".host")
	ic.DBName = gas.GetString(path + ".db_name")
	ic.UserName = gas.GetString(path + ".username")
	ic.Password = gas.GetString(path + ".password")

	if ic.Host == "" {
		log.Panic("InfluxDB Connection Host cannot be blank")
	} else if ic.DBName == "" {
		log.Panic("InfluxDB Connection DBName cannot be blank")
	} else if ic.UserName == "" {
		log.Panic("InfluxDB Connection UserName cannot be blank")
	} else if ic.Password == "" {
		log.Panic("InfluxDB Connection Password cannot be blank")
	} else {
		log.Info("(config) InfluxDB Host: " + ic.Host)
		log.Info("(config) InfluxDB DB Name: " + ic.DBName)
	}

	c.InfluxDBConnections[connName] = ic
}
