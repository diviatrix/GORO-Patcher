//go:build !release

package engine

import "testing"

func TestVerifyManifestDevFailsOpenWithoutKey(t *testing.T) {
	if !VerifyManifest(nil, "") {
		t.Error("dev build must accept manifest when no key is configured")
	}
}