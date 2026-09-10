package config

import "testing"

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDefaults(t *testing.T) {
	c, err := FromEnv(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.DBPath != "data/deepcuts.sqlite" || c.DataDir != "data" || c.Addr != ":8080" {
		t.Fatalf("bad defaults: %+v", c)
	}
	if c.Timezone != "America/New_York" || c.Location == nil {
		t.Fatalf("bad tz: %+v", c)
	}
}

func TestOverridesAndBadTimezone(t *testing.T) {
	c, err := FromEnv(env(map[string]string{
		"DEEPCUTS_DB_PATH": "/tmp/x.sqlite", "DEEPCUTS_ADDR": ":9999",
		"DEEPCUTS_TIMEZONE": "America/Chicago", "DEEPCUTS_TLS_CERT": "c.pem", "DEEPCUTS_TLS_KEY": "k.pem",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.DBPath != "/tmp/x.sqlite" || c.Addr != ":9999" || c.Location.String() != "America/Chicago" || c.TLSCert != "c.pem" {
		t.Fatalf("overrides not applied: %+v", c)
	}
	if _, err := FromEnv(env(map[string]string{"DEEPCUTS_TIMEZONE": "Mars/Olympus"})); err == nil {
		t.Fatal("expected error for bad timezone")
	}
	if _, err := FromEnv(env(map[string]string{"DEEPCUTS_TLS_CERT": "c.pem"})); err == nil {
		t.Fatal("expected error when only one of cert/key is set")
	}
}
