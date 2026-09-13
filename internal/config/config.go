// Package config reads DEEPCUTS_* environment variables into a Config.
package config

import (
	"fmt"
	"strconv"
	"time"
)

type Config struct {
	DBPath   string
	DataDir  string
	Addr     string
	Timezone string
	Location *time.Location
	TLSCert  string
	TLSKey   string
	// SecureCookies marks session cookies Secure even when this process itself
	// isn't terminating TLS -- for deployments behind a TLS-terminating proxy.
	SecureCookies bool
}

func FromEnv(getenv func(string) string) (Config, error) {
	get := func(k, def string) string {
		if v := getenv(k); v != "" {
			return v
		}
		return def
	}
	c := Config{
		DBPath:   get("DEEPCUTS_DB_PATH", "data/deepcuts.sqlite"),
		DataDir:  get("DEEPCUTS_DATA_DIR", "data"),
		Addr:     get("DEEPCUTS_ADDR", ":8080"),
		Timezone: get("DEEPCUTS_TIMEZONE", "America/New_York"),
		TLSCert:  getenv("DEEPCUTS_TLS_CERT"),
		TLSKey:   getenv("DEEPCUTS_TLS_KEY"),
	}
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return Config{}, fmt.Errorf("DEEPCUTS_TIMEZONE %q: %w", c.Timezone, err)
	}
	c.Location = loc
	if v := getenv("DEEPCUTS_SECURE_COOKIES"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("DEEPCUTS_SECURE_COOKIES %q: %w", v, err)
		}
		c.SecureCookies = b
	}
	if (c.TLSCert == "") != (c.TLSKey == "") {
		return Config{}, fmt.Errorf("DEEPCUTS_TLS_CERT and DEEPCUTS_TLS_KEY must be set together")
	}
	return c, nil
}

// TLS reports whether the server should listen with TLS.
func (c Config) TLS() bool { return c.TLSCert != "" }
