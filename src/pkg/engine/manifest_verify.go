package engine

import "os"

const manifestKeyEnvVar = "GORO_PATCHER_PUBKEY"

func ResolveManifestPublicKey(cfgKey string) string {
	if env := os.Getenv(manifestKeyEnvVar); env != "" {
		return env
	}
	return cfgKey
}

func VerifyManifest(m *Manifest, pubKey string) bool {
	if pubKey == "" {
		return MissingKeyAllowsUnsigned()
	}
	return VerifyManifestSignatureWithKey(m, pubKey)
}