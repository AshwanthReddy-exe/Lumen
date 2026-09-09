package hermes

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"
)

// Fake-server helpers intentionally stay in tests. The production adapter has
// no fake runtime, state store, or policy implementation.
type testServer struct {
	URL      string
	server   *http.Server
	listener net.Listener
}

func newTestServer(t *testing.T, handler http.Handler) *testServer {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &testServer{URL: "http://" + listener.Addr().String(), server: &http.Server{Handler: handler}, listener: listener}
	go func() { _ = srv.server.Serve(listener) }()
	t.Cleanup(func() { _ = srv.Close() })
	return srv
}

func (s *testServer) Close() error {
	if err := s.server.Close(); err != nil {
		return err
	}
	return s.listener.Close()
}

type testPKI struct {
	Roots       *x509.CertPool
	Server      tls.Certificate
	ServerLeaf  *x509.Certificate
	Client      tls.Certificate
	OtherClient tls.Certificate
}

func makeTestPKI(t *testing.T) testPKI {
	t.Helper()
	caCert, caKey := makeCA(t, "hermes-test-ca")
	otherCert, otherKey := makeCA(t, "other-test-ca")
	server, serverLeaf := makeSignedCertificate(t, caCert, caKey, "127.0.0.1", x509.ExtKeyUsageServerAuth)
	client, _ := makeSignedCertificate(t, caCert, caKey, "lumen-client", x509.ExtKeyUsageClientAuth)
	otherClient, _ := makeSignedCertificate(t, otherCert, otherKey, "wrong-client", x509.ExtKeyUsageClientAuth)
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	return testPKI{Roots: roots, Server: server, ServerLeaf: serverLeaf, Client: client, OtherClient: otherClient}
}

func makeCA(t *testing.T, commonName string) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: serialNumber(t), Subject: pkix.Name{CommonName: commonName}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert, key
}

func makeSignedCertificate(t *testing.T, parent *x509.Certificate, parentKey *ecdsa.PrivateKey, name string, usage x509.ExtKeyUsage) (tls.Certificate, *x509.Certificate) {
	return makeSignedCertificateWindow(t, parent, parentKey, name, usage, time.Now().Add(-time.Minute), time.Now().Add(time.Hour))
}

func makeSignedCertificateWindow(t *testing.T, parent *x509.Certificate, parentKey *ecdsa.PrivateKey, name string, usage x509.ExtKeyUsage, notBefore, notAfter time.Time) (tls.Certificate, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: serialNumber(t), Subject: pkix.Name{CommonName: name}, DNSNames: []string{name}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, NotBefore: notBefore, NotAfter: notAfter, ExtKeyUsage: []x509.ExtKeyUsage{usage}, KeyUsage: x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, &key.PublicKey, parentKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return pair, cert
}

func serialNumber(t *testing.T) *big.Int {
	t.Helper()
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	if err != nil {
		t.Fatal(err)
	}
	return serial
}

func newMutualTLSServer(t *testing.T, pki testPKI, handler http.Handler) *testServer {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	tlsListener := tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{pki.Server}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pki.Roots, MinVersion: tls.VersionTLS13})
	srv := &testServer{URL: "https://" + listener.Addr().String(), server: &http.Server{Handler: handler}, listener: tlsListener}
	go func() { _ = srv.server.Serve(tlsListener) }()
	t.Cleanup(func() { _ = srv.Close() })
	return srv
}
