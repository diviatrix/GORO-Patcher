package engine

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func canonicalManifest(m *Manifest) ([]byte, error) {
	clone := *m
	clone.Signature = ""
	raw, err := json.Marshal(&clone)
	if err != nil {
		return nil, fmt.Errorf("canonicalize manifest: %w", err)
	}
	return raw, nil
}

func SignManifest(priv ed25519.PrivateKey, m *Manifest) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid ed25519 private key length %d", len(priv))
	}
	canonical, err := canonicalManifest(m)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(priv, canonical)
	return base64.StdEncoding.EncodeToString(sig), nil
}

func VerifyManifestSignatureWithKey(m *Manifest, pubKeyBase64 string) bool {
	if m == nil || pubKeyBase64 == "" {
		return false
	}
	pub, err := base64.StdEncoding.DecodeString(pubKeyBase64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	canonical, err := canonicalManifest(m)
	if err != nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pub), canonical, sig)
}
