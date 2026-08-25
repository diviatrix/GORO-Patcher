//go:build release

package engine

const manifestPublicKeyBase64 = ""

func VerifyManifestSignature(m *Manifest) bool {
	return VerifyManifestSignatureWithKey(m, manifestPublicKeyBase64)
}