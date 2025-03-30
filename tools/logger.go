package tools

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

type SyncMetrics struct {
	GroupEmail    string
	TotalUsers    int
	ADAdded       int
	ADRemoved     int
	GoogleAdded   int
	GoogleRemoved int
}

var Log = logrus.New()

func InitLogger() {
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		DisableColors:   false,
		PadLevelText:    true,
	})
	Log.SetLevel(logrus.InfoLevel)
}

func LogSyncCombined(m SyncMetrics) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	blue := color.New(color.FgCyan).SprintFunc()

	adAdd := green(fmt.Sprintf("+%3d", m.ADAdded))
	adRemove := red(fmt.Sprintf("-%3d", m.ADRemoved))
	gsAdd := green(fmt.Sprintf("+%3d", m.GoogleAdded))
	gsRemove := red(fmt.Sprintf("-%3d", m.GoogleRemoved))

	Log.Infof(
		"[SYNC] %-45s | Users: %4d | AD: %s / %s | Google: %s / %s",
		blue(m.GroupEmail),
		m.TotalUsers,
		adAdd, adRemove,
		gsAdd, gsRemove,
	)
}
