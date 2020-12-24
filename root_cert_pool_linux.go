package main

import (
	"crypto/x509"
)

func rootCertPool() (*x509.CertPool, error) {
	return x509.SystemCertPool()
}
