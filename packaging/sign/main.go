package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "sign:", err)
		os.Exit(1)
	}
}

func run(files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("usage: go run ./packaging/sign <files...>")
	}

	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(os.Getenv("UPDATE_SIGNING_KEY")))

	if err != nil || len(seed) != ed25519.SeedSize {
		return fmt.Errorf("UPDATE_SIGNING_KEY must hold a base64 ed25519 seed")
	}

	var sums strings.Builder

	for _, f := range files {
		sum, err := hash(f)

		if err != nil {
			return err
		}

		fmt.Fprintf(&sums, "%s  %s\n", sum, filepath.Base(f))
	}

	dir := filepath.Dir(files[0])
	body := []byte(sums.String())
	sig := ed25519.Sign(ed25519.NewKeyFromSeed(seed), body)

	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), body, 0o644); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "SHA256SUMS.sig"), []byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644)
}

func hash(path string) (string, error) {
	f, err := os.Open(path)

	if err != nil {
		return "", err
	}

	defer f.Close()

	h := sha256.New()

	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
