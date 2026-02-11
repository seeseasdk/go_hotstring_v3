package prettylog

import (
	"log/slog"
	"os"
	"time"
)

func init() {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "nothing" {
				return slog.Attr{}
			}
			return a
		}}

	logger := slog.New(PrettylogNewHandler(opts, getLogFilePath()))

	slog.SetDefault(logger)

}

func InitLogger() *slog.Logger {
	return slog.Default()
}

func getLogFilePath() string {
	logFileName := time.Now().Format("./log/2006-01-02") + ".log"

	if _, err := os.Stat(logFileName); err != nil {
		if _, err := os.Create(logFileName); err != nil {
			slog.Error(err.Error())
		}
	}
	return logFileName
}
