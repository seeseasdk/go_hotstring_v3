package prettylog

import (
	"log/slog"
	"os"
	"path/filepath"
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
	// 실행 파일 기준이 아닌 고정 경로 사용
	exePath, err := os.Executable()
	logDir := "./log"
	if err == nil {
		// exe와 같은 폴더의 log/ 디렉터리
		dir := exePath[:len(exePath)-len(filepath.Base(exePath))]
		logDir = dir + "log"
	}
	os.MkdirAll(logDir, 0755)
	logFileName := logDir + "/" + time.Now().Format("2006-01-02") + ".log"

	if _, err := os.Stat(logFileName); err != nil {
		if _, err := os.Create(logFileName); err != nil {
			slog.Error(err.Error())
		}
	}
	return logFileName
}
