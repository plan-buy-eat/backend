package config

import (
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

	instance = &Config{
		ServiceName:    getValue("SERVICE_NAME", ""),
		HostName:       getValue("HOSTNAME", "localhost"),
		ServiceVersion: getValue("SERVICE_VERSION", ""),
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
