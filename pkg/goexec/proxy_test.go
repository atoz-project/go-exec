package goexec

import "testing"

func TestParseProxyURI_Socks5(t *testing.T) {
	d, err := ParseProxyURI("socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatalf("ParseProxyURI returned error: %v", err)
	}
	if d == nil {
		t.Fatalf("ParseProxyURI returned nil dialer")
	}
}

func TestParseProxyURI_InvalidURI(t *testing.T) {
	if _, err := ParseProxyURI("://bad"); err == nil {
		t.Fatalf("expected error for invalid proxy URI")
	}
}

func TestParseProxyURI_UnsupportedScheme(t *testing.T) {
	if _, err := ParseProxyURI("http://127.0.0.1:8080"); err == nil {
		t.Fatalf("expected error for unsupported proxy scheme")
	}
}
