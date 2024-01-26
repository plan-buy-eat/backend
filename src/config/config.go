package config

import (
	"github.com/shoppinglist/log"
	"os"
	"sync"
)

type Config struct {
	ServiceName    string
	HostName       string
	ServiceVersion string
	Port           string
}

var instance *Config
var mu sync.Mutex

func Get() *Config {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Logger().Fatal().Err(err).Msg("getting hostname")
	}
	instance = &Config{
		ServiceName:    getValue("SERVICE_NAME", ""),
		HostName:       getValue("HOSTNAME", hostname),
		ServiceVersion: getValue("SERVICE_VERSION", "0.0"),
		Port:           getValue("PORT", "80"),
	}

	return instance
}

func getValue(key, def string) string {
	val, found := os.LookupEnv(key)
	if !found {
		val = def
	}
	return val
}
