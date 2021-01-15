package bctest

import (
	"bufio"
	"bytes"
	"os"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func SetDebug(debug bool) {

	if debug {
		os.Setenv("DEBUG", "true") //for apiclient
		log.SetReportCaller(true)
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{FullTimestamp: false, DisableColors: true})
		defer log.SetOutput(os.Stdout)

	} else {
		os.Setenv("DEBUG", "false")
		log.SetLevel(log.WarnLevel)
		var ignore bytes.Buffer
		logignore := bufio.NewWriter(&ignore)
		log.SetOutput(logignore)
	}

}
