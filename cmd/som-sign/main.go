package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"aead.dev/minisign"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "gen":
		gen()
	case "sign":
		if len(os.Args) < 3 {
			usage()
		}
		if err := sign(os.Args[2:]...); err != nil {
			fmt.Fprintln(os.Stderr, "sign:", err)
			os.Exit(1)
		}
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  som-sign gen")
	fmt.Fprintln(os.Stderr, "  SOM_MINISIGN_KEY=<base64 secret> som-sign sign <file>...")
	os.Exit(2)
}

func gen() {
	pub, priv, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		os.Exit(1)
	}

	pubText, err := pub.MarshalText()
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal pubkey:", err)
		os.Exit(1)
	}
	secret, err := minisign.EncryptKey("", priv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "encrypt secret key:", err)
		os.Exit(1)
	}

	fmt.Println("--- public key (paste into cmd/som/signkey.go) ---")
	fmt.Println(string(pubText))
	fmt.Println("--- secret key base64 (set as GitHub Actions secret SOM_MINISIGN_KEY) ---")
	fmt.Println(base64.StdEncoding.EncodeToString(secret))
}

func sign(files ...string) error {
	raw := os.Getenv("SOM_MINISIGN_KEY")
	if raw == "" {
		return fmt.Errorf("SOM_MINISIGN_KEY is empty")
	}
	keyBytes, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return fmt.Errorf("SOM_MINISIGN_KEY is not valid base64: %w", err)
	}
	priv, err := minisign.DecryptKey("", keyBytes)
	if err != nil {
		return fmt.Errorf("cannot decrypt secret key: %w", err)
	}

	for _, f := range files {
		bin, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		sig := minisign.Sign(priv, bin)
		if err := os.WriteFile(f+".minisig", sig, 0o644); err != nil {
			return err
		}
		fmt.Println("signed", f, "->", f+".minisig")
	}
	return nil
}
