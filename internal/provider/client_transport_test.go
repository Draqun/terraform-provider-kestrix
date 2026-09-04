package provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeTestCA writes a self-signed CA certificate so the CA bundle handling can
// be exercised without depending on the host trust store.
func writeTestCA(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "kestrix-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNewAPIClientRejectsRelativeEndpoint(t *testing.T) {
	if _, err := newAPIClient("localhost:8080", "", "", false, time.Second); err == nil {
		t.Fatal("expected absolute URL error")
	}
}

func TestNewAPIClientCAFileHandling(t *testing.T) {
	if _, err := newAPIClient("https://kestrix.test", "", filepath.Join(t.TempDir(), "absent.pem"), false, time.Second); err == nil {
		t.Fatal("expected missing CA file error")
	}

	garbage := filepath.Join(t.TempDir(), "garbage.pem")
	if err := os.WriteFile(garbage, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newAPIClient("https://kestrix.test", "", garbage, false, time.Second); err == nil {
		t.Fatal("expected empty CA bundle error")
	}

	client, err := newAPIClient("https://kestrix.test", "", writeTestCA(t), false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok || transport.TLSClientConfig.RootCAs == nil {
		t.Fatalf("CA bundle was not installed: %+v", client.httpClient.Transport)
	}
}

func TestNewAPIClientFallsBackToEmptyCertPool(t *testing.T) {
	original := systemCertPool
	systemCertPool = func() (*x509.CertPool, error) { return nil, errors.New("no system trust store") }
	t.Cleanup(func() { systemCertPool = original })

	client, err := newAPIClient("https://kestrix.test", "", writeTestCA(t), false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if client.httpClient.Transport.(*http.Transport).TLSClientConfig.RootCAs == nil {
		t.Fatal("expected the CA file to seed a fresh certificate pool")
	}
}

func TestAPIClientDoRejectsUnencodableInput(t *testing.T) {
	client, err := newAPIClient("http://kestrix.test", "", "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	err = client.do(context.Background(), http.MethodPost, "http://kestrix.test/api/projects", make(chan int), nil)
	if err == nil || !strings.Contains(err.Error(), "encode request") {
		t.Fatalf("encode error = %v", err)
	}
}

func TestAPIClientDoRejectsInvalidRequest(t *testing.T) {
	client, err := newAPIClient("http://kestrix.test", "", "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	err = client.do(context.Background(), "invalid method", "http://kestrix.test/api/projects", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "create request") {
		t.Fatalf("create request error = %v", err)
	}
}

func TestAPIClientReportsTruncatedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1024")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"demo"`))
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	}))
	defer server.Close()

	client, err := newAPIClient(server.URL, "", "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.getProject(context.Background(), "demo"); err == nil {
		t.Fatal("expected a read failure for a truncated response")
	}
}

func TestAPIClientDeleteIgnoresEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := newAPIClient(server.URL, "", "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.deleteProject(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
}
