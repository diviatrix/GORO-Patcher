//go:build release

package engine

// manifestPublicKeyBase64 is the publisher's Ed25519 public key (base64) that
// signed manifests must verify against in a release build. The matching private
// key lives only with the publisher and is never embedded or shipped.
//
// Set this to the base64 public key produced by `hashfile genkey` before
// shipping a release. If it is left empty the release build rejects every
// manifest (fail-closed) rather than accepting unauthenticated content.
const manifestPublicKeyBase64 = ""

// VerifyManifestSignature is the strict, release-only check. A release build
// refuses any manifest that does not carry a valid signature under the embedded
// public key.
func VerifyManifestSignature(m *Manifest) bool {
	return VerifyManifestSignatureWithKey(m, manifestPublicKeyBase64)
}