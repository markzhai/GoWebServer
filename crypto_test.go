// Testing for cryptography utilities
package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

func TestEncryptDecryptString(t *testing.T) {
	key := []byte("11111111111111111111111111111111")
	text := "The lazy programmer does something stupid."
	es, err := EncryptString(key, text)
	if err != nil {
		t.Fatalf("[Crypto] EncryptString failed: %v\n", err)
	}
	ds, err := DecryptString(key, es)
	if err != nil {
		t.Fatalf("[Crypto] DecryptString failed: %v\n", err)
	}
	if ds != text {
		t.Fatal("[Crypto] Decrypted string does not match original")
	}
}

func TestEncryptDecryptStream(t *testing.T) {
	// Setup necessary files
	in, err := os.Open("crypto.go")
	if err != nil {
		t.Fatalf("[Crypto] Cannot open crypto.go: %v\n", err)
	}
	defer in.Close()
	inb, err := os.Open("crypto.go")
	if err != nil {
		t.Fatalf("[Crypto] Cannot open crypto.go for bytes: %v\n", err)
	}
	defer inb.Close()
	oute, err := os.Create("test_crypto_enc")
	if err != nil {
		t.Fatalf("[Crypto] Cannot create temp file for enc: %v\n", err)
	}
	defer oute.Close()
	outeb, err := os.Create("test_crypto_enc_bytes")
	if err != nil {
		t.Fatalf("[Crypto] Cannot create temp file for enc of bytes: %v\n",
			err)
	}
	defer outeb.Close()
	outd, err := os.Create("test_crypto_dec")
	if err != nil {
		t.Fatalf("[Crypto] Cannot create temp file for dec: %v\n", err)
	}
	defer outd.Close()

	// Do encryption/decryption cycle
	key := []byte("11111111111111111111111111111111")
	err = EncryptStream(key, in, oute)
	if err != nil {
		t.Fatalf("[Crypto] EncryptStream failed: %v\n", err)
	}
	ine, err := os.Open("test_crypto_enc")
	if err != nil {
		t.Fatalf("[Crypto] Cannot read temp file for enc: %v\n", err)
	}
	defer ine.Close()
	err = DecryptStream(key, ine, outd)
	if err != nil {
		t.Fatalf("[Crypto] DecryptStream failed: %v\n", err)
	}
	inbb, err := ioutil.ReadAll(inb)
	if err != nil {
		t.Fatalf("[Crypto] EncryptStreamBytes failed reading: %v\n", err)
	}
	err = EncryptStreamBytes(key, inbb, outeb)
	if err != nil {
		t.Fatalf("[Crypto] EncryptStreamBytes failed: %v\n", err)
	}
	ineb, err := os.Open("test_crypto_enc_bytes")
	if err != nil {
		t.Fatalf("[Crypto] Cannot read temp file for enc bytes: %v\n", err)
	}
	defer ineb.Close()
	outb, err := DecryptStreamBytes(key, ineb)
	if err != nil {
		t.Fatalf("[Crypto] DecryptStreamBytes failed: %v\n", err)
	}

	// Compare final results
	fin, err := ioutil.ReadFile("crypto.go")
	if err != nil {
		t.Fatalf("[Crypto] Cannot read crypto.go result: %v\n", err)
	}
	fout, err := ioutil.ReadFile("test_crypto_dec")
	if err != nil {
		t.Fatalf("[Crypto] Cannot read temp file for dec result: %v\n", err)
	}
	if !bytes.Equal(fin, fout) {
		t.Fatal("[Crypto] Decrypted file does not match original")
	}
	if !bytes.Equal(fin, outb) {
		t.Fatal("[Crypto] Decrypted bytes does not match original")
	}
}
