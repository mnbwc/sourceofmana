package main

// TLS support for the Go game server's WebSocket endpoint.
//
// Browser clients (web export) must connect via wss:// (secure WebSocket)
// because browsers block mixed content and require secure contexts for
// many APIs. The Go server therefore exposes two endpoints:
//
//   - plain ws://  on WS_PORT   (default 27012) — for native Godot clients
//   - secure wss:// on WSS_PORT (default 27013) — for browser clients
//
// Both endpoints serve the same /ws handler. TLS certificates come from
// the standard certificate files (cert.pem / key.pem) in the working
// directory; if they are absent a self-signed pair is generated in-memory
// so local development works out of the box.
//
// In production, set TLS_CERT_FILE and TLS_KEY_FILE to a real certificate
// (e.g. Let's Encrypt) instead of relying on the self-signed fallback.

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

// loadTLSConfig builds a *tls.Config for the WSS listener. It prefers the
// certificate files named by TLS_CERT_FILE / TLS_KEY_FILE, falling back to a
// freshly generated self-signed certificate (valid 365 days) so local
// development needs no setup.
func loadTLSConfig() *tls.Config {
	certFile := envOr("TLS_CERT_FILE", "cert.pem")
	keyFile := envOr("TLS_KEY_FILE", "key.pem")

	var cert tls.Certificate
	var err error

	if fileExists(certFile) && fileExists(keyFile) {
		cert, err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Printf("tls: loading %s/%s failed (%v), falling back to self-signed", certFile, keyFile, err)
		} else {
			log.Printf("tls: using certificate %s", certFile)
		}
	}

	if cert.Certificate == nil {
		cert, err = selfSignedCert()
		if err != nil {
			log.Fatalf("tls: generating self-signed certificate: %v", err)
		}
		log.Printf("tls: using self-signed certificate (set TLS_CERT_FILE/TLS_KEY_FILE for production)")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		// WebSocket over TLS: modern browsers require TLS 1.2+.
		MinVersion: tls.VersionTLS12,
	}
}

// selfSignedCert generates an ECDSA P-256 self-signed certificate valid for
// 365 days. The certificate covers localhost and 127.0.0.1 so both the
// loopback hostnames work in development.
func selfSignedCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Source of Mana"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "som.manasource.org"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}

// fileExists reports whether the named file exists.
func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envInt reads an integer environment variable, returning fallback when unset
// or invalid.
func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n <= 0 {
		return fallback
	}
	return n
}
