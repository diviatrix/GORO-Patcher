package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"os"

	"github.com/diviatrix/GORO-Patcher/pkg/downloader"
	"github.com/diviatrix/GORO-Patcher/pkg/engine"
)

func main() {
	if len(os.Args) < 2 || (len(os.Args[1]) == 0 || os.Args[1][0] != '-') && !isSubcommand(os.Args[1]) {
		hashFilesSHA256(os.Args[1:])
		return
	}

	switch sub := os.Args[1]; sub {
	case "sha256":
		hashFilesSHA256(os.Args[2:])
	case "genkey":
		genkey(os.Args[2:])
	case "sign":
		sign(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n\n", sub)
		usage()
		os.Exit(2)
	}
}

func isSubcommand(arg string) bool {
	switch arg {
	case "sha256", "genkey", "sign", "verify":
		return true
	}
	return false
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: hashfile [COMMAND] [ARGS]

No command (default):
  hashfile <file>...          Print SHA-256 + size (patch-file integrity)

Commands:
  hashfile sha256 <file>...   Print SHA-256 + size (alias of the default)
  hashfile genkey             Generate an Ed25519 keypair (release signing)
      -out key.pem            Private key (PKCS#8 PEM) [default: key.pem]
      -pub pub.pem            Public key (PKIX PEM) [default: pub.pem]
  hashfile sign               Sign a manifest plist.json
      -key key.pem            Private key (from genkey)
      -in plist.json          Unsigned manifest [default: plist.json]
      -out plist.signed.json  Signed output [default: -in with signature injected]
  hashfile verify             Check a manifest's signature
      -key pub.pem            Public key PEM
      -in plist.json          Manifest to verify [default: plist.json]

The private key is the publisher's secret — keep it off-repo and never ship it.
genkey also prints the base64 public key: set it as the manifest_public_key field in
goro-config.json, or via the GORO_PATCHER_PUBKEY environment variable.
`)
}

func hashFilesSHA256(paths []string) {
	if len(paths) == 0 {
		usage()
		os.Exit(2)
	}
	for _, path := range paths {
		sum, size, err := sha256File(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			continue
		}
		fmt.Printf("%s  %d  %s\n", sum, size, path)
	}
}

func sha256File(path string) (sum string, size int64, err error) {
	sum, err = downloader.HashFile(path)
	if err != nil {
		return "", 0, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	return sum, info.Size(), nil
}

func genkey(args []string) {
	fs := flag.NewFlagSet("genkey", flag.ExitOnError)
	keyOut := fs.String("out", "key.pem", "private key output")
	pubOut := fs.String("pub", "pub.pem", "public key output")
	fs.Parse(args)

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		fatal("generate key: %v", err)
	}

	writeKey(*keyOut, "PRIVATE KEY", priv)
	writeKey(*pubOut, "PUBLIC KEY", pub)

	pubB64 := base64.StdEncoding.EncodeToString(pub)
	fmt.Printf("wrote %s (private) and %s (public)\n", *keyOut, *pubOut)
	fmt.Println("release public key (set as manifest_public_key in goro-config.json,")
	fmt.Println("or via the GORO_PATCHER_PUBKEY environment variable):")
	fmt.Println(pubB64)
}

func writeKey(path, blockType string, key any) {
	var der []byte
	var err error
	if blockType == "PRIVATE KEY" {
		der, err = x509.MarshalPKCS8PrivateKey(key)
	} else {
		der, err = x509.MarshalPKIXPublicKey(key)
	}
	if err != nil {
		fatal("marshal %s: %v", blockType, err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	if err := os.WriteFile(path, pemData, 0600); err != nil {
		fatal("write %s: %v", path, err)
	}
}

func loadPrivateKey(path string) ed25519.PrivateKey {
	data, err := os.ReadFile(path)
	if err != nil {
		fatal("read key %s: %v", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		fatal("%s: no PEM block found", path)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		fatal("parse private key %s: %v", path, err)
	}
	ed, ok := key.(ed25519.PrivateKey)
	if !ok {
		fatal("%s: not an Ed25519 private key", path)
	}
	return ed
}

func loadPublicKeyBase64(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		fatal("read key %s: %v", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		fatal("%s: no PEM block found", path)
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		fatal("parse public key %s: %v", path, err)
	}
	ed, ok := key.(ed25519.PublicKey)
	if !ok {
		fatal("%s: not an Ed25519 public key", path)
	}
	return base64.StdEncoding.EncodeToString(ed)
}

func sign(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	keyPath := fs.String("key", "", "private key PEM (required)")
	inPath := fs.String("in", "plist.json", "manifest to sign")
	outPath := fs.String("out", "", "signed output; defaults to -in (rewritten)")
	fs.Parse(args)

	if *keyPath == "" {
		usage()
		fatal("sign: -key is required")
	}

	priv := loadPrivateKey(*keyPath)

	data, err := os.ReadFile(*inPath)
	if err != nil {
		fatal("read manifest %s: %v", *inPath, err)
	}
	m, err := engine.ParseManifest(data)
	if err != nil {
		fatal("parse manifest: %v", err)
	}

	sig, err := engine.SignManifest(priv, m)
	if err != nil {
		fatal("sign manifest: %v", err)
	}
	m.Signature = sig

	out := *outPath
	if out == "" {
		out = *inPath
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fatal("marshal signed manifest: %v", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(out, raw, 0644); err != nil {
		fatal("write %s: %v", out, err)
	}
	fmt.Printf("signed %s -> %s\n", *inPath, out)
}

func verify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	keyPath := fs.String("key", "", "public key PEM (required)")
	inPath := fs.String("in", "plist.json", "manifest to verify")
	fs.Parse(args)

	if *keyPath == "" {
		usage()
		fatal("verify: -key is required")
	}

	data, err := os.ReadFile(*inPath)
	if err != nil {
		fatal("read manifest %s: %v", *inPath, err)
	}
	m, err := engine.ParseManifest(data)
	if err != nil {
		fatal("parse manifest: %v", err)
	}

	pubB64 := loadPublicKeyBase64(*keyPath)
	if engine.VerifyManifestSignatureWithKey(m, pubB64) {
		fmt.Println("signature OK")
	} else {
		fmt.Fprintln(os.Stderr, "signature INVALID")
		os.Exit(1)
	}
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "hashfile: "+format+"\n", args...)
	os.Exit(1)
}
