package log

import (
	"github.com/rs/zerolog"
	"os"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

func Logger() *zerolog.Logger {
	return &logger
}
