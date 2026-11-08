package main

import "testing"

func TestValidatePprofAddress(t *testing.T) {
	tests := []struct {
		name string
		addr string
		wantErr bool
	}{
		{name: "ipv4 loopback", addr: "127.0.0.1:6060"},
		{name: "ipv6 loopback", addr: "[::1]:6060"},
		{name: "localhost", addr: "localhost:6060"},
		{name: "wildcard", addr: ":6060", wantErr: true},
		{name: "public ip", addr: "0.0.0.0:6060", wantErr: true},
		{name: "lan ip", addr: "192.168.1.10:6060", wantErr: true},
		{name: "missing port", addr: "127.0.0.1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePprofAddress(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validatePprofAddress(%q) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			}
		})
	}
}
