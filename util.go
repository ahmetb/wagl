package main

import (
	"crypto/tls"
	"path/filepath"

	"github.com/miekg/dns"
	"wagl/tlsconfig"
)

// tlsConfig constructs a Docker TLS configuration using the certs in the
// specified directory.
func tlsConfig(certDir string, verify bool) (*tls.Config, error) {
	if certDir == "" {
		return nil, nil
	}

	return tlsconfig.Client(tlsconfig.Options{
		CAFile:             filepath.Join(certDir, "ca.pem"),
		CertFile:           filepath.Join(certDir, "cert.pem"),
		KeyFile:            filepath.Join(certDir, "key.pem"),
		InsecureSkipVerify: !verify,
	})
}

// localNameservers returns list of local nameservers.
func localNameservers() ([]string, error) {
	c, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}
	return c.Servers, nil
}
