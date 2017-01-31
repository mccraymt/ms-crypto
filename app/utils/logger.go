package utils

import (
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	cfg "github.com/mccraymt/ms-crypto/config"
	"github.com/go-martini/martini"
	loggly "github.com/sebest/logrusly"
)

// ConstEmptyMessage => loggly throws a silent error if an empty message is provided
const ConstEmptyMessage = "."

// ConstLogFieldEvent => event name (bind, click, page view, etc.)
const ConstLogFieldEvent = "e"

// ConstLogFieldRequestMethod => HTTP request method (GET, POST, etc)
const ConstLogFieldRequestMethod = "rm"

// ConstLogFieldRequestDurationInMS => request duration in milliseconds (start to finish)
const ConstLogFieldRequestDurationInMS = "rdms"

// ConstLogFieldRequestPath => request route (https://ms-crypto/{id}/quote-intents)
const ConstLogFieldRequestPath = "rp"

// ConstLogFieldRequestIP => client IP address
const ConstLogFieldRequestIP = "rip"

// ConstLogFieldRequestStatus => HTTP response code (200, 401, etc)
const ConstLogFieldRequestStatus = "rs" // => HTTP response code (200, 401, etc)

// ConstLogFieldServerEvironment => the environment config value for the server running this code (ci, qa1, prod)
const ConstLogFieldServerEvironment = "env"

// ConstLogFieldServerHostName => the host name of the server running this code
const ConstLogFieldServerHostName = "host"

var isProd = checkIfProdEnv()

func checkIfProdEnv() bool {
	return strings.ToLower(cfg.Config.Environment) == "prod"
}

func buildInfluxDBTagList() []string {
	return []string{
		ConstLogFieldEvent,
		ConstLogFieldRequestMethod,
		// ConstLogFieldRequestDurationInMS, TODO: can we/should we tag this field?
		ConstLogFieldRequestPath,
		ConstLogFieldRequestIP,
		ConstLogFieldRequestStatus,
		ConstLogFieldServerEvironment,
		ConstLogFieldServerHostName,
	}
}

func init() {
	// Output to stderr instead of stdout, could also be a file.
	log.SetOutput(os.Stderr)

	// configure logging settings based on environment
	env := cfg.Config.Environment
	logLevel := log.DebugLevel // default log level = DEBUG

	if env == "dev" {
		// Log as TEXT and force colors in terminal
		log.SetFormatter(&log.TextFormatter{ForceColors: true})

		// Set log level = DEBUG
	} else {
		// Log as JSON instead of the default ASCII formatter.
		log.SetFormatter(&log.JSONFormatter{})

		// Set log log level = INFO
		logLevel = log.InfoLevel
	}

	// configure logrus log level
	log.SetLevel(logLevel)

	// configure loggly
	hostName, _ := os.Hostname()
	logglyHook := loggly.NewLogglyHook(cfg.Config.LogglyKey, hostName, logLevel, env, "ms-crypto")
	log.Println("Adding Loggly logging hook")
	log.AddHook(logglyHook)

	// configure influxdb
	// influxdbHook, err := logrus_influxdb.NewInfluxDBHook("ec2-52-90-45-8.compute-1.amazonaws.com", "webdb", nil)
	influxdbTagList := buildInfluxDBTagList()
	influxdbHook, err := NewInfluxDBHook(InfluxDBConnection.Host, InfluxDBConnection.DBName, hostName, env, influxdbTagList)
	if err == nil { //|| err.Error() == "database already exists" {
		log.Println("Adding InfluxDB logging hook")
		log.AddHook(influxdbHook)
	} else {
		log.Error("InfluxDB hook configuration failed: " + err.Error())
	}
}

// LogError => logs a message and optional key/values with log level = ERROR
func LogError(errorMsg string, kvs *map[string]interface{}) {
	fields := log.Fields{}

	if kvs != nil {
		for key, value := range *kvs {
			fields[key] = value
		}
	}

	log.WithFields(fields).Error(errorMsg)
}

// MartiniLogger => returns a martini middleware handler that logs requests w/ response times
func MartiniLogger() martini.Handler {
	return func(res http.ResponseWriter, req *http.Request, c martini.Context) {
		var dump []byte
		start := time.Now() // start capturing duration as soon as the method is called
		rw := res.(martini.ResponseWriter)

		// ---- PRE RESPONSE PROCESSING --------------------------------------------

		// grab ip address
		addr := req.Header.Get("X-Real-IP")
		if addr == "" {
			addr = req.Header.Get("X-Forwarded-For")
			if addr == "" {
				// addr = strings.Split(req.RemoteAddr, ":")[0]
				addr = req.RemoteAddr // the above line doesn't work for mac osx, so just use the whole remote addr
			}
		}

		// grab request JSON for all non-prod environments b/c we'll send it when we log the error
		// NOTE: we won't log prod due to personally identifiable info being stored on Loggly
		if !isProd {
			dump, _ = httputil.DumpRequest(req, true)
		}

		c.Next() // call next martini handler

		// ---- POST RESPONSE PROCESSING -------------------------------------------
		// log.Debug(fmt.Sprintf("%s - %s %d \"%s %s\" %s", addr, time.Now().UTC().String(), rw.Status(), req.Method, req.URL.Path, time.Since(start)))

		// TODO: consolidate this code w/ dupe code in martini-recovery.go in a helper method
		if req.Method != "OPTIONS" { // don't log options calls
			fields := log.Fields{
				ConstLogFieldRequestDurationInMS: time.Since(start).Nanoseconds() / 1e6,
				ConstLogFieldEvent:               "api",
				ConstLogFieldRequestIP:           addr,
				ConstLogFieldRequestMethod:       req.Method,
				ConstLogFieldRequestPath:         req.URL.Path,
				ConstLogFieldRequestStatus:       rw.Status(),
			}

			if rw.Status() < 300 { // shouldn't have any 300 level api calls for this svc
				log.WithFields(fields).Info(ConstEmptyMessage)
			} else {
				// check for request dump (should be non-prod requests)
				if len(dump) > 0 {
					requestDump := string(dump)
					requestDump = strings.Replace(requestDump, "\n", "", -1)
					requestDump = strings.Replace(requestDump, `"`, "`", -1)
					fields["request"] = requestDump
				}

				log.WithFields(fields).Error(".")
			} // rw.Status() < 300
		} // req.Method != "Options"
	} // return func
} // MartiniLogger
