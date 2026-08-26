//go:build release

package downloader

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestReleaseCheckRedirectBlocksHTTP(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/redirected", nil)
	if err := checkRedirect(req, nil); err == nil {
		t.Error("release checkRedirect must reject a plain-http redirect target")
	}
}

func TestReleaseCheckRedirectAllowsHTTPS(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/redirected", nil)
	if err := checkRedirect(req, nil); err != nil {
		t.Errorf("release checkRedirect must allow an https redirect target: %v", err)
	}
}

func TestReleaseValidateURLBlocksInsecure(t *testing.T) {
	for _, url := range []string{"http://example.com/a.patch", "ftp://example.com/a.patch", "file:///tmp/a.patch"} {
		if err := validateURL(url); err == nil {
			t.Errorf("release validateURL must reject %q, got nil", url)
		}
	}
}

func TestReleaseValidateURLAllowsHTTPS(t *testing.T) {
	for _, url := range []string{"https://example.com/a.patch", "https://example.com/path?x=1#f"} {
		if err := validateURL(url); err != nil {
			t.Errorf("release validateURL must accept %q: %v", url, err)
		}
	}
}

func TestReleaseFetchBytesBlocksHTTP(t *testing.T) {
	dl := New(3)
	if _, err := dl.FetchBytes(context.Background(), "http://example.com/a.patch"); err == nil {
		t.Error("release FetchBytes must reject a plain-http URL before fetching")
	}
}

func TestReleaseFetchResourceBlocksHTTP(t *testing.T) {
	dl := New(3)
	url := "http://example.com/a.patch"
	err := dl.Fetch(context.Background(), url, t.TempDir()+"/out", nil)
	if err == nil {
		t.Error("release Fetch must reject a plain-http URL before fetching")
	}
}