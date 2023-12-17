package config

import "os"

func GetValue(key, def string) string {
	port, found := os.LookupEnv(key)
	if !found {
		port = def
	}
	return port
}
