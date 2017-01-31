package utils

import log "github.com/Sirupsen/logrus"

// P => prints BREAK
func P() {
	log.Debug("-------------------- BREAK --------------------")
}

// PM => prints BREAK followed by message text
func PM(message string) {
	log.Debug("-------------------- BREAK " + message + " --------------------")
}
