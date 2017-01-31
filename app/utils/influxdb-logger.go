package utils

/*
   COPIED DIRECTLY FROM: https://github.com/Abramovic/logrus_influxdb/blob/master/logrus_influxdb.go
   REASON: 1) connection timeout isn't configurable and it's too short
           2) username and password are hard-coded to use env var. need to use config file
*/

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	cfg "github.com/mccraymt/ms-crypto/config"
	influxdb "github.com/influxdata/influxdb/client/v2"
)

const (
	// DefaultHost default InfluxDB influxdbHostName
	DefaultHost = "localhost"

	// DefaultEnvironment default enironment where app is deployed
	DefaultEnvironment = "dev"

	// DefaultPort default InfluxDB port
	DefaultPort = 8086

	// DefaultDatabase default InfluxDB database. We'll only try to use this if one is not provided.
	DefaultDatabase = "logrus"

	// DefaultBatchInterval
	DefaultBatchInterval = 5 * time.Second
)

// InfluxDBConnection => grabs influx db connection info from config file
var InfluxDBConnection = cfg.Config.InfluxDBConnections[cfg.Config.Environment+"_"+cfg.ConstConfigInfluxDBConnectionName]

// InfluxDBHook delivers logs to an InfluxDB cluster.
type InfluxDBHook struct {
	client            influxdb.Client
	database          string
	serverHostName    string
	serverEnvironment string
	tagList           []string
	batchP            influxdb.BatchPoints
	lastBatchUpdate   time.Time
	batchInterval     time.Duration
}

// NewInfluxDBHook creates a hook to be added to an instance of logger and initializes the InfluxDB client
func NewInfluxDBHook(
	influxdbHostName, database, serverHostName, serverEnvironment string,
	tagList []string,
) (*InfluxDBHook, error) {

	if influxdbHostName == "" {
		influxdbHostName = DefaultHost
	}

	if serverEnvironment == "" {
		serverEnvironment = DefaultEnvironment
	}

	// use the default database if we're missing one in the initialization
	if database == "" {
		database = DefaultDatabase
	}

	if tagList == nil { // if no tags exist then make an empty map[string]string
		tagList = []string{}
	}

	batchInterval := DefaultBatchInterval

	client, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s:%d", influxdbHostName, DefaultPort),
		Username: InfluxDBConnection.UserName,
		Password: InfluxDBConnection.Password,
		// Username: os.Getenv("INFLUX_USER"),
		// Password: os.Getenv("INFLUX_PWD"),
		Timeout: 2000 * time.Millisecond,
	})
	if err != nil {
		return nil, fmt.Errorf("NewInfluxDBHook: Error creating InfluxDB Client, %v", err)
	}
	defer client.Close()

	hook := &InfluxDBHook{
		client:            client,
		database:          database,
		serverHostName:    serverHostName,
		serverEnvironment: serverEnvironment,
		tagList:           tagList,
		batchInterval:     batchInterval,
	}

	err = hook.autocreateDatabase()
	if err != nil {
		return nil, err
	}

	return hook, nil
}

// NewWithClientInfluxDBHook creates a hook using an initialized InfluxDB client.
func NewWithClientInfluxDBHook(
	client influxdb.Client,
	database string,
	tagList []string,
) (*InfluxDBHook, error) {
	// use the default database if we're missing one in the initialization
	if database == "" {
		database = DefaultDatabase
	}

	if tagList == nil { // if no tags exist then make an empty map[string]string
		tagList = []string{}
	}

	batchInterval := DefaultBatchInterval

	// If the configuration is nil then assume default configurations
	if client == nil {
		return NewInfluxDBHook(DefaultHost, database, DefaultHost, DefaultEnvironment, tagList)
	}
	return &InfluxDBHook{
		client:        client,
		database:      database,
		tagList:       tagList,
		batchInterval: batchInterval,
	}, nil
}

// Fire is called when an event should be sent to InfluxDB
func (hook *InfluxDBHook) Fire(entry *logrus.Entry) error {
	// Merge all of the fields from Logrus as one entry in InfluxDB
	fields := entry.Data

	// If passing a "message" field then it will be overridden by the entry Message
	if entry.Message != ConstEmptyMessage { // only set message if it's not empty
		fields["message"] = entry.Message
	}

	// Create a new point batch
	if hook.batchP == nil {
		var err error
		hook.batchP, err = influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database: hook.database,
		})
		if err != nil {
			return fmt.Errorf("Fire: %v", err)
		}
	}

	var measurement string
	var ok bool
	if measurement, ok = getTag(entry.Data, "measurement"); !ok {
		measurement = "logrus"
	}

	tags := make(map[string]string)
	// Set the level of the entry
	tags["level"] = entry.Level.String()

	// getAndDel and getAndDelRequest are taken from https://github.com/evalphobia/logrus_sentry
	if logger, ok := getTag(entry.Data, "logger"); ok {
		tags["logger"] = logger
	}

	// Set the Server Host Name
	tags[ConstLogFieldServerHostName] = hook.serverHostName

	// Set the Server environment
	tags[ConstLogFieldServerEvironment] = hook.serverEnvironment

	for _, tag := range hook.tagList {
		if tagValue, ok := getTag(entry.Data, tag); ok {
			tags[tag] = tagValue
		}
	}

	pt, err := influxdb.NewPoint(
		measurement,
		tags,
		fields,
		entry.Time,
	)
	if err != nil {
		return fmt.Errorf("Fire: %v", err)
	}

	hook.batchP.AddPoint(pt)

	// Arbitrary length of points trigger, just to make sure it doesn't overflow
	if hook.lastBatchUpdate.Add(hook.batchInterval).Before(time.Now()) ||
		len(hook.batchP.Points()) > 200 {
		err = hook.client.Write(hook.batchP)
		if err != nil {
			return fmt.Errorf("Fire: %v", err)
		}
		hook.lastBatchUpdate = time.Now()
		hook.batchP = nil
	}

	return nil
}

// queryDB convenience function to query the database
func (hook *InfluxDBHook) queryDB(cmd string) ([]influxdb.Result, error) {
	response, err := hook.client.Query(influxdb.Query{
		Command:  cmd,
		Database: hook.database,
	})
	if err != nil {
		return nil, err
	}
	if response.Error() != nil {
		return nil, response.Error()
	}

	return response.Results, nil
}

// Return back an error if the database does not exist in InfluxDB
func (hook *InfluxDBHook) databaseExists() error {
	results, err := hook.queryDB("SHOW DATABASES")
	if err != nil {
		return err
	}
	if results == nil || len(results) == 0 {
		return errors.New("Missing results from InfluxDB query response")
	}
	if results[0].Series == nil || len(results[0].Series) == 0 {
		return errors.New("Missing series from InfluxDB query response")
	}

	// This can probably be cleaned up
	for _, value := range results[0].Series[0].Values {
		for _, val := range value {
			if v, ok := val.(string); ok { // InfluxDB returns back an interface. Try to check only the string values.
				if v == hook.database { // If we the database exists, return back nil errors
					return nil
				}
			}
		}
	}
	return errors.New("No matching database can be detected")
}

// Try to detect if the database exists and if not, automatically create one.
func (hook *InfluxDBHook) autocreateDatabase() error {
	err := hook.databaseExists()
	if err == nil {
		return nil
	}

	_, err = hook.queryDB(fmt.Sprintf("CREATE DATABASE %s", hook.database))
	if err != nil {
		return err
	}

	return nil
}

// Try to return a field from logrus
// Taken from Sentry adapter (from https://github.com/evalphobia/logrus_sentry)
func getTag(d logrus.Fields, key string) (string, bool) {

	var ok bool
	var v interface{}

	if v, ok = d[key]; !ok {
		return "", false
	}

	switch vs := v.(type) {
	case fmt.Stringer:
		return vs.String(), true
	case string:
		return vs, true
	case int:
		return strconv.FormatInt(int64(vs), 10), true
	case int32:
		return strconv.FormatInt(int64(vs), 10), true
	case int64:
		return strconv.FormatInt(vs, 10), true
	case uint:
		return strconv.FormatUint(uint64(vs), 10), true
	case uint32:
		return strconv.FormatUint(uint64(vs), 10), true
	case uint64:
		return strconv.FormatUint(vs, 10), true
	default:
		return "", false
	}
}

// Try to return an http request
// Taken from Sentry adapter (from https://github.com/evalphobia/logrus_sentry)
func getRequest(d logrus.Fields, key string) (*http.Request, bool) {
	var (
		ok  bool
		v   interface{}
		req *http.Request
	)
	if v, ok = d[key]; !ok {
		return nil, false
	}
	if req, ok = v.(*http.Request); !ok || req == nil {
		return nil, false
	}
	return req, true
}

// Levels is available logging levels.
func (hook *InfluxDBHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}
