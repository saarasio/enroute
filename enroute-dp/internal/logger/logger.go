package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"github.com/rs/zerolog"
)

type EnrouteLogger struct {
	ELogger       *logrus.Logger
	BuilderLogger *logrus.Logger
	FilterLogger  *logrus.Logger
}

// Initialized when the system comes up
// logrus Logger is thread-safe
var EL EnrouteLogger

func (el *EnrouteLogger) Initialize() {
	el.ELogger = logrus.StandardLogger()
	el.ELogger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	el.ELogger.SetLevel(logrus.InfoLevel)

	el.BuilderLogger = logrus.StandardLogger()
	el.BuilderLogger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	el.BuilderLogger.SetLevel(logrus.InfoLevel)

	el.FilterLogger = logrus.StandardLogger()
	el.FilterLogger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	el.FilterLogger.SetLevel(logrus.InfoLevel)

}

func (el *EnrouteLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// level=error
	// level=info
	// level=debug
	// level=trace
	keys, ok := r.URL.Query()["level"]
	if !ok || len(keys[0]) < 1 {
		return
	}

	key := keys[0]

	fmt.Fprintf(w, "Received Log Level [%s]\n", key)

	var l logrus.Level

	if ok && EL.ELogger != nil {
		switch key {
		case "error":
			l = logrus.ErrorLevel
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
			fmt.Fprintf(w, "Set Log Level [%s]\n", key)
		case "info":
			l = logrus.InfoLevel
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			fmt.Fprintf(w, "Set Log Level [%s]\n", key)
		case "debug":
			l = logrus.DebugLevel
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
			fmt.Fprintf(w, "Set Log Level [%s]\n", key)
		case "trace":
			l = logrus.TraceLevel
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
			fmt.Fprintf(w, "Set Log Level [%s]\n", key)
		default:
			l = logrus.InfoLevel
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			fmt.Fprintf(w, "Set Log Level [%s]\n", key)
		}

		EL.ELogger.SetLevel(l)
		EL.BuilderLogger.SetLevel(l)
		EL.FilterLogger.SetLevel(l)
	}
}
