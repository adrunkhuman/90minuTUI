package site

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
)

type Client struct {
	baseURL *url.URL
	http    *http.Client
}

func NewClient() *Client {
	base, _ := url.Parse(BaseURL)

	return &Client{
		baseURL: base,
		http: &http.Client{
			Timeout: HTTPTimeout,
		},
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

func (c *Client) Document(ctx context.Context, rawURL string) (*goquery.Document, error) {
	absoluteURL := c.Resolve(rawURL)

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

	doc, err := decodeAndParse(body, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("parse %q HTML: %w", absoluteURL, err)
	}

	return doc, nil
}

func decodeAndParse(body []byte, contentType string) (*goquery.Document, error) {
	encoding, _, _ := charset.DetermineEncoding(body, contentType)
	decodedReader := encoding.NewDecoder().Reader(bytes.NewReader(body))

	return goquery.NewDocumentFromReader(decodedReader)
}
