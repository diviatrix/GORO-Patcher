package engine

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func sampleManifest() *Manifest {
	return &Manifest{
		PatchBaseURL:   "https://cdn.example.com/game",
		PatcherURL:     "https://cdn.example.com/patcher.exe",
		PatcherHash:    "abc",
		PatcherVersion: 2,
		PatcherSize:    123,
		Patches: []Patch{
			{ID: 1, Name: "a.grf", Hash: "hash-a", Type: "grf", Target: "myserver.grf"},
			{ID: 2, Name: "b.zip", Hash: "hash-b", Type: "raw"},
		},
	}
}

func TestSignAndVerifyRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	pubB64 := base64Of(pub)

	m := sampleManifest()
	sig, err := SignManifest(priv, m)
	if err != nil {
		t.Fatal(err)
	}
	m.Signature = sig

	if !VerifyManifestSignatureWithKey(m, pubB64) {
		t.Error("validly signed manifest failed verification")
	}
}

func TestSignVerifySurvivesRoundTrip(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)

	m := sampleManifest()
	sig, err := SignManifest(priv, m)
	if err != nil {
		t.Fatal(err)
	}
	m.Signature = sig

	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var reparsed Manifest
	if err := json.Unmarshal(raw, &reparsed); err != nil {
		t.Fatal(err)
	}
	if !VerifyManifestSignatureWithKey(&reparsed, base64Of(pub)) {
		t.Error("reparsed signed manifest failed verification")
	}
}

func TestVerifyRejectsTampering(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	pubB64 := base64Of(pub)

	m := sampleManifest()
	sig, _ := SignManifest(priv, m)
	m.Signature = sig

	m.Patches[0].Target = "evil.grf"
	if VerifyManifestSignatureWithKey(m, pubB64) {
		t.Error("tampered manifest verified — signature does not cover content")
	}
}

func TestVerifyFailsClosed(t *testing.T) {
	m := sampleManifest()

	if VerifyManifestSignatureWithKey(m, "") {
		t.Error("verification with empty public key must fail closed")
	}

	if VerifyManifestSignatureWithKey(m, "not-base64!!") {
		t.Error("verification with malformed public key must fail")
	}
}

func TestNeedsSelfUpdateVersionGate(t *testing.T) {
	cases := []struct {
		name string
		m    *Manifest
		want bool
	}{
		{"nil manifest", nil, false},
		{"no url", &Manifest{PatcherHash: "x", PatcherVersion: CurrentPatcherVersion + 1}, false},
		{"no hash", &Manifest{PatcherURL: "u", PatcherVersion: CurrentPatcherVersion + 1}, false},
		{"same version", &Manifest{PatcherURL: "u", PatcherHash: "h", PatcherVersion: CurrentPatcherVersion}, false},
		{"older version", &Manifest{PatcherURL: "u", PatcherHash: "h", PatcherVersion: CurrentPatcherVersion - 1}, false},
		{"zero version", &Manifest{PatcherURL: "u", PatcherHash: "h", PatcherVersion: 0}, false},
		{"newer version", &Manifest{PatcherURL: "u", PatcherHash: "h", PatcherVersion: CurrentPatcherVersion + 1}, true},
	}
	for _, tc := range cases {
		if got := NeedsSelfUpdate(tc.m); got != tc.want {
			t.Errorf("%s: NeedsSelfUpdate = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func base64Of(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}