//go:build release

package downloader

import (
	"fmt"
	"net/url"
)

// validateURL is the release-build policy: only https:// is acceptable for any
// content the patcher fetches (manifest, patch files, patcher binary, notes).
// Plain http, file, and bare local paths are rejected so a network attacker
// cannot downgrade the channel and serve tampered content. Verification of the
// manifest's Ed25519 signature is what guarantees authenticity on top of this.
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