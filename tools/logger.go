package tools

import (
	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

func InitLogger() {
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		DisableColors:   false,
		PadLevelText:    true,
	})
	Log.SetLevel(logrus.InfoLevel) // or DebugLevel
}

func LogSyncSummary(category, name string, userCount, added, removed int) {
	Log.Infof("[%s:%s] users=%d added=%d removed=%d", category, name, userCount, added, removed)
}
