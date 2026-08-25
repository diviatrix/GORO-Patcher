//go:build !release

package engine

func VerifyManifestSignature(m *Manifest) bool {
	return true
}