package config

import "testing"

func TestServerConfigValidate(t *testing.T) {
	for _, test := range []struct {
		host    string
		wantErr bool
	}{
		{host: "127.0.0.1"},
		{host: "::1"},
		{host: "localhost"},
		{host: "0.0.0.0", wantErr: true},
		{host: "192.168.1.10", wantErr: true},
		{host: "example.com", wantErr: true},
	} {
		t.Run(test.host, func(t *testing.T) {
			err := (ServerConfig{Host: test.host, Port: "3847"}).Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestServerConfigAddr(t *testing.T) {
	for _, test := range []struct {
		host string
		want string
	}{
		{host: "127.0.0.1", want: "127.0.0.1:3847"},
		{host: "::1", want: "[::1]:3847"},
		{host: "localhost", want: "localhost:3847"},
	} {
		t.Run(test.host, func(t *testing.T) {
			if got := (ServerConfig{Host: test.host, Port: "3847"}).Addr(); got != test.want {
				t.Fatalf("Addr() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestAuthConfig(t *testing.T) {
	if (AuthConfig{Mode: "disabled"}).Enabled() {
		t.Fatal("disabled auth must not be enabled")
	}
	if !(AuthConfig{Mode: "token"}).Enabled() {
		t.Fatal("token auth must be enabled")
	}
	if err := (AuthConfig{Mode: "unknown"}).Validate(); err == nil {
		t.Fatal("unknown auth mode must fail validation")
	}
}
