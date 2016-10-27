// Cryptography functions for financial data security at MarketX
// In public domain due to potential reusability.

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
)

// encryptBytes is the underlying private method to do AES encryption
// of bytes
func encryptBytes(key []byte, plaintext []byte) ([]byte, error) {
	// Create AES block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize + len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// CFB encrypt
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// Done
	return ciphertext, nil
}

// decryptBytes is the underlying private method to do AES decryption
// of bytes
func decryptBytes(key []byte, ciphertext []byte) ([]byte, error) {
	// Create AES block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// CFB decrypt
	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
	// But we should modify input arguments.
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	// Done
	return plaintext, nil
}

// EncryptString takes the usual AES-CFB encryption of a plaintext string
// and returns the encrypted text string in base64
func EncryptString(key []byte, pt string) (string, error) {
	ciphertext, err := encryptBytes(key, []byte(pt))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString takes the usual AES-CFB decryption of an encrypted string
// in base64 and returns the plaintext string
func DecryptString(key []byte, ct string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ct)
	if err != nil {
		return "", err
	}
	plaintext, err := decryptBytes(key, data)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptStream takes the usual AES-CFB encryption of plain input stream
// contents and writes the encrypted contents to output stream
// Caller handles stream closing
func EncryptStream(key []byte, in io.Reader, out io.Writer) error {
	plaintext, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	ciphertext, err := encryptBytes(key, plaintext)
	if err != nil {
		return err
	}

	_, err = out.Write(ciphertext)
	return err
}

// EncryptStreamBytes takes the usual AES-CFB encryption of plain input
// content bytes and writes the encrypted contents to output stream
func EncryptStreamBytes(key []byte, inb []byte, out io.Writer) error {
	ciphertext, err := encryptBytes(key, inb)
	if err != nil {
		return err
	}

	_, err = out.Write(ciphertext)
	return err
}

// DecryptStream takes the usual AES-CFB decryption of encrypted input stream
// contents and writes the plain contents to output stream
// Caller handles stream closing
func DecryptStream(key []byte, in io.Reader, out io.Writer) error {
	ciphertext, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	plaintext, err := decryptBytes(key, ciphertext)
	if err != nil {
		return err
	}

	_, err = out.Write(plaintext)
	return err
}

// DecryptStreamBytes takes the usual AES-CFB decryption of encrypted
// input stream contents and returns the plain content bytes directly
// Caller handles stream closing
func DecryptStreamBytes(key []byte, in io.Reader) ([]byte, error) {
	ciphertext, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	plaintext, err := decryptBytes(key, ciphertext)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
