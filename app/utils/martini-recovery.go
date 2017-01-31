package utils

/*
   COPIED DIRECTLY FROM: https://github.com/go-martini/martini/blob/master/recovery.go
   REASON: can't inject logrus logger into Martini.
           As a result, Panics aren't logged to loggly.  So.... here's a workaround
*/

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"runtime"
	"strings"

	log "github.com/Sirupsen/logrus" // this is the key... removed the standard library log package
	"github.com/codegangsta/inject"
	"github.com/go-martini/martini"
)

const (
	panicHTML = `<html>
<head><title>PANIC: %s</title>
<style type="text/css">
html, body {
	font-family: "Roboto", sans-serif;
	color: #333333;
	background-color: #ea5343;
	margin: 0px;
}
h1 {
	color: #d04526;
	background-color: #ffffff;
	padding: 20px;
	border-bottom: 1px dashed #2b3848;
}
pre {
	margin: 20px;
	padding: 20px;
	border: 2px solid #2b3848;
	background-color: #ffffff;
}
</style>
</head><body>
<h1>PANIC</h1>
<pre style="font-weight: bold;">%s</pre>
<pre>%s</pre>
</body>
</html>`
)

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")
)

// stack returns a nicely formated stack frame, skipping skip frames
func stack(skip int) []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contains dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastslash := bytes.LastIndex(name, slash); lastslash >= 0 {
		name = name[lastslash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

// Recovery returns a middleware that recovers from any panics and writes a 500 if there was one.
// While Martini is in development mode, Recovery will also output the panic as HTML.
func Recovery() martini.Handler {
	return func(req *http.Request, c martini.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := stack(3)
				// log.Printf("PANIC IN THE STREETS OF BIRMINGHAM: %s\n%s", err, stack)
				log.Errorf("PANIC IN THE STREETS OF BIRMINGHAM: %s\n%s", err, stack)

				// Lookup the current responsewriter
				val := c.Get(inject.InterfaceOf((*http.ResponseWriter)(nil)))
				res := val.Interface().(http.ResponseWriter)

				// respond with panic message while in development mode
				var body []byte
				if martini.Env == martini.Dev {
					res.Header().Set("Content-Type", "text/html")
					body = []byte(fmt.Sprintf(panicHTML, err, err, stack))
				} else {
					body = []byte("500 Internal Server Error")
				}

				res.WriteHeader(http.StatusInternalServerError)
				if nil != body {
					res.Write(body)
				}

				// finally, log request for troubleshooting purposes
				// NOTE: personally identifiable info will be stored on loggly as a result,
				//       but... a) should happen very infrequently b) need it for debugging purposes
				// TODO: 1) just log to influxdb, which is hosted internally???
				//       2) encrypt if running in prod?
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					log.Error("Panic Request Dump failed: " + err.Error())
				} else {
					requestDump := string(dump)
					requestDump = strings.Replace(requestDump, "\n", "", -1)
					requestDump = strings.Replace(requestDump, `"`, "`", -1)

					// grab ip address
					addr := req.Header.Get("X-Real-IP")
					if addr == "" {
						addr = req.Header.Get("X-Forwarded-For")
						if addr == "" {
							addr = req.RemoteAddr // the above line doesn't work for mac osx, so just use the whole remote addr
						}
					}

					// build fields
					fields := log.Fields{
						"e":    "api",        // e    => event
						"ip":   addr,         // ip   => ip address
						"m":    req.Method,   // m    => request method (GET, POST, etc)
						"p":    req.URL.Path, // p    => request path
						"code": 500,          // code => HTTP response code (200, 401, etc)
					}

					log.WithFields(fields).Error(requestDump)
				} // dump err != nil
			} //recover() err != nil
		}()

		c.Next()
	}
}
