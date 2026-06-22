package site

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
)

// documentCacheLimit caps the per-process FIFO HTML response cache.
const documentCacheLimit = 20 * 1024 * 1024

type Client struct {
	baseURL *url.URL
	http    *http.Client
	cache   *documentCache
}

type cachedDocument struct {
	body        []byte
	contentType string
}

type documentCache struct {
	maxBytes int
	bytes    int
	entries  map[string]cachedDocument
	order    []string
	mu       sync.Mutex
}

func NewClient() *Client {
	base, _ := url.Parse(BaseURL)

	return &Client{
		baseURL: base,
		http: &http.Client{
			Timeout: HTTPTimeout,
		},
		cache: newDocumentCache(documentCacheLimit),
	}
}

func (c *Client) Resolve(raw string) string {
	if raw == "" {
		return c.baseURL.String()
	}

	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if u.IsAbs() {
		return u.String()
	}

	return c.baseURL.ResolveReference(u).String()
}

func (c *Client) Cached(rawURL string) bool {
	if c.cache == nil {
		return false
	}
	_, _, ok := c.cache.get(c.Resolve(rawURL))
	return ok
}

// Document resolves rawURL, sends the project User-Agent, decodes legacy HTML charsets,
// and caches successful responses in-process. Cached responses are reparsed on each call.
// Request-build, fetch/status, body-read, and parse failures include URL context.
func (c *Client) Document(ctx context.Context, rawURL string) (*goquery.Document, error) {
	return c.document(ctx, rawURL, true)
}

// DocumentFresh bypasses the in-process cache, fetches rawURL, and replaces any cached response.
func (c *Client) DocumentFresh(ctx context.Context, rawURL string) (*goquery.Document, error) {
	return c.document(ctx, rawURL, false)
}

func (c *Client) document(ctx context.Context, rawURL string, useCache bool) (*goquery.Document, error) {
	absoluteURL := c.Resolve(rawURL)
	if useCache {
		if body, contentType, ok := c.cachedDocument(absoluteURL); ok {
			doc, err := decodeAndParse(body, contentType)
			if err != nil {
				return nil, fmt.Errorf("parse cached %q HTML: %w", absoluteURL, err)
			}
			return doc, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, absoluteURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %q: %w", absoluteURL, err)
	}

	req.Header.Set("User-Agent", "90minuTUI/0.1 (+https://github.com/adrunkhuman/90minuTUI)")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", absoluteURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %q: unexpected status %s", absoluteURL, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %q body: %w", absoluteURL, err)
	}

	contentType := resp.Header.Get("Content-Type")
	doc, err := decodeAndParse(body, contentType)
	if err != nil {
		return nil, fmt.Errorf("parse %q HTML: %w", absoluteURL, err)
	}
	c.storeDocument(absoluteURL, body, contentType)

	return doc, nil
}

func (c *Client) cachedDocument(url string) ([]byte, string, bool) {
	if c.cache == nil {
		return nil, "", false
	}
	return c.cache.get(url)
}

func (c *Client) storeDocument(url string, body []byte, contentType string) {
	if c.cache == nil {
		return
	}
	c.cache.put(url, body, contentType)
}

func newDocumentCache(maxBytes int) *documentCache {
	return &documentCache{
		maxBytes: maxBytes,
		entries:  make(map[string]cachedDocument),
	}
}

func (c *documentCache) get(url string) ([]byte, string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[url]
	return entry.body, entry.contentType, ok
}

func (c *documentCache) put(url string, body []byte, contentType string) {
	if c.maxBytes <= 0 || len(body) > c.maxBytes {
		c.delete(url)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.entries[url]; ok {
		c.bytes -= len(existing.body)
	} else {
		c.order = append(c.order, url)
	}

	c.entries[url] = cachedDocument{body: bytes.Clone(body), contentType: contentType}
	c.bytes += len(body)

	for c.bytes > c.maxBytes && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		entry, ok := c.entries[oldest]
		if !ok {
			continue
		}
		delete(c.entries, oldest)
		c.bytes -= len(entry.body)
	}
}

func (c *documentCache) delete(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[url]
	if !ok {
		return
	}
	delete(c.entries, url)
	c.bytes -= len(entry.body)
	for i, cachedURL := range c.order {
		if cachedURL == url {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}

func decodeAndParse(body []byte, contentType string) (*goquery.Document, error) {
	encoding, _, _ := charset.DetermineEncoding(body, contentType)
	decodedReader := encoding.NewDecoder().Reader(bytes.NewReader(body))

	return goquery.NewDocumentFromReader(decodedReader)
}
