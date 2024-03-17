package config

import (
	"context"
	"os"
	"sync"

	"github.com/shoppinglist/log"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
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
	CouchbaseRamQuotaMB       string

	UseStdout bool

	Tracer trace.Tracer
	Meter  metric.Meter
}

var instance *Config
var mu sync.Mutex

func Get(ctx context.Context) *Config {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Logger(ctx).Fatal().Err(err).Msg("getting hostname")
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
		CouchbaseRamQuotaMB:       getValue("COUCHBASE_RAM_QUOTA_MB", "200"),

		// hardcoded for now
		UseStdout: false,
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
