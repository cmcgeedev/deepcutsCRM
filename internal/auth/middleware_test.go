package auth

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{"no xff uses remote addr", "203.0.113.5:1234", "", "203.0.113.5"},
		{"xff from loopback remote addr is trusted", "127.0.0.1:1234", " 10.1.2.3 , 9.9.9.9", "10.1.2.3"},
		{"xff from public remote addr is ignored", "203.0.113.5:1234", "10.1.2.3", "203.0.113.5"},
		{"garbage xff falls back to remote addr", "127.0.0.1:1234", "not-an-ip", "127.0.0.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = c.remoteAddr
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			if got := ClientIP(r); got != c.want {
				t.Fatalf("ClientIP(remoteAddr=%q, xff=%q) = %q, want %q", c.remoteAddr, c.xff, got, c.want)
			}
		})
	}
}
