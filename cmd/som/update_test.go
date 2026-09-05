package main

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aead.dev/minisign"
)

func TestAllowUnverified(t *testing.T) {
	cases := map[string]bool{
		"":        false,
		"0":       false,
		"false":   false,
		"no":      false,
		"1":       true,
		"true":    true,
		"yes":     true,
		"Y":       true,
		" 1 ":     true,
		"garbage": false,
	}
	for env, want := range cases {
		t.Setenv("SOM_ALLOW_UNVERIFIED", env)
		if got := allowUnverified(); got != want {
			t.Errorf("allowUnverified(%q) = %v, want %v", env, got, want)
		}
	}
}

func TestNewUpgradeVerifierEmptyKey(t *testing.T) {
	old := somMinisignPublicKey
	somMinisignPublicKey = ""
	defer func() { somMinisignPublicKey = old }()

	if _, err := newUpgradeVerifier("http://example.invalid/som.minisig"); err == nil {
		t.Fatal("expected error when no embedded public key")
	}
}

// testKeypair tạo một cặp key minisign dùng trong test.
func testKeypair(t *testing.T) (minisign.PublicKey, minisign.PrivateKey) {
	t.Helper()
	pub, priv, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return pub, priv
}

// serveMinisig phục vụ nội dung chữ ký tại / để test newUpgradeVerifier.
func serveMinisig(t *testing.T, body []byte, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = w.Write(body)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestNewUpgradeVerifierValidSignature(t *testing.T) {
	old := somMinisignPublicKey
	defer func() { somMinisignPublicKey = old }()

	pub, priv := testKeypair(t)
	pubText, err := pub.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	somMinisignPublicKey = string(pubText)

	bin := []byte("#!/bin/sh\necho fake som binary\n")
	sig := minisign.Sign(priv, bin)
	srv := serveMinisig(t, sig, http.StatusOK)

	v, err := newUpgradeVerifier(srv.URL)
	if err != nil {
		t.Fatalf("newUpgradeVerifier: %v", err)
	}
	if err := v.Verify(bin); err != nil {
		t.Fatalf("signature should verify: %v", err)
	}
}

func TestNewUpgradeVerifierWrongKey(t *testing.T) {
	old := somMinisignPublicKey
	defer func() { somMinisignPublicKey = old }()

	pub, _ := testKeypair(t)
	pubText, err := pub.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	somMinisignPublicKey = string(pubText)

	// Ký bằng key khác (không phải key nhúng trong binary).
	_, otherPriv := testKeypair(t)
	bin := []byte("fake som binary")
	sig := minisign.Sign(otherPriv, bin)
	srv := serveMinisig(t, sig, http.StatusOK)

	v, err := newUpgradeVerifier(srv.URL)
	if err != nil {
		t.Fatalf("newUpgradeVerifier: %v", err)
	}
	if err := v.Verify(bin); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected signature verification failure, got %v", err)
	}
}

func TestNewUpgradeVerifierMissingSignature(t *testing.T) {
	old := somMinisignPublicKey
	defer func() { somMinisignPublicKey = old }()

	pub, _ := testKeypair(t)
	pubText, _ := pub.MarshalText()
	somMinisignPublicKey = string(pubText)

	srv := serveMinisig(t, nil, http.StatusNotFound)
	if _, err := newUpgradeVerifier(srv.URL); err == nil {
		t.Fatal("expected error when signature asset is missing")
	}
}
