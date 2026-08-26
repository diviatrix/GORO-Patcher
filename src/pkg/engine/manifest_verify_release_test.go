//go:build release

package engine

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func TestVerifyManifestReleaseFailsClosedWithoutKey(t *testing.T) {
	if VerifyManifest(nil, "") {
		t.Error("release build must reject manifest when no key is configured")
	}
}

func TestVerifyManifestReleaseAcceptsConfiguredKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	m := &Manifest{Patches: []Patch{{ID: 1, Name: "one.patch"}}}
	sig, err := SignManifest(priv, m)
	if err != nil {
		t.Fatal(err)
	}
	m.Signature = sig

	key := base64.StdEncoding.EncodeToString(pub)
	if !VerifyManifest(m, key) {
		t.Error("release build must verify a valid signature against the configured key")
	}
}
