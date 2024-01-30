package log

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"os"

	"github.com/rs/zerolog"
)

func init() {
	fileName := os.Getenv("LOG_FILE_NAME")
	if fileName == "" {
		fileName = "var/log/shoppinglist/item-service.log"
	}

	fileLogger := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    5, //
		MaxBackups: 10,
		MaxAge:     14,
		Compress:   false,
	}

	output := zerolog.MultiLevelWriter(os.Stderr, fileLogger)
	logger = zerolog.New(output).With().Timestamp().Caller().Logger()
}

var logger zerolog.Logger

func Logger() *zerolog.Logger {
	return &logger
}
