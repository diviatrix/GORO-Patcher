//go:build release

package downloader

import (
	"fmt"
	"net/url"
)

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("blocked: release builds only allow https:// URLs, got %q", raw)
	}
	return nil
}