//go:build !release

package downloader

// validateURL is the dev-build policy: allow the full set of schemes (http,
// https, file, and bare local paths) so local testing against a plain-HTTP or
// filesystem-backed feed keeps working. Release builds use
// scheme_release.go, which allows https only.
func validateURL(raw string) error {
	return nil
}