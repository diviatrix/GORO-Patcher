//go:build !release

package engine

// VerifyManifestSignature reports whether m carries a signature valid under the
// embedded release public key. In a dev build (no `release` tag) no signing key
// is embedded, so every manifest is accepted — local, unsigned manifests keep
// working for testing. Release builds use the strict implementation in
// manifest_verify_release.go.
func VerifyManifestSignature(m *Manifest) bool {
	return true
}