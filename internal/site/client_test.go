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

func TestClientDocumentCachesSuccessfulResponses(t *testing.T) {
	requests := 0
	body := `<html><body><p>cached</p></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	client := &Client{baseURL: base, http: server.Client(), cache: newDocumentCache(documentCacheLimit)}
	if client.Cached(server.URL) {
		t.Fatalf("expected URL to be uncached before first load")
	}

	for range 2 {
		if _, err := client.Document(context.Background(), server.URL); err != nil {
			t.Fatalf("expected document load to succeed, got %v", err)
		}
	}
	if !client.Cached(server.URL) {
		t.Fatalf("expected URL to be cached after successful load")
	}
	if requests != 1 {
		t.Fatalf("expected second document load to use cache, got %d requests", requests)
	}

	body = `<html><body><p>fresh</p></body></html>`
	doc, err := client.DocumentFresh(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("expected fresh document load to succeed, got %v", err)
	}
	if got := strings.TrimSpace(doc.Find("p").Text()); got != "fresh" {
		t.Fatalf("expected fresh document body, got %q", got)
	}
	doc, err = client.Document(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("expected cached fresh document load to succeed, got %v", err)
	}
	if got := strings.TrimSpace(doc.Find("p").Text()); got != "fresh" {
		t.Fatalf("expected fresh body to replace cached body, got %q", got)
	}
	if requests != 2 {
		t.Fatalf("expected fresh load to make one extra request, got %d requests", requests)
	}
}

func TestClientDocumentCacheEvictsOldestEntry(t *testing.T) {
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<html><body>`+strings.Repeat(r.URL.Path, 8)+`</body></html>`)
	}))
	defer server.Close()

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	client := &Client{baseURL: base, http: server.Client(), cache: newDocumentCache(80)}

	for _, path := range []string{"/one", "/two", "/one"} {
		if _, err := client.Document(context.Background(), server.URL+path); err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
	}
	if requests["/one"] != 2 {
		t.Fatalf("expected /one to be evicted and fetched again, got %d requests", requests["/one"])
	}
	if requests["/two"] != 1 {
		t.Fatalf("expected /two to stay cached, got %d requests", requests["/two"])
	}
}

func TestClientDocumentFreshOversizedResponseClearsCachedEntry(t *testing.T) {
	body := `<html><body><p>small</p></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	client := &Client{baseURL: base, http: server.Client(), cache: newDocumentCache(80)}

	if _, err := client.Document(context.Background(), server.URL); err != nil {
		t.Fatalf("expected initial document load to succeed, got %v", err)
	}
	if !client.Cached(server.URL) {
		t.Fatalf("expected small response to be cached")
	}

	body = `<html><body><p>` + strings.Repeat("large", 40) + `</p></body></html>`
	if _, err := client.DocumentFresh(context.Background(), server.URL); err != nil {
		t.Fatalf("expected oversized fresh document load to succeed, got %v", err)
	}
	if client.Cached(server.URL) {
		t.Fatalf("expected oversized fresh response to clear cached entry")
	}
}
