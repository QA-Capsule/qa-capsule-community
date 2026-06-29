package server

import (
	"net"
	"testing"
	"time"
)

func TestIsSSRFSafeURL_blocksPrivateIPs(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/admin",
		"http://localhost/",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/",
		"ftp://example.com/",
		"",
		"not-a-url",
	}
	for _, u := range cases {
		if isSSRFSafeURL(u) {
			t.Fatalf("expected blocked URL: %q", u)
		}
	}
}

func TestIsSSRFSafeURL_allowsPublicHost(t *testing.T) {
	if !isSSRFSafeURL("https://example.com/page") {
		t.Fatal("expected public URL to be allowed")
	}
}

func TestIsPrivateOrRestrictedIP(t *testing.T) {
	if !isPrivateOrRestrictedIP(net.ParseIP("127.0.0.1")) {
		t.Fatal("loopback should be private")
	}
	if isPrivateOrRestrictedIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public IP should not be private")
	}
}

func TestRateLimiter_blocksBurst(t *testing.T) {
	rl := &rateLimiter{hits: make(map[string][]time.Time)}
	key := "test:127.0.0.1"
	for i := 0; i < 5; i++ {
		if !rl.allow(key, 5, time.Minute) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if rl.allow(key, 5, time.Minute) {
		t.Fatal("6th request should be blocked")
	}
}
