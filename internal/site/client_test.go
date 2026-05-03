package site

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func (failingReadCloser) Close() error {
	return nil
}

func TestClientDocumentReportsRequestBuildFailure(t *testing.T) {
	base, err := url.Parse(BaseURL)
	if err != nil {
		t.Fatalf("parse base URL: %v", err)
	}
	client := &Client{baseURL: base, http: http.DefaultClient}

	_, err = client.Document(context.Background(), "http://[::1")
	if err == nil || !strings.Contains(err.Error(), "build request") {
		t.Fatalf("expected request-build error, got %v", err)
	}
}

func TestClientDocumentReportsFetchFailure(t *testing.T) {
	base, err := url.Parse(BaseURL)
	if err != nil {
		t.Fatalf("parse base URL: %v", err)
	}
	client := &Client{
		baseURL: base,
		http: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})},
	}

	_, err = client.Document(context.Background(), BaseURL)
	if err == nil || !strings.Contains(err.Error(), "fetch") {
		t.Fatalf("expected fetch error, got %v", err)
	}
}

func TestClientDocumentReportsUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	client := &Client{baseURL: base, http: server.Client()}

	_, err = client.Document(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "unexpected status 404") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestClientDocumentReportsBodyReadFailure(t *testing.T) {
	base, err := url.Parse(BaseURL)
	if err != nil {
		t.Fatalf("parse base URL: %v", err)
	}
	client := &Client{
		baseURL: base,
		http: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "200 OK",
				StatusCode: http.StatusOK,
				Body:       failingReadCloser{},
				Header:     make(http.Header),
			}, nil
		})},
	}

	_, err = client.Document(context.Background(), BaseURL)
	if err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("expected body-read error, got %v", err)
	}
}

func TestClientDocumentSendsUserAgent(t *testing.T) {
	var userAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent = r.UserAgent()
		_, _ = io.WriteString(w, `<html><body>ok</body></html>`)
	}))
	defer server.Close()

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	client := &Client{baseURL: base, http: server.Client()}

	if _, err := client.Document(context.Background(), server.URL); err != nil {
		t.Fatalf("expected document load to succeed, got %v", err)
	}
	if !strings.HasPrefix(userAgent, "90minuTUI/") {
		t.Fatalf("unexpected user agent: %q", userAgent)
	}
}
