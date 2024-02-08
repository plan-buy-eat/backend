package config

import (
	"os"
	"sync"

	"github.com/shoppinglist/log"
)

type Config struct {
	ServiceName       string
	HostName          string
	ServiceVersion    string
	Port              string
	OtelCollectorHost string
	//LogFileName       string
	CouchbaseConnectionString string
	CouchbaseBucketName       string
	CouchbaseUsername         string
	CouchbasePassword         string
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
		ServiceName:       getValue("SERVICE_NAME", ""),
		HostName:          getValue("HOSTNAME", hostname),
		ServiceVersion:    getValue("SERVICE_VERSION", "0.0"),
		Port:              getValue("PORT", "80"),
		OtelCollectorHost: getValue("OTEL_COLLECTOR_HOST", ""),
		// TODO: moved to log.go for now
		//LogFileName:       getValue("LOG_FILE_NAME", "/var/log/item-service.log"),
		CouchbaseConnectionString: getValue("COUCHBASE_CONNECTION_STRING", ""),
		CouchbaseBucketName:       getValue("COUCHBASE_BUCKET", ""),
		CouchbaseUsername:         getValue("COUCHBASE_USERNAME", ""),
		CouchbasePassword:         getValue("COUCHBASE_PASSWORD", ""),
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
