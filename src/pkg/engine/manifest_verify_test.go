package engine

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func TestResolveManifestPublicKeyEnvOverridesConfig(t *testing.T) {
	t.Setenv(manifestKeyEnvVar, "env-key")
	if got := ResolveManifestPublicKey("cfg-key"); got != "env-key" {
		t.Errorf("ResolveManifestPublicKey = %q, want env override %q", got, "env-key")
	}
}

func TestResolveManifestPublicKeyFallsBackToConfig(t *testing.T) {
	t.Setenv(manifestKeyEnvVar, "")
	if got := ResolveManifestPublicKey("cfg-key"); got != "cfg-key" {
		t.Errorf("ResolveManifestPublicKey = %q, want config fallback %q", got, "cfg-key")
	}
}

func TestVerifyManifestWithConfiguredKey(t *testing.T) {
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
		t.Error("valid signature must verify against the configured key")
	}

	wrongKey := base64.StdEncoding.EncodeToString([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	if VerifyManifest(m, wrongKey) {
		t.Error("signature must fail against the wrong key")
	}
}
