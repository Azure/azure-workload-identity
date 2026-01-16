package main

import (
	"crypto/tls"
	"reflect"
	"testing"
)

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    uint16
		wantErr bool
	}{
		{
			name:    "TLS 1.0",
			version: "1.0",
			want:    tls.VersionTLS10,
			wantErr: false,
		},
		{
			name:    "VersionTLS10",
			version: "VersionTLS10",
			want:    tls.VersionTLS10,
			wantErr: false,
		},
		{
			name:    "TLS 1.1",
			version: "1.1",
			want:    tls.VersionTLS11,
			wantErr: false,
		},
		{
			name:    "VersionTLS11",
			version: "VersionTLS11",
			want:    tls.VersionTLS11,
			wantErr: false,
		},
		{
			name:    "TLS 1.2",
			version: "1.2",
			want:    tls.VersionTLS12,
			wantErr: false,
		},
		{
			name:    "VersionTLS12",
			version: "VersionTLS12",
			want:    tls.VersionTLS12,
			wantErr: false,
		},
		{
			name:    "TLS 1.3",
			version: "1.3",
			want:    tls.VersionTLS13,
			wantErr: false,
		},
		{
			name:    "VersionTLS13",
			version: "VersionTLS13",
			want:    tls.VersionTLS13,
			wantErr: false,
		},
		{
			name:    "Invalid version",
			version: "1.4",
			want:    0,
			wantErr: true,
		},
		{
			name:    "Empty version",
			version: "",
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTLSVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTLSVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseTLSVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTLSCipherSuites(t *testing.T) {
	tests := []struct {
		name         string
		cipherSuites string
		want         []uint16
		wantErr      bool
	}{
		{
			name:         "Empty cipher suites",
			cipherSuites: "",
			want:         nil,
			wantErr:      false,
		},
		{
			name:         "Valid cipher suite",
			cipherSuites: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			want:         []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
			wantErr:      false,
		},
		{
			name:         "Multiple valid cipher suites",
			cipherSuites: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			want:         []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
			wantErr:      false,
		},
		{
			name:         "Valid cipher suites with spaces",
			cipherSuites: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			want:         []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
			wantErr:      false,
		},
		{
			name:         "Invalid cipher suite",
			cipherSuites: "INVALID_CIPHER",
			want:         nil,
			wantErr:      true,
		},
		{
			name:         "Mixed valid and invalid cipher suites",
			cipherSuites: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,INVALID_CIPHER",
			want:         nil,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTLSCipherSuites(tt.cipherSuites)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTLSCipherSuites() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTLSCipherSuites() = %v, want %v", got, tt.want)
			}
		})
	}
}
