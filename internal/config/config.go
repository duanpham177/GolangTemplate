package config

import (
	"errors"
	"os"
)

type Config struct {
	Port        string
	Env         string
	APIToken    string
	TCPAddr     string
	TCPToken    string
	TLSCertFile string
	TLSKeyFile  string
}

// Load reads configuration from environment variables with defaults.
func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		Env:         getEnv("APP_ENV", "development"),
		APIToken:    os.Getenv("API_TOKEN"),
		TCPAddr:     getEnv("TCP_ADDR", ":9000"),
		TCPToken:    os.Getenv("TCP_TOKEN"),
		TLSCertFile: os.Getenv("TCP_TLS_CERT_FILE"),
		TLSKeyFile:  os.Getenv("TCP_TLS_KEY_FILE"),
	}
}

func (c Config) Validate() error {
	if c.TCPToken == "" {
		return errors.New("TCP_TOKEN must be set")
	}
	if c.APIToken == "" {
		return errors.New("API_TOKEN must be set")
	}
	if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
		return errors.New("TCP_TLS_CERT_FILE and TCP_TLS_KEY_FILE must be set together")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
