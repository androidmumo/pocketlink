// fwtool produces a detached signed OTA manifest; it never prints private keys.
package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"github.com/androidmumo/pocketlink/services/relay/internal/firmware"
	"os"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	keyPath := flag.String("key", "", "signing private PEM outside repository")
	imagePath := flag.String("image", "", "application-only binary")
	version := flag.String("version", "", "release version")
	sequence := flag.Int64("sequence", 0, "strictly increasing release sequence")
	out := flag.String("out", "", "output directory")
	flag.Parse()
	if *keyPath == "" || *imagePath == "" || *out == "" {
		return fmt.Errorf("key, image and out are required")
	}
	raw, e := os.ReadFile(*keyPath)
	if e != nil {
		return e
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return firmware.ErrInvalid
	}
	parsed, e := x509.ParsePKCS8PrivateKey(block.Bytes)
	if e != nil {
		return e
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return firmware.ErrInvalid
	}
	trusted, e := firmware.PublicKey()
	if e != nil {
		return e
	}
	if !key.PublicKey.Equal(trusted) {
		return fmt.Errorf("private key does not match embedded trust key")
	}
	image, e := os.ReadFile(*imagePath)
	if e != nil {
		return e
	}
	hash := sha256.Sum256(image)
	metadata := firmware.Metadata{Format: 1, Board: "ai-passport-esp32c3", App: "pocketlink", Version: *version, Sequence: *sequence, Size: len(image), SHA256: hex.EncodeToString(hash[:])}
	if e = firmware.ValidateImage(metadata, image); e != nil {
		return e
	}
	payload, e := json.Marshal(metadata)
	if e != nil {
		return e
	}
	signedHash := sha256.Sum256(payload)
	signature, e := ecdsa.SignASN1(rand.Reader, key, signedHash[:])
	if e != nil {
		return e
	}
	envelope, e := json.Marshal(firmware.Envelope{Payload: base64.StdEncoding.EncodeToString(payload), Signature: base64.StdEncoding.EncodeToString(signature)})
	if e != nil {
		return e
	}
	if _, e = firmware.Parse(envelope); e != nil {
		return e
	}
	if e = os.MkdirAll(*out, 0755); e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(*out, "manifest.json"), envelope, 0644); e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(*out, "pocketlink-ota.bin"), image, 0644); e != nil {
		return e
	}
	fmt.Printf("Signed %s, sequence %d, sha256 %s\n", metadata.Version, metadata.Sequence, metadata.SHA256)
	return nil
}
